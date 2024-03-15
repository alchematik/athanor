package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/dependency"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/internal/repo"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type TranslatorManager struct {
	logger            hclog.Logger
	translators       map[string]*Translator
	dependencyManager *dependency.Manager
}

func NewTranslatorManager(logger hclog.Logger, dm *dependency.Manager) *TranslatorManager {
	return &TranslatorManager{
		translators:       map[string]*Translator{},
		logger:            logger,
		dependencyManager: dm,
	}
}

func (m *TranslatorManager) Translator(ctx context.Context, s repo.Source, r repo.Runtime) (*Translator, error) {
	var src any
	switch s := s.(type) {
	case repo.Local:
		src = dependency.SourceLocal{Path: s.Path}
	case repo.GitHubRelease:
		src = dependency.SourceGitHubRelease{
			RepoOwner: s.RepoOwner,
			RepoName:  s.RepoName,
			Name:      s.Name,
		}
	default:
		return nil, fmt.Errorf("invalid source type: %T", s)
	}

	dep := dependency.BinDependency{
		Type:   "translator",
		Source: src,
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
	}
	binPath, err := m.dependencyManager.FetchBinDependency(ctx, dep)
	if err != nil {
		return nil, err
	}

	if tr, ok := m.translators[binPath]; ok {
		return tr, nil
	}

	m.logger.Debug("PLUGIN PATH >>", "path", binPath)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"translator": &TranslatorPlugin{},
		},
		Cmd:              exec.Command("sh", "-c", binPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           m.logger,
	})

	tr := &Translator{plug: client}

	m.translators[binPath] = tr
	return tr, nil
}

type Translator struct {
	plug *plugin.Client
}

func (t *Translator) TranslateBlueprint(ctx context.Context, b ast.ExprBuild) (ast.Blueprint, error) {
	tc, err := t.client()
	if err != nil {
		return ast.Blueprint{}, err
	}

	// TODO: Should plugin source and blueprint source be different?
	inputPath := ""
	switch s := b.Source.(type) {
	case repo.Local:
		inputPath = s.Path
	}

	fmt.Printf("TRANSLATING>>> %v\n", inputPath)

	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return ast.Blueprint{}, err
	}

	defer os.Remove(tempFile.Name())

	configs := make([]*consumerpb.Expr, len(b.Config))
	for i, c := range b.Config {
		converted, err := exprToProto(c)
		if err != nil {
			return ast.Blueprint{}, err
		}

		configs[i] = converted
	}

	configData, err := json.Marshal(configs)
	if err != nil {
		return ast.Blueprint{}, err
	}

	configTempFile, err := os.CreateTemp("", "")
	if err != nil {
		return ast.Blueprint{}, err
	}

	defer os.Remove(configTempFile.Name())

	configFile, err := os.OpenFile(configTempFile.Name(), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return ast.Blueprint{}, err
	}

	if _, err := configFile.Write(configData); err != nil {
		return ast.Blueprint{}, err
	}

	if _, err = tc.TranslateBlueprint(ctx, &translatorpb.TranslateBlueprintRequest{
		InputPath:  inputPath,
		ConfigPath: configFile.Name(),
		OutputPath: tempFile.Name(),
	}); err != nil {
		return ast.Blueprint{}, fmt.Errorf("error translating blueprint: %v", err)
	}

	blueprintData, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return ast.Blueprint{}, err
	}

	var blueprint consumerpb.Blueprint
	if err := json.Unmarshal(blueprintData, &blueprint); err != nil {
		return ast.Blueprint{}, fmt.Errorf("error unmarshaling blueprint: %v", err)
	}

	return convertBlueprint(&blueprint)
}

func (t *Translator) TranslateProviderSchema(ctx context.Context, inputPath, outputPath string) error {
	client, err := t.client()
	if err != nil {
		return err
	}

	_, err = client.TranslateProviderSchema(ctx, &translatorpb.TranslateProviderSchemaRequest{
		OutputPath: outputPath,
		InputPath:  inputPath,
	})

	return err
}

func (t *Translator) GenerateProviderSDK(ctx context.Context, inputPath, outputPath string, args map[string]string) error {
	client, err := t.client()
	if err != nil {
		return err
	}

	_, err = client.GenerateProviderSDK(ctx, &translatorpb.GenerateProviderSDKRequest{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Args:       args,
	})

	return err
}

