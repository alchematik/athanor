package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alchematik/athanor/internal/ast"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type Translator struct {
	Dir     string
	clients map[string]translatorpb.TranslatorClient
}

func NewTranslator() *Translator {
	return &Translator{
		clients: map[string]translatorpb.TranslatorClient{},
	}
}

func (t *Translator) Translate(ctx context.Context, b ast.StmtBuild) (ast.Blueprint, error) {
	var dir string
	switch r := b.Translator.Repo.(type) {
	case ast.RepoLocal:
		dir = r.Path
	default:
		return ast.Blueprint{}, fmt.Errorf("invalid repo type: %T", b.Translator.Repo)
	}

	pluginPath := filepath.Join(dir, b.Translator.Name, b.Translator.Version, "translator")

	c, ok := t.clients[pluginPath]
	if !ok {
		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: plugin.HandshakeConfig{
				ProtocolVersion:  1,
				MagicCookieKey:   "COOKIE",
				MagicCookieValue: "hi",
			},
			Plugins: map[string]plugin.Plugin{
				"translator": &TranslatorPlugin{},
			},
			Cmd:              exec.Command("sh", "-c", pluginPath),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			// Logger:           hclog.NewNullLogger(),
		})

		dispensor, err := client.Client()
		if err != nil {
			return ast.Blueprint{}, err
		}

		rawPlug, err := dispensor.Dispense("translator")
		if err != nil {
			return ast.Blueprint{}, err
		}

		c, ok = rawPlug.(translatorpb.TranslatorClient)
		if !ok {
			return ast.Blueprint{}, fmt.Errorf("expected TranslatorClient, got %T", rawPlug)
		}

		t.clients[pluginPath] = c
	}

	var inputPath string
	switch r := b.Repo.(type) {
	case ast.RepoLocal:
		inputPath = r.Path
	default:
		return ast.Blueprint{}, fmt.Errorf("invalid repo type: %T", b.Repo)
	}

	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return ast.Blueprint{}, err
	}

	// defer os.Remove(tempFile.Name())

	config, err := exprToProto(b.Config)
	if err != nil {
		return ast.Blueprint{}, err
	}

	configData, err := json.Marshal(config)
	if err != nil {
		return ast.Blueprint{}, err
	}

	configTempFile, err := os.CreateTemp("", "")
	if err != nil {
		return ast.Blueprint{}, err
	}

	// defer os.Remove(configTempFile.Name())

	configFile, err := os.OpenFile(configTempFile.Name(), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return ast.Blueprint{}, err
	}

	if _, err := configFile.Write(configData); err != nil {
		return ast.Blueprint{}, err
	}

	// fmt.Printf("INPUT PATH: %v\n", inputPath)
	// fmt.Printf("CONFIG: %v\n", configTempFile.Name())
	// fmt.Printf("OUTPUT PATH: %v\n", tempFile.Name())

	if _, err = c.TranslateBlueprint(ctx, &translatorpb.TranslateBlueprintRequest{
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

// TODO: replace.
func (t Translator) Client(name, version string) (translatorpb.TranslatorClient, func(), error) {
	pluginPath := filepath.Join(t.Dir, name, version, "translator")

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"translator": &TranslatorPlugin{},
		},
		Cmd:              exec.Command("sh", "-c", pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           hclog.NewNullLogger(),
	})

	dispensor, err := client.Client()
	if err != nil {
		return nil, nil, err
	}

	rawPlug, err := dispensor.Dispense("translator")
	if err != nil {
		return nil, nil, err
	}

	plug, ok := rawPlug.(translatorpb.TranslatorClient)
	if !ok {
		return nil, nil, fmt.Errorf("expected TranslatorClient, got %T", rawPlug)
	}

	return plug, client.Kill, nil
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
		config, err := convertExpr(s.Build.GetConfig())
		if err != nil {
			return nil, err
		}

		runtimeConfig, err := convertExpr(s.Build.GetRuntimeConfig())
		if err != nil {
			return nil, err
		}

		repo, err := convertRepo(s.Build.GetRepo())
		if err != nil {
			return nil, err
		}

		translatorRepo, err := convertRepo(s.Build.Translator.GetRepo())
		if err != nil {
			return nil, err
		}

		return ast.StmtBuild{
			Alias:         s.Build.GetAlias(),
			Repo:          repo,
			Config:        config,
			RuntimeConfig: runtimeConfig,
			Translator: ast.Translator{
				Name:    s.Build.Translator.Name,
				Version: s.Build.Translator.Version,
				Repo:    translatorRepo,
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
		repo, err := convertRepo(e.Provider.GetRepo())
		if err != nil {
			return nil, err
		}

		return ast.ExprProvider{
			Name:    e.Provider.GetName(),
			Version: e.Provider.GetVersion(),
			Repo:    repo,
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

func convertRepo(r *consumerpb.Repo) (ast.Repo, error) {
	switch r := r.GetType().(type) {
	case *consumerpb.Repo_Local:
		return ast.RepoLocal{
			Path: r.Local.GetPath(),
		}, nil
	default:
		return nil, fmt.Errorf("invalid repo type: %T", r)
	}
}
