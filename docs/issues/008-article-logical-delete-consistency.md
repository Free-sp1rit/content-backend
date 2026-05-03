# Issue: 为文章补逻辑删除链路并加固删除一致性

## Status

已完成。当前任务已为文章补充 `DELETE /me/articles/{id}` 逻辑删除链路，采用 `deleted_at`，不引入 `unpublished` 状态，也不一次性展开完整文章生命周期状态机。

## Background

当前项目已经支持创建草稿、编辑草稿、发布文章、公开查询文章和 Redis 阅读计数。文章发布和编辑已经通过 PostgreSQL 条件更新保护作者、状态和重复请求边界。

下一阶段学习重点从“发布/编辑的状态一致性”延伸到“删除动作对文章可见性、缓存和并发边界的影响”。

本阶段不把 `draft` 和 `unpublished` 拆成两个状态。当前项目暂不需要表达“曾发布后被下架，但未来可以重新上架”的独立语义；先保持文章状态只有 `draft` 和 `published`，删除通过 `deleted_at` 表示。

## Goals

- 新增作者删除文章能力，完成从 handler 到 service、repository、database 的完整链路。
- 使用逻辑删除保留文章记录，避免物理删除导致后续审计、恢复或关联数据学习空间被提前关闭。
- 使用条件 SQL 保证删除动作在重复请求、非作者请求和并发请求下语义清晰。
- 删除后的文章不再出现在公开列表、公开详情和作者文章列表中。
- 删除 published 文章后，公开文章列表缓存必须失效。

## Decisions

- 删除使用逻辑删除：新增 `articles.deleted_at TIMESTAMPTZ NULL`。
- 不新增 `unpublished` 状态。
- 不新增恢复文章能力。
- 不删除 Redis 阅读计数 key。
- 删除后的文章对公开接口和作者接口都视作不可见。
- 重复删除返回 `404 Not Found`，因为已删除文章对当前 API 语义等同不可见资源。

## Invariants

- 只有作者能删除自己的文章。
- 已删除文章不能再次发布。
- 已删除文章不能再次编辑。
- 已删除文章不能再次被公开读取。
- 已删除文章不能出现在 `GET /articles`。
- 已删除文章不能出现在 `GET /me/articles`。
- 重复删除不应重复产生状态迁移或额外副作用。
- 删除成功后如果影响公开可见性，必须删除 `articles:published` 缓存。
- 删除失败、权限失败、文章不存在或文章已删除时，不应删除公开文章列表缓存。

## Scope

- 新增迁移文件，例如 `migrations/002_add_article_deleted_at.sql`。
- 调整 `internal/repository/article_repository.go`：
  - 查询类方法过滤 `deleted_at IS NULL`。
  - 发布、编辑条件更新增加 `deleted_at IS NULL`。
  - 新增逻辑删除条件更新方法，让调用方知道是否真实删除成功，并能判断被删除文章原状态。
- 调整 `internal/service/article_service.go`：
  - 新增 `DeleteArticle`。
  - 删除失败后由 service 查询并解释失败原因。
  - 只有真实删除成功后才处理缓存失效。
- 调整 `internal/handler/article_handler.go`：
  - 为 `DELETE /me/articles/{id}` 接入删除动作。
  - 成功删除返回 `204 No Content`。
- 调整 `cmd/server/main.go` 中 `/me/articles/{id}` 的方法分发，支持 `PUT` 和 `DELETE`。
- 调整 `internal/handler/article_errors.go` 中必要的错误映射；若复用现有错误，可保持映射不变。
- 补充 service 层测试，覆盖重复删除、非作者删除、删除后发布、删除后编辑、删除后公开读取和缓存失效。
- 补充 handler 测试，确认 `DELETE /me/articles/{id}` 的状态码稳定。
- 必要时更新 `README.md`、`docs/architecture.md`、`docs/deployment.md` 和 `scripts/smoke.sh`。

## Proposed Design

数据库新增字段：

```sql
ALTER TABLE articles
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;
```

查询公开文章时过滤逻辑删除：

```sql
SELECT id, author_id, title, content, state, created_at, updated_at
FROM articles
WHERE state = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC
```

公开详情和作者文章列表也应过滤 `deleted_at IS NULL`。

发布文章继续使用条件更新，但增加删除条件：

```sql
UPDATE articles
SET state = 'published', updated_at = NOW()
WHERE id = $1
  AND author_id = $2
  AND state = 'draft'
  AND deleted_at IS NULL
```

编辑文章同样增加删除条件：

```sql
UPDATE articles
SET title = $1, content = $2, updated_at = NOW()
WHERE id = $3
  AND author_id = $4
  AND state = 'draft'
  AND deleted_at IS NULL
```

删除文章使用条件更新：

```sql
UPDATE articles
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1
  AND author_id = $2
  AND deleted_at IS NULL
RETURNING state
```

repository 只返回“是否删除成功”和“被删除前的 state”。service 根据返回结果决定是否删除公开列表缓存。

