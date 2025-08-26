package mysql

import (
	"database/sql"
	"fmt"
	"integration/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

func New(params config.MySQLConnect) (*sql.DB, error) {
	const op = "storage.driver.mysql.New"

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		params.Username,
		params.Password,
		params.Host,
		params.Port,
		params.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		return nil, fmt.Errorf("%s", pingErr)
	}

	return db, nil

}
