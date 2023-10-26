package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/vlasdash/redditclone/internal/comment"
	"github.com/vlasdash/redditclone/internal/post"
	"github.com/vlasdash/redditclone/internal/session"
	"github.com/vlasdash/redditclone/internal/user"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
)

type PostResponse struct {
	ID               string             `json:"id"`
	Category         string             `json:"category"`
	CreateDate       string             `json:"created"`
	Text             string             `json:"text,omitempty"`
	URL              string             `json:"url,omitempty"`
	Title            string             `json:"title"`
	Type             string             `json:"type"`
	Score            int                `json:"score"`
	UpvotePercentage uint               `json:"upvotePercentage"`
	Views            int                `json:"views"`
	Votes            []*post.Vote       `json:"votes"`
	Comments         []*CommentResponse `json:"comments"`
	Author           *user.User         `json:"author"`
}

type CommentResponse struct {
	ID         string     `json:"id"`
	Author     *user.User `json:"author"`
	CreateDate string     `json:"created"`
	Body       string     `json:"body"`
}

type CommentRequest struct {
	Body string `json:"comment"`
}

type PostHandler struct {
	PostRepo    post.PostRepo
	UserRepo    user.UserRepo
	CommentRepo comment.CommentRepo
	Logger      *logrus.Entry
}

func NewPostHandler(pr post.PostRepo, ur user.UserRepo, cr comment.CommentRepo, log *logrus.Entry) *PostHandler {
	return &PostHandler{
		PostRepo:    pr,
		CommentRepo: cr,
		UserRepo:    ur,
		Logger:      log,
	}
}

func (h *PostHandler) createResponse(posts []*post.Post) ([]*PostResponse, error) {
	resp := make([]*PostResponse, 0, len(posts))

	for _, p := range posts {
		r := &PostResponse{
			ID:         p.ID,
			Category:   p.Category,
			CreateDate: p.CreateDate,
			Text:       p.Text,
			URL:        p.URL,
			Title:      p.Title,
			Type:       p.Type,
			Views:      p.Views,
			Votes:      p.Votes,
		}

		r.Score = p.UpvotesCount - p.DownvotesCount
		if p.UpvotesCount+p.DownvotesCount != 0 {
			r.UpvotePercentage = uint(math.Round(float64(p.UpvotesCount) / float64(p.UpvotesCount+p.DownvotesCount) * 100))
		} else {
			r.UpvotePercentage = 0
		}

		r.Comments = make([]*CommentResponse, 0, len(p.CommentIDs))
		for _, id := range p.CommentIDs {
			c, err := h.CommentRepo.GetByID(id)
			if err != nil {
				return nil, err
			}

			commentResp := &CommentResponse{
				ID:         c.ID,
				CreateDate: c.CreateDate,
				Body:       c.Body,
			}
			commentResp.Author, err = h.UserRepo.GetByID(c.AuthorID)
			if err != nil {
				return nil, err
			}
			commentResp.Author.Password = ""

			r.Comments = append(r.Comments, commentResp)
		}

		sort.SliceStable(r.Comments, func(i, j int) bool {
			return r.Comments[i].CreateDate < r.Comments[j].CreateDate
		})

		var err error
		r.Author, err = h.UserRepo.GetByID(p.AuthorID)
		if err != nil {
			return nil, err
		}
		r.Author.Password = ""

		resp = append(resp, r)
	}

	return resp, nil
}

func (h *PostHandler) GetList(w http.ResponseWriter, r *http.Request) {
	posts, err := h.PostRepo.GetAll()
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable get posts from memory repository: ", err)

		http.Error(w, "unable get posts from server", http.StatusInternalServerError)
		return
	}

	resp, err := h.createResponse(posts)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at get post all: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at get all posts: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}

func (h *PostHandler) Add(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		http.Redirect(w, r, "/api/login", http.StatusUnauthorized)
		return
	}

	defer func(r *http.Request, logger *logrus.Entry) {
		err := r.Body.Close()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
			}).Error("unable request`s body close at add post: ", err)
		}
	}(r, h.Logger)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable read body at add post: ", err)
		http.Error(w, "unable read body", http.StatusInternalServerError)
		return
	}

	req := &post.Post{}
	err = json.Unmarshal(body, req)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable unmarshal json from client at add post: ", err)
		http.Error(w, "can't unmarshal request from json", http.StatusInternalServerError)
		return
	}

	req.AuthorID = sess.UserID

	id, err := h.PostRepo.Create(req)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create post: ", err)
		http.Error(w, "unable create post", http.StatusInternalServerError)
		return
	}

	viewsUpdate := 0
	p, err := h.PostRepo.GetByID(id, viewsUpdate)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create post: ", err)
		http.Error(w, "unable create post", http.StatusInternalServerError)
		return
	}

	posts := make([]*post.Post, 0, 1)
	posts = append(posts, p)
	resp, err := h.createResponse(posts)
	if err != nil || len(resp) != 1 {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at add post: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp[0])

	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at add post: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusCreated,
	}).Info()
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]
	viewsUpdate := 1

	p, err := h.PostRepo.GetByID(postID, viewsUpdate)
	if err == post.ErrNotExist {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at get post: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}

		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusBadRequest,
		}).Info()
		return
	}
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable get post from repository: ", err)
		http.Error(w, "unable get post get post from repository", http.StatusInternalServerError)
		return
	}

	posts := make([]*post.Post, 0, 1)
	posts = append(posts, p)
	resp, err := h.createResponse(posts)
	if err != nil || len(resp) != 1 {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at get post: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp[0])

	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at get post: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}

