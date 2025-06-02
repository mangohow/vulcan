package sqlbuilder

import "strings"

type DeleteBuilder struct {
	TableName string
	Condition []string
}

func (b *DeleteBuilder) Build() string {
	builder := strings.Builder{}
	builder.WriteString("DELETE FROM ")
	builder.WriteString(b.TableName)
	if len(b.Condition) == 0 {
		return builder.String()
	}
	builder.WriteString(" WHERE ")
	for i, cond := range b.Condition {
		if i > 0 {
			builder.WriteString(" AND ")
		}
		builder.WriteString(cond + " = ?")
	}

	return builder.String()
}
