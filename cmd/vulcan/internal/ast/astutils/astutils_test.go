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
	"reflect"
	"strings"
	"testing"

	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
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
	return ast.Fprint(file, fileSet, f, nil)
}

func TestParseAst(t *testing.T) {
	err := ParseAst("../../../../../internal/example/db/mapper/usermapper.go", "usermapper.ast")
	if err != nil {
		log.Println(err)
	}
	ParseAst("../../../../../internal/example/db/mapper/usermapper_gen.go", "usermapper_gen.ast")
	ParseAst("../../../../../internal/example/model/user.go", "user.ast")
}

func TestGenericAst(t *testing.T) {
	source := `
package test

type TestGeneric struct {
	NullTime sql.Null[string]
}
`
	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, "", source, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile("generic.ast", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	err = ast.Fprint(file, fileSet, f, nil)
	if err != nil {
		log.Fatal(err)
	}
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

func TestBuildInitAssignExpr(t *testing.T) {
	p1 := types.Param{
		Type: types.TypeSpec{
			Name: "User",
			Package: &types.PackageInfo{
				PackageName: "model",
			},
			Kind: reflect.Struct,
		},
	}
	printSource(BuildInitAssignExpr(&p1, "res", "mapper"))
	p2 := types.Param{
		Type: types.TypeSpec{
			Name: "User",
			Package: &types.PackageInfo{
				PackageName: "model",
			},
			Kind: reflect.Struct,
		},
	}
	printSource(BuildInitAssignExpr(&p2, "res", "model"))
	p3 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Int,
		},
	}
	printSource(BuildInitAssignExpr(&p3, "res", ""))
	p4 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.String,
		},
	}
	printSource(BuildInitAssignExpr(&p4, "res", ""))
	p5 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Int32,
		},
	}
	printSource(BuildInitAssignExpr(&p5, "res", ""))
	p6 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Float32,
		},
	}
	printSource(BuildInitAssignExpr(&p6, "res", ""))
	p7 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Bool,
		},
	}
	printSource(BuildInitAssignExpr(&p7, "res", ""))

	p8 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Pointer,
			ValueType: &types.TypeSpec{
				Name: "User",
				Package: &types.PackageInfo{
					PackageName: "model",
				},
				Kind: reflect.Struct,
			},
		},
	}
	printSource(BuildInitAssignExpr(&p8, "res", "mapper"))

	p9 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Slice,
			ValueType: &types.TypeSpec{
				Name: "User",
				Package: &types.PackageInfo{
					PackageName: "model",
				},
				Kind: reflect.Struct,
			},
		},
	}
	printSource(BuildInitAssignExpr(&p9, "res", "mapper"))

	p10 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Pointer,
			ValueType: &types.TypeSpec{
				Kind: reflect.Slice,
				ValueType: &types.TypeSpec{
					Name: "User",
					Package: &types.PackageInfo{
						PackageName: "model",
					},
					Kind: reflect.Struct,
				},
			},
		},
	}
	printSource(BuildInitAssignExpr(&p10, "res", "mapper"))

	p11 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Slice,
			ValueType: &types.TypeSpec{
				Kind: reflect.Int,
			},
		},
	}
	printSource(BuildInitAssignExpr(&p11, "res", "mapper"))

	p12 := types.Param{
		Type: types.TypeSpec{
			Kind: reflect.Slice,
			ValueType: &types.TypeSpec{
				Kind: reflect.String,
			},
		},
	}
	printSource(BuildInitAssignExpr(&p12, "res", "mapper"))
}

func TestGenCompositeLit(t *testing.T) {
	compositeLit := &ast.CompositeLit{
		Type: &ast.SelectorExpr{
			X:   ast.NewIdent("model"),
			Sel: ast.NewIdent("User"),
		},
		Lbrace: 65535,
		Elts: []ast.Expr{
			&ast.KeyValueExpr{
				Key:   ast.NewIdent("Name"),
				Value: ast.NewIdent("name"),
				Colon: 65536,
			},
			&ast.KeyValueExpr{
				Key:   ast.NewIdent("Password"),
				Value: ast.NewIdent("password"),
				Colon: 65537,
			},
		},
		Rbrace:     65538,
		Incomplete: false,
	}

	format.Node(os.Stdout, token.NewFileSet(), compositeLit)
}
