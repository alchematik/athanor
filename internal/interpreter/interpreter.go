package interpreter

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	external_ast "github.com/alchematik/athanor/ast"

	"github.com/bytecodealliance/wasmtime-go/v20"
)

type Interpreter struct {
	Logger *slog.Logger
}

func (it *Interpreter) InterpretBlueprint(source external_ast.BlueprintSource, input map[string]any) (external_ast.Blueprint, error) {
	engine := wasmtime.NewEngine()
	module, err := wasmtime.NewModuleFromFile(engine, source.LocalFile.Path)
	if err != nil {
		return external_ast.Blueprint{}, err
	}

	linker := wasmtime.NewLinker(engine)
	if err := linker.DefineWasi(); err != nil {
		return external_ast.Blueprint{}, err
	}

	wasiConfig := wasmtime.NewWasiConfig()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return external_ast.Blueprint{}, err
	}
	defer os.RemoveAll(dir)

	if err := wasiConfig.PreopenDir(dir, "/"); err != nil {
		return external_ast.Blueprint{}, err
	}

	store := wasmtime.NewStore(engine)
	store.SetWasi(wasiConfig)

	instance, err := linker.Instantiate(store, module)
	if err != nil {
		return external_ast.Blueprint{}, err
	}

	nom := instance.GetFunc(store, "_start")
	_, err = nom.Call(store)
	if err != nil {
		var wasmtimeError *wasmtime.Error
		if errors.As(err, &wasmtimeError) {
			st, ok := wasmtimeError.ExitStatus()
			if ok && st != 0 {
				return external_ast.Blueprint{}, fmt.Errorf("non-0 exit status: %d", st)
			}
		} else {
			return external_ast.Blueprint{}, err
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "blueprint.json"))
	if err != nil {
		return external_ast.Blueprint{}, err
	}

	var bp external_ast.Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return external_ast.Blueprint{}, err
	}

	return bp, nil
}
