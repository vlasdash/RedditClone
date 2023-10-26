package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/vlasdash/redditclone/internal/test/mock"
	"github.com/vlasdash/redditclone/internal/user"
	"github.com/vlasdash/redditclone/pkg/handlers"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type errAuthReader struct{}

func (errAuthReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

type AuthorizationError struct {
	Errors []handlers.ResponseError `json:"errors"`
}

type TestAuthCase struct {
	Request     handlers.AuthorizationRequest
	User        *user.User
	ResponseErr AuthorizationError
	Token       string
}

func TestLoginCorrect(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "password",
		},
		Token: "tokenUser",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(test.User, nil)
	sessionRepo.EXPECT().Add(test.User.Username, test.User.ID).Return(test.Token, nil)
	hasher.EXPECT().IsPassword(test.User.Password, test.Request.Password).Return(true)

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/login", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected resp status %d, got %d", http.StatusOK, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(test.Token)) {
		t.Errorf("expected token %s, got %s", test.Token, body)
	}
}

func TestLoginAuthorizationError(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "secret",
		},
		Token: "",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	// тестирование неправильного логина пользователя
	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(nil, user.ErrNoExist)

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/login", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}

	expectedMessage := "user not found"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}

	// тестирование неправильного пароля пользователя
	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(test.User, nil)
	hasher.EXPECT().IsPassword(test.User.Password, test.Request.Password).Return(false)

	b = bytes.NewBufferString("")
	err = json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req = httptest.NewRequest("POST", "/api/login", b)
	req.Header.Add("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.Login(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected resp status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		return
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	expectedMessage = "invalid password"
	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestLoginSessionError(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "password",
		},
		Token: "",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(test.User, nil)
	hasher.EXPECT().IsPassword(test.User.Password, test.Request.Password).Return(true)
	sessionRepo.EXPECT().Add(test.User.Username, test.User.ID).Return("", fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/login", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	expectedMessage := "unable generate token"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestLoginReadBodyError(t *testing.T) {
	expectedMessage := "can't read request"
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	req := httptest.NewRequest("POST", "/api/login", errAuthReader{})
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestLoginUnmarshalError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	b := bytes.NewBufferString("bad body")
	req := httptest.NewRequest("POST", "/api/login", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	expectedMessage := "can't unmarshal request from json"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestRegisterCorrect(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "password",
		},
		Token: "tokenUser",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(nil, user.ErrNoExist)
	userRepo.EXPECT().Create(test.User.Username, test.User.Password).Return(test.User.ID, nil)
	hasher.EXPECT().GetHashPassword(test.Request.Password).Return(test.User.Password, nil)
	sessionRepo.EXPECT().Add(test.User.Username, test.User.ID).Return(test.Token, nil)

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/register", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected resp status %d, got %d", http.StatusCreated, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(test.Token)) {
		t.Errorf("expected token %s, got %s", test.Token, body)
	}
}

func TestRegisterAlreadyExistError(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "password",
		},
		Token: "",
		ResponseErr: AuthorizationError{
			[]handlers.ResponseError{
				{
					Location: "body",
					Param:    "username",
					Value:    "username",
					Message:  "already exists",
				},
			},
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(test.User, nil)

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/register", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected resp status %d, got %d", http.StatusUnprocessableEntity, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}
	errResp := AuthorizationError{
		Errors: make([]handlers.ResponseError, 0),
	}
	err = json.Unmarshal(body, &errResp)
	if err != nil {
		t.Fatalf("unable unmarshal json: %v", err)
	}

	if !reflect.DeepEqual(errResp, test.ResponseErr) {
		t.Errorf("wrong result, expected %#v, got %#v", test.ResponseErr, errResp)
	}
}

func TestRegisterReadBodyError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	req := httptest.NewRequest("POST", "/api/register", errAuthReader{})
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	expectedMessage := "can't unmarshal request from json"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestRegisterUnmarshalError(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	b := bytes.NewBufferString("bad body")
	req := httptest.NewRequest("POST", "/api/register", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	expectedMessage := "can't unmarshal request from json"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestRegisterHashError(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "password",
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(nil, user.ErrNoExist)
	hasher.EXPECT().GetHashPassword(test.Request.Password).Return("", fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/register", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	expectedMessage := "unable process hash"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestRegisterCreateError(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "password",
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(nil, user.ErrNoExist)
	hasher.EXPECT().GetHashPassword(test.Request.Password).Return(test.User.Password, nil)
	userRepo.EXPECT().Create(test.User.Username, test.User.Password).Return(test.User.ID, fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/register", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	expectedMessage := "unable create user"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}

func TestRegisterAddSessionError(t *testing.T) {
	test := TestAuthCase{
		Request: handlers.AuthorizationRequest{
			Username: "username",
			Password: "password",
		},
		User: &user.User{
			ID:       1,
			Username: "username",
			Password: "password",
		},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})
	contextLogger.Logger.Out = ioutil.Discard

	userRepo := mock.NewMockUserRepo(controller)
	sessionRepo := mock.NewMockSessionRepo(controller)
	hasher := mock.NewMockPasswordHasher(controller)
	handler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)

	userRepo.EXPECT().GetByUsername(test.Request.Username).Return(nil, user.ErrNoExist)
	hasher.EXPECT().GetHashPassword(test.Request.Password).Return(test.User.Password, nil)
	userRepo.EXPECT().Create(test.User.Username, test.User.Password).Return(test.User.ID, nil)
	sessionRepo.EXPECT().Add(test.User.Username, test.User.ID).Return("", fmt.Errorf("something went wrong"))

	b := bytes.NewBufferString("")
	err := json.NewEncoder(b).Encode(test.Request)
	if err != nil {
		t.Fatalf("unable encode json: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/register", b)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected resp status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		return
	}

	expectedMessage := "unable generate token"
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable read body response: %v", err)
	}

	if !bytes.Contains(body, []byte(expectedMessage)) {
		t.Errorf("expected error message %s, got %s", expectedMessage, body)
	}
}
