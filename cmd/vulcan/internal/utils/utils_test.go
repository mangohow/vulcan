package utils

import (
	"fmt"
	"testing"
)

func TestGetCurrentPackagePath(t *testing.T) {
	path, err := GetCurrentPackagePath("E:\\go_workspace\\src\\projects\\vulcan\\internal\\utils\\utils.go")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(path)
}

func TestGetPackageNameByDir(t *testing.T) {
	packageName, err := GetPackageNameByDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(packageName)
}
