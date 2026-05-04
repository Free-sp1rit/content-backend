package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"content-backend/internal/model"

	_ "github.com/lib/pq"
)

const integrationDatabaseDSNEnv = "CONTENT_BACKEND_TEST_DATABASE_DSN"

func TestIntegrationMigrationsFreshDatabase(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationSchema(t, db)
	runAllMigrations(t, db)

	assertArticleDeletedAtColumnExists(t, db)
	assertArticleIndexExists(t, db, "idx_articles_not_deleted_state_created_at")
	assertArticleIndexExists(t, db, "idx_articles_not_deleted_author_id_created_at")
}

func TestIntegrationMigrationExistingDatabaseUpgrade(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationSchema(t, db)
	runMigrationFiles(t, db, migrationPath(t, "001_init.sql"))
	runMigrationFiles(t, db, migrationPath(t, "002_add_article_deleted_at.sql"))
	runMigrationFiles(t, db, migrationPath(t, "002_add_article_deleted_at.sql"))

	assertArticleDeletedAtColumnExists(t, db)

	ctx := integrationContext(t)
	repo := NewArticleRepository(db)
	authorID := insertIntegrationUser(t, db, "upgrade-author@example.com")

	articleID, err := repo.Create(ctx, model.Article{
		AuthorID: authorID,
		Title:    "upgrade article",
		Content:  "created after migration upgrade",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got, err := repo.GetByID(ctx, articleID)
	if err != nil {
		t.Fatalf("GetByID returned error after migration upgrade: %v", err)
	}
	if got.ID != articleID {
		t.Fatalf("got article id %d, want %d", got.ID, articleID)
	}
}

func TestIntegrationArticleRepositorySQLBehavior(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationSchema(t, db)
	runAllMigrations(t, db)

	ctx := integrationContext(t)
	repo := NewArticleRepository(db)
	authorID := insertIntegrationUser(t, db, "article-author@example.com")
	otherAuthorID := insertIntegrationUser(t, db, "other-author@example.com")

	articleID, err := repo.Create(ctx, model.Article{
		AuthorID: authorID,
		Title:    "draft title",
		Content:  "draft content",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	updated, err := repo.UpdateStateIfAuthorAndState(ctx, articleID, otherAuthorID, model.ArticleStateDraft, model.ArticleStatePublished)
	if err != nil {
		t.Fatalf("UpdateStateIfAuthorAndState for other author returned error: %v", err)
	}
	if updated {
		t.Fatal("expected non-author publish to affect 0 rows")
	}

	updated, err = repo.UpdateContentIfAuthorAndState(ctx, articleID, otherAuthorID, model.ArticleStateDraft, "other title", "other content")
	if err != nil {
		t.Fatalf("UpdateContentIfAuthorAndState for other author returned error: %v", err)
	}
	if updated {
		t.Fatal("expected non-author edit to affect 0 rows")
	}

	updated, err = repo.UpdateContentIfAuthorAndState(ctx, articleID, authorID, model.ArticleStateDraft, "updated draft", "updated content")
	if err != nil {
		t.Fatalf("UpdateContentIfAuthorAndState returned error: %v", err)
	}
	if !updated {
		t.Fatal("expected author draft edit to update one row")
	}

	updated, err = repo.UpdateStateIfAuthorAndState(ctx, articleID, authorID, model.ArticleStateDraft, model.ArticleStatePublished)
	if err != nil {
		t.Fatalf("UpdateStateIfAuthorAndState returned error: %v", err)
	}
	if !updated {
		t.Fatal("expected author draft publish to update one row")
	}

	updated, err = repo.UpdateStateIfAuthorAndState(ctx, articleID, authorID, model.ArticleStateDraft, model.ArticleStatePublished)
	if err != nil {
		t.Fatalf("duplicate UpdateStateIfAuthorAndState returned error: %v", err)
	}
	if updated {
		t.Fatal("expected duplicate publish to affect 0 rows")
	}

	updated, err = repo.UpdateContentIfAuthorAndState(ctx, articleID, authorID, model.ArticleStateDraft, "after publish", "should not update")
	if err != nil {
		t.Fatalf("UpdateContentIfAuthorAndState after publish returned error: %v", err)
	}
	if updated {
		t.Fatal("expected published article edit to affect 0 rows")
	}

	assertArticlePresentInLists(t, repo, ctx, articleID, authorID)

	deletedState, deleted, err := repo.DeleteIfAuthorAndNotDeleted(ctx, articleID, otherAuthorID)
	if err != nil {
		t.Fatalf("DeleteIfAuthorAndNotDeleted for other author returned error: %v", err)
	}
	if deleted {
		t.Fatalf("expected non-author delete to affect 0 rows, got state %q", deletedState)
	}

	deletedState, deleted, err = repo.DeleteIfAuthorAndNotDeleted(ctx, articleID, authorID)
	if err != nil {
		t.Fatalf("DeleteIfAuthorAndNotDeleted returned error: %v", err)
	}
	if !deleted {
		t.Fatal("expected author delete to update one row")
	}
	if deletedState != model.ArticleStatePublished {
		t.Fatalf("got deleted state %q, want %q", deletedState, model.ArticleStatePublished)
	}

	deletedState, deleted, err = repo.DeleteIfAuthorAndNotDeleted(ctx, articleID, authorID)
	if err != nil {
		t.Fatalf("duplicate DeleteIfAuthorAndNotDeleted returned error: %v", err)
	}
	if deleted {
		t.Fatalf("expected duplicate delete to affect 0 rows, got state %q", deletedState)
	}

	_, err = repo.GetByID(ctx, articleID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetByID after delete returned %v, want sql.ErrNoRows", err)
	}

	assertArticleHiddenFromLists(t, repo, ctx, articleID, authorID)

	updated, err = repo.UpdateStateIfAuthorAndState(ctx, articleID, authorID, model.ArticleStateDraft, model.ArticleStatePublished)
	if err != nil {
		t.Fatalf("UpdateStateIfAuthorAndState after delete returned error: %v", err)
	}
	if updated {
		t.Fatal("expected deleted article publish to affect 0 rows")
	}

	updated, err = repo.UpdateContentIfAuthorAndState(ctx, articleID, authorID, model.ArticleStateDraft, "deleted edit", "should not update")
	if err != nil {
		t.Fatalf("UpdateContentIfAuthorAndState after delete returned error: %v", err)
	}
	if updated {
		t.Fatal("expected deleted article edit to affect 0 rows")
	}
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv(integrationDatabaseDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run PostgreSQL integration tests", integrationDatabaseDSNEnv)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close integration database: %v", err)
		}
	})

	ctx := integrationContext(t)
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	requireTestDatabase(t, db)

	return db
}

func requireTestDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := integrationContext(t)
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		t.Fatalf("query current database: %v", err)
	}
	if !strings.Contains(strings.ToLower(databaseName), "test") {
		t.Fatalf("refusing to run destructive integration test against database %q; database name must contain test", databaseName)
	}
}

func integrationContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func resetIntegrationSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := integrationContext(t)
	statements := []string{
		`DROP SCHEMA IF EXISTS public CASCADE`,
		`CREATE SCHEMA public`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("reset integration schema with %q: %v", statement, err)
		}
	}
}

func runAllMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("list migration files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no migration files found")
	}
	sort.Strings(files)
	runMigrationFiles(t, db, files...)
}

func runMigrationFiles(t *testing.T, db *sql.DB, files ...string) {
	t.Helper()

	ctx := integrationContext(t)
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("execute migration %s: %v", file, err)
		}
	}
}

func migrationPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "migrations", name)
}

func assertArticleDeletedAtColumnExists(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := integrationContext(t)
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'articles'
			  AND column_name = 'deleted_at'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("query deleted_at column: %v", err)
	}
	if !exists {
		t.Fatal("expected articles.deleted_at column to exist")
	}
}

func assertArticleIndexExists(t *testing.T, db *sql.DB, indexName string) {
	t.Helper()

	ctx := integrationContext(t)
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE schemaname = 'public'
			  AND tablename = 'articles'
			  AND indexname = $1
		)
	`, indexName).Scan(&exists)
	if err != nil {
		t.Fatalf("query article index %q: %v", indexName, err)
	}
	if !exists {
		t.Fatalf("expected article index %q to exist", indexName)
	}
}

func insertIntegrationUser(t *testing.T, db *sql.DB, email string) int64 {
	t.Helper()

	ctx := integrationContext(t)
	var id int64
	err := db.QueryRowContext(ctx, `
		INSERT INTO users(email, password_hash)
		VALUES($1, $2)
		RETURNING id
	`, email, "integration-password-hash").Scan(&id)
	if err != nil {
		t.Fatalf("insert integration user %q: %v", email, err)
	}
	return id
}

func assertArticlePresentInLists(t *testing.T, repo *ArticleRepository, ctx context.Context, articleID, authorID int64) {
	t.Helper()

	publishedArticles, err := repo.ListByState(ctx, model.ArticleStatePublished)
	if err != nil {
		t.Fatalf("ListByState returned error: %v", err)
	}
	if !containsArticleID(publishedArticles, articleID) {
		t.Fatalf("expected published article list to contain article %d", articleID)
	}

	authorArticles, err := repo.ListByAuthorID(ctx, authorID)
	if err != nil {
		t.Fatalf("ListByAuthorID returned error: %v", err)
	}
	if !containsArticleID(authorArticles, articleID) {
		t.Fatalf("expected author article list to contain article %d", articleID)
	}
}

func assertArticleHiddenFromLists(t *testing.T, repo *ArticleRepository, ctx context.Context, articleID, authorID int64) {
	t.Helper()

	publishedArticles, err := repo.ListByState(ctx, model.ArticleStatePublished)
	if err != nil {
		t.Fatalf("ListByState after delete returned error: %v", err)
	}
	if containsArticleID(publishedArticles, articleID) {
		t.Fatalf("expected published article list to hide deleted article %d", articleID)
	}

	authorArticles, err := repo.ListByAuthorID(ctx, authorID)
	if err != nil {
		t.Fatalf("ListByAuthorID after delete returned error: %v", err)
	}
	if containsArticleID(authorArticles, articleID) {
		t.Fatalf("expected author article list to hide deleted article %d", articleID)
	}
}

func containsArticleID(articles []model.Article, id int64) bool {
	for _, article := range articles {
		if article.ID == id {
			return true
		}
	}
	return false
}
