package astutils

import (
	"fmt"
	"github.com/mangohow/mangokit/tools/collection"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"go/ast"
	gotoken "go/token"
	"path/filepath"
	"reflect"
)

const (
	EmptyLineSign = "//g:empty"
)

// BuildSelectorExpr 构建 a.b.c 表达式
func BuildSelectorExpr(names []string) *ast.SelectorExpr {
	if len(names) < 2 {
		return nil
	}

	se := &ast.SelectorExpr{
		X:   ast.NewIdent(names[0]),
		Sel: ast.NewIdent(names[1]),
	}
	i := 2
	for ; i < len(names); i++ {
		se = &ast.SelectorExpr{
			X:   se,
			Sel: ast.NewIdent(names[i]),
		}
	}

	return se
}

// BuildAssignStmt 构建赋值语句 a.b = c.d
func BuildAssignStmt(left []string, right []string) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: BuildIdentOrSelectorExprList(left),
		Rhs: BuildIdentOrSelectorExprList(right),
		Tok: gotoken.ASSIGN,
	}
}

// BuildAssignStmtByExpr 使用Expr构建赋值表达式  a.xx = b.yyy
func BuildAssignStmtByExpr(left []ast.Expr, right []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: left,
		Rhs: right,
		Tok: gotoken.ASSIGN,
	}
}

// BuildDefineStmtByExpr 使用Expr构建定义赋值表达式 a.b = c.d
func BuildDefineStmtByExpr(left []ast.Expr, right []ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: left,
		Rhs: right,
		Tok: gotoken.DEFINE,
	}
}

// BuildUnaryExpr 构建 &a *b
func BuildUnaryExpr(token string, x ast.Expr) *ast.UnaryExpr {
	var op gotoken.Token
	switch token {
	case "&":
		op = gotoken.AND
	case "*":
		op = gotoken.MUL
	case "-":
		op = gotoken.SUB
	}
	return &ast.UnaryExpr{
		Op: op,
		X:  x,
	}
}

// BuildTypeAssertExpr 构建断言 b.(int) b.(*int)
func BuildTypeAssertExpr(x ast.Expr, typename string, isPointer bool) *ast.TypeAssertExpr {
	res := &ast.TypeAssertExpr{
		X: x,
	}
	t := ast.NewIdent(typename)
	if !isPointer {
		return res
	}

	res.Type = &ast.StarExpr{
		X: t,
	}

	return res
}

// BuildCallExpr 构建函数调用 fn(arg1, arg2) 或 fn(arg1, arg2...)
func BuildCallExpr(fn ast.Expr, args []ast.Expr, ellipsis bool) *ast.CallExpr {
	ce := &ast.CallExpr{
		Fun:  fn,
		Args: args,
	}
	if ellipsis {
		ce.Ellipsis = gotoken.Pos(1)
	}
	return ce
}

func BuildBasicLit(kind gotoken.Token, val string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  kind,
		Value: val,
	}
}

func BuildImportSpec(pkg string) *ast.ImportSpec {
	return &ast.ImportSpec{
		Path: &ast.BasicLit{
			ValuePos: 0,
			Kind:     gotoken.STRING,
			Value:    fmt.Sprintf("\"%s\"", pkg),
		},
	}
}

func BuildVarNameExpr(name []string, isPointer bool) ast.Expr {
	var expr ast.Expr
	if len(name) == 1 {
		expr = ast.NewIdent(name[0])
	} else {
		expr = BuildSelectorExpr(name)
	}

	if isPointer {
		expr = BuildUnaryExpr("*", expr)
	}

	return expr
}

func BuildIdentList(args ...string) []ast.Expr {
	res := make([]ast.Expr, 0, len(args))
	for _, arg := range args {
		res = append(res, ast.NewIdent(arg))
	}
	return res
}

// BuildIfStmt 构建if语句
func BuildIfStmt(cond ast.Expr, bodyStmts []ast.Stmt) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: cond,
		Body: &ast.BlockStmt{
			List: bodyStmts,
		},
	}
}

// BuildBasicTypeConvertExpr 构建基本类型转换的表达式, 例如：
// int(n)  uint(n)
func BuildBasicTypeConvertExpr(typeName, arg string) ast.Expr {
	return &ast.CallExpr{
		Fun:  ast.NewIdent(typeName),
		Args: []ast.Expr{ast.NewIdent(arg)},
	}
}

// BuildEmptyStmt 构建空行, 使用注释, 否则空行会被删掉, 生成源码后再进行处理
func BuildEmptyStmt() ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.Ident{
			NamePos: 1,             // 必须设置位置
			Name:    EmptyLineSign, // 注释内容
		},
	}
}

// BuildStringBasicLit 构建一些字面量, 比如字符串
func BuildStringBasicLit(value string, dot bool) *ast.BasicLit {
	if dot {
		value = fmt.Sprintf("`%s`", value)
	} else {
		value = fmt.Sprintf("%q", value)
	}
	return &ast.BasicLit{
		Kind:  gotoken.STRING,
		Value: value,
	}
}

func BuildForRangeStmt(key, val, collection string, body *ast.BlockStmt) ast.Stmt {
	// 构建 range 表达式: users
	rangeExpr := ast.NewIdent(collection)

	// 构建 key/value 变量: _, user
	k := ast.NewIdent(key)
	v := ast.NewIdent(val)

	return &ast.RangeStmt{
		Key:   k,
		Value: v,
		Tok:   gotoken.DEFINE,
		X:     rangeExpr,
		Body:  body,
	}
}

func StringList(args ...string) []string {
	return args
}

