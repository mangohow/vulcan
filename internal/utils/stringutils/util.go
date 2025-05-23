package stringutils

import "strings"

// TrimTrailingRedundantSpaces 使字符串尾部只保留一个空格, 前面的空格也去掉
func TrimTrailingRedundantSpaces(s string) string {
	return strings.TrimRight(s, " ") + " "
}
