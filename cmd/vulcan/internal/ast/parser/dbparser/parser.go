package dbparser

import (
	"fmt"
	"go/ast"
	astparser "go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"

	"github.com/mangohow/gowlb/tools/collection"
	"github.com/mangohow/gowlb/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/astutils"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils/sqlutils"
)

const (
	dbOperatorPackageName = "database/sql"
	dbOperatorRefName     = "sql"
	dbOperatorTypeName    = "DB"
)

type FileParser struct {
	fst               *token.FileSet
	dependencyManager *parser.DependencyManager
	filterPackages    []string // 生成的代码中需要过滤掉的package
	addPackages       []string // 生成的代码中需要添加的package
	necessaryPackages []string // 必须要导入的包
	typeParser        *TypeParser
	typeDeclarations  []*ast.TypeSpec
}

func NewFileParser(fst *token.FileSet, dm *parser.DependencyManager) *FileParser {
	return &FileParser{
		fst:               fst,
		dependencyManager: dm,
		filterPackages: []string{
			"github.com/mangohow/vulcan/annotation",
		},
		addPackages: []string{
			"github.com/mangohow/vulcan",
		},
		necessaryPackages: []string{
			dbOperatorPackageName,
		},
		typeParser: NewTypeParser(dm),
	}
}

func (p *FileParser) Parse(filename string) (*types.File, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Errorf("reading file failed, reason: %v", err)
	}
	source = utils.TrimLineWithPrefix(source, []byte("//go:build "), []byte("// +build"), []byte("//go:generate"))
	f, err := astparser.ParseFile(p.fst, "", source, astparser.ParseComments)
	if err != nil {
		return nil, errors.Errorf("parse file %s failed, reason: %s", filename, err)
	}

	// 处理导入的包
	packageInfo := parseImports(f, filename)
	currentPkg, err := utils.GetCurrentPackagePath(filename)
	if err != nil {
		return nil, err
	}
	packageInfo.PackagePath = currentPkg

	if err := checkNecessaryPackageImport(p.necessaryPackages, packageInfo.Imports); err != nil {
		return nil, err
	}

	// 对包进行过滤, 如果没有包含注解相关的包, 则不生成代码
	for _, pkg := range p.filterPackages {
		if _, ok := packageInfo.ImportsMap[pkg]; !ok {
			return nil, errors.Errorf("package %s is not imported", pkg)
		}
	}

	fileInfo := &types.File{
		AstFile: f,
		PkgInfo: packageInfo,
	}
	// 处理类型、函数声明
	if err := p.parseDeclares(f, fileInfo); err != nil {
		return nil, errors.Wrapf(err, "parse declares failed")
	}

	for i := 0; i < len(fileInfo.Declarations); i++ {
		fileInfo.Declarations[i].PkgInfo = packageInfo
	}

	// 处理包导入信息, 过滤掉无用的包导入
	fileInfo.PkgInfo.AstImports = stream.Filter(fileInfo.PkgInfo.AstImports, func(spec *ast.ImportSpec) bool {
		return !utils.Contains(p.filterPackages, strings.Trim(spec.Path.Value, `"`))
	})
	// 增加需要导入的包
	fileInfo.PkgInfo.AstImports = append(fileInfo.PkgInfo.AstImports, stream.Map(p.addPackages, func(name string) *ast.ImportSpec {
		return &ast.ImportSpec{
			Path: astutils.BuildBasicLit(token.STRING, fmt.Sprintf("%q", name)),
		}
	})...)

	return fileInfo, nil
}

// 解析所有声明
func (p *FileParser) parseDeclares(af *ast.File, file *types.File) error {
	// 先解析出当前文件中的type定义
	for _, decl := range af.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			p.typeDeclarations = append(p.typeDeclarations, ts)
		}
	}

	for _, decl := range af.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// 在函数中寻找注解并解析
			declare, err := p.parseFuncDeclare(d, file.PkgInfo)
			if err != nil {
				return errors.Wrapf(err, "in func %s", d.Name.Name)
			}

			if declare != nil {
				file.AddDeclaration(decl, declare)
			} else {
				file.AddAstDecl(decl) // 如果没用注解, 则该函数无需处理, 直接添加ast
			}
		case *ast.GenDecl:
			if len(d.Specs) > 0 {
				if _, ok := d.Specs[0].(*ast.ImportSpec); ok {
					continue
				}
			}

			file.AddAstDecl(d)
			// TODO 解析类型定义

		default:
			file.AddAstDecl(decl) // 非函数声明或类型定义直接添加ast, 无需做任何修改
		}

	}

	return nil
}

