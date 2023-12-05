package main

import (
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

var DefaultYamlTypes = map[string]any{
	"int": 0, "int16": 0, "int32": 0, "int64": 0,
	"string": "",
}

type YamlMarshaler struct {
	Indent     int
	commentMap yaml.CommentMap
}

func NewYamlMarshaler(indent int) *YamlMarshaler {
	return &YamlMarshaler{
		Indent:     indent,
		commentMap: make(map[string][]*yaml.Comment),
	}
}

func (y *YamlMarshaler) Marshal(w io.Writer, baseField *Field) error {
	baseField.Name = "$"

	item, err := y.MarshalYAML(baseField, "")
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(w, yaml.Indent(2), yaml.WithComment(y.commentMap))
	defer encoder.Close()

	ms, ok := item.Value.(yaml.MapSlice)
	if !ok {
		return err
	}

	err = encoder.Encode(ms)
	if err != nil {
		panic(err)
	}

	return nil
}

func (y *YamlMarshaler) MarshalYAML(field *Field, yamlPath string) (yaml.MapItem, error) {
	key := field.Name
	if field.Alias != "" {
		key = field.Alias
	}

	if yamlPath == "" {
		yamlPath = key
	} else {
		yamlPath = yamlPath + "." + key
	}

	if field.Comment != "" {
		comment := []*yaml.Comment{yaml.LineComment(" " + field.Comment)}
		y.commentMap[yamlPath] = comment
	}

	var value any
	var ok bool

	if field.IsStruct {
		value := yaml.MapSlice{}

		for _, v := range field.Fields {
			item, err := y.MarshalYAML(v, yamlPath)
			if err != nil {
				return yaml.MapItem{}, err
			}

			value = append(value, item)
		}

		return yaml.MapItem{
			Key:   key,
			Value: value,
		}, nil
	}

	value, ok = DefaultYamlTypes[field.Value]
	if !ok {
		return yaml.MapItem{}, fmt.Errorf("cannot get default type of %s", field.Value)
	}

	item := yaml.MapItem{
		Key:   key,
		Value: value,
	}

	return item, nil
}
