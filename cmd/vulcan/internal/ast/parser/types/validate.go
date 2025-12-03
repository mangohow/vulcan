package types

import (
	"log"
	"strings"

	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
)

var (
	supportedBasicType = []string{
		"int",
		"int64",
		"int32",
		"int16",
		"int8",
		"uint",
		"uint64",
		"uint32",
		"uint16",
		"uint8",
		"float64",
		"float32",
		"string",
		"bool",
		"[]byte",
		"time.Time",
	}
	supportedColumnType = []string{
		"sql.NullTime",
		"sql.NullInt64",
		"sql.NullInt32",
		"sql.NullInt16",
		"sql.NullFloat64",
		"sql.NullBool",
		"sql.NullString",
		"sql.NullByte",
		"sql.Null",
		"sql.RawBytes",
	}
)

func IsTypeSupported(typeName string, isGenericType bool) bool {
	if isGenericType {
		s1, s2, found := strings.Cut(typeName, "[")
		if !found {
			log.Fatal("type %s is invalid", typeName)
		}
		s2 = strings.TrimSuffix(s2, "]")
		return s1 == "sql.Null" && utils.Contains(supportedBasicType, s2)
	}

	if utils.ContainsPrefix(supportedColumnType, typeName) || utils.Contains(supportedBasicType, typeName) {
		return true
	}

	return false
}

func IsNullableType(typeName string) bool {
	for _, s := range supportedColumnType {
		if strings.HasPrefix(typeName, s) {
			return true
		}
	}

	return false
}
