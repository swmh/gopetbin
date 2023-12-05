package main

import (
	"fmt"
	"io"
	"strings"
)

var DefaultEnvTypes = map[string]any{
	"int": 0, "int16": 0, "int32": 0, "int64": 0,
	"string": "string",
}

type EnvMarshaler struct{}

func (e *EnvMarshaler) Marshal(w io.Writer, baseField *Field) error {
	baseField.Name = ""

	item, err := e.MarshalEnv(baseField, "")
	if err != nil {
		return err
	}

	w.Write([]byte(item))

	return nil
}

func (e *EnvMarshaler) MarshalEnv(field *Field, prefix string) (string, error) {
	key := field.Name
	if field.Alias != "" {
		key = field.Alias
	}

	key = strings.ToUpper(key)

	var value any
	var ok bool

	if field.IsStruct {
		bu := strings.Builder{}

		if prefix == "" && key != "" {
			prefix = key + "_"
		} else if key != "" {
			prefix = prefix + "_" + key
		}

		for _, v := range field.Fields {
			item, err := e.MarshalEnv(v, prefix)
			if err != nil {
				return "", err
			}

			bu.WriteString(item + "\n")
		}

		return bu.String(), nil
	}

	value, ok = DefaultEnvTypes[field.Value]
	if !ok {
		return "", fmt.Errorf("cannot get default type of %s", field.Value)
	}

	return fmt.Sprintf("%s%s=%v", prefix, key, value), nil
}
