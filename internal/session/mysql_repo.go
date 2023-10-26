package session

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"strings"
	"time"
)

type MySQLRepo struct {
	DB        *sql.DB
	Generator TokenGenerator
}

var _ SessionRepo = (*MySQLRepo)(nil)

func NewMySQLRepo(db *sql.DB, generator TokenGenerator) *MySQLRepo {
	return &MySQLRepo{
		DB:        db,
		Generator: generator,
	}
}

func (r *MySQLRepo) Get(accessToken string) (*Session, error) {
	accessToken = strings.Split(accessToken, " ")[1]
	row := r.DB.QueryRow(
		"SELECT username, user_id, expiration_date FROM sessions WHERE token = ?",
		accessToken,
	)

	session := &Session{}
	err := row.Scan(&session.Username, &session.UserID, &session.ExpirationDate)
	if err == sql.ErrNoRows {
		return nil, ErrBadToken
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	expirationDate, err := strconv.Atoi(session.ExpirationDate)
	if err != nil {
		return nil, err
	}
	if int(now) > expirationDate {
		return nil, ErrTokenExpired
	}

	return session, nil
}

func (r *MySQLRepo) Add(username string, userID uint) (tokenStr string, err error) {
	token, exp, err := r.Generator.Generate(username, userID)
	if err != nil {
		return "", ErrUnableGenerateToken
	}

	_, err = r.DB.Exec(
		"INSERT INTO sessions (`token`, `username`, `user_id`, `expiration_date`) VALUES (?, ?, ?, ?)",
		token,
		username,
		userID,
		exp,
	)
	if err != nil {
		return "", err
	}

	return token, nil
}
