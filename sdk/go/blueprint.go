package sdk

import (
	"encoding/json"
	"fmt"
	blueprintpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	"os"
)

type Blueprint struct {
	stmts []Stmt
}

func (b Blueprint) WithResource(r Type) Blueprint {
	b.stmts = append(b.stmts, resourceStmt{
		resource: r,
	})
	return b
}

func (b Blueprint) toProto() *blueprintpb.Blueprint {
	stmts := make([]*blueprintpb.Stmt, len(b.stmts))
	for i, stmt := range b.stmts {
		stmts[i] = stmt.ToStmtProto()
	}

	return &blueprintpb.Blueprint{
		Stmts: stmts,
	}
}

type Stmt interface {
	ToStmtProto() *blueprintpb.Stmt
}

type resourceStmt struct {
	resource Type
}

func (r resourceStmt) ToStmtProto() *blueprintpb.Stmt {
	return &blueprintpb.Stmt{
		Type: &blueprintpb.Stmt_Resource{
			Resource: &blueprintpb.ResourceStmt{
				Expr: r.resource.ToExpr().ToExprProto(),
			},
		},
	}
}

type Expr interface {
	ToExprProto() *blueprintpb.Expr
}

type Type interface {
	ToExpr() Expr
}

func Bool(b bool) BoolValue {
	return BoolValue(b)
}

type BoolValue bool

func (b BoolValue) ToExpr() Expr {
	return BoolExpr(b)
}

func String(s string) StringType {
	return StringType(s)
}

type StringType string

func (s StringType) ToExpr() Expr {
	return StringExpr(s)
}

func Map(m map[string]Type) MapType {
	return MapType(m)
}

type MapType map[string]Type

func (m MapType) ToExpr() Expr {
	ex := MapExpr{}
	for k, v := range m {
		ex[k] = v.ToExpr()
	}

	return ex
}

func ResourceIdentifier(resourceType, alias string, value Type) ResourceIdentifierType {
	return ResourceIdentifierType{
		ResourceType: resourceType,
		Alias:        alias,
		Value:        value,
	}
}

type ResourceIdentifierType struct {
	Alias        string
	ResourceType string
	Value        Type
}

func (r ResourceIdentifierType) ToExpr() Expr {
	return ResourceIdentifierExpr{
		Alias:        r.Alias,
		ResourceType: r.ResourceType,
		Value:        r.Value.ToExpr(),
	}
}

func Resource(exists, provider, identifier, config Type) ResourceType {
	return ResourceType{
		Exists:     exists,
		Provider:   provider,
		Identifier: identifier,
		Config:     config,
	}
}

type ResourceType struct {
	Exists     Type
	Provider   Type
	Identifier Type
	Config     Type
}

func (r ResourceType) ToExpr() Expr {
	return ResourceExpr{
		Exists:     r.Exists.ToExpr(),
		Provider:   r.Provider.ToExpr(),
		Identifier: r.Identifier.ToExpr(),
		Config:     r.Config.ToExpr(),
	}
}

func Provider(alias string, name, version Type) ProviderType {
	return ProviderType{
		Alias:   alias,
		Name:    name,
		Version: version,
	}
}

type ProviderType struct {
	Alias   string
	Name    Type
	Version Type
}

func (p ProviderType) ToExpr() Expr {
	return ProviderExpr{
		Alias:   p.Alias,
		Name:    p.Name.ToExpr(),
		Version: p.Version.ToExpr(),
	}
}

func GetResource(alias string) GetResourceType {
	return GetResourceType{Alias: alias}
}

type GetResourceType struct {
	Alias string
}

func (g GetResourceType) GetAttrs() GetType {
	return GetType{
		Name:   "attrs",
		Object: g,
	}
}

func (g GetResourceType) GetIdentifier() GetType {
	return GetType{
		Name:   "identifier",
		Object: g,
	}
}

