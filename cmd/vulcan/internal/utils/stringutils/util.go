package stringutils

import "strings"

// TrimTrailingRedundantSpaces 使字符串尾部只保留一个空格, 前面的空格也去掉
func TrimTrailingRedundantSpaces(s string) string {
	return strings.TrimRight(s, " ") + " "
}

func ToPascalCase(snake string) string {
	parts := strings.Split(snake, "_")
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, strings.Title(part))
		}
	}
	return strings.Join(result, "")
}

func ToPascalCaseByList(parts []string) string {
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, strings.Title(part))
		}
	}
	return strings.Join(result, "")
}

func UpperFirstLittle(str string) string {
	if str == "" {
		return str
	}

	return strings.ToUpper(str[:1]) + str[1:]
}

func LowerFirstLittle(str string) string {
	if str == "" {
		return str
	}

	return strings.ToLower(str[:1]) + str[1:]
}

func IsUpperLetter(str string) bool {
	for i := range str {
		if !(str[i] >= 'A' && str[i] <= 'Z') {
			return false
		}
	}

	return true
}
