package middleware

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
)

func CheckContentType(logger *logrus.Entry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")

			err := json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "content type must be json",
			})
			if err != nil {
				logger.WithFields(logrus.Fields{
					"method":      r.Method,
					"remote_addr": r.RemoteAddr,
					"url":         r.URL.Path,
					"status_code": http.StatusInternalServerError,
				}).Error("unable send json to client: ", err)
				http.Error(w, "can't encode answer to json", http.StatusInternalServerError)
				return
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}
