package main

import (
	"content-backend/internal/handler"
	"content-backend/internal/repository"
	"content-backend/internal/service"
	"database/sql"
	"log"
	"net/http"

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

	authService := service.NewAuthService(userRepo)
	articleService := service.NewArticleService(articleRepo)

	authHandler := handler.NewAuthHandler(authService)
	_ = articleService

	http.HandleFunc("/register", authHandler.Register)
	http.HandleFunc("/login", authHandler.Login)

	log.Println("server listening on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
