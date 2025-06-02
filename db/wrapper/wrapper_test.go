package wrapper

import "github.com/mangohow/vulcan/db/types"

type User struct {
	types.TableName `tableName:"t_user"`
	Id              int    `tableField:"id"`
	Username        string `tableField:"username"`
	Password        string `tableField:"password"`
}
