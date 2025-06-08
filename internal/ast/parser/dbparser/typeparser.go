package dbparser

import (
	"github.com/mangohow/mangokit/tools/math"
	"github.com/mangohow/mangokit/tools/stream"
	"github.com/mangohow/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/internal/errors"
	"github.com/mangohow/vulcan/internal/utils"
	"go/ast"
	"reflect"
	"strings"
)

type TypeInfo struct {
	AstType *parser.TypeInfo
	Type    *types.TypeSpec
}

var kindNames = map[string]reflect.Kind{
	"bool":    reflect.Bool,
	"int":     reflect.Int,
	"int8":    reflect.Int8,
	"int16":   reflect.Int16,
	"int32":   reflect.Int32,
	"int64":   reflect.Int64,
	"uint":    reflect.Uint,
	"uint8":   reflect.Uint8,
	"uint16":  reflect.Uint16,
	"uint32":  reflect.Uint32,
	"uint64":  reflect.Uint64,
	"float32": reflect.Float32,
	"float64": reflect.Float64,
	"string":  reflect.String,
}

var unsupportedKind = map[string]reflect.Kind{
	"uintptr":    reflect.Uintptr,
	"complex64":  reflect.Complex64,
	"complex128": reflect.Complex128,
}

// 无需解析的一些类型，比如time.Time
var innerType = map[[2]string][]innerTypeInfo{
	// [2]string{absPkg, shortPkg}: typeName
	[2]string{"time", "time"}: {
		{
			typeName: "Time",
			kind:     reflect.Struct,
		},
	},
	[2]string{"github.com/jmoiron/sqlx", "sqlx"}: {
		{
			typeName: "DB",
			kind:     reflect.Struct,
		},
	},
	[2]string{"github.com/mangohow/vulcan", "vulcan"}: {
		{
			typeName: "Page",
			kind:     reflect.Interface,
		},
	},
}

type TypeCache map[string]map[string]*TypeInfo

func (c TypeCache) get(pkgPath, typeName string) *TypeInfo {
	if cc, ok := c[pkgPath]; ok {
		return cc[typeName]
	}

	return nil
}

func (c TypeCache) set(pkgPath, typeName string, info *TypeInfo) {
	cc, ok := c[pkgPath]
	if !ok {
		cc = make(map[string]*TypeInfo)
		c[pkgPath] = cc
	}
	cc[typeName] = info
}

type innerTypeInfo struct {
	typeName string
	kind     reflect.Kind
}

type TypeParser struct {
	dependencyManager *parser.DependencyManager
	typeCache         TypeCache
}

func NewTypeParser(manager *parser.DependencyManager) *TypeParser {
	return &TypeParser{
		dependencyManager: manager,
		typeCache:         make(TypeCache),
	}
}

func (p *TypeParser) GetTypeInfo(filePath, pkgPath, typeName string) (*TypeInfo, error) {
	typeInfo := p.typeCache.get(filePath, typeName)
	if typeInfo != nil {
		return typeInfo, nil
	}

	parsedType, err := p.dependencyManager.GetTypeInfo(filePath, pkgPath, typeName)
	if err != nil {
		return nil, err
	}
	st := &types.TypeSpec{
		Package: &types.PackageInfo{
			PackageName: utils.GetPackageName(pkgPath),
			PackagePath: parsedType.PackagePath,
			FilePath:    parsedType.FilePath,
		},
	}
	err = p.parseAstType(parsedType, st)
	if err != nil {
		return nil, errors.Errorf("%v, typeName: %s", err, typeName)
	}

	typeInfo = &TypeInfo{
		AstType: parsedType,
		Type:    st,
	}

	p.typeCache.set(filePath, typeName, typeInfo)

	return typeInfo, nil
}

func (p *TypeParser) parseAstType(typeInfo *parser.TypeInfo, st *types.TypeSpec) error {
	spec := typeInfo.AstType
	if spec.Name != nil {
		st.Name = spec.Name.Name
	}

	err := p.parseFieldExpr(spec.Type, typeInfo, st)
	if err != nil {
		return err
	}

	return nil
}