func (t *Translator) GenerateConsumerSDK(ctx context.Context, inputPath, outputPath string) error {
	client, err := t.client()
	if err != nil {
		return err
	}

	_, err = client.GenerateConsumerSDK(ctx, &translatorpb.GenerateConsumerSDKRequest{
		InputPath:  inputPath,
		OutputPath: outputPath,
	})

	return err
}

func (t *Translator) client() (translatorpb.TranslatorClient, error) {
	dispensor, err := t.plug.Client()
	if err != nil {
		return nil, err
	}

	rawPlug, err := dispensor.Dispense("translator")
	if err != nil {
		return nil, err
	}

	tc, ok := rawPlug.(translatorpb.TranslatorClient)
	if !ok {
		return nil, fmt.Errorf("expected TranslatorClient, got %T", rawPlug)
	}

	return tc, nil
}

func (t *Translator) Stop() {
	t.plug.Kill()
}

func (m *TranslatorManager) Stop() {
	for _, t := range m.translators {
		t.Stop()
	}
}

type TranslatorPlugin struct {
	plugin.Plugin

	TranslatorServer translatorpb.TranslatorServer
}

func (p *TranslatorPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	translatorpb.RegisterTranslatorServer(s, p.TranslatorServer)
	return nil
}

func (p *TranslatorPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return translatorpb.NewTranslatorClient(conn), nil
}

func convertBlueprint(bp *consumerpb.Blueprint) (ast.Blueprint, error) {
	out := ast.Blueprint{}
	for _, stmt := range bp.GetStmts() {
		converted, err := convertStmt(stmt)
		if err != nil {
			return ast.Blueprint{}, err
		}

		out.Stmts = append(out.Stmts, converted)
	}

	return out, nil
}