// 解析函数声明
func (p *FileParser) parseFuncDeclare(fd *ast.FuncDecl, pkgInfo types.PackageInfo) (*types.FuncDecl, error) {
	res := &types.FuncDecl{
		InputParam:  make(map[string]*types.Param),
		OutputParam: make(map[string]*types.Param),
		FuncName:    fd.Name.Name,
	}
	annotationInfos := astutils.FindAnnotationsInFuncBody(types.SQLAnnotationFuncs, types.AnnotationPackageName, fd.Body, pkgInfo)
	// 如果找不到, 则无需为该函数生成样板代码
	if len(annotationInfos) == 0 {
		return nil, nil
	}

	// 校验注解
	if err := p.validateAnnotation(annotationInfos); err != nil {
		return nil, err
	}

	// 解析注解
	// 1. 先处理接收器
	// 如果没有接收器, 则返回错误
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return nil, errors.Errorf("func must have a receiver")
	}

	// 判断接收器是否为结构体, 并且包含sqlx.DB对象
	if err := p.checkReceiverInvalid(fd.Recv.List[0], res); err != nil {
		return nil, err
	}

	// 2. 处理入参
	if err := p.parseInputParameter(fd.Type.Params.List, res, pkgInfo); err != nil {
		return nil, err
	}

	// 3. 处理出参
	var outputFields []*ast.Field
	if fd.Type.Results != nil {
		outputFields = fd.Type.Results.List
	}
	if err := p.parseOutputParameter(outputFields, res, pkgInfo); err != nil {
		return nil, err
	}

	// 4. 处理注解调用
	if err := p.parseAnnotations(res); err != nil {
		return nil, err
	}

	return res, nil
}

func (p *FileParser) checkReceiverInvalid(receiver *ast.Field, fnDecl *types.FuncDecl) error {
	expr := receiver.Type
	name := ""
	fnDecl.Receiver = &types.Param{}
	receiverType := &fnDecl.Receiver.Type
loop:
	for {
		// 检查receiver中是否包含*sqlx.db
		switch typeSpec := expr.(type) {
		case *ast.StarExpr: // 指针接收者
			expr = typeSpec.X
			receiverType.Kind = reflect.Pointer
			receiverType.ValueType = &types.TypeSpec{
				Kind: reflect.Struct,
			}
			receiverType = receiverType.ValueType
		case *ast.Ident: // 值接收者
			name = typeSpec.Name
			break loop
		}
	}

	if len(receiver.Names) == 0 || receiver.Names[0].Name == "_" {
		// 生成receiver name
		receiver.Names = []*ast.Ident{ast.NewIdent(genReceiverName(name))}
	}
	fnDecl.Receiver.Name = receiver.Names[0].Name

	receiverType.Name = name

	ts, ok := utils.Find(p.typeDeclarations, func(spec *ast.TypeSpec) bool {
		return spec.Name.Name == name
	})
	if !ok {
		return errors.Errorf("type %s not found", name)
	}

	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return errors.Errorf("receiver must be struct type")
	}

	if st.Fields == nil {
		return errors.Errorf("type %s must have a field which type is *sqlx.DB", name)
	}

	for _, field := range st.Fields.List {
		starExpr, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		se, ok := starExpr.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		if ident, ok := se.X.(*ast.Ident); ok && ident.Name == dbOperatorRefName && se.Sel.Name == dbOperatorTypeName {
			if len(field.Names) == 0 || field.Names[0].Name == "_" {
				return errors.Errorf("the *sqlx.DB field of type %s must have a name", name)
			}

			receiverType.Fields = append(receiverType.Fields, &types.Param{
				Name: field.Names[0].Name,
				Type: types.TypeSpec{
					Kind: reflect.Pointer,
					ValueType: &types.TypeSpec{
						Name: dbOperatorTypeName,
						Package: &types.PackageInfo{
							PackageName: dbOperatorRefName,
							PackagePath: dbOperatorPackageName,
						},
						Kind: reflect.Struct,
					},
				},
			})

			return nil
		}
	}

	return errors.Errorf("type %s must have a field which type is *sqlx.DB", name)
}

func genReceiverName(name string) string {
	if name == "" {
		return "s"
	}

	return strings.ToLower(name[:1])
}

