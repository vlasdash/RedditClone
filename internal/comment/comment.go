package comment

import (
	"errors"
)

var (
	ErrNotExist  = errors.New("comment with specified id not exist")
	ErrNoAccess  = errors.New("hasn`t access to delete comment")
	ErrInvalidID = errors.New("comment id is invalid")
)

type Comment struct {
	ID         string
	AuthorID   uint
	CreateDate string
	Body       string
}

type CommentRepo interface {
	GetByID(id string) (*Comment, error)
	Add(userID uint, body string) (string, error)
	Delete(id string, userID uint) error
}
