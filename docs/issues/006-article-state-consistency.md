# Issue: 加固文章发布和编辑的状态一致性

## Status

已完成。当前文章发布和编辑已经改为使用 PostgreSQL 条件更新保护状态流转不变量；重复发布返回状态冲突，发布失败不会删除公开文章列表缓存。

## Background

当前项目已进入 Post-MVP Alpha 阶段，文章发布和编辑是核心内容链路。

现有文章发布/编辑逻辑存在典型的“先查再判断再更新”形态。在重复请求、并发发布、编辑与发布同时发生时，状态判断和实际更新之间可能出现竞态窗口。

本任务聚焦文章状态流转一致性：用 PostgreSQL 条件更新保护业务不变量，并让 service 在更新失败后负责解释失败原因。

## Goals

- 保证文章发布和编辑在并发或重复请求下仍符合业务规则。
- 使用条件 SQL 保护状态流转，不引入通用事务框架。
- 拆清发布失败和编辑失败的状态语义。
- 确保只有真实发布成功后才删除公开文章列表缓存。

## Invariants

- 只有作者能发布自己的文章。
- 只有作者能编辑自己的文章。
- 只有 `draft` 状态文章能被发布。
- 只有 `draft` 状态文章能被编辑。
- 重复发布不应重复产生状态迁移或额外副作用。
- 发布成功后公开文章列表缓存必须失效。
- 发布失败、权限失败、文章不存在时不应删除公开文章列表缓存。

## Scope

- 调整 `internal/repository/article_repository.go`，增加或改造文章条件更新能力。
- 调整 `internal/service/article_service.go` 中的 `PublishArticle` 和 `UpdateArticle`。
- 新增更清晰的发布状态错误，例如 `ErrArticleNotPublishable`。
- 调整 `internal/handler/article_errors.go` 中的错误映射。
- 补充 service 层测试，覆盖重复发布、发布后编辑、非作者操作和条件更新失败后的错误解释。
- 必要时补 handler 测试，确认 HTTP 状态码稳定。

## Proposed Design

发布文章使用条件更新：

```sql
UPDATE articles
SET state = 'published', updated_at = NOW()
WHERE id = $1
  AND author_id = $2
  AND state = 'draft'
```

编辑文章使用条件更新：

```sql
UPDATE articles
SET title = $1, content = $2, updated_at = NOW()
WHERE id = $3
  AND author_id = $4
  AND state = 'draft'
```

条件更新成功时直接返回成功。

条件更新影响 0 行时，由 service 再查询文章解释失败原因：

- 查不到文章：`ErrArticleNotFound`
- 作者不匹配：`ErrPermissionDenied`
- 发布时状态不是 `draft`：`ErrArticleNotPublishable`
- 编辑时状态不是 `draft`：`ErrArticleNotEditable`

## Error Mapping

- `ErrArticleNotFound` -> `404 Not Found`
- `ErrPermissionDenied` -> `403 Forbidden`
- `ErrArticleNotEditable` -> `409 Conflict`
- `ErrArticleNotPublishable` -> `409 Conflict`

## Tasks

- [x] 增加或改造 repository 条件更新方法，让调用方能判断是否真实更新成功。
- [x] 调整 `PublishArticle`，用条件更新完成 `draft -> published` 状态迁移。
- [x] 调整 `UpdateArticle`，只允许作者更新 `draft` 文章。
- [x] 条件更新失败后，由 service 使用查询结果解释失败原因。
- [x] 新增 `ErrArticleNotPublishable`，并同步 handler 错误映射。
- [x] 确认只有发布成功路径会删除 `articles:published` 缓存。
- [x] 补充 service 层边界测试。
- [x] 必要时补充 handler 测试。
- [x] 运行 `gofmt -l .` 和 `go test ./...`。

## Verification

- `gofmt -l .`
- `go test ./...`

## Non Goals

- 不做通用数据库事务封装。
- 不做 Redis 分布式锁。
- 不做 `Idempotency-Key` 机制。
- 不做文章版本历史。
- 不做已发布文章再次编辑、撤回或审核流。
- 不做阅读计数落库。
- 不改变现有公开 API 路径。

## Acceptance Criteria

- `gofmt -l .` 无输出。
- `go test ./...` 通过。
- 草稿文章可以正常发布。
- 重复发布同一文章返回明确状态冲突，不重复产生副作用。
- 已发布文章不能再次编辑。
- 非作者不能发布或编辑文章。
- 不存在文章返回 not found。
- 只有发布成功时才删除 `articles:published` 缓存。
- 发布失败、权限失败、文章不存在时不删除公开文章列表缓存。
- service 测试覆盖条件更新失败后的错误解释逻辑。
- 收口前检查 `AGENTS.md`、`README.md`、`ROADMAP.md`、`docs/` 是否因本任务变旧。

## AI Agent Notes

- 优先读取 `AGENTS.md`、`docs/architecture.md`、`internal/service/article_service.go`、`internal/service/article_service_test.go`、`internal/repository/article_repository.go`、`internal/handler/article_errors.go`、`internal/handler/article_handler_test.go` 和 `internal/model/article.go`。
- repository 只提供条件更新能力，不承载业务语义解释。
- service 决定如何解释条件更新失败。
- 更新安全性靠条件 SQL；失败原因解释可以靠二次查询。
- Redis 缓存删除保持 best-effort 策略，不把 Redis 纳入 SQL 事务。
- 收口时询问是否需要同步更新 `docs/issues/` 和 `.github/ISSUE_TEMPLATE/`。