func convertStmt(st *consumerpb.Stmt) (ast.Stmt, error) {
	switch s := st.GetType().(type) {
	case *consumerpb.Stmt_Resource:
		ex, err := convertExpr(s.Resource.GetExpr())
		if err != nil {
			return nil, err
		}

		return ast.StmtResource{
			Expr: ex,
		}, nil
	case *consumerpb.Stmt_Build:
		configs := make([]ast.Expr, len(s.Build.GetBuild().GetConfig()))
		for i, c := range s.Build.GetBuild().GetConfig() {
			config, err := convertExpr(c)
			if err != nil {
				return nil, err
			}

			configs[i] = config
		}

		runtimeConfig, err := convertExpr(s.Build.GetBuild().GetRuntimeConfig())
		if err != nil {
			return nil, err
		}

		src, err := convertSource(s.Build.GetBuild().GetSource())
		if err != nil {
			return nil, err
		}

		translatorSource, err := convertSource(s.Build.Translator.GetSource())
		if err != nil {
			return nil, err
		}

		return ast.StmtBuild{
			Translator: ast.Translator{
				Source: translatorSource,
			},
			Build: ast.ExprBuild{
				Alias:         s.Build.GetBuild().GetAlias(),
				Config:        configs,
				RuntimeConfig: runtimeConfig,
				Source:        src,
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid stmt: %T", st.GetType())
	}
}

func convertExpr(ex *consumerpb.Expr) (ast.Expr, error) {
	switch e := ex.GetType().(type) {
	case *consumerpb.Expr_Blueprint:
		stmts := make([]ast.Stmt, len(e.Blueprint.GetStmts()))
		for i, s := range e.Blueprint.GetStmts() {
			converted, err := convertStmt(s)
			if err != nil {
				return nil, err
			}

			stmts[i] = converted
		}

		return ast.ExprBlueprint{Stmts: stmts}, nil
	case *consumerpb.Expr_Provider:
		s, err := convertSource(e.Provider.GetSource())
		if err != nil {
			return nil, err
		}

		return ast.ExprProvider{
			Name:   e.Provider.GetName(),
			Source: s,
		}, nil
	case *consumerpb.Expr_Resource:
		provider, err := convertExpr(e.Resource.GetProvider())
		if err != nil {
			return nil, err
		}

		id, err := convertExpr(e.Resource.GetIdentifier())
		if err != nil {
			return nil, err
		}

		config, err := convertExpr(e.Resource.GetConfig())
		if err != nil {
			return nil, err
		}

		exists, err := convertExpr(e.Resource.GetExists())
		if err != nil {
			return nil, err
		}

		return ast.ExprResource{
			Provider:   provider,
			Identifier: id,
			Config:     config,
			Exists:     exists,
		}, nil
	case *consumerpb.Expr_ResourceIdentifier:
		val, err := convertExpr(e.ResourceIdentifier.GetValue())
		if err != nil {
			return ast.ExprResourceIdentifier{}, err
		}

		return ast.ExprResourceIdentifier{
			Alias:        e.ResourceIdentifier.GetAlias(),
			ResourceType: e.ResourceIdentifier.GetType(),
			Value:        val,
		}, nil
	case *consumerpb.Expr_StringLiteral:
		return ast.ExprString{Value: e.StringLiteral}, nil
	case *consumerpb.Expr_BoolLiteral:
		return ast.ExprBool{Value: e.BoolLiteral}, nil
	case *consumerpb.Expr_File:
		return ast.ExprFile{Path: e.File.Path}, nil
	case *consumerpb.Expr_Map:
		m := ast.ExprMap{Entries: map[string]ast.Expr{}}
		for k, v := range e.Map.GetEntries() {
			var err error
			m.Entries[k], err = convertExpr(v)
			if err != nil {
				return nil, err
			}
		}

		return m, nil
	case *consumerpb.Expr_List:
		l := make([]ast.Expr, len(e.List.Elements))
		for i, val := range e.List.Elements {
			converted, err := convertExpr(val)
			if err != nil {
				return nil, err
			}
			l[i] = converted
		}

		return ast.ExprList{
			Elements: l,
		}, nil
	case *consumerpb.Expr_Get:
		obj, err := convertExpr(e.Get.GetObject())
		if err != nil {
			return nil, err
		}

		g := ast.ExprGet{
			Name:   e.Get.GetName(),
			Object: obj,
		}

		return g, nil
	case *consumerpb.Expr_GetRuntimeConfig:
		return ast.ExprGetRuntimeConfig{}, nil
	case *consumerpb.Expr_Nil:
		return ast.ExprNil{}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", ex.GetType())
	}
}

func exprToProto(expr ast.Expr) (*consumerpb.Expr, error) {
	switch expr := expr.(type) {
	case ast.ExprString:
		return &consumerpb.Expr{
			Type: &consumerpb.Expr_StringLiteral{
				StringLiteral: expr.Value,
			},
		}, nil
	case ast.ExprBool:
		return &consumerpb.Expr{
			Type: &consumerpb.Expr_BoolLiteral{
				BoolLiteral: expr.Value,
			},
		}, nil
	case ast.ExprMap:
		m := map[string]*consumerpb.Expr{}
		for k, v := range expr.Entries {
			val, err := exprToProto(v)
			if err != nil {
				return nil, err
			}
			m[k] = val
		}

		return &consumerpb.Expr{
			Type: &consumerpb.Expr_Map{
				Map: &consumerpb.MapExpr{
					Entries: m,
				},
			},
		}, nil
	case ast.ExprList:
		l := make([]*consumerpb.Expr, len(expr.Elements))
		return &consumerpb.Expr{
			Type: &consumerpb.Expr_List{
				List: &consumerpb.ListExpr{
					Elements: l,
				},
			},
		}, nil
	case ast.ExprNil:
		return &consumerpb.Expr{
			Type: &consumerpb.Expr_Nil{},
		}, nil
	case nil:
		return &consumerpb.Expr{
			Type: &consumerpb.Expr_Nil{},
		}, nil
	default:
		return nil, fmt.Errorf("invalid expr type: %T", expr)
	}
}

func convertSource(src *consumerpb.Source) (repo.Source, error) {
	switch s := src.GetType().(type) {
	case *consumerpb.Source_FilePath:
		return repo.Local{
			Path: s.FilePath.GetPath(),
		}, nil
	case *consumerpb.Source_GitHubRelease:
		return repo.GitHubRelease{
			RepoOwner: s.GitHubRelease.GetRepoOwner(),
			RepoName:  s.GitHubRelease.GetRepoName(),
			Name:      s.GitHubRelease.GetName(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported source: %T", s)
	}
}
