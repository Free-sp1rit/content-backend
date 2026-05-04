# Issue: 验证 PostgreSQL migration 与 repository 真实 SQL 行为

## Status

已完成。当前任务已补充 opt-in repository 集成测试，确认 `001_init.sql` 和 `002_add_article_deleted_at.sql` 能支持新库初始化、已有库升级，并用真实 PostgreSQL 覆盖文章条件更新和逻辑删除 SQL 行为。

## Background

项目已经从单个初始化 SQL 进入数据库结构演进阶段。文章逻辑删除引入了 `migrations/002_add_article_deleted_at.sql`，repository 查询、发布、编辑和删除都开始依赖 `deleted_at IS NULL`。

当前 `go test ./...` 主要使用 fake repository，能验证 service/handler 业务语义，但不能验证真实 SQL 是否和 PostgreSQL schema 匹配。`scripts/smoke.sh` 覆盖了真实 HTTP 主链路，但它不是专门的 repository SQL 边界测试。

本阶段目标是先把 migration 和 repository 真实 SQL 验证补起来，不急于引入完整 migration 平台或 CI 数据库服务。

## Goals

- 验证 `migrations/` 按文件名顺序执行后，新库 schema 可用。
- 验证已有库先执行 `001_init.sql` 后，再执行 `002_add_article_deleted_at.sql` 的升级路径可用。
- 用真实 PostgreSQL 覆盖文章 repository 的关键 SQL 行为。
- 保持默认 `go test ./...` 不依赖本地 PostgreSQL。
- 明确真实数据库验证的运行命令、环境变量和失败排查入口。

## Decisions

- 第一版不引入 `goose`、`golang-migrate`、Atlas 等 migration 工具。
- 第一版不把 PostgreSQL 集成测试接入 GitHub Actions。
- 第一版使用 Go 标准 `testing` + `database/sql` + 当前已有 `github.com/lib/pq`。
- repository 集成测试使用显式环境变量启用；环境变量未设置时跳过，不影响默认测试口径。
- 集成测试必须使用一次性测试数据库或测试 schema，不允许污染开发/生产数据库。

推荐环境变量：

```text
CONTENT_BACKEND_TEST_DATABASE_DSN
```

推荐运行方式：

```bash
CONTENT_BACKEND_TEST_DATABASE_DSN='postgres://user:password@127.0.0.1:5432/content_backend_test?sslmode=disable' go test ./internal/repository -run Integration -count=1
```

## Scope

- 新增 repository 集成测试，例如 `internal/repository/article_repository_integration_test.go`。
- 新增测试 helper，按文件名顺序执行 `migrations/*.sql`。
- 集成测试在 `CONTENT_BACKEND_TEST_DATABASE_DSN` 未设置时 `t.Skip`。
- 集成测试需要清理自己创建的数据或明确要求使用一次性数据库。
- 必要时补充 README / docs 说明真实 PostgreSQL 验证命令。
- 必要时补充 `docs/deployment.md` 的 migration 验证说明。
- 更新 `docs/issues/README.md` 和 `ROADMAP.md`。

## Proposed Test Coverage

### Migration order

- 在空测试数据库中按文件名顺序执行 `migrations/001_init.sql` 和 `migrations/002_add_article_deleted_at.sql`。
- 验证 `articles.deleted_at` 字段存在。
- 验证未删除文章相关索引存在，至少确认 migration 执行不报错。

### Existing database upgrade path

- 在空测试数据库中先执行 `001_init.sql`。
- 再执行 `002_add_article_deleted_at.sql`。
- 验证升级后 repository 查询和条件更新可用。
- 重复执行 `002_add_article_deleted_at.sql` 不应失败，因为当前 migration 使用 `IF NOT EXISTS`。

### Repository SQL behavior

建议覆盖：

