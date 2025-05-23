package types

import (
	"encoding/json"
	"go/ast"
	"reflect"
)

type PackageInfo struct {
	PackageName string
	FilePath    string
	Imports     []ImportInfo
	ImportsMap  map[string]string
	AstImports  []*ast.ImportSpec `json:"-"`
}

type ImportInfo struct {
	AbsPackagePath string
	Name           string
}

func (p *PackageInfo) String() string {
	bytes, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return "marshal json error: " + err.Error()
	}
	return string(bytes)
}

type File struct {
	PkgInfo      PackageInfo   // 导入的包信息
	Declarations []Declaration // 文件中的声明声明
}

func (f *File) AddAstDecl(decl ast.Decl) {
	f.Declarations = append(f.Declarations, Declaration{
		AstDecl: decl,
	})
}

func (f *File) AddDeclaration(decl ast.Decl, funcDecl *FuncDecl) {
	fd := decl.(*ast.FuncDecl)
	fd.Body = nil
	f.Declarations = append(f.Declarations, Declaration{
		AstDecl:     fd,
		SqlFuncDecl: funcDecl,
	})
}

type Declaration struct {
	AstDecl     ast.Decl
	SqlFuncDecl *FuncDecl
}

type FuncDecl struct {
	FuncName    string            // 函数名
	Receiver    *Param            // 接收器参数信息
	InputParam  map[string]*Param // 入参信息
	OutputParam map[string]*Param // 出参信息
	Sql         []SQL             // SQL体
	Annotation  string            // SQL类型 Insert、Delete、Update、Select
}

// Param 参数类型
type Param struct {
	Name string   // 参数名称
	Type TypeSpec // 参数类型
}

type TypeSpec struct {
	Name      string       // 类型名称
	Package   *PackageInfo // 所属包
	Tag       string       // 结构体字段tag
	Kind      reflect.Kind // 类型
	ValueType *TypeSpec    // 如果是指针或切片, 该类型指向指针指向的类型
	Fields    []*TypeSpec  // 如果是结构体, 该类型为结构体类型
}
