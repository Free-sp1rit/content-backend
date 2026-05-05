# Issue: 验证文章公开列表 Redis 缓存一致性

## Status

已完成。当前任务已补充文章公开列表缓存 `articles:published` 的一致性验证：确认发布、删除和查询链路在 Redis 缓存命中、缓存失效、缓存失败时仍以 PostgreSQL 事实为准。

## Background

项目当前已经具备文章发布、编辑、逻辑删除、PostgreSQL migration/repository 真实 SQL 验证，以及 Docker Compose smoke 验证。

公开文章列表目前使用 Redis 缓存：

```text
key: articles:published
TTL: 5m
```

当前实现已在 service 层通过 fake cache 测试了若干局部行为，例如发布成功后删除缓存、删除 published 文章后删除缓存、删除 draft 文章不删除公开列表缓存。`scripts/smoke.sh` 也覆盖了发布后公开列表可见、删除后公开列表不可见。

但这些验证还没有把“先暖缓存，再发生状态变化，再查询公开列表”的 stale cache 场景作为明确验收目标。下一阶段需要把这个边界补清楚，避免 Redis 中的旧公开列表压过 PostgreSQL 的最新事实。

## Goals

- 明确公开文章列表缓存是 PostgreSQL 公开文章列表的派生数据，不是事实来源。
- 验证发布成功后会使旧的空公开列表缓存失效，后续公开查询能看到新发布文章。
- 验证删除 published 文章后会使旧的公开列表缓存失效，后续公开查询不再看到已删除文章。
- 验证删除 draft 文章不会产生多余公开列表缓存失效。
- 验证发布/删除失败不会删除公开列表缓存。
- 验证 Redis 缓存删除失败时，发布/删除业务仍按当前 best-effort 策略返回成功，并记录为可接受的短期 stale 风险。
- 让 smoke 验证更明确覆盖缓存预热后的发布/删除链路。

## Decisions

- 不引入 Redis 分布式锁。
- 不引入消息队列、outbox 或后台修复任务。
- 不把 Redis 删除纳入 PostgreSQL 事务。
- 不改变 `articles:published` key 名称和 TTL。
- 不新增文章状态，不做下架、恢复或审核流。
- 默认 `go test ./...` 不依赖真实 Redis；真实 Redis 行为仍通过 redismock、Compose smoke 或手工验证覆盖。

## Scope

- 调整或补充 `internal/service/article_service_test.go`，覆盖缓存预热后的发布/删除一致性场景。
- 必要时新增 `internal/service/redis_cache_test.go`，用 redismock 覆盖 `RedisCache` 的 `GET` miss、`SET` TTL 和 `DEL` 行为。
- 调整 `scripts/smoke.sh`，让它显式执行：
  - 发布前先查询公开列表，预热空列表或旧列表缓存。
  - 发布后再查询公开列表，确认新文章可见。
  - 删除前先查询公开列表，预热包含该文章的缓存。
  - 删除后再查询公开列表，确认文章不可见。
- 必要时更新 `docs/redis.md`，补充删除 published 文章后的缓存失效说明和 stale cache 风险边界。
- 必要时更新 `README.md` 或 `docs/deployment.md` 中的 smoke 覆盖说明。
- 收口时更新 `docs/issues/010-redis-cache-consistency-verification.md`、`docs/issues/README.md` 和 `ROADMAP.md`。

## Proposed Test Coverage

### Service cache consistency

建议补充面向流程的 service 测试，而不是只测单个方法是否调用 `Delete`：

- `ListPublishedArticles` 先写入或命中空缓存。
- `PublishArticle` 成功后删除 `articles:published`。
- 后续 `ListPublishedArticles` 应 miss 缓存，重新从 repository 读取 published 文章并写回缓存。
- `ListPublishedArticles` 先缓存包含某篇 published 文章的列表。
- `DeleteArticle` 删除该 published 文章后删除 `articles:published`。
- 后续 `ListPublishedArticles` 应 miss 缓存，重新从 repository 读取列表，并且不包含已删除文章。
- `DeleteArticle` 删除 draft 文章时不删除公开列表缓存。
- `PublishArticle` 条件更新失败时不删除缓存。
- `DeleteArticle` 条件更新失败时不删除缓存。
- `cache.Delete` 返回错误时，`PublishArticle` 和删除 published 文章仍返回成功。

### Redis cache command behavior

如果新增 `redis_cache_test.go`，建议覆盖：

- Redis `GET` 返回 `redis.Nil` 时，`RedisCache.Get` 返回空字符串和 nil error。
- Redis `GET` 命中时返回字符串。
- `RedisCache.Set` 使用指定 TTL 写入 key。
- `RedisCache.Delete` 使用 `DEL` 删除 key。
- Redis 命令错误原样返回给 service，由 service 决定 fail-open 或记录日志。