func (p *TypeParser) parseFieldExpr(expr ast.Expr, typeInfo *parser.TypeInfo, ts *types.TypeSpec) error {
	var err error
	switch at := expr.(type) {
	case *ast.Ident:
		err = p.parseBasicType(at, ts, typeInfo)
	case *ast.StructType:
		err = p.parseStructType(at, ts, typeInfo)
	case *ast.ArrayType:
		err = p.parseArrType(at, ts, typeInfo)
	case *ast.StarExpr:
		err = p.parseStarType(at, ts, typeInfo)
	case *ast.SelectorExpr:
		id, ok := at.X.(*ast.Ident)
		if !ok {
			return errors.Errorf("unsupported type")
		}
		shortPkgName := id.Name
		typeName := at.Sel.Name
		pkgInfo, _ := utils.Find(typeInfo.Imports, func(info types.ImportInfo) bool {
			return (info.Name != "" && info.Name == shortPkgName) || (utils.GetPackageName(info.AbsPackagePath) == shortPkgName)
		})
		allTypes, ok := innerType[[2]string{pkgInfo.AbsPackagePath, shortPkgName}]
		if !ok {
			return errors.Errorf("unsupported type")
		}
		foundType, ok := utils.Find(allTypes, func(info innerTypeInfo) bool {
			return info.typeName == typeName
		})
		if !ok {
			return errors.Errorf("unsupported type")
		}

		ts.Kind = foundType.kind
		ts.Name = typeName
		ts.Package = &types.PackageInfo{
			PackageName: pkgInfo.Name,
			PackagePath: pkgInfo.AbsPackagePath,
		}
	case *ast.InterfaceType:
		shortPkgName := ts.Package.PackageName
		pkgPath := ts.Package.PackagePath
		allTypes, _ := innerType[[2]string{pkgPath, shortPkgName}]
		foundType, ok := utils.Find(allTypes, func(info innerTypeInfo) bool {
			return info.typeName == ts.Name
		})
		if !ok {
			return errors.Errorf("unsupported type: interface")
		}
		ts.Kind = foundType.kind
	case *ast.ChanType:
		return errors.Errorf("unsupported type: chan")
	case *ast.FuncType:
		return errors.Errorf("unsupported type: func")
	case *ast.MapType:
		return errors.Errorf("unsupported type: map")
	default:
		return errors.Errorf("unsupported type")
	}

	if err != nil {
		return err
	}

	return nil
}

// parseBasicType 解析基础类型
//
// 参数:
//
//	info: 标识符节点，包含类型名称
//	typeSpec: 类型规范对象，用于存储解析后的类型信息
//
// 返回值:
//
//	error: 如果类型不支持或解析过程中发生错误，则返回错误信息；否则返回nil
func (p *TypeParser) parseBasicType(info *ast.Ident, typeSpec *types.TypeSpec, typeInfo *parser.TypeInfo) error {
	name := info.Name
	if _, ok := unsupportedKind[name]; ok {
		return errors.Errorf("unsupported type %s", name)
	}

	rt, ok := kindNames[name]
	if ok {
		typeSpec.Kind = rt
		typeSpec.Name = name
		return nil
	}

	// 可能是结构体
	// 目前, 先不允许结构体嵌套结构体
	if info.Obj == nil {
		return errors.Errorf("unsupport type: %s", name)
	}

	obj := info.Obj
	if obj.Decl == nil {
		return errors.Errorf("unsupport type: %s", name)
	}
	innerTypeSpec, ok := obj.Decl.(*ast.TypeSpec)
	if !ok {
		return errors.Errorf("unsupport type: %s", name)
	}

	structType, ok := innerTypeSpec.Type.(*ast.StructType)
	if !ok {
		return errors.Errorf("unsupport type: %s", name)
	}
	typeSpec.Name = innerTypeSpec.Name.Name

	return p.parseStructType(structType, typeSpec, typeInfo)
}

func (p *TypeParser) parseStructType(info *ast.StructType, typeSpec *types.TypeSpec, typeInfo *parser.TypeInfo) error {
	typeSpec.Kind = reflect.Struct
	if info.Fields == nil || len(info.Fields.List) == 0 {
		return nil
	}

	var err error
	params := stream.Map(info.Fields.List, func(field *ast.Field) []*types.Param {
		n := math.Max(1, len(field.Names))
		params := make([]*types.Param, n)
		if len(field.Names) == 0 {
			params[0] = &types.Param{}
		}
		for i := 0; i < len(field.Names); i++ {
			params[i] = &types.Param{
				Name: field.Names[i].Name,
			}
		}
		if field.Tag != nil {
			stream.ForEach(params, func(param *types.Param) bool {
				param.Type.Tag = reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				return true
			})
		}

		err = p.parseFieldExpr(field.Type, typeInfo, &params[0].Type)
		if err != nil {
			return nil
		}

		stream.ForEach(params[1:], func(param *types.Param) bool {
			param.Type = params[0].Type
			return true
		})

		return params
	})

	if err != nil {
		return err
	}

	typeSpec.Fields = stream.Flatten(params...)

	return nil
}

func (p *TypeParser) parseArrType(info *ast.ArrayType, typeSpec *types.TypeSpec, typeInfo *parser.TypeInfo) error {
	typeSpec.Kind = reflect.Slice
	typeSpec.ValueType = &types.TypeSpec{}
	return p.parseFieldExpr(info.Elt, typeInfo, typeSpec.ValueType)
}

func (p *TypeParser) parseStarType(info *ast.StarExpr, typeSpec *types.TypeSpec, typeInfo *parser.TypeInfo) error {
	typeSpec.Kind = reflect.Pointer
	typeSpec.ValueType = &types.TypeSpec{}
	return p.parseFieldExpr(info.X, typeInfo, typeSpec.ValueType)
}
