package analyzer

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

type Finding struct {
	Line      int
	Package   string
	Func      string
	IsPhantom bool
}

func Analyze(src string) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	info := &types.Info{
		Uses:       map[*ast.Ident]types.Object{},
		Types:      map[ast.Expr]types.TypeAndValue{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	conf := types.Config{Importer: importer.Default(), Error: func(error) {}}
	pkg, err := conf.Check("sample", fset, []*ast.File{file}, info)
	if err != nil && len(info.Uses) == 0 {
		return nil, err
	}
	findings := make([]Finding, 0)
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok {
			if pkgName, ok := info.Uses[ident].(*types.PkgName); ok {
				obj := pkgName.Imported().Scope().Lookup(sel.Sel.Name)
				findings = append(findings, Finding{Line: fset.Position(call.Pos()).Line, Package: ident.Name, Func: sel.Sel.Name, IsPhantom: obj == nil || !obj.Exported()})
				return true
			}
		}
		tv, ok := info.Types[sel.X]
		if !ok || tv.Type == nil || tv.Type == types.Typ[types.Invalid] {
			return true
		}
		_, selected := info.Selections[sel]
		findings = append(findings, Finding{Line: fset.Position(call.Pos()).Line, Package: typeLabel(tv.Type, pkg), Func: sel.Sel.Name, IsPhantom: !selected})
		return true
	})
	return findings, nil
}

func typeLabel(t types.Type, currentPkg *types.Package) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil {
			return ""
		}
		if currentPkg != nil && p.Path() == currentPkg.Path() {
			return ""
		}
		return p.Name()
	})
}
