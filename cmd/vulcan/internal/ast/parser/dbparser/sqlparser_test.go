package dbparser

import (
	"encoding/json"
	"fmt"
	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"testing"
)

func TestParseCreationFields(t *testing.T) {
	createTable := `CREATE TABLE example_table (
    -- int 类型
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT 'bigint -> int64',
    int_col INT NOT NULL COMMENT 'int -> int32',
    mediumint_col MEDIUMINT COMMENT 'mediumint -> int64',
    smallint_col SMALLINT COMMENT 'smallint -> int16',
    tinyint_col TINYINT COMMENT 'tinyint -> int8',
    
    -- uint 类型
    uint_col INT UNSIGNED COMMENT 'uint -> uint32',


    -- float 类型
    float_col FLOAT COMMENT 'float -> float32',
    double_col DOUBLE COMMENT 'double -> float64',

    -- string 类型
    char_col CHAR(255) COMMENT 'char -> string',
    varchar_col VARCHAR(255) COMMENT 'varchar -> string',
    text_col TEXT COMMENT 'text -> string',
    tinytext_col TINYTEXT COMMENT 'tinytext -> string',
    mediumtext_col MEDIUMTEXT COMMENT 'mediumtext -> string',
    longtext_col LONGTEXT COMMENT 'longtext -> string',

    -- []byte 类型
    binary_col BINARY(255) COMMENT 'binary -> []byte',
    varbinary_col VARBINARY(255) COMMENT 'varbinary -> []byte',
    blob_col BLOB COMMENT 'blob -> []byte',
    tinyblob_col TINYBLOB COMMENT 'tinyblob -> []byte',
    mediumblob_col MEDIUMBLOB COMMENT 'mediumblob -> []byte',
    longblob_col LONGBLOB COMMENT 'longblob -> []byte',

    -- 时间日期类型
    date_col DATE COMMENT 'date -> time.Time',
    time_col TIME COMMENT 'time -> time.Time',
    datetime_col DATETIME COMMENT 'datetime -> time.Time',
    timestamp_col BIGINT COMMENT 'timestamp -> int64 (存储为 UNIX 时间戳)',
    year_col YEAR COMMENT 'year -> int8'
)
ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;`

	statement, err := sqlparser.Parse(createTable)
	if err != nil {
		t.Fatal(err)
	}
	stmt, ok := statement.(*sqlparser.CreateTable)
	if !ok {
		t.Fail()
	}

	spec := ParseCreationFields(stmt)
	content, _ := json.MarshalIndent(spec, "", "    ")
	t.Log(string(content))
}

func TestParseSqlFile(t *testing.T) {
	specs, err := ParseSqlFile("E:\\go_workspace\\src\\vulcan_test\\db\\blog.sql")
	if err != nil {
		t.Fatal(err)
	}
	content, err := json.MarshalIndent(specs, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(content))
}
