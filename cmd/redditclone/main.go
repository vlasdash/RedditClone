package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/vlasdash/redditclone/config"
	"github.com/vlasdash/redditclone/init/db"
	"github.com/vlasdash/redditclone/internal/comment"
	"github.com/vlasdash/redditclone/internal/post"
	"github.com/vlasdash/redditclone/internal/session"
	"github.com/vlasdash/redditclone/internal/user"
	"github.com/vlasdash/redditclone/pkg/handlers"
	"github.com/vlasdash/redditclone/pkg/middleware"
	"html/template"
	"net/http"
)

const ConfigPath = "./config/"

func main() {
	contextLogger := logrus.WithFields(logrus.Fields{
		"logger": "LOGRUS",
	})

	err := config.LoadConfig(ConfigPath)
	if err != nil {
		contextLogger.Fatal("read config failed: %v\n", err)
		return
	}

	tmpl := template.Must(template.ParseFiles("static/html/index.html"))

	mongoDB, err := db.InitMongo()
	if err != nil {
		contextLogger.Fatal(err)
		return
	}

	mysqlDB, err := db.InitMySQL()
	if err != nil {
		contextLogger.Fatal(err)
		return
	}
	defer func() {
		if err = mysqlDB.Close(); err != nil {
			contextLogger.Error(err)
		}
	}()

	generator := &session.JWTGenerator{}
	hasher := &user.BcryptHasher{}
	userRepo := user.NewMySQLRepo(mysqlDB)
	sessionRepo := session.NewMySQLRepo(mysqlDB, generator)
	postRepo := post.NewMongoRepo(mongoDB)
	commentRepo := comment.NewMongoRepo(mongoDB)
	sessionManager := session.NewManager(sessionRepo, userRepo)

	authorizationHandler := handlers.NewAuthorizationHandler(userRepo, sessionRepo, contextLogger, hasher)
	postHandler := handlers.NewPostHandler(postRepo, userRepo, commentRepo, contextLogger)
	homepageHandler := handlers.NewHomepageHandler(tmpl, contextLogger)
	authenticationMiddleware := middleware.NewAuthenticationMiddleware(sessionManager, contextLogger)

	r := mux.NewRouter()
	fileServer := http.StripPrefix("/static/", http.FileServer(http.Dir("static/")))
	r.PathPrefix("/static/").Handler(fileServer).Methods("GET")
	r.HandleFunc("/api/login", authorizationHandler.Login).Methods("POST")
	r.HandleFunc("/api/register", authorizationHandler.Register).Methods("POST")
	r.HandleFunc("/api/posts/", postHandler.GetList).Methods("GET")
	r.HandleFunc("/api/post/{id}", postHandler.GetPost).Methods("GET")
	r.HandleFunc("/api/posts/{category}", postHandler.GetByCategory).Methods("GET")
	r.HandleFunc("/api/user/{username}", postHandler.GetByUsername).Methods("GET")

	s := r.PathPrefix("/api").Subrouter()
	s.HandleFunc("/posts", postHandler.Add).Methods("POST")
	s.HandleFunc("/post/{id}", postHandler.AddComment).Methods("POST")
	s.HandleFunc("/post/{id}/{comment_id}", postHandler.DeleteComment).Methods("DELETE")
	s.HandleFunc("/post/{id}", postHandler.Delete).Methods("DELETE")
	s.HandleFunc("/post/{id}/upvote", postHandler.Upvote).Methods("GET")
	s.HandleFunc("/post/{id}/downvote", postHandler.Downvote).Methods("GET")
	s.HandleFunc("/post/{id}/unvote", postHandler.Unvote).Methods("GET")
	s.Use(authenticationMiddleware.Authenticate)

	r.PathPrefix("/").Handler(homepageHandler)

	h := middleware.CheckContentType(contextLogger, r)
	h = middleware.AccessLog(contextLogger, h)

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", config.C.App.Port),
		Handler: h,
	}
	if err := server.ListenAndServe(); err != nil {
		contextLogger.Fatalf("unable to start server: %v\n", err)
	}
}
