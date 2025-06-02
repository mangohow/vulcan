package types

const (
	// TableNameTagKey 使用该Tag Key指定表名
	TableNameTagKey = "tableName"
	// TableFieldTagKey 使用该Tag Key指定表中字段名
	TableFieldTagKey = "tableField"
	// TablePrimaryIdTagValue 使用该Tag Value指定主键
	TablePrimaryIdTagValue = "primary"

	// TableAutoFillTagValue 使用该Tag时自动填充Id, 但是要保证主键Id是自增的
	TableAutoFillTagValue = "autoIncrement"
)