func (h *PostHandler) GetByCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars["category"]

	posts, err := h.PostRepo.GetByCategory(category)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable get posts from memory repository: ", err)

		http.Error(w, "unable get posts from server:", http.StatusInternalServerError)
		return
	}

	resp, err := h.createResponse(posts)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at get post by categore: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at get posts by category: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]

	defer func(r *http.Request, logger *logrus.Entry) {
		err := r.Body.Close()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
			}).Error("unable request`s body close at add comment: ", err)
		}
	}(r, h.Logger)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable read body at add comment: ", err)
		http.Error(w, "unable read body", http.StatusInternalServerError)
		return
	}

	req := &CommentRequest{}
	err = json.Unmarshal(body, req)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable unmarshal json from client at add comment: ", err)
		http.Error(w, "can't unmarshal request from json", http.StatusInternalServerError)
		return
	}

	commentID, err := h.CommentRepo.Add(sess.UserID, req.Body)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable add comment to bd: ", err)
		http.Error(w, "unable add comment to bd", http.StatusInternalServerError)
		return
	}

	err = h.PostRepo.AddComment(postID, commentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at add comment: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}
		return
	}

	viewsUpdate := 0
	p, err := h.PostRepo.GetByID(postID, viewsUpdate)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable add comment: ", err)
		http.Error(w, "unable get post by id", http.StatusInternalServerError)
		return
	}

	posts := make([]*post.Post, 0, 1)
	posts = append(posts, p)
	resp, err := h.createResponse(posts)
	if err != nil || len(resp) != 1 {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at add comment: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp[0])
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at add comment: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusCreated,
	}).Info()
}

func (h *PostHandler) Upvote(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]

	err = h.PostRepo.Upvote(postID, sess.UserID)
	if err == post.ErrNotExist {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at vote: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}
		return
	}
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable upvote: ", err)
		http.Error(w, "unable upvote", http.StatusInternalServerError)
		return
	}

	viewsUpdate := 0
	p, err := h.PostRepo.GetByID(postID, viewsUpdate)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable upvote: ", err)
		http.Error(w, "unable upvote", http.StatusInternalServerError)
		return
	}

	posts := make([]*post.Post, 0, 1)
	posts = append(posts, p)
	resp, err := h.createResponse(posts)
	if err != nil || len(resp) != 1 {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at vote: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp[0])

	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
		}).Error("unable send json to client at vote: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}

func (h *PostHandler) Downvote(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]

	err = h.PostRepo.Downvote(postID, sess.UserID)
	if err == post.ErrNotExist {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at vote: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}

		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusBadRequest,
		}).Info()
		return
	}
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable downvote: ", err)
		http.Error(w, "unable downvote", http.StatusInternalServerError)
		return
	}

	viewsUpdate := 0
	p, err := h.PostRepo.GetByID(postID, viewsUpdate)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable downvote: ", err)
		http.Error(w, "unable downvote", http.StatusInternalServerError)
		return
	}

	posts := make([]*post.Post, 0, 1)
	posts = append(posts, p)
	resp, err := h.createResponse(posts)
	if err != nil || len(resp) != 1 {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at vote: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp[0])

	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at vote: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}

func (h *PostHandler) Unvote(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]

	err = h.PostRepo.Unvote(postID, sess.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at vote: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}

		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusBadRequest,
		}).Info()
		return
	}

	viewsUpdate := 0
	p, err := h.PostRepo.GetByID(postID, viewsUpdate)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable unvote: ", err)
		http.Error(w, "unable unvote", http.StatusInternalServerError)
		return
	}

	posts := make([]*post.Post, 0, 1)
	posts = append(posts, p)
	resp, err := h.createResponse(posts)
	if err != nil || len(resp) != 1 {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at unvote: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp[0])

	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at vote: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}

func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]

	err = h.PostRepo.Delete(postID, sess.UserID)
	if err == post.ErrNotExist || err == post.ErrNoAccess {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at delete post: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}

		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusBadRequest,
		}).Info()
		return
	}
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable delete post: ", err)
		http.Error(w, "unable delete post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "success",
	})

	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at add comment: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]
	commentID := vars["comment_id"]

	err = h.CommentRepo.Delete(commentID, sess.UserID)
	if err == comment.ErrNotExist || err == comment.ErrNoAccess {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at delete comment: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}
		return
	}
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at delete comment: ", err)
		http.Error(w, "unable delete comment", http.StatusInternalServerError)
		return
	}

	err = h.PostRepo.DeleteComment(postID, commentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": err.Error(),
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at delete comment: ", err)
			http.Error(w, "unable send json", http.StatusInternalServerError)
			return
		}

		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusBadRequest,
		}).Info()
		return
	}

	viewsUpdate := 0
	p, err := h.PostRepo.GetByID(postID, viewsUpdate)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable delete comment: ", err)
		http.Error(w, "unable delete comment", http.StatusInternalServerError)
		return
	}

	posts := make([]*post.Post, 0, 1)
	posts = append(posts, p)
	resp, err := h.createResponse(posts)
	if err != nil || len(resp) != 1 {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at delete comment: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp[0])

	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at add comment: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}

func (h *PostHandler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	u, err := h.UserRepo.GetByUsername(username)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable get user from db: ", err)
		http.Error(w, "unable get user from db", http.StatusInternalServerError)
		return
	}

	posts, err := h.PostRepo.GetByAuthor(u.ID)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable get posts from memory repository: ", err)

		http.Error(w, "unable get posts from server", http.StatusInternalServerError)
		return
	}

	resp, err := h.createResponse(posts)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create response to client at unvote: ", err)
		http.Error(w, "unable create response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at get posts by username: ", err)
		http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
		return
	}

	h.Logger.WithFields(logrus.Fields{
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"url":         r.URL.Path,
		"status_code": http.StatusOK,
	}).Info()
}
