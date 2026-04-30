# Issue: 增加登录用户阅读去重 Redis 原型

## Status

已完成。当前实现了公开文章详情可选 JWT 认证、登录用户阅读去重 Redis 原型、相关测试和 `docs/redis.md` 文档更新。

## Background

当前项目已经有公开文章详情阅读计数原型：访问 `GET /articles/{id}` 成功后，对 Redis key `article:views:<article_id>` 执行 `INCR`。这个计数表示原始访问次数，不做去重。

经过本阶段讨论，暂不为了非核心浏览量功能引入匿名 cookie / viewer 模块。下一步选择复用现有 JWT 认证能力，在公开文章详情上增加“可选认证”，并基于登录用户做短期阅读去重。

这个方向符合当前项目边界：

- 公开文章详情仍然公开可访问。
- 未登录用户仍然可以读取公开文章详情。
- 如果请求携带合法 JWT，系统可以识别当前用户。
- 登录用户阅读去重是独立指标，不替代原始访问计数。
- 不引入 cookie、匿名设备 ID、浏览器指纹或复杂统计系统。

## Goals

- 学习 Redis `SET NX EX` 的短期去重用法。
- 增加可选认证能力：公开接口不强制登录，但能识别合法登录用户。
- 保留现有原始访问计数 `article:views:<article_id>`。
- 新增登录用户去重阅读计数原型。
- 明确 JWT、可选认证、Redis 去重计数之间的职责边界。

## Proposed Design

### 可选认证

新增 middleware 能力，例如：

```text
OptionalLogin
```

建议规则：

```text
没有 Authorization header -> 匿名访问，继续
Authorization header 合法 -> 注入 user_id，继续
Authorization header 存在但格式错误、签名无效或已过期 -> 返回 401
```

原因：

- 公开接口不能因为未登录而拒绝访问。
- 但客户端显式带了错误认证信息时，应得到明确的认证失败反馈。
- 可选认证只复用现有 JWT，不引入 cookie。

### Redis key

保留原始访问计数：

```text
article:views:<article_id>
```

新增登录用户去重计数：

```text
article:user_views:<article_id>
```

新增登录用户短期去重标记：

```text
article:viewed:<article_id>:user:<user_id>
```

### Redis 命令

使用：

```text
SET article:viewed:<article_id>:user:<user_id> 1 NX EX <window>
```

含义：

- `NX`：只有去重标记不存在时才设置成功。
- `EX`：给去重标记设置窗口 TTL。
- 设置成功表示该用户在当前窗口内第一次阅读这篇文章。

设置成功后再执行：

```text
INCR article:user_views:<article_id>
```

### 文章详情流程

```text
GET /articles/{id}
-> OptionalLogin 尝试识别 user_id
-> ArticleHandler.GetArticle 读取可选 user_id
-> ArticleService.GetArticle 查询公开文章详情
-> 原始计数：article:views:<article_id> 执行 INCR
-> 如果存在 user_id：
   -> SET article:viewed:<article_id>:user:<user_id> NX EX <window>
   -> SET 成功才 INCR article:user_views:<article_id>
-> 返回文章详情
```

### 分层边界

- `auth`：继续只负责 JWT 生成和校验。
- `middleware`：负责强制认证和可选认证，将合法 user_id 写入 context。
- `handler`：读取可选 user_id，传给 service；不直接执行 Redis 命令。
- `service`：决定公开详情成功后如何触发原始计数和登录用户去重计数。
- Redis counter 实现：封装 `INCR`、`SET NX EX` 和 key 设计。
- `main`：负责把 OptionalLogin 和新的 counter 依赖装配到路由。

## Tasks

- [x] 新增可选认证 middleware，并补 middleware 测试。
- [x] 将 `GET /articles/{id}` 包装为可选认证：未登录可访问，合法登录态可注入 user_id。
- [x] 调整 handler/service 接口，使文章详情链路能接收可选 viewer user id。
- [x] 保留 `article:views:<article_id>` 原始访问计数。
- [x] 新增登录用户去重计数 Redis 实现：
  - [x] `article:viewed:<article_id>:user:<user_id>`
  - [x] `article:user_views:<article_id>`
  - [x] `SET NX EX` 成功后才 `INCR`
- [x] 明确去重窗口默认值，例如 24 小时；如需要配置，补环境变量和示例文件。
- [x] Redis 去重失败时记录日志，不阻断文章详情。
- [x] 补 service 测试：
  - [x] 未登录访问只走原始访问计数。
  - [x] 登录用户首次访问触发用户去重计数。
  - [x] Redis 去重失败不阻断文章详情。
- [x] 补 Redis 命令行为测试。
  - [x] `SET NX EX` 成功后递增登录用户去重计数。
  - [x] 登录用户窗口内重复访问不重复增加用户去重计数。
- [x] 更新 `docs/redis.md`，说明 key、TTL、失败策略和边界。
- [x] 运行 `gofmt` 和 `go test ./...`。

## Non Goals

- 不引入 cookie、匿名 viewer id、设备指纹或前端存储。
- 不要求公开文章详情必须登录。
- 不把登录用户去重计数返回到 API。
- 不实现浏览历史、阅读者列表、排行榜或批量落库。
- 不把 Redis 计数作为 PostgreSQL 事实来源。

## Acceptance Criteria

- 未登录用户仍能访问公开文章详情。
- 合法登录用户访问公开文章详情时，可用于登录用户阅读去重。
- 无效或过期 JWT 在携带 `Authorization` header 时返回 `401`。
- `article:views:<article_id>` 继续记录原始访问次数。
- `article:user_views:<article_id>` 只在同一用户去重窗口内首次访问时递增。
- Redis 故障不会导致公开文章详情不可访问。
- `docs/redis.md` 已记录新增 key、TTL、失败策略和当前非目标。
- `go test ./...` 通过。

## AI Agent Notes

- 这是 Redis `SET NX EX` 学习实践，不是完整浏览统计系统。
- 优先保持分层清楚，不要让 service 直接解析 HTTP header 或 JWT。
- OptionalLogin 应复用现有 `TokenManager` 和 context 注入方式。
- 如果需要调整 `articleService` 接口，优先选择清晰表达可选 viewer 的参数或小结构，不要把 HTTP 概念泄漏进 service。
- 当前不迁移任何规则到 `AGENTS.md`，除非实现过程中沉淀出长期通用规则。
