package annotation

import "time"

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

func Cacheable(key string, cacheNil bool, queryTimeOut time.Duration) {
	panic(tip)
}

func CacheEvict(key string, beforeInvocation bool) {
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

// TableProperty 使用该类型在一个model结构体中通过tag指定生成的代码所需的配置
// 1、使用tableName指定表名称
// tableName: xxx
// 例如:
//
//		type User struct {
//		    vulcan.TableProperty `tableName:"t_user"`
//			Id       int     `db:"id,pk"`
//			UserName string  `db:"username"`
//	     	Password string  `db:"password"`
//			Email    string  `db:"email"`
//			Address  string  `db:"address"`
//		}
//
// 2、使用gen指定需要生成的函数列表, 如果不指定, 则默认全部生成
// 函数列表如下：
// Add: 新增操作
// AddBatch: 批量新增
// DeleteById: 根据主键Id删除
// DeleteBatchIds: 根据主键Id列表删除
// SelectById: 根据主键Id查询
// SelectBatchIds: 根据主键Id列表查询
// SelectAll: 查询全部
// SelectCount: 查询总数
// 以下函数需要添加参数, 其中有三种参数：
//	Where条件参数：该参数用于指定查询时根据哪些列进行查询, 使用AND[]或OR[], 如果两者都有使用|进行分割, 列在中括号中指定, 也可以省略AND, 默认为AND
//				可以使用列的索引（从0开始）, 比如 AND[2 3 5]表示根据索引为2 3 5的列进行查询
//				索引也可以指定一个范围, 比如 AND[2-4]表示根据索引为2 3 4的列进行查询
//				也可以使用列的名称, 比如 AND[username password]表示根据根据username和password查询
//				这三种条件可以混用
//  查询或更新条件参数：该参数用于指定查询时查询出哪些列或者更新时更新哪些列, 使用[], 中括号内部指定索引或名称
//  查询时或更新时是否判空：该参数为一个bool字面量, 用于指定在查询或更新时是否需要对字段进行判空, 如果为空则不作为查询或更新条件
// DeleteBy[2&6]: 根据Where条件参数进行删除
//
// UpdateById[2-4,6]: 根据Id更新指定的列
// UpdateByXXX[2,4,6][1]: 根据Where条件参数更新指定的列, 其中XXX后缀可以由用户任意指定, 第一个参数为Where条件参数, 第二个参数为要更新的列
//
// SelectOneByXXX[][]: 根据Where条件参数查询指定的一列, 其中XXX后缀可以由用户任意指定, 比如SelectOneByName; 参数为Where条件参数
// SelectListByXXX[][]: 根据Where条件参数查询指定多列, 其中XXX后缀可以由用户任意指定, 比如SelectListByName; 第一个参数为Where条件参数, 第二个参数为要查询的列
// SelectCountByXXX[]: 根据Where条件参数查询数量, 其中XXX后缀可以由用户任意指定, 比如SelectListByName; 参数为Where条件参数
// SelectPageByXXX[][]: 根据Where条件参数分页查询, 其中XXX后缀可以由用户任意指定, 比如SelectListByName; 第一个参数为Where条件参数,  第二个参数为要查询的列
// 注意： 上面的参数中, 如果有空参数, 则可以省略[]
//  	 在生成代码时, 不带参数的函数默认生成, 带参数的函数需要手动指定, 否则不会生成
// Where条件使用规则： [<字段标识>.[<操作符>][<逻辑运算符>...]]
//	[4, OR{NE(5), 6}]
// 支持的逻辑判断： EQ/NE/LT/GT/LE/GE/IN/LIKE/INL/IOL

// updateById中的参数指定需要更新的字段, 使用字段在结构体中的index, 可以使用单数字, 也可以使用index1-index2表示, 闭区间
// 例如：下面的示例指定了要生成的函数有Add、DeleteById、UpdateById和GetById
//
//	      其中UpdateById函数中根据Id更新索引为2、3、4的字段，即Password、Email、Address, true表示在更新时需要判断该字段是不是空(默认值, 字符串为空字符串, int为0...)
//
//			type User struct {
//			    vulcan.TableProperty `gen:"Add,DeleteById,UpdateById([2-4], true),GetById"`
//		 	}
type TableProperty struct{}
