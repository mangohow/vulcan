package dbparser

import (
	"bytes"
	"fmt"
	"github.com/mangohow/mangokit/tools/collection"
	"github.com/mangohow/mangokit/tools/stream"
	"github.com/mangohow/vulcan/internal/ast/astutils"
	"github.com/mangohow/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/internal/errors"
	"github.com/mangohow/vulcan/internal/utils"
	"go/ast"
	astparser "go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

const (
	dbOperatorPackageName = "github.com/jmoiron/sqlx"
	dbOperatorRefName     = "sqlx"
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
	index := bytes.Index(source, []byte("package"))
	if index == -1 {
		return nil, errors.Errorf("source file is invalid, no package name declared")
	}
	f, err := astparser.ParseFile(p.fst, "", source[index:], astparser.ParseComments)
	if err != nil {
		return nil, errors.Errorf("parse file failed, reason: %s", err)
	}

	// 处理导入的包
	packageInfo := p.parseImports(f, filename)
	currentPkg, err := utils.GetCurrentPackagePath(filename)
	if err != nil {
		return nil, err
	}
	packageInfo.PackagePath = currentPkg

	if err := p.checkNecessaryPackageImport(packageInfo.Imports); err != nil {
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

	// 处理包导入信息
	fileInfo.PkgInfo.AstImports = stream.Filter(fileInfo.PkgInfo.AstImports, func(spec *ast.ImportSpec) bool {
		return !utils.Contains(p.filterPackages, strings.Trim(spec.Path.Value, `"`))
	})
	fileInfo.PkgInfo.AstImports = append(fileInfo.PkgInfo.AstImports, stream.Map(p.addPackages, func(name string) *ast.ImportSpec {
		return &ast.ImportSpec{
			Path: astutils.BuildBasicLit(token.STRING, fmt.Sprintf("%q", name)),
		}
	})...)

	return fileInfo, nil
}

// 解析文件导入的包
func (p *FileParser) parseImports(af *ast.File, filepath string) types.PackageInfo {
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
func (p *FileParser) checkNecessaryPackageImport(imports []types.ImportInfo) error {
	for _, name := range p.necessaryPackages {
		_, exist := utils.Find(imports, func(info types.ImportInfo) bool {
			return info.AbsPackagePath == name
		})
		if !exist {
			return errors.Errorf("package %s is not imported", name)
		}
	}

	return nil
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
	callExpr, annoName := astutils.FindCallsInFuncBody(types.SQLAnnotationFuncs, types.AnnotationPackageName, fd.Body, pkgInfo)
	if callExpr == nil {
		return nil, nil
	}
	res.Annotation = annoName

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
	if err := p.parseOutputParameter(fd.Type.Results.List, res, pkgInfo); err != nil {
		return nil, err
	}

	// 4. 处理函数体
	if err := p.parseFuncBody(fd.Body, res, pkgInfo); err != nil {
		return nil, err
	}

	return res, nil
}

func (p *FileParser) checkReceiverInvalid(receiver *ast.Field, fnDecl *types.FuncDecl) error {
	expr := receiver.Type
	name := ""
	fnDecl.Receiver = &types.Param{
		Name: receiver.Names[0].Name,
	}
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
		return errors.Errorf("receiver must have a name")
	}

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

// 解析入参数据类型
func (p *FileParser) parseInputParameter(params []*ast.Field, res *types.FuncDecl, pkgInfo types.PackageInfo) error {
	var (
		err         error
		inputParams = make(map[string]*types.Param)
	)

	stream.ForEach(params, func(field *ast.Field) bool {
		paramList := make([]*types.Param, len(field.Names))
		for i := range field.Names {
			paramList[i] = &types.Param{
				Name: field.Names[i].Name,
			}
		}

		var param types.Param
		if err = p.parseFieldExpr(field.Type, &param, pkgInfo); err != nil {
			return false
		}

		for i := range paramList {
			paramList[i].Type = param.Type
			inputParams[paramList[i].Name] = paramList[i]
		}

		return true
	})

	if err != nil {
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
		case *ast.Ident:
			typeName = et.Name
			break loop
		case *ast.SelectorExpr:
			i, ok := et.X.(*ast.Ident)
			if !ok {
				return errors.Errorf("unsupported input parameter type: %s", typeParam.Name)
			}
			typeName = et.Sel.Name
			typePkgName = i.Name
			break loop
		case *ast.StarExpr:
			if isPointer {
				return errors.Errorf("unsupported multi level pointer, type: %s", typeParam.Name)
			}
			typeSpec.Kind = reflect.Pointer
			typeSpec.ValueType = &types.TypeSpec{}
			typeSpec = typeSpec.ValueType
			isPointer = true
			expr = et.X
		case *ast.ArrayType:
			if isPointer {
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
			return errors.Errorf("can't find type %s.%s's declartion, %w", typePkgName, typeName, err)
		}

		*typeSpec = *typeInfo.Type
	}

	return nil
}

// 解析出参数据类型
// 出参的数量为1~2个
// 如果只有一个参数那么必须是error
// 如果有两个参数, 第二个必须是error, 第一个参数可以为结构体、结构体指针、切片（切片元素为结构体、切片元素为结构体指针、基本类型）、基本类型）
// 第一个参数不能为基本类型的指针, 没有意义
func (p *FileParser) parseOutputParameter(params []*ast.Field, fnDecl *types.FuncDecl, pkgInfo types.PackageInfo) error {
	// 检查出参类型
	// 查询时: 第一个为结构体、基本类型、切片或指针
	// 增删改: 第一个参数为Number类型(int, int64, ..., uint, uint64...), 或只有一个error参数
	// 第二个为error类型
	if len(params) < 1 || len(params) > 2 {
		return errors.Errorf("function must return 1 or 2 results, and the last must be error type")
	}

	for _, p := range params {
		if len(p.Names) != 0 {
			return errors.Errorf("output parameter must not have default name")
		}
	}

	errField := params[0]
	if len(params) == 2 {
		errField = params[1]
	}
	if err := p.checkErrOutputParameter(errField); err != nil {
		return err
	}

	if len(params) == 1 {
		return nil
	}

	paramType := &types.Param{}
	if len(params[0].Names) > 0 && params[0].Names[0] != nil {
		paramType.Name = params[0].Names[0].Name
	}
	if err := p.parseFieldExpr(params[0].Type, paramType, pkgInfo); err != nil {
		return err
	}

	fnDecl.FuncReturnResultParam = paramType

	if fnDecl.Annotation == types.SQLSelectFunc {
		return nil
	}

	// 增删改, 第一个参数必须为int或uint族类型
	switch paramType.Type.Kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	default:
		return errors.Errorf("invalid output parameter %s", paramType.Type.Kind)
	}

	return nil
}

func (p *FileParser) checkErrOutputParameter(field *ast.Field) error {
	ident, ok := field.Type.(*ast.Ident)
	if !ok || ident.Name != "error" {
		return errors.Errorf("function must return 1 or 2 results, and the last must be error type")
	}

	return nil
}

func (p *FileParser) parseFuncBody(body *ast.BlockStmt, fnDecl *types.FuncDecl, pkgInfo types.PackageInfo) error {
	name := pkgInfo.ImportsMap[types.AnnotationPackageName]
	sqlExpr, anno := astutils.FindCallsInBlockStmt(types.SQLAnnotationFuncs, name, body)
	if sqlExpr == nil {
		return errors.Errorf("SQL annotation not found")
	}
	fnDecl.Annotation = anno

	if len(sqlExpr.Args) != 1 {
		return errors.Errorf("SQL annotation call is invalid, has more than 1 parameters")
	}

	// 静态sql
	arg := sqlExpr.Args[0]
	if sbl, ok := arg.(*ast.BasicLit); ok {
		if sbl.Kind != token.STRING {
			return errors.Errorf("sql is invalid")
		}
		fnDecl.Sql = append(fnDecl.Sql, types.NewRawSQL(strings.Trim(sbl.Value, "`\"")))

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
		s, err := parseSqlOperate(sc)
		if err != nil {
			return errors.Errorf("parse sql operate %s error, %v", sc.funcName, err)
		}
		sqls = append(sqls, s)
	}

	fnDecl.Sql = sqls

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
