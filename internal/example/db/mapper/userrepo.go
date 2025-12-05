//go:build vulcan

//go:generate ${GOPATH}/bin/vulcan gen db
package mapper

import (
	"database/sql"

	"github.com/mangohow/vulcan"
	. "github.com/mangohow/vulcan/annotation"
	"github.com/mangohow/vulcan/internal/example/model"
)

type UserRepo struct {
	db           *sql.DB
	cacheManager vulcan.CacheManger[model.User]
}

func NewUserRepo(db *sql.DB, cacheManager vulcan.CacheManger[model.User]) *UserRepo {
	return &UserRepo{
		db:           db,
		cacheManager: cacheManager,
	}
}

func (m *UserRepo) Add(user *model.User) {
	Insert(`INSERT INTO t_user (id, username, password, create_at, email, address) 
            VALUES (#{user.Id}, #{user.Username}, #{user.Password}, #{user.CreateAt}, #{user.Email}, #{user.Address})`)
}

func (m *UserRepo) Add1(user *model.User) int {
	Insert(`INSERT INTO t_user (id, username, password, create_at, email, address) 
            VALUES (#{user.Id}, #{user.Username}, #{user.Password}, #{user.CreateAt}, #{user.Email}, #{user.Address})`)
}

func (m *UserRepo) DeleteById(id int) int {
	Delete("DELETE FROM t_user WHERE id = #{id}")
	return 0
}

func (m *UserRepo) FindById(id int) *model.User {
	Select("SELECT * FROM t_user WHERE id = #{id}")
	return nil
}

func (m *UserRepo) UpdateById(user *model.User) int {
	Update(SQL().Stmt("UPDATE t_user").
		Set(If(user.Password != "", "password = #{user.Password}").
			If(user.Email != "", "email = #{user.Email}").
			If(user.Address != "", "address = #{user.Address}")).
		Stmt("WHERE id = #{user.Id}").Build())
	return 0
}

func (m *UserRepo) Find(user *model.User) *model.User {
	Select(SQL().
		Stmt("SELECT * FROM t_user").
		Where(If(user.Username != "", "username = #{user.Username}").
			If(user.Address != "", "address = #{user.Address}")).
		Build())
	return nil
}

func (m *UserRepo) Find2(user *model.User) model.User {
	Select(SQL().
		Stmt("SELECT * FROM t_user").
		Where(If(user.Username != "", "username = #{user.Username}").
			If(user.Address != "", "address = #{user.Address}")).
		Build())
	return nil
}

func (m *UserRepo) BatchAdd(users []*model.User) {
	Insert(SQL().
		Stmt("INSERT INTO t_user (id, username, password, create_at, email, address) VALUES ").
		Foreach("users", "user", ", ", "", "",
			"(#{user.Id}, #{user.Username}, #{user.Password}, #{user.CreateAt}, #{user.Email}, #{user.Address})").Build())
}

func (m *UserRepo) UpdateByIdOrUsername(user *model.User) {
	Update(SQL().
		Stmt("UPDATE t_user").
		Set(If(user.Password != "", "password = #{user.Password}").
			If(user.Email != "", "email = #{user.Email}")).
		Where(Choose().When(user.Id > 0, "id = #{user.Id}").
			When(user.Username != "", "username = #{user.Username}")).
		Build())
}

func (u *UserRepo) SelectPage(page vulcan.Page, cond *model.QueryCond) []*model.User {
	Select(SQL().
		Stmt("SELECT * FROM t_user").
		Where(If(cond.Username != "", "And username = #{cond.Username}").
			If(cond.Address != "", "AND address = #{cond.Address} ")).Build())
	return nil
}

func (u *UserRepo) SelectBatchIds(ids []int) []*model.User {
	Select(SQL().
		Stmt("SELECT * FROM t_user WHERE id IN").
		Foreach("ids", "id", ", ", "(", ")", "#{id}").
		Build())
}

func (m *UserRepo) FindByIdCached(id int) *model.User {
	Select("SELECT * FROM t_user WHERE id = #{id}")
	Cacheable("user:id:#{id}", false)
	return nil
}

func (m *UserRepo) UpdateByIdEvict(user *model.User) int {
	Update(SQL().Stmt("UPDATE t_user").
		Set(If(user.Password != "", "password = #{user.Password}").
			If(user.Email != "", "email = #{user.Email}").
			If(user.Address != "", "address = #{user.Address}")).
		Stmt("WHERE id = #{user.Id}").Build())
	CacheEvict("user:id:#{user.Id}", false)
	return 0
}
