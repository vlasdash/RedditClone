package user

import (
	"sync"
)

type MemoryRepo struct {
	idCount uint
	users   []*User
	mu      *sync.RWMutex
}

var _ UserRepo = (*MemoryRepo)(nil)

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		idCount: 0,
		users:   make([]*User, 0, 2),
		mu:      &sync.RWMutex{},
	}
}

func (r *MemoryRepo) Create(username string, password string) (id uint, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.idCount++
	r.users = append(r.users, &User{
		ID:       r.idCount,
		Username: username,
		Password: password,
	})

	return r.idCount, nil
}

func (r *MemoryRepo) GetByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Username != username {
			continue
		}

		return user, nil
	}

	return nil, ErrNoExist
}

func (r *MemoryRepo) GetByID(id uint) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.ID != id {
			continue
		}

		return user, nil
	}

	return nil, ErrNoExist
}
