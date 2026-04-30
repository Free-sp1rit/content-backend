# Issue: 补充 Redis 阅读计数运行验证和边界测试

## Status

已完成。当前采用 Docker Compose smoke 步骤验证真实 Redis 行为，并补充 service 边界测试确认找不到文章时不会触发阅读计数。

本地验证结果：

- 已发布文章详情访问两次后，`article:views:<article_id>` 返回 `2`。
- 草稿文章详情返回 `404`，对应阅读计数 key 不存在。
- 不存在文章详情返回 `404`，对应阅读计数 key 不存在。
- 停止 Redis 后，公开文章详情仍返回 `200`，应用日志记录阅读计数自增失败；随后 Redis 已重新启动并恢复 healthy。

## Background

`001-redis-atomic-counter-integration` 已经在 service 层接入 Redis `INCR` 阅读计数，并补充了单元测试和 Redis 命令行为测试。下一步需要用更接近真实运行的方式验证这个功能，确认公开文章详情访问会递增 Redis key，并把可复现的验证步骤沉淀下来。

## Goals

- 验证 `article:views:<article_id>` 在真实 Redis 或集成环境中会按访问次数递增。
- 验证未公开、找不到或无权访问的文章不会产生阅读计数。
- 验证 Redis 阅读计数失败不会阻断公开文章详情查询。
- 将验证命令、预期输出和边界结论补充到合适文档。

## Tasks

- [x] 选择验证方式：Docker Compose smoke、集成测试或最小手工验证。
- [x] 准备一篇已发布文章，并访问公开文章详情至少两次。
- [x] 通过 Redis 查询确认 `article:views:<article_id>` 递增。
- [x] 验证草稿文章、缺失文章或非公开详情不会递增阅读计数。
- [x] 验证或说明 Redis 不可用时文章详情查询的降级表现。
- [x] 将验证步骤和结论更新到 `docs/redis.md`，必要时同步 `docs/deployment.md` 或 `README.md`。
- [x] 运行 `go test ./...`。

## Non Goals

- 不返回阅读数到公开 API。
- 不实现阅读数落库、排行榜或去重。
- 不扩大文章详情接口契约。

## Acceptance Criteria

- 有一组可复现的 Redis 阅读计数验证步骤或自动化测试。
- 文档能说明如何确认 `article:views:<article_id>` 的递增行为。
- Redis 故障降级策略仍与 `docs/redis.md` 一致。
- `go test ./...` 通过。

## AI Agent Notes

- 这是 Redis 运行验证任务，优先补验证和文档；除非发现实现 bug，否则不要扩大业务代码范围。
- 验证命令不要包含真实密码、token、服务器 IP 或本机代理配置。
- 如果选择手工 smoke，记录关键命令和预期结果即可，不需要把个人环境输出原样提交。
