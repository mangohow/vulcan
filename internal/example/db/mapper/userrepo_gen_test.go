package mapper

import (
	"database/sql"
	"fmt"
	"github.com/mangohow/vulcan"
	"github.com/mangohow/vulcan/internal/example/model"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var testDB *sql.DB

type MockCacheManager struct {
	store map[string]*model.User
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		store: make(map[string]*model.User),
	}
}

func (m *MockCacheManager) Get(key string) (*model.User, bool) {
	val, ok := m.store[key]
	if !ok {
		return nil, false
	}

	clone := *val
	return &clone, true
}

func (m *MockCacheManager) Set(key string, value *model.User) {
	m.store[key] = value
}

func (m *MockCacheManager) Delete(key string) {
	delete(m.store, key)
}

type Debugger struct {
}

func (d Debugger) Debug(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func TestMain(m *testing.M) {
	// 初始化测试数据库连接
	// 注意：这里使用环境变量或者测试数据库配置
	db, err := sql.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&loc=Local")
	if err != nil {
		// 如果无法连接到数据库，则跳过测试
		println("Skipping database tests: unable to connect to test database")
		os.Exit(0)
	}

	testDB = db
	defer testDB.Close()

	// 创建测试表
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS t_user (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(50) NOT NULL,
			password VARCHAR(100) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			email VARCHAR(100),
			address VARCHAR(200)
		)
	`)
	if err != nil {
		panic(err)
	}

	vulcan.SetupSqlDebugInterceptor(Debugger{})

	// 运行测试
	code := m.Run()

	// 清理测试数据
	testDB.Exec("DROP TABLE IF EXISTS t_user")

	// 退出
	os.Exit(code)
}

func setupTestData() (*UserRepo, *model.User) {
	repo := &UserRepo{
		db:           testDB,
		cacheManager: NewMockCacheManager(),
	}

	user := &model.User{
		Username:  "testuser",
		Password:  "password",
		CreatedAt: time.Now(),
		Email:     "test@example.com",
		Address:   "test address",
	}

	return repo, user
}

func TestUserRepo_Add(t *testing.T) {
	repo, user := setupTestData()

	err := repo.Add(user, nil)
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	if user.Id <= 0 {
		t.Error("Expected user ID to be set")
	}
}

func TestUserRepo_Add1(t *testing.T) {
	repo, user := setupTestData()
	user.Username = "testuser_add1"

	affected, err := repo.Add1(user, nil)
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}

	if user.Id <= 0 {
		t.Error("Expected user ID to be set")
	}
}

func TestUserRepo_DeleteById(t *testing.T) {
	repo, user := setupTestData()
	user.Username = "testuser_delete"

	// 先插入一条记录
	err := repo.Add(user, nil)
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	// 删除记录
	affected, err := repo.DeleteById(int(user.Id), nil)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func TestUserRepo_FindById(t *testing.T) {
	repo, user := setupTestData()
	user.Username = "testuser_find"

	// 先插入一条记录
	err := repo.Add(user, nil)
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	// 查询记录
	foundUser, err := repo.FindById(int(user.Id), nil)
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	if foundUser == nil {
		t.Fatal("Expected user to be found")
	}

	if foundUser.Id != user.Id {
		t.Errorf("Expected user ID %d, got %d", user.Id, foundUser.Id)
	}

	if foundUser.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, foundUser.Username)
	}
}

func TestUserRepo_UpdateById(t *testing.T) {
	repo, user := setupTestData()
	user.Username = "testuser_update"

	// 先插入一条记录
	err := repo.Add(user, nil)
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	// 更新记录
	user.Password = "newpassword"
	user.Email = "new@example.com"
	user.Address = "new address"

	affected, err := repo.UpdateById(user, nil)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}

	// 验证更新结果
	foundUser, err := repo.FindById(int(user.Id), nil)
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	if foundUser.Password != user.Password {
		t.Errorf("Expected password %s, got %s", user.Password, foundUser.Password)
	}

	if foundUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, foundUser.Email)
	}

	if foundUser.Address != user.Address {
		t.Errorf("Expected address %s, got %s", user.Address, foundUser.Address)
	}
}

func TestUserRepo_Find(t *testing.T) {
	repo, user := setupTestData()
	user.Username = "testuser_find_cond"

	// 先插入一条记录
	err := repo.Add(user, nil)
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	// 条件查询
	cond := &model.User{
		Username: "testuser_find_cond",
		Password: "password",
	}

	foundUser, err := repo.Find(cond)
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	if foundUser == nil {
		t.Fatal("Expected user to be found")
	}

	if foundUser.Username != cond.Username {
		t.Errorf("Expected username %s, got %s", cond.Username, foundUser.Username)
	}
}

func TestUserRepo_Find2(t *testing.T) {
	repo, user := setupTestData()
	user.Username = "testuser_find2"

	// 先插入一条记录
	err := repo.Add(user, nil)
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	// 条件查询
	cond := &model.User{
		Username: "testuser_find2",
		Password: "password",
	}

	foundUser, err := repo.Find2(cond)
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	if foundUser.Username != cond.Username {
		t.Errorf("Expected username %s, got %s", cond.Username, foundUser.Username)
	}
}

func TestUserRepo_BatchAdd(t *testing.T) {
	repo := &UserRepo{db: testDB}

	users := []*model.User{
		{
			Username:  "testuser_batch1",
			Password:  "password1",
			CreatedAt: time.Now(),
			Email:     "test1@example.com",
			Address:   "test address 1",
		},
		{
			Username:  "testuser_batch2",
			Password:  "password2",
			CreatedAt: time.Now(),
			Email:     "test2@example.com",
			Address:   "test address 2",
		},
	}

	affected, err := repo.BatchAdd(users, nil)
	if err != nil {
		t.Fatalf("Failed to batch add users: %v", err)
	}

	if affected != 2 {
		t.Errorf("Expected 2 rows affected, got %d", affected)
	}
}

func TestUserRepo_SelectBatchIds(t *testing.T) {
	repo := &UserRepo{db: testDB}

	// 先插入几条记录
	users := []*model.User{
		{
			Username:  "testuser_batch_ids1",
			Password:  "password1",
			CreatedAt: time.Now(),
			Email:     "test1@example.com",
			Address:   "test address 1",
		},
		{
			Username:  "testuser_batch_ids2",
			Password:  "password2",
			CreatedAt: time.Now(),
			Email:     "test2@example.com",
			Address:   "test address 2",
		},
	}

	_, err := repo.BatchAdd(users, nil)
	if err != nil {
		t.Fatalf("Failed to batch add users: %v", err)
	}

	// 获取插入记录的ID
	user1, err := repo.Find(&model.User{Username: "testuser_batch_ids1"})
	if err != nil {
		t.Fatalf("Failed to find user1: %v", err)
	}

	user2, err := repo.Find(&model.User{Username: "testuser_batch_ids2"})
	if err != nil {
		t.Fatalf("Failed to find user2: %v", err)
	}

	ids := []int{int(user1.Id), int(user2.Id)}
	result, err := repo.SelectBatchIds(ids)
	if err != nil {
		t.Fatalf("Failed to select batch ids: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 users, got %d", len(result))
	}
}
