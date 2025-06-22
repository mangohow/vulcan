package dbparser

import (
	"reflect"

	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
)

var (
	// 白名单类型, 除了基本类型之外, 这些类型也可以作为查询返回的结构体的字段
	registeredWhitelistTypes = map[string]map[string]reflect.Kind{
		"time": {
			"Time": reflect.Struct,
		},
		"database/sql": {
			"NullString":  reflect.Struct,
			"NullInt64":   reflect.Struct,
			"NullInt32":   reflect.Struct,
			"NullInt16":   reflect.Struct,
			"NullByte":    reflect.Struct,
			"NullFloat64": reflect.Struct,
			"NullBool":    reflect.Struct,
			"NullTime":    reflect.Struct,
			"Null":        reflect.Struct,
		},
	}
)

func isTypeInWhitelist(param *types.Param) bool {
	if param == nil {
		return false
	}

	valueType := param.Type.GetValueType()

	if valueType.IsBasicType() || (valueType.IsInterface() && valueType.Name == "interface{}") {
		return true
	}

	if valueType.Package == nil {
		return false
	}

	if valueType.Package.PackagePath == "" {
		return false
	}

	typeMap, ok := registeredWhitelistTypes[valueType.Package.PackagePath]
	if !ok {
		return false
	}

	kind, ok := typeMap[valueType.Name]
	if !ok {
		return false
	}

	return kind == valueType.Kind
}