条件删除影响 0 行时，由 service 再查询文章解释失败原因：

- 查不到文章，或文章已删除后被查询层隐藏：`ErrArticleNotFound`
- 作者不匹配：`ErrPermissionDenied`

## API Behavior

- `DELETE /me/articles/{id}` 需要登录。
- 删除成功：`204 No Content`
- 未登录：`401 Unauthorized`
- 路径 id 非法：`400 Bad Request`
- 文章不存在或已删除：`404 Not Found`
- 非作者删除：`403 Forbidden`
- 未知 service error：`500 Internal Server Error`

删除后：

- `GET /articles/{id}` 返回 `404 Not Found`
- `GET /articles` 不再包含该文章
- `GET /me/articles` 不再包含该文章
- `PUT /me/articles/{id}` 返回 `404 Not Found`
- `POST /articles/publish` 对已删除文章返回 `404 Not Found`

## Tasks

- [x] 新增 `migrations/002_add_article_deleted_at.sql`。
- [x] 明确已有数据库升级路径，并在必要文档中说明已有 volume 不会自动重跑旧 migration。
- [x] 调整 `internal/repository/article_repository.go` 的查询方法，统一过滤 `deleted_at IS NULL`。
- [x] 调整发布和编辑条件更新，防止已删除文章被重新发布或编辑。
- [x] 新增 repository 逻辑删除方法，使用条件 SQL 并返回删除前状态。
- [x] 调整 `internal/service/article_service.go`，新增 `DeleteArticle`。
- [x] 删除失败后由 service 解释 `not found` 和 `permission denied`。
- [x] 确保只有真实删除成功路径会触发公开文章列表缓存失效。
- [x] 调整 `internal/handler/article_handler.go`，新增删除 handler。
- [x] 调整 `cmd/server/main.go` 的 `/me/articles/{id}` 路由方法分发。
- [x] 补充 service 层测试。
- [x] 补充 handler 层测试。
- [x] 必要时更新 README、架构文档、部署文档和 smoke 验证。
- [x] 运行 `gofmt -l .`、`git diff --check` 和 `go test ./...`。
- [x] 若修改了 Compose/migration 运行路径，补充清晰手工验证步骤。

## Verification

- `gofmt -l .`
- `git diff --check`
- `bash -n scripts/smoke.sh`
- `go test ./...`
- `docker compose --env-file .env.compose config`
- `docker compose --env-file .env.compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f /docker-entrypoint-initdb.d/002_add_article_deleted_at.sql'`
- `docker compose --env-file .env.compose up --build -d`
- `NO_PROXY='*' no_proxy='*' scripts/smoke.sh http://127.0.0.1:8080`

## Non Goals

- 不做物理删除。
- 不新增 `unpublished` 状态。
- 不做文章恢复。
- 不做文章版本历史。
- 不做审计日志表。
- 不删除 Redis 阅读计数 key。
- 不新增后台管理接口。
- 不做批量删除。
- 不引入通用事务框架或 Redis 分布式锁。

## Acceptance Criteria

- `gofmt -l .` 无输出。
- `git diff --check` 通过。
- `go test ./...` 通过。
- 作者可以删除自己的 draft 文章。
- 作者可以删除自己的 published 文章。
- 非作者不能删除文章。
- 不存在文章返回 `404 Not Found`。
- 重复删除返回 `404 Not Found`。
- 删除后的文章不能被发布。
- 删除后的文章不能被编辑。
- 删除后的文章不能通过公开详情读取。
- 删除后的文章不出现在公开文章列表。
- 删除后的文章不出现在作者文章列表。
- 删除 published 文章成功后，`articles:published` 缓存失效。
- 删除失败、权限失败、文章不存在或文章已删除时，不删除公开文章列表缓存。
- migration 同时说明新库初始化路径和已有库升级路径。
- 收口前检查 `AGENTS.md`、`README.md`、`ROADMAP.md`、`docs/`、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否需要同步。

## AI Agent Notes

- 优先读取 `AGENTS.md`、`docs/architecture.md`、`migrations/001_init.sql`、`internal/model/article.go`、`internal/repository/article_repository.go`、`internal/service/article_service.go`、`internal/service/article_service_test.go`、`internal/handler/article_handler.go`、`internal/handler/article_handler_test.go`、`internal/handler/article_errors.go`、`cmd/server/main.go` 和 `scripts/smoke.sh`。
- 本 issue 的重点是“删除动作的局部状态边界”，不是一次性设计完整生命周期状态机。
- repository 只提供条件删除能力，不承载业务语义解释。
- service 决定删除失败后如何解释错误。
- PostgreSQL 继续作为文章内容、作者归属、发布状态和删除状态的事实来源。
- Redis 阅读计数不参与删除事务；本阶段不清理阅读计数 key。
- 缓存删除继续保持 best-effort 策略，不把 Redis 纳入 SQL 事务。
- 如果决定让 smoke 在最后删除测试文章，先保证不会破坏现有 smoke 对重复发布和发布后编辑的验证顺序。
