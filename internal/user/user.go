package user

import (
	"errors"
)

var (
	ErrNoExist = errors.New("user doesn`t exist")
)

type User struct {
	ID       uint   `json:"id,string"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type UserRepo interface {
	GetByUsername(username string) (*User, error)
	GetByID(id uint) (*User, error)
	Create(username string, password string) (id uint, err error)
}

type PasswordHasher interface {
	IsPassword(hash string, password string) bool
	GetHashPassword(password string) (string, error)
}
