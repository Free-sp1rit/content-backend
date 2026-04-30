# Issue: Redis 阅读计数原子化与验证收口

## Status

已完成。当前登录用户阅读去重已经从客户端侧 `SET NX EX + INCR` 两步命令改为 Redis Lua 脚本；相关测试、`docs/redis.md`、本地学习笔记和 Compose smoke 验证已同步。

## Background

当前项目已经完成两步 Redis 阅读计数学习实践：

- `001-redis-atomic-counter-integration`：公开文章详情访问后，使用 `INCR article:views:<article_id>` 记录原始访问次数。
- `004-redis-authenticated-view-dedup`：公开文章详情增加可选 JWT 认证，并使用 `SET NX EX + INCR` 记录登录用户去重阅读次数。

其中登录用户去重计数当前由客户端侧两步 Redis 命令完成：

```text
SET article:viewed:<article_id>:user:<user_id> 1 NX EX 24h
如果 SET 成功 -> INCR article:user_views:<article_id>
```

这个实现可以工作，但不是一个 Redis 内部原子整体。如果 `SET NX EX` 成功后 `INCR` 失败，会出现“用户已被标记为看过，但登录用户阅读数没有增加”的中间不一致。下一步需要学习 Redis Lua 脚本，把这个复合动作收口成 Redis 内部一次原子执行。

同时，当前 `SetNX` 调用在编辑器中会提示 deprecated。改成 Lua 脚本后，这个接口选择问题会自然消失，不需要单独做 `SetArgs` 迁移。

## Goals

- 学习 Redis Lua 脚本如何保证复合 Redis 操作的原子执行。
- 将登录用户阅读去重的 `SET NX EX + INCR` 改为 Lua 脚本。
- 消除 `SetNX` deprecated 提示。
- 补充真实运行验证，确认原始计数、登录用户去重计数和无效 JWT 行为。
- 更新 `docs/redis.md` 和 `.notes/Redis原子计数器.md`。

## Proposed Design

### Lua 脚本语义

目标脚本逻辑：

```lua
if redis.call("SET", KEYS[1], "1", "NX", "EX", ARGV[1]) then
  return redis.call("INCR", KEYS[2])
end

return 0
```

参数含义：

```text
KEYS[1] = article:viewed:<article_id>:user:<user_id>
KEYS[2] = article:user_views:<article_id>
ARGV[1] = 去重窗口秒数
```

结果含义：

```text
返回 0：窗口内重复访问，未增加登录用户去重计数
返回正整数：首次访问，返回递增后的 article:user_views:<article_id>
```

### 保留原始访问计数

`article:views:<article_id>` 暂时继续使用独立 `INCR`，因为它表示原始访问次数，不参与登录用户去重判断。

### 失败策略

- Lua 脚本执行失败时记录日志，不阻断公开文章详情。
- Redis 不可用时，公开文章详情仍然可读，只是计数可能丢失。
- 这是学习阶段的 fail-open 策略；如果未来阅读数成为核心产品数据，需要重新设计持久化、补偿和恢复。

## Tasks

- [x] 将登录用户阅读去重的 `SET NX EX + INCR` 改为 Redis Lua 脚本。
- [x] 移除对 deprecated `SetNX` API 的直接调用。
- [x] 补充或调整 Redis 命令行为测试：
  - [x] 首次登录用户访问时写入去重标记并递增 `article:user_views:<article_id>`。
  - [x] 同一登录用户窗口内重复访问时不递增 `article:user_views:<article_id>`。
  - [x] Redis 脚本失败时返回错误给 service，由 service 记录日志并继续返回文章详情。
- [x] 对比登录限流 Lua 和阅读去重 Lua 的相同点与差异。
- [x] 补充 Docker Compose smoke 验证：
  - [x] 未登录访问公开详情会递增 `article:views:<article_id>`。
  - [x] 登录用户首次访问公开详情会递增 `article:user_views:<article_id>`。
  - [x] 同一登录用户重复访问不会重复递增 `article:user_views:<article_id>`。
  - [x] 携带无效 JWT 访问公开详情返回 `401`。
- [x] 更新 `docs/redis.md`，说明 Lua 脚本、key、TTL、失败策略和验证方式。
- [x] 更新 `.notes/Redis原子计数器.md`，追加“从 `SET NX EX + INCR` 到 Lua 脚本”的学习笔记。
- [x] 运行 `gofmt`、`git diff --check` 和 `go test ./...`。

## Verification

- `gofmt -w internal/service/redis_article_view_counter.go internal/service/redis_article_view_counter_test.go`
- `go test ./...`
- `git diff --check`
- Docker Compose smoke：
  - `/healthz` 返回 `ok`
  - 注册返回 `201`，登录返回 `200`
  - 创建文章返回 `201`，发布返回 `200`
  - 匿名访问公开详情返回 `200`，`article:views:<article_id>` 为 `1`
  - 同一登录用户访问两次均返回 `200`，`article:user_views:<article_id>` 从 `1` 保持为 `1`
  - `article:viewed:<article_id>:user:<user_id>` 存在
  - 携带无效 JWT 访问公开详情返回 `401`，原始访问计数不增加

## Non Goals

- 不新增 API 响应字段，不返回原始阅读数或登录用户去重阅读数。
- 不实现匿名用户去重、cookie、设备 ID 或浏览器指纹。
- 不实现排行榜、浏览历史、阅读者列表或批量落库。
- 不改 PostgreSQL schema。
- 不把 Redis 计数升级为核心事实来源。

## Acceptance Criteria

- 登录用户阅读去重由 Redis Lua 脚本完成。
- 代码中不再直接使用 deprecated `SetNX` API。
- 同一登录用户在去重窗口内重复访问不会重复增加 `article:user_views:<article_id>`。
- Redis 脚本失败不会阻断公开文章详情读取。
- Compose smoke 验证覆盖原始计数、登录用户去重计数和无效 JWT。
- `docs/redis.md` 与实现一致。
- `.notes/Redis原子计数器.md` 有对应学习总结。
- `go test ./...` 通过。

## AI Agent Notes

- 这是 Redis 阅读计数专题的原子化收口任务，不要扩大到产品级统计系统。
- 优先参考 `internal/service/redis_login_limiter.go` 中 Lua 脚本的写法和测试方式。
- Lua 脚本只处理登录用户去重计数，不要把原始访问计数和去重计数强行合并成一个复杂脚本。
- 保持 service fail-open：Redis 计数失败只记录日志，不影响公开文章详情。
- `.notes/` 是本地学习资料，当前被 `.gitignore` 忽略；如果需要提交笔记策略，先与用户确认。