// 解析入参数据类型
func (p *FileParser) parseInputParameter(params []*ast.Field, res *types.FuncDecl, pkgInfo types.PackageInfo) error {
	var (
		err         error
		inputParams = make(map[string]*types.Param)
	)

	if err = stream.ForEachE(params, func(field *ast.Field) error {
		paramList := make([]*types.Param, len(field.Names))
		for i := range field.Names {
			paramList[i] = &types.Param{
				Name: field.Names[i].Name,
			}
		}

		var param types.Param
		if err = p.parseFieldExpr(field.Type, &param, pkgInfo); err != nil {
			return err
		}

		for i := range paramList {
			paramList[i].Type = param.Type
			inputParams[paramList[i].Name] = paramList[i]
		}

		return nil
	}); err != nil {
		return err
	}

	res.InputParam = inputParams

	return nil
}

func (p *FileParser) parseFieldExpr(expr ast.Expr, typeParam *types.Param, pkgInfo types.PackageInfo) error {
	var (
		typeSpec    = &typeParam.Type
		typeName    string
		typePkgName string
		typeInfo    *TypeInfo
		isPointer   = false
		err         error
	)

loop:
	for {
		switch et := expr.(type) {
		case *ast.Ident: // 类型名只包含一个标识符, 比如 int、string、User
			typeName = et.Name
			break loop
		case *ast.SelectorExpr: // 包名.类型名, 比如 model.User
			i, ok := et.X.(*ast.Ident)
			if !ok {
				return errors.Errorf("unsupported input parameter type: %s", typeParam.Name)
			}
			typeName = et.Sel.Name
			typePkgName = i.Name
			break loop
		case *ast.StarExpr: // 指针类型, 比如 *int、*model.User
			if isPointer { // 避免多级指针
				return errors.Errorf("unsupported multi level pointer, type: %s", typeParam.Name)
			}
			typeSpec.Kind = reflect.Pointer
			typeSpec.ValueType = &types.TypeSpec{}
			typeSpec = typeSpec.ValueType
			isPointer = true
			expr = et.X
		case *ast.ArrayType: // 切片或数组类型, 比如 []string
			if isPointer { // 避免切片指针
				return errors.Errorf("unsupported pointer slice type")
			}
			typeSpec.Kind = reflect.Slice
			typeSpec.ValueType = &types.TypeSpec{}
			typeSpec = typeSpec.ValueType
			expr = et.Elt
		default:
			return errors.Errorf("unsupported input parameter type: %s", typeParam.Name)
		}
	}

	if typePkgName == "" {
		// 普通类型
		if rn, ok := kindNames[typeName]; ok {
			typeSpec.Name = typeName
			typeSpec.Kind = rn
		} else {
			// 本包的结构体, 不允许
			return errors.Errorf("can't use types %s in %s", typeName, pkgInfo.PackagePath)
		}
	} else {
		// 其他包的结构体, 进行解析
		found, ok := utils.Find(pkgInfo.Imports, func(info types.ImportInfo) bool {
			return utils.GetPackageName(info.AbsPackagePath) == typePkgName
		})
		if !ok {
			return errors.Errorf("can't find %s.%s's type declartion", typePkgName, typeName)
		}

		typeInfo, err = p.typeParser.GetTypeInfo(pkgInfo.FilePath, found.AbsPackagePath, typeName)
		if err != nil {
			return errors.Wrapf(err, "type %s.%s invalid", typePkgName, typeName)
		}

		*typeSpec = *typeInfo.Type
	}

	return nil
}

// 解析出参数据类型
// 出参的数量为0~1个
// 如果有参数, 该参数可以为结构体、结构体指针、切片（切片元素为结构体、切片元素为结构体指针、基本类型）、基本类型）
// 参数不能为基本类型的指针, 没有意义
func (p *FileParser) parseOutputParameter(params []*ast.Field, fnDecl *types.FuncDecl, pkgInfo types.PackageInfo) error {
	if fnDecl.SQLAnnotation.Name == types.SQLSelectFunc && len(params) == 0 {
		return errors.Errorf("Select stmt must have return parameter")
	}

	if len(params) == 0 {
		return nil
	}

	// 检查出参类型
	// 查询时: 第一个为结构体、基本类型、切片或指针
	// 增删改: 第一个参数为Number类型(int, int64, ..., uint, uint64...)
	if len(params) > 1 {
		return errors.Errorf("function must return 0 or 1 results")
	}

	paramType := &types.Param{}
	resultField := params[0]
	if err := p.parseFieldExpr(resultField.Type, paramType, pkgInfo); err != nil {
		return err
	}

	fnDecl.FuncReturnResultParam = paramType

	if fnDecl.SQLAnnotation.Name == types.SQLSelectFunc {
		// 如果出参为结构体，对其中的字段进行校验
		return p.checkOutputParameterInvalid(paramType)
	}

	// 增删改, 第一个参数必须为int或uint族类型
	switch paramType.Type.Kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	default:
		return errors.Errorf("invalid output parameter %s", paramType.Type.Kind)
	}

	return nil
}

