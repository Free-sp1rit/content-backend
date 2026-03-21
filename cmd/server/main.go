package main

import (
	"content-backend/internal/auth"
	"content-backend/internal/handler"
	"content-backend/internal/middleware"
	"content-backend/internal/repository"
	"content-backend/internal/service"
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open(
		"postgres",
		"host=localhost port=5432 user=postgres dbname=content_backend sslmode=disable",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUserRepository(db)
	articleRepo := repository.NewArticleRepository(db)

	tokenManager := auth.NewTokenManager(
		"dev-secret",
		"content-backend",
		24*time.Hour,
	)

	authService := service.NewAuthService(userRepo, tokenManager)
	articleService := service.NewArticleService(articleRepo)

	authMiddleware := middleware.NewAuthMiddleware(tokenManager)

	authHandler := handler.NewAuthHandler(authService)
	articleHandler := handler.NewArticleHandler(articleService)

	publicListArticlesHandler := http.HandlerFunc(articleHandler.ListPublishedArticles)
	protectedCreateArticleHandler := authMiddleware.RequireLogin(http.HandlerFunc(articleHandler.CreateArticle))

	http.HandleFunc("/register", authHandler.Register)
	http.HandleFunc("/login", authHandler.Login)
	http.HandleFunc("/articles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			publicListArticlesHandler.ServeHTTP(w, r)
		case http.MethodPost:
			protectedCreateArticleHandler.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.Handle(
		"/articles/publish",
		authMiddleware.RequireLogin(http.HandlerFunc(articleHandler.PublishArticle)),
	)
	http.Handle(
		"/me/articles",
		authMiddleware.RequireLogin(http.HandlerFunc(articleHandler.ListMyArticles)),
	)

	log.Println("server listening on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
