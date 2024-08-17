package ast

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Blueprint struct {
	Stmts []Stmt `json:"stmts"`
}

type Stmt struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

func (s *Stmt) UnmarshalJSON(data []byte) error {
	var inner struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}

	if err := json.Unmarshal(data, &inner); err != nil {
		return err
	}

	switch inner.Type {
	case "":
		return errors.New("must specify statement type")
	case "resource":
		value := &DeclareResource{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}

		s.Value = *value
	case "build":
		value := &DeclareBuild{}
		if err := json.Unmarshal(inner.Value, &value); err != nil {
			return err
		}

		s.Value = *value
	default:
		return fmt.Errorf("unsupported statement type: %q", inner.Type)
	}

	s.Type = inner.Type

	return nil
}

type DeclareResource struct {
	Name       string `json:"name"`
	Exists     Expr   `json:"exists"`
	Type       Expr   `json:"type"`
	Provider   Expr   `json:"provider"`
	Identifier Expr   `json:"identifier"`
	Config     Expr   `json:"config"`
}

type DeclareBuild struct {
	Name            string          `json:"name"`
	Exists          Expr            `json:"exists"`
	Input           map[string]any  `json:"input"`
	Runtimeinput    Expr            `json:"runtime_input"`
	BlueprintSource BlueprintSource `json:"source"`
}

type BlueprintSource struct {
	LocalFile BlueprintSourceLocalFile `json:"local_file"`
}

type BlueprintSourceLocalFile struct {
	Path string `json:"path"`
}
