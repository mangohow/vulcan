package parser

import (
	"fmt"
	"github.com/mangohow/mangokit/tools/stream"
	"github.com/mangohow/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/internal/utils"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/packages"
	"strings"
)

// DependencyManager 依赖管理器
// 管理导入的依赖
type DependencyManager struct {
	fset          *token.FileSet
	declaredTypes map[string]map[string]*TypeInfo // 声明的类型 包名 类型名 类型
}

type TypeInfo struct {
	AstType     *ast.TypeSpec      // ast
	FilePath    string             // 所属文件
	PackagePath string             // 所属包
	Imports     []types.ImportInfo // 导入的包
}

func NewDependencyManager(fset *token.FileSet) *DependencyManager {
	return &DependencyManager{
		fset:          fset,
		declaredTypes: make(map[string]map[string]*TypeInfo),
	}
}

// GetTypeInfo 获取类型声明信息
// param:
//
//	filePath: 当前分析的文件路径
//	pkg: 类型的包名
//	typeName: 类型名称
func (m *DependencyManager) GetTypeInfo(filePath, pkg, typeName string) (*TypeInfo, error) {
	spec := m.getTypeInfo(pkg, typeName)
	if spec != nil {
		return spec, nil
	}

	// 加载依赖
	if err := m.loadDependency(filePath, pkg, typeName); err != nil {
		return nil, err
	}

	return m.getTypeInfo(pkg, typeName), nil
}

// getTypeInfo 用于获取指定包中指定类型的名字的类型定义。
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
func (m *DependencyManager) getTypeInfo(pkg, typeName string) *TypeInfo {
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
	p, err := utils.FindGoModuleRoot(filePath)
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
		pkgCache = make(map[string]*TypeInfo)
		m.declaredTypes[pkgName] = pkgCache
	}

	// 遍历包内所有文件AST
	for _, pkg := range pkgs {
		for j, file := range pkg.Syntax {
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
					pkgCache[typeName] = &TypeInfo{
						AstType:     typeSpec,
						FilePath:    pkg.GoFiles[j],
						PackagePath: pkg.PkgPath,
						Imports: stream.Map(file.Imports, func(importSpec *ast.ImportSpec) types.ImportInfo {
							res := types.ImportInfo{}
							if importSpec.Name != nil {
								res.Name = importSpec.Name.Name
							}
							res.AbsPackagePath = strings.Trim(`"`, importSpec.Path.Value)
							return res
						}),
					}
				}
			}
		}
	}

	return nil
}
