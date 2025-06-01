package types

import (
	"encoding/json"
	"go/ast"
	"reflect"
)

type PackageInfo struct {
	PackageName string // 包名, 短命
	PackagePath string // 绝对包名
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
	AstFile      *ast.File
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
	PkgInfo     PackageInfo
}

type FuncDecl struct {
	FuncName              string            // 函数名
	Receiver              *Param            // 接收器参数信息
	InputParam            map[string]*Param // 入参信息
	OutputParam           map[string]*Param // 出参信息
	FuncReturnResultParam *Param            // 函数出参1类型
	Sql                   []SQL             // SQL体
	Annotation            string            // SQL类型 Insert、Delete、Update、Select
}

// 是否是基本类型

// Param 参数类型
type Param struct {
	Name string   // 参数名称
	Type TypeSpec // 参数类型
}

type TypeSpec struct {
	Name      string            // 类型名称
	Package   *PackageInfo      // 所属包
	Tag       reflect.StructTag // 结构体字段tag
	Kind      reflect.Kind      // 类型
	ValueType *TypeSpec         // 如果是指针或切片, 该类型指向指针指向的类型
	Fields    []*Param          // 如果是结构体, 该类型为结构体类型
}

// IsStruct 是否是结构体
func (t *TypeSpec) IsStruct() bool {
	return t.Kind == reflect.Struct
}

// IsPointer 是否是指针
func (t *TypeSpec) IsPointer() bool {
	return t.Kind == reflect.Ptr
}

// IsSlice 是否是切片
func (t *TypeSpec) IsSlice() bool {
	return t.Kind == reflect.Slice
}

func (t *TypeSpec) IsBasicType() bool {
	switch t.Kind {
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}

	return false
}