/*
*
对返回值结构体中的字段类型进行校验
只有基本类型和处于白名单类型列表中的类型是被允许的
并且结构体的所有字段都不能为指针类型
*/
func (p *FileParser) checkOutputParameterInvalid(param *types.Param) error {
	if !param.Type.GetValueType().IsStruct() {
		return nil
	}

	for _, field := range param.Type.GetValueType().Fields {
		if field.Type.IsPointer() {
			return errors.Errorf("fields of pointer type are not allowed in the struct, field: %s, struct: %s", field.Name, param.Type.GetValueType().Name)
		}

		ok := isTypeInWhitelist(field)
		if !ok {
			return errors.Errorf("invalid field type %s in struct %s", field.Type.Name, param.Type.Name)
		}
	}

	return nil
}

func (p *FileParser) parseAnnotations(fnDecl *types.FuncDecl) error {
	if err := p.parseSQLAnnotation(fnDecl); err != nil {
		return err
	}

	// TODO 处理其它注解
	return nil
}

// 解析SQL注解Select/Insert/Update/Delete中的静态或动态SQL
func (p *FileParser) parseSQLAnnotation(fnDecl *types.FuncDecl) error {
	var (
		sqlExpr  = fnDecl.SQLAnnotation.CallExpr
		annoName = fnDecl.SQLAnnotation.Name
	)

	// 静态sql
	arg := sqlExpr.Args[0]
	if lit, ok := arg.(*ast.BasicLit); ok {
		if lit.Kind != token.STRING {
			return errors.Errorf("sql is invalid")
		}

		sqlStr := strings.Trim(lit.Value, "\r\n`\"")
		// 对sql进行解析, 解析出参数列表, 并替换为?
		sqlInfo := sqlutils.ParseSQLStmt(sqlStr)
		sqlStr = sqlInfo.SQL
		fnDecl.SqlParseResult = sqlInfo

		// 如果是select语句, 则需要找到select表字段和结构体字段的对应关系
		if annoName == types.SQLSelectFunc {
			var err error
			sqlStr, err = p.parseStaticSelectSqlStmt(sqlStr, fnDecl)
			if err != nil {
				return err
			}
		}

		fnDecl.Sql = append(fnDecl.Sql, types.NewRawSQL(sqlStr))

		return nil
	}

	// 动态sql
	// 倒叙解析出所有的CallExpr
	scs := parseAllCallExprDepth(arg)
	// 进行校验
	set := collection.NewSetFromSlice(types.SQLOperateNames)
	for _, v := range scs {
		if !set.Has(v.funcName) {
			return errors.Errorf("invalid operate func name %s", v.funcName)
		}
	}

	sqls := make([]types.SQL, 0, len(scs))
	// 解析为接口
	for _, sc := range scs {
		sc.basicInfo = fnDecl
		s, err := parseSqlOperate(sc)
		if err != nil {
			return errors.Errorf("parse sql operate %s error, %v", sc.funcName, err)
		}
		sqls = append(sqls, s)
	}

	// 如果是Select语句，处理SELECT *
	if annoName == types.SQLSelectFunc {
		if err := p.parseDynamicSelectSqlStmt(sqls, fnDecl); err != nil {
			return err
		}
	}

	fnDecl.Sql = sqls

	return nil
}

func (p *FileParser) parseStaticSelectSqlStmt(sql string, fnDecl *types.FuncDecl) (string, error) {
	// 如果返回值不为结构体类型
	returnType := &fnDecl.FuncReturnResultParam.Type
	returnType = returnType.GetValueType()
	if returnType.IsBasicType() {
		fnDecl.SelectFields = append(fnDecl.SelectFields, fnDecl.FuncReturnResultParam.Name)
		return sql, nil
	}

	tableFields, structFields, star, err := ParseSelectFields(sql, fnDecl.FuncReturnResultParam)
	if err != nil {
		return "", errors.Wrapf(err, "parse sql %s error", sql)
	}

	// 替换SELECT *中的*为具体的字段
	if star {
		sql = strings.Replace(sql, " * ", " "+strings.Join(tableFields, ", ")+" ", 1)
	}
	fnDecl.SelectFields = structFields

	return sql, nil
}

