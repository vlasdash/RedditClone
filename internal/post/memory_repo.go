package post

import (
	"strconv"
	"sync"
	"time"
)

type MemoryRepo struct {
	idCount uint
	posts   []*Post
	mu      *sync.RWMutex
}

var _ PostRepo = (*MemoryRepo)(nil)

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		idCount: 0,
		posts:   make([]*Post, 0, 2),
		mu:      &sync.RWMutex{},
	}
}

func (r *MemoryRepo) GetAll() ([]*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	posts := make([]*Post, 0, 2)

	posts = append(posts, r.posts...)

	return posts, nil

}

func (r *MemoryRepo) Create(p *Post) (id string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.idCount++
	id = strconv.Itoa(int(r.idCount))
	p.ID = id
	p.CreateDate = time.Now().Format(time.RFC3339)
	p.Views = 0
	p.Votes = []*Vote{
		{
			UserID: p.AuthorID,
			Value:  Like,
		},
	}
	p.UpvotesCount = 1
	p.CommentIDs = make([]string, 0)

	r.posts = append(r.posts, p)

	return id, nil
}

func (r *MemoryRepo) GetByID(id string, viewsUpdate int) (*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, post := range r.posts {
		if post.ID != id {
			continue
		}

		post.Views += viewsUpdate
		return post, nil
	}

	return nil, ErrNotExist
}

func (r *MemoryRepo) GetByCategory(category string) ([]*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	posts := make([]*Post, 0, 2)

	for _, post := range r.posts {
		if post.Category != category {
			continue
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (r *MemoryRepo) GetByAuthor(id uint) ([]*Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	posts := make([]*Post, 0, 2)

	for _, post := range r.posts {
		if post.AuthorID != id {
			continue
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (r *MemoryRepo) AddComment(postID string, commentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.posts {
		if r.posts[i].ID != postID {
			continue
		}

		r.posts[i].CommentIDs = append(r.posts[i].CommentIDs, commentID)

		return nil
	}

	return ErrNotExist
}

func (r *MemoryRepo) Upvote(postID string, voter uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.posts {
		if r.posts[i].ID != postID {
			continue
		}

		for j := range r.posts[i].Votes {
			if r.posts[i].Votes[j].UserID != voter {
				continue
			}

			r.posts[i].Votes[j].Value = Like
			r.posts[i].UpvotesCount += 1
			r.posts[i].DownvotesCount -= 1

			return nil
		}

		r.posts[i].Votes = append(r.posts[i].Votes, &Vote{
			UserID: voter,
			Value:  Like,
		})
		r.posts[i].UpvotesCount += 1

		return nil
	}

	return ErrNotExist
}

func (r *MemoryRepo) Downvote(postID string, voter uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.posts {
		if r.posts[i].ID != postID {
			continue
		}

		for j := range r.posts[i].Votes {
			if r.posts[i].Votes[j].UserID != voter {
				continue
			}

			r.posts[i].Votes[j].Value = Unlike
			r.posts[i].DownvotesCount += 1
			r.posts[i].UpvotesCount -= 1

			return nil
		}

		r.posts[i].Votes = append(r.posts[i].Votes, &Vote{
			UserID: voter,
			Value:  Unlike,
		})
		r.posts[i].DownvotesCount += 1

		return nil
	}

	return ErrNotExist
}

func (r *MemoryRepo) Unvote(postID string, voter uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.posts {
		if r.posts[i].ID != postID {
			continue
		}

		for j := range r.posts[i].Votes {
			if r.posts[i].Votes[j].UserID != voter {
				continue
			}

			if r.posts[i].Votes[j].Value == Like {
				r.posts[i].UpvotesCount -= 1
			} else {
				r.posts[i].DownvotesCount -= 1
			}
			copy(r.posts[i].Votes[j:], r.posts[i].Votes[j+1:])
			r.posts[i].Votes[len(r.posts[i].Votes)-1] = nil
			r.posts[i].Votes = r.posts[i].Votes[:len(r.posts[i].Votes)-1]

			return nil
		}
	}

	return ErrNotExist
}

func (r *MemoryRepo) Delete(postID string, userID uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.posts {
		if r.posts[i].ID != postID {
			continue
		}
		if r.posts[i].AuthorID != userID {
			return ErrNoAccess
		}

		copy(r.posts[i:], r.posts[i+1:])
		r.posts[len(r.posts)-1] = nil
		r.posts = r.posts[:len(r.posts)-1]

		return nil
	}

	return ErrNotExist
}

func (r *MemoryRepo) DeleteComment(postID string, commentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := 0
	for ; i < len(r.posts); i++ {
		if r.posts[i].ID != postID {
			continue
		}
	}

	if i == len(r.posts) {
		return ErrNotExist
	}

	for j := range r.posts[i].CommentIDs {
		if r.posts[i].CommentIDs[j] != commentID {
			continue
		}

		copy(r.posts[i].CommentIDs[j:], r.posts[i].CommentIDs[j+1:])
		r.posts[i].CommentIDs[len(r.posts[i].CommentIDs)-1] = ""
		r.posts[i].CommentIDs = r.posts[i].CommentIDs[:len(r.posts[i].CommentIDs)-1]

		return nil
	}

	return ErrCommentNotExist
}
