package post

import (
	"errors"
)

const (
	Like   = 1
	Unlike = -1
)

var (
	ErrNotExist        = errors.New("post with specified id not exist")
	ErrCommentNotExist = errors.New("comment with specified id not exist")
	ErrNoAccess        = errors.New("hasn`t access to delete post")
	ErrInvalidID       = errors.New("post id is invalid")
)

type Vote struct {
	UserID uint `json:"user,string" bson:"user_id"`
	Value  int  `json:"vote" bson:"value"`
}

type Post struct {
	ID             string
	Category       string
	CreateDate     string
	Text           string
	URL            string
	Title          string
	Type           string
	Views          int
	Votes          []*Vote
	CommentIDs     []string
	AuthorID       uint
	UpvotesCount   int
	DownvotesCount int
}

type PostRepo interface {
	GetAll() ([]*Post, error)
	Create(post *Post) (id string, err error)
	GetByID(id string, viewsUpdate int) (*Post, error)
	GetByCategory(category string) ([]*Post, error)
	GetByAuthor(id uint) ([]*Post, error)
	AddComment(postID string, commentID string) error
	Upvote(postID string, voter uint) error
	Downvote(postID string, voter uint) error
	Unvote(postID string, voter uint) error
	Delete(postID string, userID uint) error
	DeleteComment(postID string, commentID string) error
}
