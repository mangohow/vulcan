package types

import (
	"reflect"
)

/*
*
扩展类型, 用于在preHandle中进行一些特殊处理
比如Page可以实现物理分页
*/
type ExtensionType struct {
	Kind        reflect.Kind
	Name        string
	PackagePath string
	PackageName string
}

var (
	RegisteredExtensions = []ExtensionType{
		{
			Kind:        reflect.Interface,
			Name:        "Page",
			PackagePath: "github.com/mangohow/vulcan",
			PackageName: "vulcan",
		},
	}
)

func IsRegisteredExtension(param *Param) bool {
	for _, ext := range RegisteredExtensions {
		if param.Type.Kind == ext.Kind && param.Type.Name == ext.Name && param.Type.Package.PackagePath == ext.PackagePath {
			return true
		}
	}

	return false
}
