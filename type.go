package simplemock

import (
	"go/types"
	"path/filepath"
)

// TypeZeroValue returns zero value of type.
func TypeZeroValue(t types.Type) string {
	switch v := t.Underlying().(type) {
	case *types.Basic:
		return typeBasicZeroValue(v)
	case *types.Struct:
		return types.TypeString(t, qualifier) + `{}`
	default:
		return `nil`
	}
}

func typeBasicZeroValue(basic *types.Basic) string {
	switch basic.Info() {
	case types.IsBoolean:
		return `false`
	case types.IsInteger:
		return `0`
	case types.IsUnsigned:
		return `0`
	case types.IsFloat:
		return `0`
	case types.IsComplex:
		return `0`
	case types.IsString:
		return `""`
	default:
		return `nil`
	}
}

func qualifier(pkg *types.Package) string {
	if pkg.Path() == "" {
		return ""
	}
	return filepath.Base(pkg.Path())
}

// TypeString convert types.Type to string
func TypeString(t types.Type) string {
	switch v := t.(type) {
	case *types.Array:
		return "[]" + TypeString(v.Elem())
	case *types.Slice:
		return "[]" + TypeString(v.Elem())
	case *types.Pointer:
		return "*" + TypeString(v.Elem())
	case *types.Map:
		return "map[" + TypeString(v.Key()) + "]" + TypeString(v.Elem())
	default:
		return types.TypeString(t, qualifier)
	}
}