func (p *FileParser) parseDynamicSelectSqlStmt(sqls []types.SQL, fnDecl *types.FuncDecl) error {
	// 如果返回值不为结构体类型
	returnType := &fnDecl.FuncReturnResultParam.Type
	returnType = returnType.GetValueType()
	if returnType.IsBasicType() {
		fnDecl.SelectFields = append(fnDecl.SelectFields, fnDecl.FuncReturnResultParam.Name)
		return nil
	}

	builder := strings.Builder{}
	builder.Grow(128)
	var possibleTarger []*types.SimpleStmt
	for _, sq := range sqls {
		switch s := sq.(type) {
		case *types.WhereStmt:
			builder.WriteString("WHERE 1=1 ")
			switch ss := s.Cond.(type) {
			case *types.IfStmt:
				builder.WriteString(ss.Sql)
			case *types.IfChainStmt:
				builder.WriteString(ss.Stmts[0].Sql)
			case *types.ChooseStmt:
				builder.WriteString(ss.Whens[0].Sql)
			}
		case *types.SetStmt:
			builder.WriteString("SET ")
			switch ss := s.Cond.(type) {
			case *types.IfStmt:
				builder.WriteString(ss.Sql)
			case *types.IfChainStmt:
				builder.WriteString(ss.Stmts[0].Sql)
			case *types.ChooseStmt:
				builder.WriteString(ss.Whens[0].Sql)
			}
		case *types.IfStmt:
			builder.WriteString(s.Sql)
		case *types.ForeachStmt:
			builder.WriteString(s.Open)
			builder.WriteString(s.Sql)
			builder.WriteString(s.Close)
		case *types.SimpleStmt:
			builder.WriteString(s.Sql)
			possibleTarger = append(possibleTarger, s)
		case *types.EmptySQLImpl: // 什么也不做
		default:
			return errors.Errorf("unknown annotaion type: %T", sq)
		}
	}

	sql := builder.String()
	tableFields, structFields, star, err := ParseSelectFields(sql, fnDecl.FuncReturnResultParam)
	if err != nil {
		return errors.Wrapf(err, "parse sql error")
	}
	fnDecl.SelectFields = structFields
	if !star {
		return nil
	}

	// 替换*
	for _, sq := range possibleTarger {
		if strings.Contains(sq.Sql, " * ") {
			sq.Sql = strings.Replace(sq.Sql, " * ", " "+strings.Join(tableFields, ", ")+" ", 1)
			break
		}
	}

	return nil
}

// TODO
func (p *FileParser) validateAnnotation(annotations []types.AnnotationInfo) error {

	return nil
}

func parseAllCallExprDepth(expr ast.Expr) []*sqlCall {
	var (
		fn  func(expr ast.Expr) *sqlCall
		res = make([]*sqlCall, 0, 8)
	)

	fn = func(er ast.Expr) *sqlCall {
		switch e := er.(type) {
		case *ast.CallExpr:
			sc := fn(e.Fun)
			if sc == nil {
				return nil
			}
			sc.args = e.Args
			res = append(res, sc)
			return nil
		case *ast.SelectorExpr:
			fn(e.X)
			return &sqlCall{
				funcName: e.Sel.Name,
			}
		case *ast.Ident:
			return &sqlCall{
				funcName: e.Name,
			}
		}

		return nil
	}

	fn(expr)
	return res
}

// 解析文件导入的包
func parseImports(af *ast.File, filepath string) types.PackageInfo {
	pkg := types.PackageInfo{
		FilePath:    filepath,
		PackageName: af.Name.Name,
		AstImports:  af.Imports,
		ImportsMap:  make(map[string]string),
	}

	for _, imp := range af.Imports {
		impInfo := types.ImportInfo{}
		if imp.Path != nil {
			impInfo.AbsPackagePath = strings.Trim(imp.Path.Value, "`\"")
		}
		if imp.Name != nil {
			impInfo.Name = imp.Name.Name
		}
		pkg.Imports = append(pkg.Imports, impInfo)
		pkg.ImportsMap[impInfo.AbsPackagePath] = impInfo.Name
	}

	return pkg
}

// 检查必要的包导入情况
func checkNecessaryPackageImport(neededPackages []string, imports []types.ImportInfo) error {
	for _, name := range neededPackages {
		_, exist := utils.Find(imports, func(info types.ImportInfo) bool {
			return info.AbsPackagePath == name
		})
		if !exist {
			return errors.Errorf("package %s is not imported", name)
		}
	}

	return nil
}
