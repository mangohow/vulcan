package dbparser

import (
	"fmt"
	"github.com/mangohow/gowlb/tools/collection"
	"github.com/mangohow/gowlb/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/command"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils/stringutils"
	"go/ast"
	astparser "go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

const (
	tablePropertyName     = "TableProperty"
	annotationPackageName = "annotation"

	tableNameKey = "tableName"
	genKey       = "gen"
)

type ModelStructParser struct {
	fst               *token.FileSet
	manager           *parser.DependencyManager
	options           *command.CommandOptions
	necessaryPackages []string
}

func NewModelStructParser(fst *token.FileSet, manager *parser.DependencyManager, options *command.CommandOptions) *ModelStructParser {
	return &ModelStructParser{
		fst:     fst,
		manager: manager,
		options: options,
		necessaryPackages: []string{
			"github.com/mangohow/vulcan/annotation",
		},
	}
}

func (p *ModelStructParser) Parse() ([]*types.ModelSpec, error) {
	source, err := os.ReadFile(p.options.File)
	if err != nil {
		return nil, errors.Errorf("reading file failed, reason: %v", err)
	}
	source = utils.TrimLineWithPrefix(source, []byte("//go:build "), []byte("// +build"), []byte("//go:generate"))
	f, err := astparser.ParseFile(p.fst, "", source, astparser.ParseComments)
	if err != nil {
		return nil, errors.Errorf("parse source file failed, reason: %s", err)
	}

	packageInfo := parseImports(f, p.options.File)
	// 检查是否导入了必要的包
	if err := checkNecessaryPackageImport(p.necessaryPackages, packageInfo.Imports); err != nil {
		return nil, err
	}

	// 过滤合法的结构体model声明
	astTypeSpecs := make([]*ast.TypeSpec, 0)
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// 匿名类型
			if ts.Name == nil || ts.Name.Name == "" {
				continue
			}

			// 未导出的
			if !stringutils.IsUpperLetter(ts.Name.Name[:1]) {
				continue
			}

			structType, ok := ts.Type.(*ast.StructType)
			// 非结构体声明
			if !ok {
				continue
			}

			// 结构体中没有字段
			if structType.Fields == nil || len(structType.Fields.List) <= 1 {
				continue
			}

			astTypeSpecs = append(astTypeSpecs, ts)
		}
	}

	if len(astTypeSpecs) == 0 {
		return nil, IgnoreError
	}

	modelSpecList := make([]*types.ModelSpec, 0, len(astTypeSpecs))
	for _, astTypeSpec := range astTypeSpecs {
		modelSpec, err := p.parseStructDecls(astTypeSpec)
		if err != nil && err != IgnoreError {
			return nil, errors.Wrapf(err, "parse model %s failed", astTypeSpec.Name.Name)
		}

		if err != IgnoreError {
			modelSpecList = append(modelSpecList, modelSpec)
		}
	}

	return modelSpecList, nil
}

var (
	IgnoreError = errors.Errorf("no error")
)

func (p *ModelStructParser) parseStructDecls(typeSpec *ast.TypeSpec) (*types.ModelSpec, error) {
	modelSpec := &types.ModelSpec{
		ModelName: typeSpec.Name.Name,
	}

	if err := p.fillMoreInfo(modelSpec); err != nil {
		return nil, err
	}

	var (
		structSpec = typeSpec.Type.(*ast.StructType)
		fieldName  string
	)

	field0 := structSpec.Fields.List[0]
	switch ft := field0.Type.(type) {
	case *ast.SelectorExpr:
		fieldName = ft.Sel.Name
	case *ast.Ident:
		fieldName = ft.Name
	}

	if fieldName != tablePropertyName {
		return nil, IgnoreError
	}

	if field0.Tag == nil || field0.Tag.Value == "" {
		return nil, errors.Errorf("The tag in the TableProperty field of the model struct %s is null", modelSpec.ModelName)
	}

	// 先解析model中对应数据库的字段
	modelFields, err := p.parseColumns(modelSpec.ModelName, structSpec.Fields.List[1:])
	if err != nil {
		return nil, err
	}
	// 寻找主键
	primaryKeys := stream.Filter(modelFields, func(field *types.ModelField) bool {
		return field.IsPrimaryKey
	})
	if len(primaryKeys) > 1 {
		return nil, errors.Errorf("invalid multiple primary key field in model struct %s", modelSpec.ModelName)
	}
	if len(primaryKeys) != 0 {
		modelSpec.PrimaryKey = primaryKeys[0]
	}
	modelSpec.ModelFields = modelFields

	tableName, genFuncSpec, err := p.parseTablePropertyTag(field0.Tag.Value, modelSpec.ModelName, modelFields, modelSpec.PrimaryKey != nil)
	if err != nil {
		return nil, err
	}
	modelSpec.FuncSpecs = genFuncSpec
	modelSpec.TableName = tableName

	return modelSpec, nil
}

