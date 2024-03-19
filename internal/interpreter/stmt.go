package interpreter

import (
	"context"
	"fmt"
	"runtime"
	"slices"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/repo"
	"github.com/alchematik/athanor/internal/spec"
)

func (in *Interpreter) buildStmt(ctx context.Context, s spec.Spec, stmt ast.StmtBuild) error {
	runtimeConfig, children, err := in.Expr(ctx, stmt.Build.RuntimeConfig)
	if err != nil {
		return err
	}

	subSpec := spec.Spec{
		DependencyMap: map[string][]string{},
		Components:    map[string]spec.Component{},
		RuntimeConfig: runtimeConfig,
	}

	var translatorSrc any
	switch src := stmt.Translator.Source.(type) {
	case repo.Local:
		translatorSrc = dependency.SourceLocal{Path: src.Path}
	case repo.GitHubRelease:
		translatorSrc = dependency.SourceGitHubRelease{
			RepoOwner: src.RepoOwner,
			RepoName:  src.RepoName,
			Name:      src.Name,
		}
	default:
		return fmt.Errorf("unsupported source type: %T", stmt.Translator.Source)
	}

	binPath, err := in.DepManager.FetchBinDependency(ctx, dependency.BinDependency{
		Type:   "translator",
		Source: translatorSrc,
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
	})
	if err != nil {
		return fmt.Errorf("interpreter: error getting translator binary: %s", err)
	}

	tr, err := in.PlugManager.Translator(binPath)
	if err != nil {
		return fmt.Errorf("interpreter: error getting translator: %s", err)
	}

	bp, err := tr.TranslateBlueprint(ctx, stmt.Build)
	if err != nil {
		return fmt.Errorf("interpreter: error translating blueprint: %s", err)
	}

	// TODO: ast.ExprBlueprint vs ast.Blueprint?
	// _, _, err = in.Expr(ctx, ast.ExprBlueprint{Stmts: bp.Stmts})
	// if err != nil {
	// 	return err
	// }

	alias := stmt.Build.Alias

	in.Lock()
	for _, s := range bp.Stmts {
		in.stmts = append(in.stmts, Stmt{
			Spec: subSpec,
			Stmt: s,
		})
	}
	s.DependencyMap[alias] = append(s.DependencyMap[alias], children...)
	s.Components[alias] = spec.ComponentBuild{Spec: subSpec}
	in.Unlock()

	return nil
}

func (in *Interpreter) resourceStmt(ctx context.Context, b spec.Spec, s ast.StmtResource) error {
	resource, children, err := in.resource(ctx, s.Expr)
	if err != nil {
		return err
	}

	provider, providerChildren, err := in.provider(ctx, s.Provider)
	if err != nil {
		return err
	}

	exists, existsChildren, err := in.Expr(ctx, s.Exists)
	if err != nil {
		return err
	}

	children = append(children, providerChildren...)
	children = append(children, existsChildren...)

	// TODO: Probably Put provider and exists on component.
	resource.Provider = provider
	resource.Exists = exists

	alias := resource.Identifier.Alias
	in.Lock()
	b.DependencyMap[alias] = slices.Compact(append(b.DependencyMap[alias], children...))
	b.Components[alias] = spec.ComponentResource{
		Value: resource,
	}
	in.Unlock()

	return nil
}
