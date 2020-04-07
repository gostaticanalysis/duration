package duration

import (
	"bytes"
	"go/ast"
	"go/constant"
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
	"golang.org/x/tools/go/ast/astutil"
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
		exprPos  []token.Pos
	)

	inspect.Nodes(nil, func(n ast.Node, push bool) bool {
		if !push {
			return false
		}

		expr, ok := n.(ast.Expr)
		if !ok {
			return true
		}

		switch expr := expr.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			return false
		case *ast.CallExpr:
			if tv := pass.TypesInfo.Types[expr]; tv.Value != nil {
				return false
			}
			for _, arg := range expr.Args {
				tv := pass.TypesInfo.Types[arg]
				if tv.Value != nil &&
					types.Identical(tv.Type, typDuration) {
					exprPos = append(exprPos, arg.Pos())
					strExprs = append(strExprs, exprToString(expandNamedConstAll(pass, arg)))
				}
			}
			return false
		}

		tv := pass.TypesInfo.Types[expr]
		if tv.Value == nil ||
			!types.Identical(tv.Type, typDuration) {
			return false
		}

		exprPos = append(exprPos, expr.Pos())
		strExprs = append(strExprs, exprToString(expandNamedConstAll(pass, expr)))

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

		pos := exprPos[idx]
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

func exprToString(expr ast.Expr) string {
	fset := token.NewFileSet()
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

func expandNamedConstAll(pass *analysis.Pass, expr ast.Expr) ast.Expr {
	r, ok := astutil.Apply(expr, func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.Ident:
			tv := pass.TypesInfo.Types[n]
			if tv.Value != nil {
				v := expandNamedConst(pass, tv.Value)
				c.Replace(v)
			}
			return false
		}
		return true
	}, nil).(ast.Expr)

	if ok {
		return r
	}

	return nil
}

func expandNamedConst(pass *analysis.Pass, cnst constant.Value) ast.Expr {
	switch cnst.Kind() {
	case constant.Bool:
		return &ast.Ident{
			Name: cnst.String(),
		}
	case constant.String:
		return &ast.BasicLit{
			Kind:  token.STRING,
			Value: cnst.ExactString(),
		}
	case constant.Int:
		return &ast.BasicLit{
			Kind:  token.INT,
			Value: cnst.ExactString(),
		}
	case constant.Float:
		return &ast.BasicLit{
			Kind:  token.FLOAT,
			Value: cnst.ExactString(),
		}
	case constant.Complex:
		real := constant.Real(cnst)
		imag := constant.Imag(cnst)
		return &ast.BinaryExpr{
			X:  expandNamedConst(pass, real),
			Op: token.ADD,
			Y:  expandNamedConst(pass, imag),
		}
	}
	return nil
}
