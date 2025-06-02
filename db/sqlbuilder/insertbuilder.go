package sqlbuilder

import (
	"strings"
)

type InsertBuilder struct {
	Field     []string
	Batch     int // 批量插入时需要
	TableName string
}

func (b *InsertBuilder) Build() string {
	if b.Batch <= 0 {
		b.Batch = 1
	}
	builder := strings.Builder{}
	builder.Grow(64)
	builder.WriteString("INSERT INTO ")
	builder.WriteString(b.TableName)
	builder.WriteString(" (")
	builder.WriteString(strings.Join(b.Field, ", "))
	builder.WriteString(") VALUES")
	bb := strings.Builder{}
	bb.Grow(len(b.Field)*3 + 2)
	bb.WriteString(" (")
	for i := 0; i < len(b.Field); i++ {
		if i == len(b.Field)-1 {
			bb.WriteString("?")
		} else {
			bb.WriteString("?, ")
		}
	}
	bb.WriteString(")")
	values := bb.String()
	for i := 0; i < b.Batch; i++ {
		builder.WriteString(values)
	}

	return builder.String()
}
