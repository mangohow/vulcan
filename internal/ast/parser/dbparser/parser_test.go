package dbparser

import (
	"encoding/json"
	"fmt"
	parser2 "github.com/mangohow/vulcan/internal/ast/parser"
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

func TestTypeParser(t *testing.T) {
	dependencyManager := parser2.NewDependencyManager(token.NewFileSet())
	filePath := "E:\\go_workspace\\src\\projects\\vulcan\\internal\\example\\db\\mapper\\usermapper_gen.go"
	pkgName := "github.com/mangohow/vulcan/internal/example/model"
	typeName := "User"
	typeParser := NewTypeParser(dependencyManager)
	info, err := typeParser.GetTypeInfo(&AdditionalOption{
		FilePath: filePath,
		PkgPath:  pkgName,
		TypeName: typeName,
		Imports: []types.ImportInfo{
			{
				AbsPackagePath: "time",
				Name:           "time",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := json.MarshalIndent(info.Type, "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(content))
}