func (p *ModelStructParser) parseColumns(modelName string, fields []*ast.Field) ([]*types.ModelField, error) {
	var err error
	modelFields := stream.Map(fields, func(field *ast.Field) *types.ModelField {
		if err != nil {
			return nil
		}

		if len(field.Names) == 0 || len(field.Names) > 1 {
			err = errors.Errorf("invalid anonymous field or repeated field")
			return nil
		}

		res := &types.ModelField{
			Name: field.Names[0].Name,
		}

		isGenericType := false
		switch ft := field.Type.(type) {
		case *ast.Ident:
			res.Type = ft.Name
		case *ast.SelectorExpr:
			ident, ok := ft.X.(*ast.Ident)
			if !ok {
				err = errors.Errorf("invalid type in field %s", res.Name)
				return nil
			}
			res.Type = ident.Name + "." + ft.Sel.Name
		case *ast.ArrayType:
			ident, ok := ft.Elt.(*ast.Ident)
			if !ok || ident.Name != "byte" {
				err = errors.Errorf("invalid type in field %s", res.Name)
				return nil
			}
			res.Type = "[]byte"
		case *ast.IndexExpr: // 泛型类型
			isGenericType = true
			selector, ok := ft.X.(*ast.SelectorExpr)
			if !ok {
				err = errors.Errorf("invalid type in field %s", res.Name)
				return nil
			}
			if x, ok := selector.X.(*ast.Ident); !ok || x.Name != "sql" || selector.Sel.Name != "Null" {
				err = errors.Errorf("invalid type in field %s", res.Name)
				return nil
			}
			switch indexName := ft.Index.(type) {
			case *ast.SelectorExpr:
				x, ok := indexName.X.(*ast.Ident)
				if !ok {
					err = errors.Errorf("invalid type in field %s", res.Name)
					return nil
				}
				res.Type = fmt.Sprintf("sql.Null[%s.%s]", x.Name, indexName.Sel.Name)
			case *ast.Ident:
				res.Type = fmt.Sprintf("sql.Null[%s]", indexName.Name)
			default:
				err = errors.Errorf("invalid type in field %s", res.Name)
				return nil
			}
		default:
			err = errors.Errorf("type invalid, model struct %s, field name %s", modelName, res.Name)
			return nil
		}

		// 对Type进行校验
		if !types.IsTypeSupported(res.Type, isGenericType) {
			err = errors.Errorf("type %s of field %s is not supported in model struct %s", res.Type, res.Name, modelName)
			return nil
		}

		if field.Tag == nil || field.Tag.Value == "" {
			err = errors.Errorf("field %s has no db tag", res.Name)
			return nil
		}

		structTag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		dbTag := structTag.Get("db")
		if dbTag == "" {
			err = errors.Errorf("field %s has empty db tag", res.Name)
			return nil
		}

		res.ColumnName = dbTag
		dbTagVals := strings.Split(dbTag, ",")
		stream.Map(dbTagVals, func(t string) string {
			return strings.TrimSpace(t)
		})
		if len(dbTagVals) > 1 {
			res.ColumnName = dbTagVals[0]
		}
		if len(dbTagVals) > 1 && utils.Contains(dbTagVals[1:], "pk") {
			res.IsPrimaryKey = true
		}
		if len(dbTagVals) > 1 && utils.Contains(dbTagVals[1:], "auto_incr") {
			res.IsAutoIncrement = true
		}

		return res
	})

	if err != nil {
		return nil, err
	}

	return modelFields, nil
}

var (
	defaultGenFuncs = []string{
		"Add", "AddBatch", "SelectCount", "SelectAll", "DeleteById", "DeleteBatchIds", "SelectById", "SelectBatchIds",
	}
)

