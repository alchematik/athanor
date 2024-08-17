package diff

import (
	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/state"
)

type StmtBuild struct {
	ID      string
	Name    string
	BuildID string
	// Exists            Expr[Literal[bool]]
	Stmts             []any
	StateRuntimeInput state.Expr[map[string]any]
	PlanRuntimeInput  plan.Expr[map[plan.Maybe[string]]plan.Maybe[any]]
}

type StmtResource struct {
	ID      string
	Name    string
	BuildID string

	Type       state.Expr[string]
	Identifier state.Expr[any]
	Provider   state.Expr[state.Provider]

	PlanExists     plan.Expr[bool]
	PlanType       plan.Expr[string]
	PlanProvider   plan.Expr[plan.Provider]
	PlanIdentifier plan.Expr[any]
	PlanConfig     plan.Expr[any]
}
