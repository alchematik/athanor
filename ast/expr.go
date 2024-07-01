package ast

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Expr struct {
	Type  string
	Value any
}

func (e Expr) IsEmpty() bool {
	return e.Type == "" && e.Value == nil
}

func (e *Expr) UnmarshalJSON(data []byte) error {
	var inner struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}

	if err := json.Unmarshal(data, &inner); err != nil {
		return err
	}

	switch inner.Type {
	case "":
		return errors.New("must specify expression type")
	case "string":
		value := &StringLiteral{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "bool":
		value := &BoolLiteral{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "integer":
		value := &IntegerLiteral{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "map":
		value := &MapCollection{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "resource":
		value := &Resource{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "provider":
		value := &Provider{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "environment":
		value := &Environment{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "local_file":
		value := &LocalFile{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "get_environment":
		value := &GetEnvironment{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	case "get_resource":
		value := &GetResource{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}
		e.Value = *value
	default:
		return fmt.Errorf("unsupported expression type: %q", inner.Type)
	}

	e.Type = inner.Type

	return nil
}

type BoolLiteral struct {
	Value bool `json:"bool_literal"`
}

type StringLiteral struct {
	Value string `json:"string_literal"`
}

type IntegerLiteral struct {
	Value int `json:"integer_literal"`
}

type MapCollection struct {
	Value map[string]Expr `json:"map_collection"`
}

type Resource struct {
	Type       Expr `json:"type"`
	Provider   Expr `json:"provider"`
	Identifier Expr `json:"identifier"`
	Config     Expr `json:"config"`
}

type Provider struct {
	Name    Expr `json:"name"`
	Version Expr `json:"version"`
}

type Environment struct {
}

type LocalFile struct {
	Path Expr
}

type LocalFileSource struct {
	File Expr
}

type GetResource struct {
	Name string `json:"name"`
	From Expr   `json:"from"`
}

type GetEnvironment struct{}
