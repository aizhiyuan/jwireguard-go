package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// ----------------------------------------------------------------------------------------------------------
// 初始化SQLite3 数据库
// ----------------------------------------------------------------------------------------------------------
func InitDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
