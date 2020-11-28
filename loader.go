package simplemock

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type loadFunc func(pkgname string, node ast.Node, info typeInfo, err error) error

func load(patterns []string, f loadFunc) error {
	var err error
	conf := &packages.Config{
		Mode: packages.NeedName | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
	}
	loaded, err := packages.Load(conf, patterns...)
	if err != nil {
		return fmt.Errorf("load package error: %w", err)
	}
	if len(loaded) == 0 {
		return errors.New("not found package")
	} else if len(loaded) > 1 {
		return errors.New("you should only 1 package")
	}
	pkg := loaded[0]
	for _, file := range pkg.Syntax {
		err = f(pkg.Name, file, pkg.TypesInfo, err)
	}

	return nil
}

type typeInfo interface {
	TypeOf(e ast.Expr) types.Type
}

type walkFunc func(iface string, ifaceType *types.Interface, err error) error

func walk(node ast.Node, info typeInfo, f walkFunc) error {
	var err error
	ast.Inspect(node, func(node ast.Node) bool {
		switch t := node.(type) {
		case *ast.TypeSpec:
			// only public
			if t.Name.IsExported() {
				switch v := t.Type.(type) {
				case *ast.InterfaceType:
					ifaceType, ok := info.TypeOf(v).Underlying().(*types.Interface)
					if ok {
						err = f(t.Name.Name, ifaceType, err)
					}
				}
			}
		}
		return true
	})

	return err
}
