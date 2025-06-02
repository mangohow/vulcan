package types

type SqlKeyWord string

const (
	And     SqlKeyWord = "AND"
	Or      SqlKeyWord = "OR"
	Not     SqlKeyWord = "NOT"
	In      SqlKeyWord = "IN"
	NotIn   SqlKeyWord = "NOT IN"
	Like    SqlKeyWord = "LIKE"
	NotLike SqlKeyWord = "NOT LIKE"
	Eq      SqlKeyWord = "="
	Ne      SqlKeyWord = "<>"
	Gt      SqlKeyWord = ">"
	Ge      SqlKeyWord = ">="
	Lt      SqlKeyWord = "<"
	Le      SqlKeyWord = "<="
	GroupBy SqlKeyWord = "GROUP BY"
	Asc     SqlKeyWord = "ASC"
	Desc    SqlKeyWord = "DESC"
)
