package main

import (
	"content-backend/internal/auth"
	"content-backend/internal/handler"
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

	authHandler := handler.NewAuthHandler(authService)
	articleHandler := handler.NewArticleHandler(articleService)

	http.HandleFunc("/register", authHandler.Register)
	http.HandleFunc("/login", authHandler.Login)
	http.HandleFunc("/articles", articleHandler.Articles)
	http.HandleFunc("/articles/publish", articleHandler.PublishArticle)
	http.HandleFunc("/me/articles", articleHandler.ListMyArticles)

	log.Println("server listening on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
