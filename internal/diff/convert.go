package diff

import (
	"fmt"

	external "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/scope"
	"github.com/alchematik/athanor/internal/state"
)

type Converter struct {
	BlueprintInterpreter BlueprintInterpreter
	PlanConverter        *plan.Converter
	StateConverter       *state.Converter
}

type BlueprintInterpreter interface {
	InterpretBlueprint(source external.BlueprintSource, input map[string]any) (external.Blueprint, error)
}

func (c *Converter) ConvertStmt(d *DiffResult, sc *scope.Scope, stmt external.Stmt) (any, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareResource:
		return c.ConvertResourceStmt(d, sc, stmt)
	case external.DeclareBuild:
		return c.ConvertBuildStmt(d, sc, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertBuildStmt(d *DiffResult, sc *scope.Scope, stmt external.DeclareBuild) (StmtBuild, error) {
	blueprint, err := c.BlueprintInterpreter.InterpretBlueprint(stmt.BlueprintSource, stmt.Input)
	if err != nil {
		return StmtBuild{}, err
	}

	planRuntimeInput, err := c.PlanConverter.ConvertMapExpr(stmt.Name, stmt.Runtimeinput)
	if err != nil {
		return StmtBuild{}, err
	}

	stateRuntimeInput, err := c.StateConverter.ConvertMapExpr(stmt.Name, stmt.Runtimeinput)
	if err != nil {
		return StmtBuild{}, err
	}

	buildID := sc.ComponentID(stmt.Name)
	d.State.Builds[buildID] = state.NewBuildState(stmt.Name)
	d.Plan.Builds[buildID] = plan.NewBuildPlan(stmt.Name)
	d.Builds[buildID] = &BuildDiff{name: stmt.Name}

	var stmts []any
	sub := sc.Sub(stmt.Name)
	for _, s := range blueprint.Stmts {
		converted, err := c.ConvertStmt(d, sub, s)
		if err != nil {
			return StmtBuild{}, err
		}

		stmts = append(stmts, converted)
	}
	b := StmtBuild{
		ID:                buildID,
		Name:              stmt.Name,
		BuildID:           sc.ID(),
		Stmts:             stmts,
		PlanRuntimeInput:  planRuntimeInput,
		StateRuntimeInput: stateRuntimeInput,
	}
	sc.SetBuild(buildID, b)
	return b, nil
}

func (c *Converter) ConvertResourceStmt(d *DiffResult, sc *scope.Scope, stmt external.DeclareResource) (StmtResource, error) {
	resourceID := sc.ComponentID(stmt.Name)

	d.Plan.Resources[resourceID] = plan.NewResourcePlan(stmt.Name)
	d.State.Resources[resourceID] = state.NewResourceState(stmt.Name)
	d.Resources[resourceID] = &ResourceDiff{name: stmt.Name}

	t, err := c.StateConverter.ConvertStringExpr(stmt.Name, stmt.Type)
	if err != nil {
		return StmtResource{}, err
	}

	id, err := c.StateConverter.ConvertAnyExpr(stmt.Name, stmt.Identifier)
	if err != nil {
		return StmtResource{}, err
	}

	provider, err := c.StateConverter.ConvertProviderExpr(stmt.Name, stmt.Provider)
	if err != nil {
		return StmtResource{}, err
	}

	planExists, err := c.PlanConverter.ConvertBoolExpr(stmt.Name, stmt.Exists)
	if err != nil {
		return StmtResource{}, err
	}

	planType, err := c.PlanConverter.ConvertStringExpr(stmt.Name, stmt.Type)
	if err != nil {
		return StmtResource{}, err
	}

	planProvider, err := c.PlanConverter.ConvertProviderExpr(stmt.Name, stmt.Provider)
	if err != nil {
		return StmtResource{}, err
	}

	planIdentifier, err := c.PlanConverter.ConvertAnyExpr(stmt.Name, stmt.Identifier)
	if err != nil {
		return StmtResource{}, err
	}

	planConfig, err := c.PlanConverter.ConvertAnyExpr(stmt.Name, stmt.Config)
	if err != nil {
		return StmtResource{}, err
	}

	sr := StmtResource{
		ID:      resourceID,
		Name:    stmt.Name,
		BuildID: sc.ID(),

		Type:       t,
		Identifier: id,
		Provider:   provider,

		PlanExists:     planExists,
		PlanType:       planType,
		PlanProvider:   planProvider,
		PlanIdentifier: planIdentifier,
		PlanConfig:     planConfig,
	}
	sc.SetResource(resourceID, sr)

	return sr, nil
}

// func (c *Converter) ConvertAnyExpr(name string, expr external.Expr) (Expr[any], error) {
// 	switch expr.Value.(type) {
// 	case external.BoolLiteral:
// 		expr, err := c.ConvertBoolExpr(name, expr)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		return ExprAny[Literal[bool]]{
// 			Value: expr,
// 		}, nil
// 	default:
// 		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
// 	}
//
// }

// func (c *Converter) ConvertBoolExpr(name string, expr external.Expr) (Expr[Literal[bool]], error) {
// 	switch expr.Value.(type) {
// 	case external.BoolLiteral:
// 		p, err := c.PlanConverter.ConvertBoolExpr(name, expr)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		s, err := c.StateConverter.ConvertBoolExpr(name, expr)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		return ExprLiteral[bool]{
// 			Plan:  p,
// 			State: s,
// 		}, nil
// 	default:
// 		return nil, fmt.Errorf("invalid bool expr: %T", expr)
// 	}
// }
