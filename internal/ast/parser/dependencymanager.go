package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
)

// DependencyManager 依赖管理器
// 管理导入的依赖
type DependencyManager struct {
	fset          *token.FileSet
	declaredTypes map[string]map[string]*ast.TypeSpec // 声明的类型 包名 类型名 类型
}

func NewDependencyManager(fset *token.FileSet) *DependencyManager {
	return &DependencyManager{
		fset:          fset,
		declaredTypes: make(map[string]map[string]*ast.TypeSpec),
	}
}

// GetTypeSpec 获取类型声明信息
// param:
//
//	filePath: 当前分析的文件路径
//	pkg: 类型的包名
//	typeName: 类型名称
func (m *DependencyManager) GetTypeSpec(filePath, pkg, typeName string) (*ast.TypeSpec, error) {
	spec := m.getTypeSpec(pkg, typeName)
	if spec != nil {
		return spec, nil
	}

	// 加载依赖
	if err := m.loadDependency(filePath, pkg, typeName); err != nil {
		return nil, err
	}

	return m.getTypeSpec(pkg, typeName), nil
}

// getTypeSpec 用于获取指定包中指定类型的名字的类型定义。
// 此函数主要用于在已缓存的类型信息中查找特定类型定义。
// 参数:
//
//	pkg: 表示包的名字，用于查找缓存中的类型信息。
//	typeName: 表示要查找的类型名称。
//
// 返回值:
//
//	如果找到了对应的类型定义，则返回指向该类型定义的指针。
//	如果没有找到，则返回 nil。
func (m *DependencyManager) getTypeSpec(pkg, typeName string) *ast.TypeSpec {
	// 检查在已声明的类型缓存中是否存在指定的包。
	if pkgCache, ok := m.declaredTypes[pkg]; ok {
		// 检查在该包的缓存中是否存在指定的类型名称。
		if spec, ok := pkgCache[typeName]; ok {
			// 如果找到了，返回类型定义。
			return spec
		}
	}

	// 如果没有找到，返回 nil。
	return nil
}

// 获取跨包结构体声明的AST
// loadDependency 加载并解析指定依赖的AST节点。
// 该函数根据文件路径、包名和结构体名定位到具体的类型声明，并缓存起来。
// 参数:
//
//	filePath - 依赖文件的路径，用于寻找go.mod文件。
//	pkgName - 需要加载的包名。
//	structName - 需要查找的结构体名。
//
// 返回值:
//
//	如果加载或解析过程中发生错误，返回该错误。
func (m *DependencyManager) loadDependency(filePath, pkgName, structName string) error {
	// 寻找模块根目录
	p, err := findModuleRoot(filePath)
	if err != nil {
		return fmt.Errorf("find go.mod error, %v", err)
	}

	// 配置包加载参数
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles |
			packages.NeedSyntax | packages.NeedTypes,
		Fset: m.fset,
		Dir:  p,
	}

	// 加载目标包
	pkgs, err := packages.Load(cfg, pkgName)
	if err != nil || len(pkgs) == 0 {
		return err
	}

	// 初始化或获取包的缓存
	pkgCache, ok := m.declaredTypes[pkgName]
	if !ok {
		pkgCache = make(map[string]*ast.TypeSpec)
	}

	// 遍历包内所有文件AST
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				// 查找类型声明节点
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}

				// 匹配目标结构体名称
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || typeSpec.Name.Name != structName {
						continue
					}

					// 保存到cache
					typeName := typeSpec.Name.Name
					pkgCache[typeName] = typeSpec
				}
			}
		}
	}

	return nil
}

// findModuleRoot 用于查找从给定目录开始的 Go 模块根目录。
// 它通过向上遍历目录树，直到找到 go.mod 文件或到达文件系统的根目录为止。
// 参数:
//
//	dir (string): 开始搜索的目录路径。
//
// 返回值:
//
//	string: Go 模块的根目录路径。
//	error: 如果没有找到 go.mod 文件，则返回错误。
func findModuleRoot(dir string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found")
}
