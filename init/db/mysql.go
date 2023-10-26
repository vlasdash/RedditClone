package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vlasdash/redditclone/config"
)

func InitMySQL() (*sql.DB, error) {
	username := config.C.MySQL.User
	password := config.C.MySQL.Password
	host := config.C.MySQL.Host
	port := config.C.MySQL.Port
	dbName := config.C.MySQL.Name

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?&charset=utf8&interpolateParams=true",
		username,
		password,
		host,
		port,
		dbName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
