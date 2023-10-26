package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/vlasdash/redditclone/internal/comment"
	"github.com/vlasdash/redditclone/internal/post"
	"github.com/vlasdash/redditclone/internal/session"
	"github.com/vlasdash/redditclone/internal/test/mock"
	"github.com/vlasdash/redditclone/internal/user"
	"github.com/vlasdash/redditclone/pkg/handlers"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type errPostReader struct{}

func (errPostReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

type PostResponseErr struct {
	Message string `json:"message"`
}

type TestPostCase struct {
	CommentRequest handlers.CommentRequest
	Comment        []*comment.Comment
	Post           []*post.Post
	User           []*user.User
	PostsResponse  []*handlers.PostResponse
}

func TestGetListCorrect(t *testing.T) {
	commentFirstID := primitive.NewObjectID()
	commentSecondID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentFirstID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
			{
				ID:         commentSecondID.Hex(),
				AuthorID:   1,
				CreateDate: "10.11.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            0,
			UpvotePercentage: 0,
			Views:            0,
			Votes:            test.Post[0].Votes,
			Comments: []*handlers.CommentResponse{
				{
					ID:         test.Comment[0].ID,
					Author:     test.User[0],
					CreateDate: test.Comment[0].CreateDate,
					Body:       test.Comment[0].Body,
				},
				{
					ID:         test.Comment[1].ID,
					Author:     test.User[0],
					CreateDate: test.Comment[1].CreateDate,
					Body:       test.Comment[1].Body,
				},
			},
			Author: test.User[0],
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentFirstID.Hex())
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentSecondID.Hex())

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetAll().Return(test.Post, nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[0]).Return(test.Comment[0], nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[1]).Return(test.Comment[1], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(test.User[0], nil).Times(3)

	req := httptest.NewRequest("GET", "/api/posts/", nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.GetList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := make([]*handlers.PostResponse, 0)
	err = json.Unmarshal(body, &postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestGetListPostRepoError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)
	expectedErrMessage := "unable get posts from server"

	postRepo.EXPECT().GetAll().Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", "/api/posts/", nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.GetList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestGetListCreateResponseError(t *testing.T) {
	postID := primitive.NewObjectID()
	commentID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentID.Hex())
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetAll().Return(test.Post, nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[0]).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", "/api/posts/", nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.GetList(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCorrect(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            1,
			UpvotePercentage: 100,
			Views:            1,
			Votes:            test.Post[0].Votes,
			Comments: []*handlers.CommentResponse{
				{
					ID:         test.Comment[0].ID,
					Author:     test.User[0],
					CreateDate: test.Comment[0].CreateDate,
					Body:       test.Comment[0].Body,
				},
			},
			Author: test.User[0],
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentID.Hex())

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Create(test.Post[0]).Return(test.Post[0].ID, nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[0]).Return(test.Comment[0], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(test.User[0], nil).Times(2)

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Post[0])
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/posts/", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)

	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected resp status %d, got %d", http.StatusCreated, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &handlers.PostResponse{}
	err = json.Unmarshal(body, postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestAddSessionError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	b := bytes.NewBufferString("bad body")
	req := httptest.NewRequest("POST", "/api/posts/", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), "key", "value")

	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}
}

func TestAddUnmarshalError(t *testing.T) {
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "can't unmarshal request from json"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	b := bytes.NewBufferString("bad body")
	req := httptest.NewRequest("POST", "/api/posts/", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)

	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddReadBodyError(t *testing.T) {
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable read body"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	req := httptest.NewRequest("POST", "/api/posts/", errPostReader{})
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCreateError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable create post"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Create(test.Post[0]).Return("", fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Post[0])
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/posts/", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)

	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddPostRepoError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable create post"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Create(test.Post[0]).Return(test.Post[0].ID, nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(nil, fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Post[0])
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/posts/", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)

	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCreateResponseError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentID.Hex())
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Create(test.Post[0]).Return(test.Post[0].ID, nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[0]).Return(test.Comment[0], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Post[0])
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/posts/", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)

	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestGetPostCorrect(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            0,
			UpvotePercentage: 0,
			Views:            0,
			Votes:            test.Post[0].Votes,
			Comments: []*handlers.CommentResponse{
				{
					ID:         test.Comment[0].ID,
					Author:     test.User[0],
					CreateDate: test.Comment[0].CreateDate,
					Body:       test.Comment[0].Body,
				},
			},
			Author: test.User[0],
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentID.Hex())

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetByID(test.Post[0].ID, 1).Return(test.Post[0], nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[0]).Return(test.Comment[0], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(test.User[0], nil).Times(2)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/post/%s", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetPost(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &handlers.PostResponse{}
	err = json.Unmarshal(body, &postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse[0], postsResponse)
	}
}