func (p *ModelStructParser) parseTablePropertyTag(tag, modelName string, modelFields []*types.ModelField, hasPrimaryKey bool) (string, []*types.GenFuncSpec, error) {
	// 解析tag
	propertyTag := reflect.StructTag(strings.Trim(tag, "`"))
	tabName := propertyTag.Get(tableNameKey)
	if tabName == "" {
		return "", nil, errors.Errorf("tableName tag is not specified, model struct is %s", modelName)
	}
	genTag := propertyTag.Get(genKey)
	if genTag == "" {
		// 添加默认生成的CRUD函数
		defaultGen := defaultGenFuncs
		if !hasPrimaryKey {
			defaultGen = defaultGenFuncs[:3]
		}
		res := make([]*types.GenFuncSpec, 0, len(defaultGen))
		for _, name := range defaultGen {
			res = append(res, &types.GenFuncSpec{
				FuncName:    name,
				KeyFuncName: name,
			})
		}

		return tabName, res, nil
	}

	genFunDecls := strings.Split(genTag, "|")
	genFuncMap := make(map[string]*types.GenFuncSpec)
	for _, genFunDecl := range genFunDecls {
		genFunDecl = strings.TrimSpace(genFunDecl)
		funcName, argStr, found := strings.Cut(genFunDecl, "(")
		if !found {
			switch funcName {
			case "Add", "AddBatch", "SelectCount", "SelectAll":
			case "DeleteById", "DeleteBatchIds", "SelectById", "SelectBatchIds":
				if !hasPrimaryKey {
					return "", nil, errors.Errorf("model %s has no primary key field, can not generate funcs select by id", modelName)
				}
			default:
				return "", nil, errors.Errorf("unsupported function %s without args", funcName)
			}
			genFuncMap[funcName] = &types.GenFuncSpec{
				FuncName:    funcName,
				KeyFuncName: funcName,
			}
			continue
		}

		argStr = strings.Trim(argStr, ") ")
		args := strings.Split(argStr, ",")
		args = stream.Map(args, func(arg string) string {
			return strings.TrimSpace(arg)
		})

		var (
			genFuncSpec *types.GenFuncSpec
			err         error
			keyFuncName = funcName
		)
		switch {
		case funcName == "DeleteBy":
			if len(args) != 1 {
				return "", nil, errors.Errorf("DeleteBy must have 1 parameters")
			}
			genFuncSpec, err = p.parseGenFuncArgs(args[0], "", "", "", modelFields)
		case funcName == "UpdateById":
			if len(args) != 2 {
				return "", nil, errors.Errorf("UpdateById must have 2 parameters")
			}
			genFuncSpec, err = p.parseGenFuncArgs("", args[0], "", args[1], modelFields)
		case strings.HasPrefix(funcName, "UpdateBy"):
			if len(args) != 4 {
				return "", nil, errors.Errorf("%s must have 4 parameters", funcName)
			}
			keyFuncName = "UpdateBy"
			genFuncSpec, err = p.parseGenFuncArgs(args[0], args[1], args[2], args[3], modelFields)
		case strings.HasPrefix(funcName, "SelectOneBy"),
			strings.HasPrefix(funcName, "SelectListBy"),
			strings.HasPrefix(funcName, "SelectCountBy"),
			strings.HasPrefix(funcName, "SelectPageBy"):
			if len(args) != 3 {
				return "", nil, errors.Errorf("%s must have 4 parameters", funcName)
			}
			index := strings.Index(funcName, "By")
			keyFuncName = funcName[:index+2]
			genFuncSpec, err = p.parseGenFuncArgs(args[0], args[1], args[2], "", modelFields)
		default:
			return "", nil, errors.Errorf("unsupported function %s", funcName)
		}

		if err != nil {
			return "", nil, errors.Wrapf(err, "parse %s failed", funcName)
		}

		genFuncSpec.FuncName = funcName
		genFuncSpec.KeyFuncName = keyFuncName
		genFuncMap[funcName] = genFuncSpec
	}

	// 添加默认生成的CRUD函数
	defaultGen := defaultGenFuncs
	if !hasPrimaryKey {
		defaultGen = defaultGenFuncs[:3]
	}

	for _, name := range defaultGen {
		genFuncMap[name] = &types.GenFuncSpec{
			FuncName:    name,
			KeyFuncName: name,
		}
	}

	return tabName, collection.Values(genFuncMap), nil
}

