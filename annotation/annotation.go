package annotation

const (
	tip = "use vulcan to generate code"
)

func Select(sql string) string {
	panic(tip)
}

func Update(sql string) string {
	panic(tip)
}

func Insert(sql string) string {
	panic(tip)
}

func Delete(sql string) string {
	panic(tip)
}

type sqBuilder interface {
	Stmt(string) sqBuilder
	Where(cond) sqBuilder
	Set(cond) sqBuilder
	If(bool, string) sqBuilder
	Foreach(collection, itemName, separator, open, close, sql string) sqBuilder
	Build() string
}

type cond interface {
	noImplement()
}

type ifStmt interface {
	cond
	If(bool, string) ifStmt
}

type chooseStmt interface {
	cond
	When(bool, string) chooseStmt
	Otherwise(string)
}

func SQL() sqBuilder {
	return sqBuilder(nil)
}
func If(bool, string) ifStmt {
	return ifStmt(nil)
}

func Choose() chooseStmt {
	return chooseStmt(nil)
}
