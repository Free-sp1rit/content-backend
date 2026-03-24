package main

import (
	"content-backend/internal/auth"
	"content-backend/internal/config"
	"content-backend/internal/handler"
	"content-backend/internal/middleware"
	"content-backend/internal/repository"
	"content-backend/internal/service"
	"database/sql"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open(
		"postgres",
		cfg.Database.DSN(),
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
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.TokenTTL,
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
	http.Handle(
		"/me/articles/{id}",
		authMiddleware.RequireLogin(http.HandlerFunc(articleHandler.UpdateArticle)),
	)
	http.HandleFunc("/articles/{id}", articleHandler.GetArticle)

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           nil,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	log.Printf("server listening on :%s", cfg.Server.Port)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}