func (g GetResourceType) GetConfig() GetType {
	return GetType{
		Name:   "config",
		Object: g,
	}
}

func (e GetResourceType) ToExpr() Expr {
	return GetResourceExpr{Alias: e.Alias}
}

type GetType struct {
	Name   string
	Object Type
}

func (g GetType) ToExpr() Expr {
	return GetExpr{
		Name:   g.Name,
		Object: g.Object.ToExpr(),
	}
}

func (g GetType) IOGet(name string) IOGetType {
	return IOGetType{
		Name:   name,
		Object: g,
	}
}

type IOGetType struct {
	Name   string
	Object Type
}

func (g IOGetType) ToExpr() Expr {
	return IOGetExpr{
		Name:   g.Name,
		Object: g.Object.ToExpr(),
	}
}

func (g IOGetType) Get(name string) GetType {
	return GetType{
		Name:   name,
		Object: g,
	}
}

func Build(bp Blueprint) error {

	p := bp.toProto()

	f, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}

	fmt.Printf("TEMP FILE: %v\n", f.Name())

	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = f.Write(data)

	return err
}

type ProviderExpr struct {
	Alias   string
	Name    Expr
	Version Expr
}

func (p ProviderExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_Provider{
			Provider: &blueprintpb.ProviderExpr{
				Identifier: &blueprintpb.Expr{
					Type: &blueprintpb.Expr_ProviderIdentifier{
						ProviderIdentifier: &blueprintpb.ProviderIdentifierExpr{
							Alias:   p.Alias,
							Name:    p.Name.ToExprProto(),
							Version: p.Version.ToExprProto(),
						},
					},
				},
			},
		},
	}
}

type ResourceExpr struct {
	Provider   Expr
	Identifier Expr
	Config     Expr
	Exists     Expr
}

func (r ResourceExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_Resource{
			Resource: &blueprintpb.ResourceExpr{
				Provider:   r.Provider.ToExprProto(),
				Identifier: r.Identifier.ToExprProto(),
				Config:     r.Config.ToExprProto(),
				Exists:     r.Exists.ToExprProto(),
			},
		},
	}
}

type BoolExpr bool

func (b BoolExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_BoolLiteral{
			BoolLiteral: bool(b),
		},
	}
}

type StringExpr string

func (s StringExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_StringLiteral{
			StringLiteral: string(s),
		},
	}
}

type MapExpr map[string]Expr

func (m MapExpr) ToExprProto() *blueprintpb.Expr {
	p := map[string]*blueprintpb.Expr{}
	for k, v := range m {
		p[k] = v.ToExprProto()
	}

	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_Map{
			Map: &blueprintpb.MapExpr{
				Entries: p,
			},
		},
	}
}

type ResourceIdentifierExpr struct {
	Alias        string
	ResourceType string
	Value        Expr
}

func (r ResourceIdentifierExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_ResourceIdentifier{
			ResourceIdentifier: &blueprintpb.ResourceIdentifierExpr{
				Alias: r.Alias,
				Type:  r.ResourceType,
				Value: r.Value.ToExprProto(),
			},
		},
	}
}

type GetResourceExpr struct {
	Alias string
}

func (e GetResourceExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_GetResource_{
			GetResource_: &blueprintpb.GetResourceExpr{
				Alias: e.Alias,
			},
		},
	}
}

type GetExpr struct {
	Name   string
	Object Expr
}

func (g GetExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_Get{
			Get: &blueprintpb.GetExpr{
				Name:   g.Name,
				Object: g.Object.ToExprProto(),
			},
		},
	}
}

type IOGetExpr struct {
	Name   string
	Object Expr
}

func (g IOGetExpr) ToExprProto() *blueprintpb.Expr {
	return &blueprintpb.Expr{
		Type: &blueprintpb.Expr_IoGet{
			IoGet: &blueprintpb.IOGetExpr{
				Name:   g.Name,
				Object: g.Object.ToExprProto(),
			},
		},
	}
}
