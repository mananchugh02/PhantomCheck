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
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: importer.Default(), Error: func(error) {}}
	if _, err := conf.Check("sample", fset, []*ast.File{file}, info); err != nil && len(info.Uses) == 0 {
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
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		pkgName, ok := info.Uses[ident].(*types.PkgName)
		if !ok {
			return true
		}
		obj := pkgName.Imported().Scope().Lookup(sel.Sel.Name)
		findings = append(findings, Finding{Line: fset.Position(call.Pos()).Line, Package: ident.Name, Func: sel.Sel.Name, IsPhantom: obj == nil || !obj.Exported()})
		return true
	})
	return findings, nil
}
