package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func AccessLog(logger *logrus.Entry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"time":        time.Since(start),
		}).Info()
	})
}