### Compose smoke behavior

建议让 `scripts/smoke.sh` 的公开列表链路更有意图：

```text
GET /articles                         # 发布前预热公开列表缓存
POST /articles/publish                # 发布成功后删除缓存
GET /articles                         # 应看到新文章
GET /articles                         # 删除前预热包含文章的缓存
DELETE /me/articles/{id}              # 删除 published 文章后删除缓存
GET /articles                         # 应看不到已删除文章
```

这个 smoke 不需要直接读取 `articles:published` 的 JSON 内容；验收重点是从 HTTP 视角证明缓存失效后对外结果正确。

## Tasks

- [x] 阅读 `AGENTS.md`、`docs/redis.md`、`internal/service/article_service.go`、`internal/service/article_service_test.go`、`internal/service/redis_cache.go` 和 `scripts/smoke.sh`。
- [x] 补充 service 层缓存一致性流程测试。
- [x] 补充 `RedisCache` 命令行为测试，或确认现有测试已经覆盖并说明原因。
- [x] 调整 smoke 脚本，使缓存预热后的发布/删除公开列表结果成为明确验收步骤。
- [x] 检查并更新 `docs/redis.md` 的 published articles cache 说明。
- [x] 检查并更新 `README.md` / `docs/deployment.md` 的 smoke 覆盖说明。
- [x] 运行 `gofmt -l .`。
- [x] 运行 `git diff --check`。
- [x] 运行 `go test ./...`。
- [x] 如本地 Compose 环境可用，运行 `scripts/smoke.sh` 并记录结果。
- [x] 收口前检查 `AGENTS.md`、`ROADMAP.md`、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否需要同步。

## Verification

- `gofmt -l .`
- `git diff --check`
- `go test ./...`
- `docker compose --env-file .env.compose ps`
- `scripts/smoke.sh http://127.0.0.1:8080`

Smoke 结果包含：

```text
[ok] warm published articles cache before publish
[ok] list published articles after publish
[ok] warm published articles cache before delete
[ok] list published articles after delete
[ok] deleted article hidden from public list
```

## Non Goals

- 不新增公开 API。
- 不新增文章状态。
- 不实现下架、恢复、审核流。
- 不引入 Redis 锁。
- 不引入消息队列、outbox 或后台任务。
- 不把 Redis 变成文章公开状态的事实来源。
- 不改变当前缓存 TTL 和 key 设计，除非测试暴露出必须修复的问题。
- 不要求默认测试依赖真实 Redis 或 Docker Compose。

## Acceptance Criteria

- `gofmt -l .` 无输出。
- `git diff --check` 通过。
- `go test ./...` 通过。
- 发布前已存在旧公开列表缓存时，发布成功后公开列表能看到新文章。
- 删除前已存在包含该文章的公开列表缓存时，删除 published 文章成功后公开列表不再看到该文章。
- 删除 draft 文章不删除 `articles:published` 缓存。
- 发布失败、删除失败、权限失败、文章不存在时不删除 `articles:published` 缓存。
- `cache.Delete` 失败不会让发布成功或删除 published 成功变成业务失败。
- smoke 脚本明确覆盖缓存预热后的发布和删除公开查询链路。
- `docs/redis.md` 准确描述发布和删除 published 文章的缓存失效策略、失败策略和短期 stale 风险。
- 收口前检查 `README.md`、`docs/deployment.md`、`ROADMAP.md`、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否因本任务变旧。

## AI Agent Notes

- 优先读取 `AGENTS.md`、`docs/redis.md`、`internal/service/article_service.go`、`internal/service/article_service_test.go`、`internal/service/redis_cache.go`、`scripts/smoke.sh`、`README.md` 和 `docs/deployment.md`。
- 当前缓存 key 是 `articles:published`，TTL 是 5 分钟。
- PostgreSQL 仍是事实来源；Redis 只保存可删除、可过期、可重建的公开列表缓存。
- 缓存删除保持 best-effort，不把 Redis 纳入 PostgreSQL 状态变更事务。
- service 测试应优先验证业务流程和缓存边界，不把 Redis 命令细节散落到 service 主流程。
- `RedisCache` 命令级测试可使用 `github.com/go-redis/redismock/v9`。
- smoke 脚本应从 HTTP 结果验证缓存一致性，只有必要时才直接读取 Redis key。
- 如果发现现有实现存在真实一致性 bug，先最小修复，再补测试；不要顺手扩展文章生命周期。
