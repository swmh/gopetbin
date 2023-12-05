package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"reflect"
	"strings"
)

type Field struct {
	Name     string
	Alias    string
	IsStruct bool
	Fields   []*Field
	Value    string
	Comment  string
}

type Marshaler interface {
	Marshal(w io.Writer, baseField *Field) error
}

type Parser struct {
	fset      *token.FileSet
	file      *ast.File
	callLine  int
	baseField *Field
}

func NewParser(path string, line int) (*Parser, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("cannot parse file: %w", err)
	}

	return &Parser{
		fset:     fset,
		file:     file,
		callLine: line,
	}, nil
}

func (p *Parser) getLine(pos token.Pos) int {
	return p.fset.Position(pos).Line
}

func (p *Parser) getTargetLine(comments *ast.CommentGroup) (int, error) {
	var targetLine int

	for _, v := range comments.List {
		line := p.getLine(v.Pos())
		if line == p.callLine {
			targetLine = p.getLine(comments.List[len(comments.List)-1].Pos()) + 1
		}
	}

	if targetLine == 0 {
		return 0, errors.New("cannot find")
	}

	return targetLine, nil
}

func (p *Parser) getTargetStruct(decl *ast.GenDecl, line int) (*ast.StructType, *Field, error) {
	var target *ast.StructType
	var name string

	ast.Inspect(decl, func(node ast.Node) bool {
		targetSpec, ok := node.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if p.getLine(targetSpec.Pos()) != line {
			return true
		}

		name = targetSpec.Name.Name

		target, ok = targetSpec.Type.(*ast.StructType)
		return !ok
	})

	if target == nil {
		return target, nil, errors.New("cannot find target struct")
	}

	field := &Field{
		Name:     name,
		Alias:    "",
		IsStruct: true,
		Fields:   []*Field{},
		Value:    "",
	}

	return target, field, nil
}

func (p *Parser) getGenDecl() (*ast.GenDecl, int, error) {
	var targetLine int
	var genDecl *ast.GenDecl

	ast.Inspect(p.file, func(node ast.Node) bool {
		s, ok := node.(*ast.GenDecl)
		if !ok {
			return true
		}

		if s.Tok != token.TYPE {
			return true
		}

		genDecl = s

		ast.Inspect(s, func(node ast.Node) bool {
			comments, ok := node.(*ast.CommentGroup)
			if !ok {
				return true
			}

			line, err := p.getTargetLine(comments)
			if err != nil {
				return true
			}

			targetLine = line
			return false
		})

		return false
	})

	if targetLine == 0 {
		return nil, 0, errors.New("cannot find target line")
	}

	if genDecl == nil {
		return nil, 0, errors.New("cannot find genDecl")
	}

	return genDecl, targetLine, nil
}

func (p *Parser) ParseField(v *ast.Field) (*Field, error) {
	name := v.Names[0].Name

	var comment string
	if v.Comment != nil {
		comment = strings.TrimSpace(v.Comment.Text())
	}

	var tag string
	if v.Tag != nil {
		tag = reflect.StructTag(strings.Trim(v.Tag.Value, "`")).Get("mapstructure")
	}

	var field *Field

	switch f := v.Type.(type) {
	case *ast.StructType:
		field = &Field{
			Name:     name,
			Alias:    tag,
			IsStruct: true,
			Fields:   []*Field{},
			Value:    "",
			Comment:  comment,
		}

		for _, df := range f.Fields.List {
			rf, err := p.ParseField(df)
			if err != nil {
				return nil, err
			}

			field.Fields = append(field.Fields, rf)
		}

	case *ast.Ident:
		field = &Field{
			Name:     name,
			Alias:    tag,
			IsStruct: false,
			Fields:   []*Field{},
			Value:    f.Name,
			Comment:  comment,
		}

	default:
		return nil, errors.New("cannot parse")
	}

	return field, nil
}

func (p *Parser) Parse() error {
	genDecl, targetLine, err := p.getGenDecl()
	if err != nil {
		return err
	}

	st, baseField, err := p.getTargetStruct(genDecl, targetLine)
	if err != nil {
		return err
	}

	for _, v := range st.Fields.List {
		f, err := p.ParseField(v)
		if err != nil {
			return err
		}

		baseField.Fields = append(baseField.Fields, f)
	}

	p.baseField = baseField

	return nil
}

func (p *Parser) Marshal(w io.Writer, marshaler Marshaler) error {
	return marshaler.Marshal(w, p.baseField)
}
