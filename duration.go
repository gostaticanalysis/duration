package duration

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"text/template"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "duration finds using untyped constant as time.Duration"

// Analyzer finds using untyped constant as time.Duration.
var Analyzer = &analysis.Analyzer{
	Name: "duration",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

var tmpl = template.Must(template.New("a.go").Parse(`package a
import "time"
var _ time.Duration = 0 // dummy
func f() {
{{- range $i, $expr := .}}
	var v{{$i}} = {{$expr}}
	_ = v{{$i}}
{{end -}}
}
`))

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	typDuration := analysisutil.TypeOf(pass, "time", "Duration")
	if typDuration == nil {
		// skip
		return nil, nil
	}

	var (
		strExprs []string
		exprs    []ast.Expr
	)

	inspect.Nodes(nil, func(n ast.Node, push bool) bool {
		if !push {
			return false
		}

		expr, ok := n.(ast.Expr)
		if !ok {
			return true
		}

		switch expr.(type) {
		case *ast.Ident, *ast.SelectorExpr, *ast.CallExpr:
			return false
		}

		tv := pass.TypesInfo.Types[expr]
		if tv.Value == nil ||
			!types.Identical(tv.Type, typDuration) {
			return false
		}

		fset := token.NewFileSet()
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, expr); err != nil {
			return false
		}

		exprs = append(exprs, expr)
		strExprs = append(strExprs, buf.String())

		return false
	})

	var src bytes.Buffer
	if err := tmpl.Execute(&src, strExprs); err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "a.go", &src, 0)
	if err != nil {
		return nil, err
	}

	info := &types.Info{
		Defs: map[*ast.Ident]types.Object{},
	}
	config := &types.Config{
		Importer: importer.Default(),
	}

	pkg, err := config.Check("a", fset, []*ast.File{f}, info)
	if err != nil {
		return nil, err
	}

	done := map[token.Pos]bool{}
	dt := getDurationType(pkg)
	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 {
			return true
		}

		ident := spec.Names[0]
		if ident.Name[0] != 'v' {
			return true
		}

		t := info.TypeOf(ident)
		if t == nil || types.Identical(t, dt) {
			return true
		}

		idx, err := strconv.Atoi(ident.Name[1:])
		if err != nil {
			return true
		}

		pos := exprs[idx].Pos()
		if done[pos] {
			return true
		}
		done[pos] = true
		pass.Reportf(pos, "must not use untyped constant as a time.Duration type")

		return true
	})

	return nil, nil
}

func getDurationType(pkg *types.Package) types.Type {
	for _, p := range pkg.Imports() {
		if p.Path() != "time" {
			continue
		}

		obj := p.Scope().Lookup("Duration")
		if obj == nil {
			return nil
		}

		return obj.Type()
	}
	return nil
}
