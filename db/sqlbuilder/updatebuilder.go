package sqlbuilder

import "strings"

type UpdateBuilder struct {
	Fields    []string
	TableName string
	Condition []string
}

func (b *UpdateBuilder) Build() string {
	builder := strings.Builder{}
	builder.Grow(64)
	builder.WriteString("UPDATE ")
	builder.WriteString(b.TableName)
	builder.WriteString(" SET ")
	for i, field := range b.Fields {
		if i == len(b.Fields)-1 {
			builder.WriteString(field + " = ?")
		} else {
			builder.WriteString(field + " = ?, ")
		}
	}
	if len(b.Condition) > 0 {
		builder.WriteString(" WHERE ")
	}
	for i, cond := range b.Condition {
		if i == len(b.Condition)-1 {
			builder.WriteString(cond + " = ?")
		} else {
			builder.WriteString(cond + " = ?, ")
		}
	}

	return builder.String()
}
