package dbparser

import (
	"fmt"
	"github.com/mangohow/vulcan/internal/ast/parser/types"
	"go/parser"
	"go/token"
	"log"
	"os"
	"testing"
)

func parseHelper(src string) {
	expr, err := parser.ParseExpr(src)
	if err != nil {
		log.Fatal(err)
	}

	calls := parseAllCallExprDepth(expr)
	for _, c := range calls {
		log.Print(c.funcName)
	}
}

func TestParseAllCallExprDepth(t *testing.T) {

	src := `
	SQL().Stmt("UPDATE t_user").
		Set(If(user.Password != "", "password = #{user.Password}").
			If(user.Email != "", "email = #{user.Email}").
			If(user.Address != "" && (user.Id > 0 || a), "address = #{user.Address}")).
		Stmt("WHERE id = #{user.Id}").Build()
`
	parseHelper(src)
}

func TestParseAllCallExprDepth2(t *testing.T) {
	src1 := `
If(user.Username != "", "username = #{user.Username}").
			If(user.Address != "", "address = #{user.Address}")
`
	src2 := `
Choose().When(user.Id > 0, "id = #{user.Id}").
			When(user.Username != "", "username = #{user.Username}")
`
	parseHelper(src1)
	parseHelper(src2)
}

func TestParser(t *testing.T) {
	fileParser := NewFileParser(token.NewFileSet(), nil)
	parsed, err := fileParser.Parse("E:\\go_workspace\\src\\projects\\vulcan\\internal\\example\\db\\mapper\\usermapper.go")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(parsed.PkgInfo.String())

	for _, d := range parsed.Declarations {
		f := d.SqlFuncDecl
		if f == nil {
			continue
		}
		log.Printf("Func Name: %s", f.FuncName)
		for _, s := range f.Sql {
			types.PrintSQLHelper(s, os.Stdout)
		}
		fmt.Println()
	}
}
