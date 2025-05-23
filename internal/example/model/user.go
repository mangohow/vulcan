package model

import "time"

type User struct {
	Id        int64     `db:"id,pk"`
	Username  string    `db:"username"`
	Password  string    `db:"password"`
	CreatedAt time.Time `db:"created_at"`
	Email     string    `db:"email"`
	Address   string    `db:"address"`
}

type QueryCond struct {
	Username string
	Address  string
}
