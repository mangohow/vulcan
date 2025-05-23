package dbparser

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mangohow/mangokit/tools/collection"
	"github.com/mangohow/vulcan/internal/ast/astutils"
	"github.com/mangohow/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/internal/ast/parser/types"
	"go/ast"
	astparser "go/parser"
	"go/token"
	"os"
	"strings"
)

type FileParser struct {
	fst               *token.FileSet
	dependencyManager *parser.DependencyManager
	filterPackages    []string // 生成的代码中需要过滤掉的package
	addPackages       []string // 生成的代码中需要添加的package
}

func NewFileParser(fst *token.FileSet, dm *parser.DependencyManager) *FileParser {
	return &FileParser{
		fst:               fst,
		dependencyManager: dm,
		filterPackages: []string{
			"github.com/mangohow/vulcan/annotation",
		},
		addPackages: []string{
			"github.com/mangohow/vulcan/db",
		},
	}
}

func (p *FileParser) Parse(filename string) (*types.File, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filename, err)
	}
	index := bytes.Index(source, []byte("package"))
	if index == -1 {
		return nil, fmt.Errorf("source file invalid, no package in %s", filename)
	}
	f, err := astparser.ParseFile(p.fst, "", source[index:], astparser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", filename, err)
	}

	// 处理导入的包
	packageInfo := p.parseImports(f, filename)
	//log.Infof("parsed package info: %s", packageInfo.String())

	// 对包进行过滤, 如果没有包含注解相关的包, 则不生成代码
	for _, pkg := range p.filterPackages {
		if _, ok := packageInfo.ImportsMap[pkg]; !ok {
			return nil, fmt.Errorf("package %s not imported in %s", pkg, filename)
		}
	}

	fileInfo := &types.File{
		PkgInfo: packageInfo,
	}
	// 处理类型、函数声明
	if err := p.parseDeclares(f, fileInfo); err != nil {
		return nil, fmt.Errorf("parse declares eror, %v", err)
	}

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

// 解析所有声明
func (p *FileParser) parseDeclares(af *ast.File, file *types.File) error {
	for _, decl := range af.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// 在函数中寻找注解并解析
			declare, err := p.parseFuncDeclare(d, file.PkgInfo)
			if err != nil {
				return err
			}

			if declare != nil {
				file.AddDeclaration(decl, declare)
			} else {
				file.AddAstDecl(decl) // 如果没用注解, 则该函数无需处理, 直接添加ast
			}
		case *ast.GenDecl:
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
	callExpr := astutils.FindCallsInFuncBody(types.SQLAnnotationFuncs, types.AnnotationPackageName, fd.Body, pkgInfo)
	if callExpr == nil {
		return nil, nil
	}

	// 解析注解
	// 1. 先处理接收器
	// 如果没有接收器, 则返回错误
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return nil, errors.New("function must have a receiver")
	}

	// 判断接收器是否为结构体, 并且包含sqlx.DB对象
	if err := p.checkReceiverInvalid(fd.Recv.List[0]); err != nil {
		return nil, err
	}

	// 2. 处理入参
	if err := p.parseInputParameter(fd.Type.Params.List, res); err != nil {
		return nil, err
	}

	// 3. 检查出参, 出参数量只有1或2个
	if results := fd.Type.Results.List; len(results) < 1 || len(results) > 2 {
		return nil, errors.New("function must return 1 or 2 results, and the last must be error type")
	}

	// 检查出参类型
	// 查询时: 第一个为结构体或基本类型
	// 增删改: 第一个参数为Number类型(int, int64, ..., uint, uint64...), 或只有一个error参数
	// 第二个为error类型
	if err := p.checkOutputParameterInvalid(fd.Type.Results.List); err != nil {
		return nil, err
	}

	// 4. 处理出参
	if err := p.parseOutputParameter(fd.Type.Results.List, res); err != nil {
		return nil, err
	}

	// 5. 处理函数体
	if err := p.parseFuncBody(fd.Body, res, pkgInfo); err != nil {
		return nil, err
	}

	return res, nil
}

func (p *FileParser) checkReceiverInvalid(receiver *ast.Field) error {
	return nil
}

func (p *FileParser) parseInputParameter(params []*ast.Field, res *types.FuncDecl) error {
	return nil
}

func (p *FileParser) checkOutputParameterInvalid(params []*ast.Field) error {
	return nil
}

func (p *FileParser) parseOutputParameter(params []*ast.Field, fnDecl *types.FuncDecl) error {
	return nil
}

func (p *FileParser) parseFuncBody(body *ast.BlockStmt, fnDecl *types.FuncDecl, pkgInfo types.PackageInfo) error {
	name := pkgInfo.ImportsMap[types.AnnotationPackageName]
	sqlExpr, anno := astutils.FindCallsInBlockStmt(types.SQLAnnotationFuncs, name, body)
	if sqlExpr == nil {
		return errors.New("SQL annotation not found")
	}
	fnDecl.Annotation = anno

	if len(sqlExpr.Args) != 1 {
		return errors.New("SQL annotation call invalid, has more than 1 parameters")
	}

	// 静态sql
	arg := sqlExpr.Args[0]
	if sbl, ok := arg.(*ast.BasicLit); ok {
		if sbl.Kind != token.STRING {
			return errors.New("sql invalid")
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
			return fmt.Errorf("invalid operate func name %s", v.funcName)
		}
	}

	sqls := make([]types.SQL, 0, len(scs))
	// 解析为接口
	for _, sc := range scs {
		s, err := parseSqlOperate(sc)
		if err != nil {
			return fmt.Errorf("parse sql operate %s error, %v", sc.funcName, err)
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
