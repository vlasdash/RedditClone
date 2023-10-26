package comment

import (
	"strconv"
	"sync"
	"time"
)

type MemoryRepo struct {
	idCount  uint
	comments []*Comment
	mu       *sync.RWMutex
}

var _ CommentRepo = (*MemoryRepo)(nil)

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		idCount:  0,
		comments: make([]*Comment, 0, 2),
		mu:       &sync.RWMutex{},
	}
}

func (r *MemoryRepo) Add(userID uint, body string) (id string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.idCount++
	id = strconv.Itoa(int(r.idCount))
	r.comments = append(r.comments, &Comment{
		ID:         id,
		CreateDate: time.Now().Format(time.RFC3339),
		Body:       body,
		AuthorID:   userID,
	})

	return id, nil
}

func (r *MemoryRepo) GetByID(id string) (*Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, comment := range r.comments {
		if id != comment.ID {
			continue
		}

		return comment, nil
	}

	return nil, ErrNotExist
}

func (r *MemoryRepo) Delete(id string, userID uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.comments {
		if r.comments[i].ID != id {
			continue
		}
		if r.comments[i].AuthorID != userID {
			return ErrNoAccess
		}

		copy(r.comments[i:], r.comments[i+1:])
		r.comments[len(r.comments)-1] = nil
		r.comments = r.comments[:len(r.comments)-1]

		return nil
	}

	return ErrNotExist
}
