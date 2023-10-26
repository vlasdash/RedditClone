package user

import "golang.org/x/crypto/bcrypt"

type BcryptHasher struct{}

func (h *BcryptHasher) IsPassword(hash string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}

func (h *BcryptHasher) GetHashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
