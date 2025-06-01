package utils

import "testing"

func TestGetCurrentPackagePath(t *testing.T) {
    path, err := GetCurrentPackagePath("E:\\go_workspace\\src\\projects\\vulcan\\internal\\utils\\utils.go")
    if err != nil {
        t.Fatal(err)
    }
    t.Log(path)
}
