package user

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLRepo struct {
	DB *sql.DB
}

var _ UserRepo = (*MySQLRepo)(nil)

func NewMySQLRepo(db *sql.DB) *MySQLRepo {
	return &MySQLRepo{
		DB: db,
	}
}

func (r *MySQLRepo) Create(username string, password string) (id uint, err error) {
	result, err := r.DB.Exec(
		"INSERT INTO users (`username`, `password`) VALUES (?, ?)",
		username,
		password,
	)
	if err != nil {
		return 0, err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint(userID), nil
}

func (r *MySQLRepo) GetByUsername(username string) (*User, error) {
	row := r.DB.QueryRow(
		"SELECT id, username, password FROM users WHERE username = ?",
		username,
	)

	user := &User{}
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		return nil, ErrNoExist
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *MySQLRepo) GetByID(id uint) (*User, error) {
	row := r.DB.QueryRow(
		"SELECT id, username, password FROM users WHERE id = ?",
		id,
	)

	user := &User{}
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		return nil, ErrNoExist
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}