func (p *ModelStructParser) parseGenFuncArgs(whereArg, selectArg, selectValidate, setValidate string, modelFields []*types.ModelField) (*types.GenFuncSpec, error) {
	res := &types.GenFuncSpec{}
	if whereArg != "" {
		parts := strings.Split(whereArg, "&")
		parts = stream.Map(parts, func(part string) string {
			return strings.TrimSpace(part)
		})
		set := collection.NewSet[types.Pair[string, string]]()
		for _, part := range parts {
			var (
				indexes []string
				key     = "AND"
			)
			switch {
			case strings.HasPrefix(part, "AND[") && strings.HasSuffix(part, "]"):
				indexes = strings.Split(part[4:len(part)-1], " ")
			case strings.HasPrefix(part, "OR[") && strings.HasSuffix(part, "]"):
				indexes = strings.Split(part[3:len(part)-1], " ")
				key = "OR"
			case strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]"):
				indexes = strings.Split(part[1:len(part)-1], " ")
			default:
				return nil, errors.Errorf("invalid where arg %s", whereArg)
			}

			err := parseIndexToColumnName(indexes, "where", key, set, modelFields)
			if err != nil {
				return nil, err
			}

			set.ForEach(func(v types.Pair[string, string]) {
				res.WhereColumnNames = append(res.WhereColumnNames, v)
			})
		}
	}

	if selectArg != "" {
		if !(strings.HasPrefix(selectArg, "[") && strings.HasSuffix(selectArg, "]")) {
			return nil, errors.Errorf("invalid select arg %s", selectArg)
		}

		indexes := strings.Split(selectArg[1:len(selectArg)-1], " ")
		set := collection.NewSet[types.Pair[string, string]]()
		if err := parseIndexToColumnName(indexes, "select", "", set, modelFields); err != nil {
			return nil, err
		}

		set.ForEach(func(v types.Pair[string, string]) {
			res.SelectColumnNames = append(res.SelectColumnNames, v.Val)
		})
	}

	if selectValidate != "" {
		ok, err := strconv.ParseBool(selectValidate)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid select validate %s", selectValidate)
		}
		res.SelectValidateEmpty = ok
	}

	if setValidate != "" {
		ok, err := strconv.ParseBool(setValidate)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid set validate %s", setValidate)
		}
		res.SetValidateEmpty = ok
	}

	return res, nil
}

func parseIndexToColumnName(indexes []string, op, key string, set collection.Set[types.Pair[string, string]], modelFields []*types.ModelField) error {
	for _, index := range indexes {
		startEnd := strings.Split(index, "-")
		if ranged := (len(startEnd) == 1 && strings.Contains(index, "-")); len(startEnd) == 2 || ranged {
			start, err1 := strconv.Atoi(startEnd[0])
			var (
				end  = len(modelFields) - 1
				err2 error
			)
			if !ranged {
				end, err2 = strconv.Atoi(startEnd[1])
			}
			// 可能是字段名, 找到对应的索引
			if err1 != nil {
				start = findIndexByName(startEnd[0], modelFields)
				if start == -1 {
					return errors.Errorf("invalid %s arg index", op)
				}
			}
			if err2 != nil {
				end = findIndexByName(startEnd[1], modelFields)
				if end == -1 {
					return errors.Errorf("invalid %s arg index", op)
				}
			}

			if start < 0 || start >= len(modelFields) || start > end || end >= len(modelFields) {
				return errors.Errorf("invalid %s arg index", op)
			}

			for ; start <= end; start++ {
				set.Add(types.Pair[string, string]{key, modelFields[start].ColumnName})
			}
			continue
		}
		idx, err := strconv.Atoi(index)
		if err != nil {
			idx = findIndexByName(index, modelFields)
			if idx == -1 {
				return errors.Errorf("invalid %s arg %s", op, index)
			}
		}
		if idx < 0 || idx >= len(modelFields) {
			return errors.Errorf("invalid %s arg %s", op, index)
		}
		set.Add(types.Pair[string, string]{key, modelFields[idx].ColumnName})
	}

	return nil
}

func findIndexByName(name string, modelFields []*types.ModelField) int {
	for i, field := range modelFields {
		if name == field.Name || name == field.ColumnName {
			return i
		}
	}

	return -1
}

func (p *ModelStructParser) fillMoreInfo(spec *types.ModelSpec) error {
	modelFile := p.options.File
	abs, err := filepath.Abs(modelFile)
	if err != nil {
		return err
	}

	modelDir := filepath.Dir(abs)
	// 获取model的包名
	packageName, err := utils.GetPackageNameByDir(modelDir)
	if err != nil {
		return errors.Wrapf(err, "get model package name error")
	}

	spec.PackageName = packageName
	spec.FilePath = abs

	// 获取model包导入路径
	cwd, err := os.Getwd()
	if err != nil {
		return errors.Wrapf(err, "get working dir error")
	}
	if err := os.Chdir(modelDir); err != nil {
		return errors.Wrapf(err, "change working dir error")
	}
	output, err := exec.Command("go", "list", "-f", "{{.ImportPath}}").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "get model import package path error")
	}
	spec.ImportPath = strings.Trim(string(output), "\r\n ")
	if err := os.Chdir(cwd); err != nil {
		return errors.Wrapf(err, "change working dir error")
	}

	return nil
}