// FindCallsInFuncBody 在函数体中寻找函数调用
// 例如FindCallInFuncBody("Select", "github.com/mangohow/vulcan/annocation", block, pkgInfo)
// 为在block函数体中寻找包为github.com/mangohow/vulcan/annocation的Select函数调用
func FindCallsInFuncBody(fnName []string, pkgName string, block *ast.BlockStmt, pkgInfo types.PackageInfo) (*ast.CallExpr, string) {
	// 先查找包里面是否有目标包
	var targetPkg *types.ImportInfo
	for i, imp := range pkgInfo.Imports {
		if imp.AbsPackagePath == pkgName {
			targetPkg = &pkgInfo.Imports[i]
			break
		}
	}

	var annoName string
	if targetPkg == nil {
		return nil, ""
	}

	var (
		res   *ast.CallExpr
		found bool
		set   = collection.NewSet[string]()
	)
	for _, s := range fnName {
		set.Add(s)
	}

	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}

		expr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		var (
			name, funcName string
		)

		fn := expr.Fun
		switch f := fn.(type) {
		case *ast.SelectorExpr:
			i, ok := f.X.(*ast.Ident)
			if !ok {
				return true
			}
			name = i.Name
			funcName = f.Sel.Name
			annoName = funcName
		case *ast.Ident:
			funcName = f.Name
			annoName = funcName
		}

		if set.Has(funcName) && (targetPkg.Name == "." || name == filepath.Base(pkgName)) {
			found = true
			res = expr
			return false
		}

		return true
	})

	return res, annoName
}

func FindCallsInBlockStmt(fnName []string, pkgName string, block *ast.BlockStmt) (*ast.CallExpr, string) {
	var (
		set = collection.NewSet[string]()
	)
	for _, s := range fnName {
		set.Add(s)
	}

	for _, stmt := range block.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}

		callExpr, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		if pkgName == "" || pkgName == "." {
			ident, ok := callExpr.Fun.(*ast.Ident)
			if ok && ident != nil && set.Has(ident.Name) {
				return callExpr, ident.Name
			}
			continue
		}

		se, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		x, ok := se.X.(*ast.Ident)
		if !ok || x.Name != pkgName {
			continue
		}

		if set.Has(se.Sel.Name) {
			return callExpr, x.Name
		}

	}

	return nil, ""
}

// BuildEllipsisField 构建一个表示变长参数的字段节点。
// 该函数接受变量名称和类型名称作为参数，返回一个指向ast.Field的指针，
// 该指针描述了一个变长参数字段。
// 参数:
//
//	varName - 变量名称
//	typeName - 类型名称
//
// 返回值:
//
//	*ast.Field - 指向表示变长参数的字段节点的指针
func BuildEllipsisField(varName, typeName string) *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{
			ast.NewIdent(varName),
		},
		Type: &ast.Ellipsis{
			Elt: BuildIdentOrSelectorExpr(typeName),
		},
	}
}

// BuildKeyValueBasicLitExpr 构建key: value
func BuildKeyValueBasicLitExpr(key, val string, kind gotoken.Token) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key:   ast.NewIdent(key),
		Value: BuildBasicLit(kind, val),
	}
}

// BuildKeyValueExpr 构建key: value
func BuildKeyValueExpr(key string, val ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key:   ast.NewIdent(key),
		Value: val,
	}
}

// BuildInitAssignExpr 构建定义表达式
// res := model.User{}
// res := User{}
// res := 0
// res := ""
// res := int32(0)
// res := float32(0.0)
// res := false
// res := &model.User{}
// res := []model.User{}
// res := []*model.User{}
// res := []int{}
// res := []string{}
func BuildInitAssignExpr(param *types.Param, name string, curPkgName string) *ast.AssignStmt {
	assign := &ast.AssignStmt{
		Lhs: []ast.Expr{BuildIdentOrSelectorExpr(name)},
		Tok: gotoken.DEFINE,
	}
	var (
		expr      ast.Expr
		paramType = &param.Type
		isPointer = false
		isSlice   = false
	)

	if paramType.IsBasicType() {
		switch paramType.Kind {
		case reflect.Int:
			expr = BuildBasicLit(gotoken.INT, "0")
		case reflect.String:
			expr = BuildBasicLit(gotoken.STRING, `""`)
		case reflect.Bool:
			expr = ast.NewIdent("false")
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			expr = &ast.CallExpr{
				Fun:  ast.NewIdent(paramType.Kind.String()),
				Args: []ast.Expr{BuildBasicLit(gotoken.INT, "0")},
			}
		case reflect.Float32, reflect.Float64:
			expr = &ast.CallExpr{
				Fun:  ast.NewIdent(paramType.Kind.String()),
				Args: []ast.Expr{BuildBasicLit(gotoken.FLOAT, "0.0")},
			}
		}

		assign.Rhs = []ast.Expr{expr}
		return assign
	}

loop:
	for {
		switch {
		case paramType.IsSlice():
			isSlice = true
			paramType = paramType.ValueType
		case paramType.IsPointer():
			isPointer = true
			paramType = paramType.ValueType
		case paramType.IsStruct():
			name := paramType.Name
			if paramType.Package.PackageName != curPkgName {
				name = paramType.Package.PackageName + "." + name
			}
			expr = &ast.CompositeLit{
				Type: BuildIdentOrSelectorExpr(name),
			}
			break loop
		case paramType.IsBasicType():
			expr = &ast.CompositeLit{
				Type: ast.NewIdent(paramType.Kind.String()),
			}
			break loop
		default:
			break loop
		}
	}

	if isPointer && isSlice {
		expr = BuildUnaryExpr("*", expr)
	} else if isPointer {
		expr = BuildUnaryExpr("&", expr)
	}

	if isSlice {
		expr = &ast.ArrayType{
			Elt: expr,
		}
	}

	assign.Rhs = []ast.Expr{expr}
	return assign
}
