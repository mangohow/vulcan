package astutils

import (
	"fmt"
	"go/ast"
	gotoken "go/token"
	"strings"
)

type FuncArg struct {
	Name         string
	StarFlag     bool
	AndFlag      bool
	BasicLitFlag gotoken.Token
}

func buildFuncArg(name string) *FuncArg {
	return &FuncArg{Name: name}
}

func (f *FuncArg) star() *FuncArg {
	f.StarFlag = true
	return f
}

func (f *FuncArg) and() *FuncArg {
	f.AndFlag = true
	return f
}

func (f *FuncArg) basicLit(token gotoken.Token) *FuncArg {
	f.BasicLitFlag = token
	return f
}

func (f *FuncArg) buildExpr() ast.Expr {
	if f.BasicLitFlag != gotoken.ILLEGAL {
		return BuildBasicLit(f.BasicLitFlag, fmt.Sprintf("%q", f.Name))
	}
	expr := BuildIdentOrSelectorExpr(f.Name)

	if f.StarFlag {
		return &ast.StarExpr{X: expr}
	}

	if f.AndFlag {
		return BuildUnaryExpr("&", expr)
	}

	return expr
}

// BuildConstSqlCreateAndAssignStmt 创建并赋值语句 const sql = "SELECT ..."
func BuildConstSqlCreateAndAssignStmt(left, right string) ast.Stmt {
	return &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: gotoken.CONST,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names:  []*ast.Ident{ast.NewIdent(left)},
					Values: []ast.Expr{BuildStringBasicLit(right, true)},
				},
			},
		},
	}
}

// BuildCallAssign
// 构建 result, err := m.db.Exec(a, b, c)
// err := m.db.Select(a, b, args...)
// args = append(args, e)
// ddd表示最后一个参数是否是变参, 如 ...string
func BuildCallAssign(left []string, assign string, fn string, args []*FuncArg, ddd bool) *ast.AssignStmt {
	lhs := BuildIdentOrSelectorExprList(left)
	fnExpr := BuildIdentOrSelectorExpr(fn)
	argExpr := make([]ast.Expr, 0, len(args))
	for _, arg := range args {
		argExpr = append(argExpr, arg.buildExpr())
	}

	tok := gotoken.DEFINE
	if assign == "=" {
		tok = gotoken.ASSIGN
	}

	return &ast.AssignStmt{
		Lhs: lhs,
		Tok: tok,
		Rhs: []ast.Expr{BuildCallExpr(fnExpr, argExpr, ddd)},
	}
}

// BuildIdentOrSelectorExprList 根据传入的字符串来判断返回Ident还是SelectorExpr
// 比如 a.b.c 则生成SelectorExpr
// a 则生成IdentExpr
func BuildIdentOrSelectorExprList(args []string) []ast.Expr {
	argExpr := make([]ast.Expr, 0, len(args))
	for _, arg := range args {
		argExpr = append(argExpr, BuildIdentOrSelectorExpr(arg))
	}

	return argExpr
}

func BuildIdentOrSelectorExpr(arg string) ast.Expr {
	items := strings.Split(arg, ".")
	if len(items) > 1 {
		return BuildSelectorExpr(items)
	}

	return ast.NewIdent(arg)
}

// BuildIfErrNENilReturn 构建下面语句
/**
if err != nil {
    return err
}
*/
func BuildIfErrNENilReturn(returnArgs ...interface{}) *ast.IfStmt {
	cond := &ast.BinaryExpr{
		X:  ast.NewIdent("err"),
		Op: gotoken.NEQ,
		Y:  ast.NewIdent("nil"),
	}

	return BuildIfStmt(cond, []ast.Stmt{BuildReturnStmt(returnArgs...)})
}

// BuildReturnStmt 构建返回语句, 例如:
// return err
// return 0, err
// return result
// return nil
func BuildReturnStmt(returnArgs ...interface{}) *ast.ReturnStmt {
	results := make([]ast.Expr, 0, len(returnArgs))
	for _, arg := range returnArgs {
		switch v := arg.(type) {
		case string:
			results = append(results, BuildBasicLit(gotoken.STRING, v))
		case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
			results = append(results, BuildBasicLit(gotoken.INT, fmt.Sprintf("%d", v)))
		case bool:
			results = append(results, ast.NewIdent("false"))
		case float32, float64:
			results = append(results, BuildBasicLit(gotoken.FLOAT, fmt.Sprintf("%f", v)))
		}
	}

	return &ast.ReturnStmt{
		Results: results,
	}
}

// BuildReturnStmtByExpr 还存在这样的返回语句 return int(n), nil
func BuildReturnStmtByExpr(exprs ...ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: exprs,
	}
}

// BuildStructInitAndAssignExpr 构建创建结构体的语句, 例如:
// s := User{}
// s := &User{}
func BuildStructInitAndAssignExpr(left string, right string, address bool) *ast.AssignStmt {
	var x ast.Expr = &ast.CompositeLit{
		Type: BuildIdentOrSelectorExpr(right),
	}
	if address {
		x = BuildUnaryExpr("&", x)
	}

	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			BuildIdentOrSelectorExpr(left),
		},
		Tok: gotoken.DEFINE,
		Rhs: []ast.Expr{x},
	}
}

// BuildSimpleCallAssign
// 构建简单的函数调用, 无返回值
// builder.WriteString()
// ddd表示最后一个参数是否是变参, 如 ...string
func BuildSimpleCallAssign(fn string, args []*FuncArg, ddd bool) *ast.CallExpr {
	fnExpr := BuildSelectorExpr(strings.Split(fn, "."))
	argExpr := make([]ast.Expr, 0, len(args))
	for _, arg := range args {
		argExpr = append(argExpr, arg.buildExpr())
	}

	return BuildCallExpr(fnExpr, argExpr, ddd)
}

// BuildSqlForRangeStmt 构建下面语句
//
//	for i, v := range target {
//		   if i != 0 {
//	        builder.WriteString(", ")
//	    }
//	    collection = append(collection, args[0], args[1], ...)
//	    builderName.WriteString(writeVal)
//
// {
func BuildSqlForRangeStmt(k, v, target, collection, builderName, writeVal, separator string, args []string) ast.Stmt {
	appendCall := &ast.CallExpr{
		Fun: ast.NewIdent("append"),
	}

	// 构建 append 调用: args = append(args, user.Id, user.Username, ...)
	appendCall.Args = append(appendCall.Args, BuildIdentOrSelectorExpr(collection))
	for _, arg := range args {
		appendCall.Args = append(appendCall.Args, BuildIdentOrSelectorExpr(arg))
	}

	// 构建 append 赋值语句
	appendStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent(collection)},
		Tok: gotoken.DEFINE,
		Rhs: []ast.Expr{appendCall},
	}

	// 构建if
	ifStmt := BuildIfStmt(&ast.BinaryExpr{
		X:  ast.NewIdent("i"),
		Op: gotoken.NEQ,
		Y:  BuildBasicLit(gotoken.INT, "0"),
	}, []ast.Stmt{&ast.ExprStmt{X: BuildSimpleCallAssign(builderName+".WriteString", []*FuncArg{buildFuncArg(fmt.Sprintf("%q", separator+" "))}, false)}})

	// 构建 builder.WriteString 调用
	sqlWriteExpr := BuildSimpleCallAssign(builderName+".WriteString", []*FuncArg{buildFuncArg(fmt.Sprintf("%q", writeVal))}, false)
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			appendStmt,
			ifStmt,
			&ast.ExprStmt{X: sqlWriteExpr},
		},
	}

	return BuildForRangeStmt(k, v, target, body)
}
