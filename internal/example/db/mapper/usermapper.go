//go:build vulcan

package mapper

//go:generate ${GOPATH}/bin/vulcan gen db
import (
	"database/sql"
	"github.com/mangohow/vulcan"
	. "github.com/mangohow/vulcan/annotation"
	"github.com/mangohow/vulcan/internal/example/model"
)

type UserMapper struct {
	db *sql.DB
}

func NewUserMapper(db *sql.DB) *UserMapper {
	return &UserMapper{
		db: db,
	}
}

func (m *UserMapper) Add(user *model.User) {
	Insert(`INSERT INTO t_user (id, username, password, create_at, email, address) 
            VALUES (#{user.Id}, #{user.Username}, #{user.Password}, #{user.Create_at}, #{user.Email}, #{user.Address})`)
}

func (m *UserMapper) DeleteById(id int) int {
	Delete("DELETE FROM t_user WHERE id = #{id}")
	return 0
}

func (m *UserMapper) UpdateById(user *model.User) int {
	a := true
	Update(SQL().Stmt("UPDATE t_user").
		Set(If(user.Password != "", "password = #{user.Password}").
			If(user.Email != "", "email = #{user.Email}").
			If(user.Address != "" && (user.Id > 0 || a), "address = #{user.Address}")).
		Stmt("WHERE id = #{user.Id}").Build())
	return 0
}

func (m *UserMapper) FindById(id int) *model.User {
	Select("SELECT * FROM t_user WHERE id = #{id}")
	return nil
}

func (m *UserMapper) Find(user *model.User) *model.User {
	Select(SQL().
		Stmt("SELECT * FROM t_user").
		Where(If(user.Username != "", "username = #{user.Username}").
			If(user.Address != "", "address = #{user.Address}")).
		Build())
	panic("")
}

func (m *UserMapper) BatchAdd(users []*model.User) {
	Insert(SQL().
		Stmt("INSERT INTO t_user (id, username, password, create_at, email, address) VALUES ").
		Foreach("users", "user", " ", "", "",
			"(#{user.Id}, #{user.Username}, #{user.Password}, #{user.CreateAt}, #{user.Email}, #{user.Address})").Build())
}

func (m *UserMapper) UpdateByIdOrUsername(user *model.User) {
	Update(SQL().
		Stmt("UPDATE t_user").
		Set(If(user.Password != "", "password = #{user.Password}").
			If(user.Email != "", "email = #{user.Email}")).
		Where(Choose().When(user.Id > 0, "id = #{user.Id}").
			When(user.Username != "", "username = #{user.Username}")).
		Build())
}

func (u *UserMapper) SelectPage(page vulcan.Page, cond *model.QueryCond) []*model.User {
	Select(SQL().
		Stmt("SELECT * FROM t_user").
		Where(If(cond.Username != "", "And username = #{cond.Username}").
			If(cond.Address != "", "AND address = #{cond.Address} ")).Build())
	return nil
}

func (u *UserMapper) SelectListByIds(ids []int) []*model.User {
	Select(SQL().
		Stmt("SELECT * FROM t_user WHERE id IN").
		Foreach("ids", "id", ", ", "(", ")", "#{id}").
		Build())
}