- `Create` 能创建 draft 文章。
- `UpdateStateIfAuthorAndState` 能把作者自己的 draft 发布为 published。
- 非作者发布影响 0 行。
- 已 published 文章再次发布影响 0 行。
- `UpdateContentIfAuthorAndState` 只能编辑作者自己的 draft。
- `DeleteIfAuthorAndNotDeleted` 能逻辑删除作者自己的文章，并返回删除前 state。
- 重复删除同一文章返回未删除成功。
- 删除后 `GetByID` 返回 `sql.ErrNoRows`。
- 删除后 `ListByState(published)` 不返回该文章。
- 删除后 `ListByAuthorID(authorID)` 不返回该文章。
- 删除后发布或编辑条件更新影响 0 行。

## Tasks

- [x] 新增 opt-in repository 集成测试文件。
- [x] 增加按文件名顺序执行 migration 的测试 helper。
- [x] 约定并实现 `CONTENT_BACKEND_TEST_DATABASE_DSN` 启用方式。
- [x] 覆盖新库初始化路径。
- [x] 覆盖已有库 `001 -> 002` 升级路径。
- [x] 覆盖文章 repository 关键 SQL 行为。
- [x] 确保未设置测试数据库 DSN 时，`go test ./...` 正常跳过集成验证。
- [x] 补充 README 或 docs 中的真实 PostgreSQL 验证命令。
- [x] 运行 `gofmt -l .`、`git diff --check` 和 `go test ./...`。
- [x] 在本地真实 PostgreSQL 或 Compose PostgreSQL 上运行一次集成验证，并记录命令和结果。

## Verification

- `gofmt -l .`
- `git diff --check`
- `go test ./...`
- `docker compose --env-file .env.compose ps`
- `docker compose --env-file .env.compose exec -T db dropdb --if-exists -U content_dev content_backend_test`
- `docker compose --env-file .env.compose exec -T db createdb -U content_dev content_backend_test`
- `docker run --rm --network content-backend_default -v /home/yimg/code/content-backend:/app -w /app -e CONTENT_BACKEND_TEST_DATABASE_DSN=postgres://content_dev:devpass@db:5432/content_backend_test?sslmode=disable golang:1.22.5 go test ./internal/repository -run Integration -count=1`

## Non Goals

- 不引入完整 migration 工具。
- 不把 PostgreSQL 服务接入 GitHub Actions。
- 不修改业务 API。
- 不新增文章状态。
- 不继续扩展删除、恢复、下架或审核能力。
- 不把 repository 集成测试变成默认必须依赖数据库的测试。
- 不使用真实开发库或生产库做破坏性测试。

## Acceptance Criteria

- `gofmt -l .` 无输出。
- `git diff --check` 通过。
- `go test ./...` 在没有 PostgreSQL 测试 DSN 时仍通过。
- 设置 `CONTENT_BACKEND_TEST_DATABASE_DSN` 后，repository 集成验证能在真实 PostgreSQL 上通过。
- migration helper 按文件名顺序执行 `migrations/*.sql`。
- 新库初始化路径被验证。
- 已有库 `001 -> 002` 升级路径被验证。
- 文章 repository 集成测试覆盖逻辑删除后的查询隐藏、重复删除失败、删除后发布/编辑失败。
- 文档说明测试数据库必须是一次性数据库或专用测试库。
- 收口前检查 `README.md`、`docs/deployment.md`、`ROADMAP.md`、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否需要同步。

## AI Agent Notes

- 优先读取 `AGENTS.md`、`README.md`、`docs/deployment.md`、`migrations/001_init.sql`、`migrations/002_add_article_deleted_at.sql`、`internal/repository/article_repository.go`、`internal/service/article_service_test.go` 和 `.github/workflows/test.yml`。
- 不要让默认 `go test ./...` 依赖本地数据库；未设置 DSN 时应跳过集成测试。
- 集成测试必须明确避免污染真实库。可以要求测试前确认数据库名包含 `test`，或在文档中要求使用一次性数据库。
- 如果测试会清空表，必须只对专用测试数据库执行。
- 当前 CI 只跑 `gofmt` 和 `go test ./...`；本 issue 暂不改变 CI。
- 如果实现中发现需要长期 migration 规则，再提炼到 `AGENTS.md`，不要只留在 issue 草稿中。
