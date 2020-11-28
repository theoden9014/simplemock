package simplemock_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"testing"

	"github.com/theoden9014/simplemock"
)

func TestTypeZeroValue(t *testing.T) {
	tests := []struct {
		name string
		decl string
		want string
	}{
		// Check default value of "test" type
		{
			name: "int",
			decl: `type test int`,
			want: `0`,
		},
		{
			name: "int64",
			decl: `type test int64`,
			want: `0`,
		},
		{
			name: "bool",
			decl: `type test bool`,
			want: `false`,
		},
		{
			name: "string",
			decl: `type test string`,
			want: `""`,
		},
		{
			name: "struct",
			decl: `type test struct {}`,
			want: `test{}`,
		},
		{
			name: "pointer",
			decl: `type test *struct {}`,
			want: `nil`,
		},
		{
			name: "slice",
			decl: `type test []int`,
			want: `nil`,
		},
		{
			name: "map",
			decl: `type test map[string]string`,
			want: `nil`,
		},
		{
			name: "external package struct",
			decl: `import "net/http"; type test http.Client`,
			want: `test{}`,
		},
		{
			name: "external package",
			decl: `import "net/http"; var test http.Client`,
			want: `http.Client{}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			fmt.Fprintln(buf, "package main")
			fmt.Fprintln(buf, tt.decl)
			f, info, err := parseGoCode(t, token.NewFileSet(), buf)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(node ast.Node) bool {
				switch v := node.(type) {
				case *ast.Ident:
					if v.Name == "test" {
						value := simplemock.TypeZeroValue(info.TypeOf(v))
						if got := value; got != tt.want {
							t.Errorf("TypeZeroValue() = %v, want %v", got, tt.want)
						}
						return true
					}
				}
				return true
			})

		})
	}
}

func parseGoCode(tb testing.TB, fset *token.FileSet, src io.Reader) (*ast.File, *types.Info, error) {
	f, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse go file: %w", err)
	}

	cfg := &types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			tb.Fatalf("check err=%v", err)
		},
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	if _, err := cfg.Check("", fset, []*ast.File{f}, info); err != nil {
		return nil, nil, err
	}

	return f, info, nil
}
