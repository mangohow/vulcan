package utils

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Keys[K comparable, V any](m map[K]V) []K {
	if m == nil {
		return nil
	}
	res := make([]K, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	return res
}

func Values[K comparable, V any](m map[K]V) []V {
	if m == nil {
		return nil
	}

	res := make([]V, 0, len(m))
	for _, v := range m {
		res = append(res, v)
	}
	return res
}

func Contains[T comparable](ss []T, s T) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}

	return false
}
func GetPackageName(pkgPath string) string {
	idx := strings.LastIndex(pkgPath, "/")
	if idx == -1 {
		return pkgPath
	}

	return pkgPath[idx+1:]
}

func Find[T comparable](s []T, fn func(T) bool) (T, bool) {
	for i := 0; i < len(s); i++ {
		if fn(s[i]) {
			return s[i], true
		}
	}

	var zero T
	return zero, false
}

// FindGoModuleRoot 用于查找从给定目录开始的 Go 模块根目录。
// 它通过向上遍历目录树，直到找到 go.mod 文件或到达文件系统的根目录为止。
// 参数:
//
//	dir (string): 开始搜索的目录路径。
//
// 返回值:
//
//	string: Go 模块的根目录路径。
//	error: 如果没有找到 go.mod 文件，则返回错误。
func FindGoModuleRoot(dir string) (string, error) {
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

// FindGoModuleName 获取go.mod中的module名称
func FindGoModuleName(modFilePath string) (string, error) {
	data, err := os.ReadFile(modFilePath)
	if err != nil {
		return "", fmt.Errorf("read go.mod failed: %v", err)
	}

	content := string(data)
	moduleIndex := strings.Index(content, "module")
	index := strings.Index(content, "\n")
	moduleName := content[moduleIndex+len("module") : index]
	return strings.Trim(moduleName, " "), nil
}

func GetCurrentPackagePath(filename string) (string, error) {
	// 获取当前包路径
	modulePath, err := FindGoModuleRoot(filepath.Dir(filename))
	if err != nil {
		return "", err
	}
	modName, err := FindGoModuleName(filepath.Join(modulePath, "go.mod"))
	if err != nil {
		return "", err
	}
	idx := strings.Index(filename, modulePath)
	if idx == -1 {
		return "", fmt.Errorf("invalid project")
	}
	pkg := filepath.Dir(filename)
	pkg = pkg[len(modulePath):]
	pkg = strings.ReplaceAll(pkg, "\\", "/")

	// E:\go_workspace\src\vulcan_test
	// module vulcan_test
	// E:\go_workspace\src\vulcan_test\mapper\usermapper.go
	// \mapper\usermapper.go

	return filepath.Join(modName, pkg), nil
}

func TrimLineWithPrefix(content []byte, sub ...[]byte) []byte {
	lines := bytes.Split(content, []byte("\n"))
	buf := bytes.Buffer{}
	buf.Grow(len(content))
loop:
	for _, line := range lines {
		for _, sb := range sub {
			if bytes.HasPrefix(line, sb) {
				continue loop
			}
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}

	return buf.Bytes()
}
