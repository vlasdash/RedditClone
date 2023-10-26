package session

import (
	"context"
	"errors"
	"github.com/vlasdash/redditclone/config"
)

const (
	SessionKey = "session-key"
)

var tokenSecret = []byte(config.C.App.SecretKey)

var (
	ErrBadSigningMethod    = errors.New("invalid signing method")
	ErrBadToken            = errors.New("bad token")
	ErrEmptyPayload        = errors.New("empty payload")
	ErrEmptyUserInfo       = errors.New("empty information about user")
	ErrEmptyExpirationDate = errors.New("empty token expiration date info")
	ErrNoAuthentication    = errors.New("unauthorized")
	ErrUnableGenerateToken = errors.New("can`t create token for user")
	ErrTokenExpired        = errors.New("token expiration date has passed")
)

type Session struct {
	UserID         uint
	Username       string
	ExpirationDate string
}

type SessionRepo interface {
	Get(accessToken string) (*Session, error)
	Add(username string, userID uint) (tokenStr string, err error)
}

type TokenGenerator interface {
	Generate(username string, userID uint) (tokenStr string, exp int64, err error)
}

func GetSessionFromContext(ctx context.Context) (*Session, error) {
	sess, ok := ctx.Value(SessionKey).(*Session)
	if !ok || sess == nil {
		return nil, ErrNoAuthentication
	}

	return sess, nil
}

func CreateContextWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, SessionKey, sess)
}
