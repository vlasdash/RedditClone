package middleware

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/vlasdash/redditclone/internal/session"
	"net/http"
)

type Authentication struct {
	manager *session.Manager
	logger  *logrus.Entry
}

func NewAuthenticationMiddleware(sm *session.Manager, l *logrus.Entry) *Authentication {
	return &Authentication{
		manager: sm,
		logger:  l,
	}
}

func (a *Authentication) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.Header.Get("Authorization")

		sess, err := a.manager.Create(accessToken)
		if err != nil {
			statusCode := 0
			if err == session.ErrEmptyPayload || err == session.ErrEmptyUserInfo || err == session.ErrBadToken {
				statusCode = http.StatusUnauthorized
			} else if err == session.ErrBadSigningMethod {
				statusCode = http.StatusBadRequest
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				a.logger.WithFields(logrus.Fields{
					"method":      r.Method,
					"remote_addr": r.RemoteAddr,
					"url":         r.URL.Path,
					"status_code": http.StatusInternalServerError,
				}).Error()
				return
			}

			w.WriteHeader(statusCode)
			err = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": err.Error(),
			})
			if err != nil {
				a.logger.WithFields(logrus.Fields{
					"method":      r.Method,
					"remote_addr": r.RemoteAddr,
					"url":         r.URL.Path,
					"status_code": http.StatusInternalServerError,
				}).Error("unable send json to client: ", err)
				http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
				return
			}

			a.logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": statusCode,
			}).Info()

			return
		}

		isExist, err := a.manager.HasUserExist(sess)
		if err != nil {
			a.logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusInternalServerError,
			}).Error(err.Error())
			http.Error(w, "can't check session", http.StatusInternalServerError)
			return
		}
		if !isExist {
			w.WriteHeader(http.StatusUnauthorized)
			err = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "you did not register",
			})
			if err != nil {
				a.logger.WithFields(logrus.Fields{
					"method":      r.Method,
					"remote_addr": r.RemoteAddr,
					"url":         r.URL.Path,
					"status_code": http.StatusInternalServerError,
				}).Error("unable send json to client: ", err)
				http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
				return
			}

			a.logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
				"url":         r.URL.Path,
				"status_code": http.StatusUnauthorized,
			}).Info()
		}

		ctx := session.CreateContextWithSession(r.Context(), sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
