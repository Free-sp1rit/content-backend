package main

import (
	"content-backend/internal/auth"
	"content-backend/internal/config"
	"content-backend/internal/handler"
	"content-backend/internal/middleware"
	"content-backend/internal/repository"
	"content-backend/internal/service"
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

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

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	redisPingCtx, redisPingCancel := context.WithTimeout(context.Background(), time.Second)
	defer redisPingCancel()
	if err := redisClient.Ping(redisPingCtx).Err(); err != nil {
		log.Printf("redis unavailable: %v", err)
	} else {
		log.Printf("redis connected: %s", cfg.Redis.Addr)
	}

	userRepo := repository.NewUserRepository(db)
	articleRepo := repository.NewArticleRepository(db)

	tokenManager := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.TokenTTL,
	)

	authService := service.NewAuthServiceWithLoginLimiters(
		userRepo,
		tokenManager,
		service.NewRedisLoginLimiterWithOptions(
			redisClient,
			cfg.LoginRateLimit.EmailMaxFailures,
			cfg.LoginRateLimit.Window,
		),
		service.NewRedisLoginLimiterWithOptions(
			redisClient,
			cfg.LoginRateLimit.IPMaxFailures,
			cfg.LoginRateLimit.Window,
		),
	)
	articleService := service.NewArticleServiceWithCacheAndViewCounter(
		articleRepo,
		service.NewRedisCache(redisClient),
		service.NewRedisArticleViewCounter(redisClient),
	)

	authMiddleware := middleware.NewAuthMiddleware(tokenManager)

	authHandler := handler.NewAuthHandler(authService)
	articleHandler := handler.NewArticleHandler(articleService)

	publicListArticlesHandler := http.HandlerFunc(articleHandler.ListPublishedArticles)
	protectedCreateArticleHandler := authMiddleware.RequireLogin(http.HandlerFunc(articleHandler.CreateArticle))

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		err := db.PingContext(ctx)
		if err != nil {
			http.Error(w, "unhealthy", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte("ok"))
		}
	})
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
