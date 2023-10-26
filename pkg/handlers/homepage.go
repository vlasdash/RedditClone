package handlers

import (
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
)

type HomepageHandler struct {
	Homepage *template.Template
	Logger   *logrus.Entry
}

func NewHomepageHandler(tmp *template.Template, log *logrus.Entry) *HomepageHandler {
	return &HomepageHandler{
		Homepage: tmp,
		Logger:   log,
	}
}

func (h *HomepageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.Homepage.Execute(w, nil)
	if err != nil {
		h.Logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"url":         r.URL.Path,
			"status_code": http.StatusInternalServerError,
		}).Errorf("unable to execute temlpate: %v\n", err)
		http.Error(w, "unable to execute start page", http.StatusInternalServerError)
	}
}
