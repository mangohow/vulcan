package model

import (
	"time"

	"github.com/mangohow/vulcan/annotation"
)

type User struct {
	annotation.TableProperty `tableName:"t_user" gen:"UpdateById([3 5 6], true)|SelectOneByUsernameAndPassword([2 3], [], false)"`
	Id                       int64     `db:"id,pk"`
	Username                 string    `db:"username"`
	Password                 string    `db:"password"`
	CreatedAt                time.Time `db:"created_at"`
	Email                    string    `db:"email"`
	Address                  string    `db:"address"`
}

type QueryCond struct {
	Username string
	Address  string
}
