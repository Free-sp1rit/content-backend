# Issue: 补充 Redis 运行验证和边界测试

## User Facing Issue

### Background

项目已经接入 Redis，用于登录失败限流、公开文章列表缓存和 singleflight 防击穿辅助。`docs/redis.md` 已记录当前 key、TTL、失效和失败策略。下一步需要把 Redis 场景从“文档可解释”推进到“真实依赖可验证”。

### Tasks

- [ ] 检查 `docs/redis.md` 是否仍匹配当前代码实现。
- [ ] 为登录失败限流补充真实 Redis 验证步骤或集成测试方案。
- [ ] 为公开文章列表缓存补充真实 Redis 验证步骤或集成测试方案。
- [ ] 验证 Redis 不可用时登录和公开列表的当前降级表现。
- [ ] 记录验证方式，避免只停留在 redismock 单元测试。

### Acceptance Criteria

- 已实现 Redis 场景的 key、TTL、失效、降级策略仍有文档说明。
- 至少一个 Redis 场景有真实依赖验证方案、手工步骤或集成测试。
- 验证方式能说明 PostgreSQL 仍是核心事实来源，Redis 只保存短期运行态。
- `go test ./...` 保持通过。

## AI Agent Notes

- 优先读取 `internal/service/redis_login_limiter.go`、`internal/service/redis_cache.go`、`internal/service/auth_service.go`、`internal/service/article_service.go`。
- 不要发明代码里不存在的能力；未来方向要标明是候选场景。
- 验证说明应服务学习，不写成完整压测或监控平台方案。
- 如果发现实现与文档目标冲突，先记录为后续 issue，不在本任务扩展实现。
