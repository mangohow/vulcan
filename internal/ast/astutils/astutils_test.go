package astutils

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"testing"
)

func ParseAst(src, dst string) error {
	fileSet := token.NewFileSet()
	source, err := os.ReadFile(src)
	if err != nil {
		log.Fatal(err)
	}
	index := bytes.Index(source, []byte("package"))
	f, err := parser.ParseFile(fileSet, "", source[index:], parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	return ast.Fprint(file, fileSet, f.Decls, nil)
}

func TestParseAst(t *testing.T) {
	err := ParseAst("../../example/db//mapper/usermapper.go", "usermapper.ast")
	if err != nil {
		log.Println(err)
	}
	ParseAst("../../example/db/mapper/usermapper_gen.go", "usermapper_gen.ast")
}

func printSource(node ast.Node) {
	buffer := bytes.NewBuffer(nil)
	format.Node(buffer, token.NewFileSet(), node)
	source := buffer.String()
	// 替换空行
	fmt.Println(strings.ReplaceAll(source, EmptyLineSign, ""))
}

func TestBuildConstSqlCreateAndAssignStmt(t *testing.T) {
	stmt := BuildConstSqlCreateAndAssignStmt("sql", "SELECT * FROM table")
	printSource(stmt)
}

// result, err := m.db.Exec(a, b, c)
// err := m.db.Select(a, b, args...)
func TestBuildCallAssign(t *testing.T) {
	printSource(BuildCallAssign(StringList("result", "err"), ":=", "m.db.Exec", []*FuncArg{
		buildFuncArg("a"),
		buildFuncArg("b"),
		buildFuncArg("c"),
	}, false))
	printSource(BuildCallAssign(StringList("err"), ":=", "m.db.Select", []*FuncArg{
		buildFuncArg("a"),
		buildFuncArg("b"),
		buildFuncArg("args"),
	}, true))
	printSource(BuildCallAssign(StringList("id", "err"), ":=", "result.LastInsertId", nil, false))
	printSource(BuildCallAssign(StringList("affected", "err"), ":=", "result.RowsAffected", nil, false))
	printSource(BuildCallAssign(StringList("args"), "=", "append", []*FuncArg{
		buildFuncArg("args"),
		buildFuncArg("user.Username"),
	}, false))
}

// *a.b
// &a.b
func TestFuncArg(t *testing.T) {
	printSource(buildFuncArg("a.b").star().buildExpr())
	printSource(buildFuncArg("a.b").and().buildExpr())
}

func TestBuildSelect(t *testing.T) {
	ast.Print(token.NewFileSet(), BuildSelectorExpr([]string{"a", "b", "c"}))
}

func TestBuildIfErrReturn(t *testing.T) {
	printSource(BuildIfErrNENilReturn("err"))
	printSource(BuildIfErrNENilReturn(0, "err"))
	printSource(BuildIfErrNENilReturn("res", "nil"))
}

func TestBuildAssignStmt(t *testing.T) {
	printSource(BuildAssignStmt(StringList("user.Id"), StringList("id")))
}

func TestBuildReturn(t *testing.T) {
	printSource(BuildReturnStmt("err"))
	printSource(BuildReturnStmt("nil"))
	printSource(BuildReturnStmt("res", "nil"))
}

func TestBuildBasicTypeConvertExpr(t *testing.T) {
	printSource(BuildBasicTypeConvertExpr("int", "n"))
	printSource(BuildBasicTypeConvertExpr("[]byte", "data"))
}

func TestBuildStructInitAndAssignExpr(t *testing.T) {
	printSource(BuildStructInitAndAssignExpr("builder", "strings.Builder", false))
	printSource(BuildStructInitAndAssignExpr("builder", "strings.Builder", true))
	printSource(BuildStructInitAndAssignExpr("user", "model.User", false))
	printSource(BuildStructInitAndAssignExpr("user", "model.User", true))
}

func TestBuildSimpleCallAssign(t *testing.T) {
	printSource(BuildSimpleCallAssign("build.WriteString", []*FuncArg{
		buildFuncArg("WHERE username = ? ").basicLit(token.STRING),
	}, false))
}

/*
*

	const sql = `INSERT INTO t_user (id, username, password, create_at, email, address) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := m.db.Exec(sql, user.Id, user.Username, user.Password, user.CreatedAt, user.Email, user.Address)
	if err != nil {
	    return err
	}

	id, err := result.LastInsertId()
	if err != nil {
	    return err
	}

	user.Id = id

	return nil
*/
func TestBuildAddSqlExe(t *testing.T) {
	bodyStmts := make([]ast.Stmt, 0, 16)
	bodyStmts = append(bodyStmts, BuildConstSqlCreateAndAssignStmt("sql", "INSERT INTO t_user (id, username, password, create_at, email, address) VALUES (?, ?, ?, ?, ?, ?)"))
	bodyStmts = append(bodyStmts, BuildCallAssign(StringList("result", "err"), ":=", "m.db.Exec", []*FuncArg{
		buildFuncArg("sql"),
		buildFuncArg("user.Id"),
		buildFuncArg("user.Username"),
		buildFuncArg("user.Password"),
		buildFuncArg("user.CreatedAt"),
		buildFuncArg("user.Email"),
		buildFuncArg("user.Address"),
	}, false))
	bodyStmts = append(bodyStmts, BuildIfErrNENilReturn("err"))
	bodyStmts = append(bodyStmts, BuildCallAssign(StringList("id", "err"), ":=", "result.LastInsertId", nil, false))
	bodyStmts = append(bodyStmts, BuildEmptyStmt())
	bodyStmts = append(bodyStmts, BuildIfErrNENilReturn("err"))
	bodyStmts = append(bodyStmts, BuildEmptyStmt())
	bodyStmts = append(bodyStmts, BuildAssignStmt(StringList("user.Id"), StringList("id")))
	bodyStmts = append(bodyStmts, BuildEmptyStmt())
	bodyStmts = append(bodyStmts, BuildReturnStmt("nil"))

	printSource(&ast.BlockStmt{
		List: bodyStmts,
	})
}

func TestBuildSqlForRangeStmt(t *testing.T) {
	printSource(BuildSqlForRangeStmt("i", "user", "users", "args",
		"builder", "(?, ?, ?)", ",", []string{"user.Name", "user.Password", "user.Email"}))
}
