package session

import (
	"github.com/dgrijalva/jwt-go"
	"strconv"
	"time"
)

type JWTGenerator struct{}

var _ TokenGenerator = (*JWTGenerator)(nil)

func (g *JWTGenerator) Generate(username string, userID uint) (tokenStr string, exp int64, err error) {
	now := time.Now()
	exp = now.Add(3 * time.Hour).Unix()
	idStr := strconv.Itoa(int(userID))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]interface{}{
			"username": username,
			"id":       idStr,
		},
		"iat": now.Unix(),
		"exp": exp,
	})

	tokenStr, err = token.SignedString(tokenSecret)
	if err != nil {
		return "", 0, ErrUnableGenerateToken
	}

	return tokenStr, exp, nil
}
