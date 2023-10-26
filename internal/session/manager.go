package session

import (
	"github.com/vlasdash/redditclone/internal/user"
)

type Manager struct {
	userRepo    user.UserRepo
	sessionRepo SessionRepo
}

func NewManager(sr SessionRepo, ur user.UserRepo) *Manager {
	return &Manager{
		userRepo:    ur,
		sessionRepo: sr,
	}
}

func (m *Manager) Create(accessToken string) (*Session, error) {
	return m.sessionRepo.Get(accessToken)
}

func (m *Manager) HasUserExist(s *Session) (bool, error) {
	u, err := m.userRepo.GetByID(s.UserID)
	if err == user.ErrNoExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if u.Username != s.Username {
		return false, nil
	}

	return true, nil
}