func TestGetPostNotFound(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: "post with specified id not exist",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetByID(test.Post[0].ID, 1).Return(nil, post.ErrNotExist)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/post/%s", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetPost(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestGetPostPostRepoError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
	}
	expectedErrMessage := "unable get post get post from repository"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetByID(test.Post[0].ID, 1).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/post/%s", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetPost(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestGetPostCreateResponseError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            0,
			UpvotePercentage: 0,
			Views:            0,
			Votes:            test.Post[0].Votes,
			Comments:         make([]*handlers.CommentResponse, 0),
			Author:           test.User[0],
		},
	}
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetByID(test.Post[0].ID, 1).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/post/%s", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetPost(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestGetPostByCategoryCorrect(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            0,
			UpvotePercentage: 0,
			Views:            0,
			Votes:            test.Post[0].Votes,
			Comments:         make([]*handlers.CommentResponse, 0),
			Author:           test.User[0],
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetByCategory(test.Post[0].Category).Return(test.Post, nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(test.User[0], nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/post/%s", test.Post[0].Category), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	vars := map[string]string{
		"category": test.Post[0].Category,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetByCategory(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := make([]*handlers.PostResponse, 0)
	err = json.Unmarshal(body, &postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestGetPostByCategoryCreateResponseError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            0,
			UpvotePercentage: 0,
			Views:            0,
			Votes:            test.Post[0].Votes,
			Comments:         make([]*handlers.CommentResponse, 0),
			Author:           test.User[0],
		},
	}
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().GetByCategory(test.Post[0].Category).Return(test.Post, nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(test.User[0], fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/post/%s", test.Post[0].Category), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	vars := map[string]string{
		"category": test.Post[0].Category,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetByCategory(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCommentCorrect(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            1,
			UpvotePercentage: 100,
			Views:            1,
			Votes:            test.Post[0].Votes,
			Comments: []*handlers.CommentResponse{
				{
					ID:         test.Comment[0].ID,
					Author:     test.User[0],
					CreateDate: test.Comment[0].CreateDate,
					Body:       test.Comment[0].Body,
				},
			},
			Author: test.User[0],
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentID.Hex())

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Add(test.User[0].ID, test.Comment[0].Body).Return(test.Comment[0].ID, nil)
	postRepo.EXPECT().AddComment(test.Post[0].ID, test.Comment[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[0]).Return(test.Comment[0], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(test.User[0], nil).Times(2)

	b := bytes.NewBufferString("")
	commentReq := &handlers.CommentRequest{
		Body: test.Comment[0].Body,
	}
	err := json.NewEncoder(b).Encode(commentReq)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", test.Post[0].ID), b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.AddComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected resp status %d, got %d", http.StatusCreated, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &handlers.PostResponse{}
	err = json.Unmarshal(body, postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestAddCommentSessionError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	b := bytes.NewBufferString("body")
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", primitive.NewObjectID()), b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), "key", "value")

	handler.AddComment(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}
}

func TestAddCommentUnmarshalError(t *testing.T) {
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "can't unmarshal request from json"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	b := bytes.NewBufferString("body")
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", primitive.NewObjectID()), b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)

	handler.AddComment(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCommentReadBodyError(t *testing.T) {
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable read body"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", primitive.NewObjectID()), errPostReader{})
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	handler.Add(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCommentRepoError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
	}
	expectedErrMessage := "unable add comment to bd"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Add(test.User[0].ID, test.Comment[0].Body).Return("", fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	commentReq := &handlers.CommentRequest{
		Body: test.Comment[0].Body,
	}
	err := json.NewEncoder(b).Encode(commentReq)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", test.Post[0].ID), b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.AddComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCommentPostRepoError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: "post with specified id not exist",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Add(test.User[0].ID, test.Comment[0].Body).Return(test.Comment[0].ID, nil)
	postRepo.EXPECT().AddComment(postID.Hex(), test.Comment[0].ID).Return(post.ErrNotExist)

	b := bytes.NewBufferString("")
	commentReq := &handlers.CommentRequest{
		Body: test.Comment[0].Body,
	}
	err := json.NewEncoder(b).Encode(commentReq)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", postID.Hex()), b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": postID.Hex(),
	}
	req = mux.SetURLVars(req, vars)

	handler.AddComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestAddCommentGetPostError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable get post by id"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Add(test.User[0].ID, test.Comment[0].Body).Return(test.Comment[0].ID, nil)
	postRepo.EXPECT().AddComment(test.Post[0].ID, test.Comment[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(nil, fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	commentReq := &handlers.CommentRequest{
		Body: test.Comment[0].Body,
	}
	err := json.NewEncoder(b).Encode(commentReq)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", test.Post[0].ID), b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.AddComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestAddCommentCreateRequestError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Add(test.User[0].ID, test.Comment[0].Body).Return(test.Comment[0].ID, nil)
	postRepo.EXPECT().AddComment(test.Post[0].ID, test.Comment[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	commentReq := &handlers.CommentRequest{
		Body: test.Comment[0].Body,
	}
	err := json.NewEncoder(b).Encode(commentReq)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/post/%s", test.Post[0].ID), b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.AddComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestDownvoteCorrect(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Unlike,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 1,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            -1,
			UpvotePercentage: 0,
			Views:            1,
			Votes:            test.Post[0].Votes,
			Comments:         make([]*handlers.CommentResponse, 0),
			Author:           test.User[0],
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Downvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(test.User[0], nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/downvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Downvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &handlers.PostResponse{}
	err = json.Unmarshal(body, postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestDownvoteSessionError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/downvote", primitive.NewObjectID()), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), "key", "value")

	handler.Downvote(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}
}

func TestDownvotePostNotFoundError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Unlike,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 1,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: "post with specified id not exist",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Downvote(test.Post[0].ID, test.User[0].ID).Return(post.ErrNotExist)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/downvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Downvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestDownvoteError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Unlike,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 1,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable downvote"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Downvote(test.Post[0].ID, test.User[0].ID).Return(fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/downvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Downvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestDownvotePostRepoError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Unlike,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 1,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable downvote"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Downvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/downvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Downvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestDownvoteCreateResponseError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Unlike,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 1,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Downvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/downvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Downvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestUpvoteCorrect(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            1,
			UpvotePercentage: 100,
			Views:            1,
			Votes:            test.Post[0].Votes,
			Comments:         make([]*handlers.CommentResponse, 0),
			Author:           test.User[0],
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Upvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(test.User[0], nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/upvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Upvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &handlers.PostResponse{}
	err = json.Unmarshal(body, postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestUpvoteSessionError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/upvote", primitive.NewObjectID()), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), "key", "value")

	handler.Upvote(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}
}

func TestUpvotePostNotFoundError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: "post with specified id not exist",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Upvote(test.Post[0].ID, test.User[0].ID).Return(post.ErrNotExist)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/upvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Upvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestUpvoteError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable upvote"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Upvote(test.Post[0].ID, test.User[0].ID).Return(fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/upvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Upvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestUpvotePostRepoError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable upvote"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Upvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/upvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Upvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestUpvoteCreateResponseError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Upvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/upvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Upvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestUnvoteCorrect(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          1,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            0,
			UpvotePercentage: 0,
			Views:            1,
			Votes:            test.Post[0].Votes,
			Comments:         make([]*handlers.CommentResponse, 0),
			Author:           test.User[0],
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Unvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(test.User[0], nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/unvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Unvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &handlers.PostResponse{}
	err = json.Unmarshal(body, postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestUnvoteSessionError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/unvote", primitive.NewObjectID()), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), "key", "value")

	handler.Unvote(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}
}

func TestUnvoteError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          1,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: "post with specified id not exist",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Unvote(test.Post[0].ID, test.User[0].ID).Return(post.ErrNotExist)

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/unvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Unvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestUnvotePostRepoError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          1,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable unvote"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Unvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/unvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Unvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestUnvoteCreateResponseError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          1,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Unvote(test.Post[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/post/%s/unvote", test.Post[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": test.Post[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Unvote(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestDeleteCorrect(t *testing.T) {
	postID := primitive.NewObjectID().Hex()
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := &PostResponseErr{
		Message: "success",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Delete(postID, test.User[0].ID).Return(nil)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/post/%s", postID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": postID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Delete(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &PostResponseErr{}
	err = json.Unmarshal(body, postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, postsResponse)
	}
}

func TestDeleteSessionError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/post/%s", primitive.NewObjectID()), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), "key", "value")

	handler.Delete(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}
}

func TestDeleteError(t *testing.T) {
	postID := primitive.NewObjectID().Hex()
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: "post with specified id not exist",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Delete(postID, test.User[0].ID).Return(post.ErrNotExist)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/post/%s", postID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": postID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Delete(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestDeletePostRepoError(t *testing.T) {
	postID := primitive.NewObjectID().Hex()
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable delete post"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	postRepo.EXPECT().Delete(postID, test.User[0].ID).Return(fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/post/%s", postID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id": postID,
	}
	req = mux.SetURLVars(req, vars)

	handler.Delete(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestDeleteCommentCorrect(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            1,
			UpvotePercentage: 100,
			Views:            1,
			Votes:            test.Post[0].Votes,
			Comments: []*handlers.CommentResponse{
				{
					ID:         test.Comment[0].ID,
					Author:     test.User[0],
					CreateDate: test.Comment[0].CreateDate,
					Body:       test.Comment[0].Body,
				},
			},
			Author: test.User[0],
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentID.Hex())

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Delete(test.Comment[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().DeleteComment(test.Post[0].ID, test.Comment[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	commentRepo.EXPECT().GetByID(test.Post[0].CommentIDs[0]).Return(test.Comment[0], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(test.User[0], nil).Times(2)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/post/%s/%s", test.Post[0].ID, test.Post[0].CommentIDs[0]), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id":         test.Post[0].ID,
		"comment_id": test.Post[0].CommentIDs[0],
	}
	req = mux.SetURLVars(req, vars)

	handler.DeleteComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := &handlers.PostResponse{}
	err = json.Unmarshal(body, postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestDeleteCommentSessionError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/post/%s/%s", primitive.NewObjectID(), primitive.NewObjectID()), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), "key", "value")

	handler.Delete(w, req.WithContext(ctx))

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}
}

func TestDeleteCommentNotFound(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: comment.ErrNotExist.Error(),
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Delete(test.Comment[0].ID, test.User[0].ID).Return(comment.ErrNotExist)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/post/%s/%s", test.Post[0].ID, test.Comment[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id":         test.Post[0].ID,
		"comment_id": test.Comment[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.DeleteComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestDeleteCommentError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable delete comment"
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Delete(test.Comment[0].ID, test.User[0].ID).Return(fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/post/%s/%s", test.Post[0].ID, test.Comment[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id":         test.Post[0].ID,
		"comment_id": test.Comment[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.DeleteComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestDeleteCommentPostNotFound(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	errResponseExpected := PostResponseErr{
		Message: post.ErrNotExist.Error(),
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Delete(test.Comment[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().DeleteComment(test.Post[0].ID, test.Comment[0].ID).Return(post.ErrNotExist)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/post/%s/%s", test.Post[0].ID, test.Comment[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id":         test.Post[0].ID,
		"comment_id": test.Comment[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.DeleteComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected resp status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	errResponse := PostResponseErr{}
	err = json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResponse, errResponseExpected) {
		t.Errorf("wrong result, expected %#v, got %#v", errResponseExpected, errResponse)
	}
}

func TestDeleteCommentGetError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable delete comment"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Delete(test.Comment[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().DeleteComment(test.Post[0].ID, test.Comment[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/post/%s/%s", test.Post[0].ID, test.Comment[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id":         test.Post[0].ID,
		"comment_id": test.Comment[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.DeleteComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestDeleteCommentCreateResponseError(t *testing.T) {
	commentID := primitive.NewObjectID()
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Comment: []*comment.Comment{
			{
				ID:         commentID.Hex(),
				AuthorID:   1,
				CreateDate: "10.10.2022",
				Body:       "body",
			},
		},
		Post: []*post.Post{
			{
				ID:         postID.Hex(),
				Category:   "music",
				CreateDate: "10.09.2022",
				Text:       "text",
				Title:      "title",
				Type:       "link",
				Views:      1,
				Votes: []*post.Vote{
					{
						UserID: 1,
						Value:  post.Like,
					},
				},
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   1,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.Post[0].CommentIDs = append(test.Post[0].CommentIDs, commentID.Hex())
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	commentRepo.EXPECT().Delete(test.Comment[0].ID, test.User[0].ID).Return(nil)
	postRepo.EXPECT().DeleteComment(test.Post[0].ID, test.Comment[0].ID).Return(nil)
	postRepo.EXPECT().GetByID(test.Post[0].ID, 0).Return(test.Post[0], nil)
	commentRepo.EXPECT().GetByID(test.Comment[0].ID).Return(test.Comment[0], nil)
	userRepo.EXPECT().GetByID(test.Comment[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/post/%s/%s", test.Post[0].ID, test.Comment[0].ID), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	sess := &session.Session{
		UserID:   test.User[0].ID,
		Username: test.User[0].Username,
	}
	ctx := session.CreateContextWithSession(req.Context(), sess)
	req = req.WithContext(ctx)
	vars := map[string]string{
		"id":         test.Post[0].ID,
		"comment_id": test.Comment[0].ID,
	}
	req = mux.SetURLVars(req, vars)

	handler.DeleteComment(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestGetPostByUsernameCorrect(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	test.PostsResponse = []*handlers.PostResponse{
		{
			ID:               test.Post[0].ID,
			Category:         test.Post[0].Category,
			CreateDate:       test.Post[0].CreateDate,
			Text:             test.Post[0].Text,
			URL:              test.Post[0].URL,
			Title:            test.Post[0].Title,
			Type:             test.Post[0].Type,
			Score:            0,
			UpvotePercentage: 0,
			Views:            0,
			Votes:            test.Post[0].Votes,
			Comments:         make([]*handlers.CommentResponse, 0),
			Author:           test.User[0],
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	userRepo.EXPECT().GetByUsername(test.User[0].Username).Return(test.User[0], nil)
	postRepo.EXPECT().GetByAuthor(test.User[0].ID).Return(test.Post, nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(test.User[0], nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/user/%s", test.User[0].Username), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	vars := map[string]string{
		"username": test.User[0].Username,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetByUsername(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	postsResponse := make([]*handlers.PostResponse, 0)
	err = json.Unmarshal(body, &postsResponse)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(postsResponse, test.PostsResponse) {
		t.Errorf("wrong result, expected %#v, got %#v", test.PostsResponse, postsResponse)
	}
}

func TestGetPostByUsernameUserRepoError(t *testing.T) {
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable get user from db"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	userRepo.EXPECT().GetByUsername(test.User[0].Username).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/user/%s", test.User[0].Username), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	vars := map[string]string{
		"username": test.User[0].Username,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetByUsername(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestGetPostByUsernamePostRepoError(t *testing.T) {
	test := TestPostCase{
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable get posts from server"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	userRepo.EXPECT().GetByUsername(test.User[0].Username).Return(test.User[0], nil)
	postRepo.EXPECT().GetByAuthor(test.User[0].ID).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/user/%s", test.User[0].Username), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	vars := map[string]string{
		"username": test.User[0].Username,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetByUsername(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}

func TestGetPostByUsernameCreateResponseError(t *testing.T) {
	postID := primitive.NewObjectID()
	test := TestPostCase{
		Post: []*post.Post{
			{
				ID:             postID.Hex(),
				Category:       "music",
				CreateDate:     "10.09.2022",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		},
		User: []*user.User{
			{
				ID:       1,
				Username: "username",
			},
		},
	}
	expectedErrMessage := "unable create response"

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	commentRepo := mock.NewMockCommentRepo(controller)
	postRepo := mock.NewMockPostRepo(controller)
	handler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)

	userRepo.EXPECT().GetByUsername(test.User[0].Username).Return(test.User[0], nil)
	postRepo.EXPECT().GetByAuthor(test.User[0].ID).Return(test.Post, nil)
	userRepo.EXPECT().GetByID(test.Post[0].AuthorID).Return(nil, fmt.Errorf("something went wrong"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/user/%s", test.User[0].Username), nil)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	vars := map[string]string{
		"username": test.User[0].Username,
	}
	req = mux.SetURLVars(req, vars)

	handler.GetByUsername(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedErrMessage)) {
		t.Errorf("expected error message %s, got %s", expectedErrMessage, body)
	}
}
