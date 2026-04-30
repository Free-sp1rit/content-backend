# Redis Runtime State

本文档记录 Redis 在项目中的使用边界、当前场景、失败策略和后续整理方向。

## Positioning

Redis 在本项目中不替代 PostgreSQL。它用于短期运行态和性能保护：

- 登录失败限流
- 公开文章列表缓存
- 缓存击穿保护的配合能力
- 文章阅读计数原型

核心原则：

- PostgreSQL 保存事实数据。
- Redis 保存可过期、可重建、可丢失后降级的数据。
- 每个 Redis 场景必须明确 key、TTL、失效策略、失败策略和测试方式。

## Current Scenarios

### Login Failure Rate Limit

当前登录限流按两个维度记录失败次数：

- email 维度：限制单个账号被连续猜密码。
- IP 维度：限制同一来源批量尝试登录。

当前 key 设计：

```text
login:failures:email:<normalized-email>
login:failures:ip:<client-ip>
```

- email 会先 `trim + lower`，避免大小写和首尾空格导致多个计数器。
- IP 来自 handler 解析后的客户端地址；在 nginx 后面运行时，只在上一跳可信时读取 `X-Forwarded-For` / `X-Real-IP`。
- 默认阈值：email 失败 5 次、IP 失败 20 次。
- 默认窗口：10 分钟，可通过 `LOGIN_RATE_LIMIT_WINDOW` 配置。
- 失败计数使用 Redis Lua 脚本完成 `INCR` 和首次 `EXPIRE`，避免计数成功但 TTL 未设置的坏状态。
- 命中限流时返回 `429 Too Many Requests`，并用 Redis key 剩余 TTL 设置 `Retry-After`。
- 登录成功后只重置 email 维度失败计数，不重置 IP 维度；一个账号成功登录不应清空该来源对其他账号造成的大量失败记录。

失败策略：

- Redis 限流检查失败时记录日志并继续登录流程，当前选择 fail-open。
- 记录失败次数或重置失败次数失败时记录日志，不阻断登录响应。
- 这个取舍优先保证核心登录链路可用；代价是 Redis 故障期间登录保护会变弱。

### Published Articles Cache

当前公开文章列表使用 Redis 缓存，并在发布文章后删除缓存。

当前缓存设计：

- key：`articles:published`
- TTL：5 分钟
- 缓存内容：公开文章列表 JSON；空列表会缓存为 `[]`
- 缓存范围：只缓存公开列表，不缓存作者侧私有文章列表
- miss 后重建：先查 Redis；miss 后进入 `singleflight`；组内再次查 Redis；仍 miss 才查 PostgreSQL 并写回 Redis
- 失效时机：文章发布状态更新成功后删除 `articles:published`

失败策略：

- Redis 读取失败时记录为 miss，回退到 PostgreSQL。
- Redis 写入失败时记录日志，仍返回 PostgreSQL 查询结果。
- 发布成功后删除缓存失败时记录日志；如果 Redis 中仍有旧值，最坏会保留到 TTL 到期。
- PostgreSQL 仍是公开文章事实来源，Redis 只影响缓存命中和短期新鲜度。

当前并发边界：

- 并发缓存 miss 由应用内 `singleflight` 合并，减少同一进程内同时打到 PostgreSQL 的请求。
- `singleflight` 不是跨进程锁；如果未来多副本部署，需要重新评估跨实例击穿保护。
- 发布和编辑的状态流转仍需在文章并发加固任务中继续处理。

### Article View Counter Prototype

当前公开文章详情访问成功后，会使用 Redis 原子自增记录阅读次数。

当前 key 设计：

```text
article:views:<article_id>
```

当前实现边界：

- 触发时机：只有公开文章详情查询成功，并且文章状态为 `published` 后才递增。
- Redis 命令：使用 `INCR`，保证单个 key 的递增在 Redis 内部原子完成。
- TTL：第一版不设置 TTL；这是学习型原型，后续如果长期保留计数，需要补清理、落库或重建策略。
- 防重复：第一版不做同一用户或同一 IP 去重；刷新一次算一次。
- API 表现：第一版不在文章详情响应中返回阅读数，避免把 Redis 原型直接扩大成公开 API 契约。
- PostgreSQL 边界：文章内容仍以 PostgreSQL 为事实来源；阅读计数暂不写入 PostgreSQL。

失败策略：

- Redis 自增失败时记录日志，不影响文章详情查询。
- Redis 不可用时，文章仍可正常读取，只是阅读计数暂时丢失或停止增长。
- 这个取舍适合当前学习阶段；如果未来阅读数成为产品核心数据，需要重新设计持久化和补偿机制。

运行验证：

第一版采用 Docker Compose smoke 步骤验证真实 Redis 行为。假设已经按 `README.md` 准备 `.env.compose`、`app.env`、`db.env` 并启动服务：

```bash
docker compose --env-file .env.compose up --build -d
```

准备一个临时用户、创建草稿并发布：

```bash
BASE_URL=http://127.0.0.1:8080
EMAIL="redis-counter-$(date +%s)@example.com"
PASSWORD="secret123"

curl -sS -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"

TOKEN=$(curl -sS -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
  | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

ARTICLE_ID=$(curl -sS -X POST "$BASE_URL/articles" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"redis counter smoke","content":"check article view counter"}' \
  | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')

curl -sS -X POST "$BASE_URL/articles/publish" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"article_id\":$ARTICLE_ID}"
```

访问公开详情两次，再读取 Redis key：

```bash
curl -sS "$BASE_URL/articles/$ARTICLE_ID" > /dev/null
curl -sS "$BASE_URL/articles/$ARTICLE_ID" > /dev/null

docker compose --env-file .env.compose exec -T redis \
  redis-cli GET "article:views:$ARTICLE_ID"
```

预期 Redis 返回 `2`。如果这个 key 已经在之前的手工验证中存在，返回值可能大于 `2`，但应该随每次成功访问公开详情继续递增。

边界验证：

```bash
DRAFT_ID=$(curl -sS -X POST "$BASE_URL/articles" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"redis counter draft","content":"should stay hidden"}' \
  | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')

curl -i "$BASE_URL/articles/$DRAFT_ID"
docker compose --env-file .env.compose exec -T redis \
  redis-cli EXISTS "article:views:$DRAFT_ID"

MISSING_ID=999999999
curl -i "$BASE_URL/articles/$MISSING_ID"
docker compose --env-file .env.compose exec -T redis \
  redis-cli EXISTS "article:views:$MISSING_ID"
```

草稿文章和不存在的文章详情都应返回 `404`，对应 Redis key 的 `EXISTS` 应返回 `0`。

Redis 故障降级可以在本地 smoke 环境中这样验证：

```bash
docker compose --env-file .env.compose stop redis
curl -i "$BASE_URL/articles/$ARTICLE_ID"
docker compose --env-file .env.compose logs app --tail=50
docker compose --env-file .env.compose start redis
```

预期文章详情仍返回 `200`，应用日志中出现阅读计数自增失败记录。验证完成后要重新启动 Redis，避免影响后续登录限流、公开列表缓存和阅读计数验证。

## Future Candidate Scenarios

这些方向可以在后续阶段逐步选择，不需要一次性完成：

- 阅读计数批量落库
- 阅读计数短期去重
- token 黑名单或会话控制
- 更细粒度的接口限流
- 异步任务状态
- 简单排行榜或趋势榜

## Non Goals

- 不把 Redis 当主数据库使用。
- 不为了学习 Redis 增加脱离业务的示例代码。
- 不在未明确一致性边界前缓存作者侧私有数据。
