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

func (c *Converter) ConvertStmt(d *Diff, sc *scope.Scope, stmt external.Stmt) (any, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareResource:
		return c.ConvertResourceStmt(d, sc, stmt)
	case external.DeclareBuild:
		return c.ConvertBuildStmt(d, sc, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertBuildStmt(d *Diff, sc *scope.Scope, stmt external.DeclareBuild) (StmtBuild, error) {
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

func (c *Converter) ConvertResourceStmt(d *Diff, sc *scope.Scope, stmt external.DeclareResource) (StmtResource, error) {
	resourceID := sc.ComponentID(stmt.Name)

	d.Plan.Resources[resourceID] = plan.NewResourcePlan(stmt.Name)
	d.State.Resources[resourceID] = state.NewResourceState(stmt.Name)
	d.Resources[resourceID] = &ResourceDiff{name: stmt.Name}

	exists, err := c.ConvertBoolExpr(stmt.Name, stmt.Exists)
	if err != nil {
		return StmtResource{}, err
	}

	r, err := c.ConvertResourceExpr(stmt.Name, stmt.Resource)
	if err != nil {
		return StmtResource{}, err
	}

	sr := StmtResource{
		ID:       resourceID,
		Name:     stmt.Name,
		BuildID:  sc.ID(),
		Exists:   exists,
		Resource: r,
	}
	sc.SetResource(resourceID, sr)

	return sr, nil
}

func (c *Converter) ConvertResourceExpr(name string, expr external.Expr) (Expr[Resource], error) {
	switch expr.Value.(type) {
	case external.Resource:
		p, err := c.PlanConverter.ConvertResourceExpr(name, expr)
		if err != nil {
			return nil, err
		}

		s, err := c.StateConverter.ConvertResourceExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ExprResource{
			Name:  name,
			Plan:  p,
			State: s,
		}, nil
	default:
		return nil, fmt.Errorf("invalid resource expr: %T", expr)
	}
}

func (c *Converter) ConvertAnyExpr(name string, expr external.Expr) (Expr[any], error) {
	switch expr.Value.(type) {
	case external.BoolLiteral:
		expr, err := c.ConvertBoolExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ExprAny[DiffLiteral[bool]]{
			Value: expr,
		}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}

}

func (c *Converter) ConvertBoolExpr(name string, expr external.Expr) (Expr[DiffLiteral[bool]], error) {
	switch expr.Value.(type) {
	case external.BoolLiteral:
		p, err := c.PlanConverter.ConvertBoolExpr(name, expr)
		if err != nil {
			return nil, err
		}

		s, err := c.StateConverter.ConvertBoolExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ExprLiteral[bool]{
			Plan:  p,
			State: s,
		}, nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}
