package handlers

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/vlasdash/redditclone/internal/session"
	"github.com/vlasdash/redditclone/internal/user"
	"io/ioutil"
	"net/http"
)

type AuthorizationHandler struct {
	UserRepo    user.UserRepo
	SessionRepo session.SessionRepo
	Logger      *logrus.Entry
	Hasher      user.PasswordHasher
}

type AuthorizationRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ResponseError struct {
	Location string `json:"body"`
	Param    string `json:"param"`
	Value    string `json:"value"`
	Message  string `json:"msg"`
}

func NewAuthorizationHandler(ur user.UserRepo, sr session.SessionRepo, log *logrus.Entry, ph user.PasswordHasher) *AuthorizationHandler {
	return &AuthorizationHandler{
		UserRepo:    ur,
		SessionRepo: sr,
		Logger:      log,
		Hasher:      ph,
	}
}

func (h *AuthorizationHandler) Login(w http.ResponseWriter, r *http.Request) {
	defer func(r *http.Request, logger *logrus.Entry) {
		err := r.Body.Close()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
			}).Error("unable request`s body close at login: ", err)
		}
	}(r, h.Logger)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable read body at login: ", err)
		http.Error(w, "can't read request", http.StatusInternalServerError)
		return
	}

	req := &AuthorizationRequest{}
	err = json.Unmarshal(body, req)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable unmarshal json from client at login: ", err)
		http.Error(w, "can't unmarshal request from json", http.StatusInternalServerError)
		return
	}

	u, err := h.UserRepo.GetByUsername(req.Username)
	if err == user.ErrNoExist {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "user not found",
		})
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at login: ", err)
			http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
			return
		}
		return
	}

	if !h.Hasher.IsPassword(u.Password, req.Password) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "invalid password",
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at login: ", err)
			http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
			return
		}
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnauthorized,
		}).Info()
		return
	}

	token, err := h.SessionRepo.Add(u.Username, u.ID)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error(err.Error())
		http.Error(w, "unable generate token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
	})
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at login: ", err)
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

func (h *AuthorizationHandler) Register(w http.ResponseWriter, r *http.Request) {
	defer func(r *http.Request, logger *logrus.Entry) {
		err := r.Body.Close()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
			}).Error("unable request`s body close at register: ", err)
		}
	}(r, h.Logger)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable read body at register: ", err)
		http.Error(w, "can't unmarshal request from json", http.StatusInternalServerError)
		return
	}

	req := &AuthorizationRequest{}
	err = json.Unmarshal(body, req)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable unmarshal json from client at register: ", err)
		http.Error(w, "can't unmarshal request from json", http.StatusInternalServerError)
		return
	}

	_, err = h.UserRepo.GetByUsername(req.Username)
	if err != user.ErrNoExist {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []ResponseError{
				{
					Location: "body",
					Param:    "username",
					Value:    req.Username,
					Message:  "already exists",
				},
			},
		})
		if err != nil {
			h.Logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error("unable send json to client at register: ", err)
			http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
			return
		}
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusUnprocessableEntity,
		}).Info()
		return
	}

	passwordHash, err := h.Hasher.GetHashPassword(req.Password)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable take hash of password at register: ", err)
		http.Error(w, "unable process hash", http.StatusInternalServerError)
		return
	}

	userID, err := h.UserRepo.Create(req.Username, passwordHash)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable create user at bd: ", err)
		http.Error(w, "unable create user", http.StatusInternalServerError)
		return
	}

	token, err := h.SessionRepo.Add(req.Username, userID)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error(err.Error())
		http.Error(w, "unable generate token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
	})
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Error("unable send json to client at login: ", err)
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
