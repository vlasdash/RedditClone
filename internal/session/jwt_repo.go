package session

import (
	"github.com/dgrijalva/jwt-go"
	"strconv"
	"strings"
)

type JWTRepo struct {
	Generator TokenGenerator
}

func NewJWTRepo(generator TokenGenerator) *JWTRepo {
	return &JWTRepo{
		Generator: generator,
	}
}

var _ SessionRepo = (*JWTRepo)(nil)

func (r *JWTRepo) Get(accessToken string) (*Session, error) {
	tokenParts := strings.Split(accessToken, " ")
	hashSecretGetter := func(token *jwt.Token) (interface{}, error) {
		method, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok || method.Alg() != "HS256" {
			return nil, ErrBadSigningMethod
		}

		return tokenSecret, nil
	}

	token, err := jwt.Parse(tokenParts[1], hashSecretGetter)
	if err != nil || !token.Valid {
		return nil, ErrBadToken
	}

	payload, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrEmptyPayload
	}
	userInfo, ok := payload["user"].(map[string]interface{})
	if !ok {
		return nil, ErrEmptyUserInfo
	}

	id, err := strconv.Atoi(userInfo["id"].(string))
	if err != nil {
		return nil, err
	}

	sess := &Session{
		UserID:   uint(id),
		Username: userInfo["username"].(string),
	}
	return sess, nil
}

func (r *JWTRepo) Add(username string, userID uint) (token string, err error) {
	token, _, err = r.Generator.Generate(username, userID)

	return token, err
}
