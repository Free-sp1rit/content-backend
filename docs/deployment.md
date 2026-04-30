# Deployment

本文档记录当前部署链路和 Post-MVP Alpha 阶段部署成熟化方向。具体启动命令仍以 `README.md` 为准。

## Current Shape

当前项目使用 Docker Compose 组织服务：

```text
host -> nginx -> app -> PostgreSQL
                 -> Redis
```

相关文件：

- `Dockerfile`
- `docker-compose.yml`
- `deploy/nginx/default.conf`
- `.env.compose.example`
- `app.env.example`
- `db.env.example`
- `README.md`

当前 Compose 服务：

- `nginx`：宿主机入口，默认把 `${APP_PORT:-8080}` 转发到内部 `app:8080`。
- `app`：Go HTTP 服务，依赖 PostgreSQL 和 Redis 健康检查通过后启动。
- `db`：PostgreSQL，使用 `postgres_data` volume 持久化数据。
- `redis`：Redis，使用 `redis_data` volume，并开启 AOF。

## Environment Files

示例文件可以进入 Git：

- `.env.example`
- `.env.compose.example`
- `app.env.example`
- `db.env.example`

真实环境文件不得进入 Git：

- `.env`
- `.env.compose`
- `app.env`
- `db.env`

真实密钥、密码、服务器 IP、代理地址和临时调试配置都应留在服务器或本机环境中。

`app.env.example` 需要覆盖应用运行配置，包括：

- HTTP 服务配置：`PORT`、`READ_HEADER_TIMEOUT`
- PostgreSQL 配置：`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`、`DB_SSLMODE`
- JWT 配置：`JWT_SECRET`、`JWT_ISSUER`、`JWT_TOKEN_TTL`
- Redis 配置：`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB`
- 登录限流配置：`LOGIN_RATE_LIMIT_EMAIL_MAX_FAILURES`、`LOGIN_RATE_LIMIT_IP_MAX_FAILURES`、`LOGIN_RATE_LIMIT_WINDOW`

## Minimum Verification

部署或更新后，至少验证：

- `docker compose ps` 显示 `db`、`redis`、`app`、`nginx` 正常运行。
- `/healthz` 返回 `ok`。当前 `/healthz` 检查应用和 PostgreSQL；Redis 由 Compose healthcheck 和业务 smoke 覆盖。
- 公开文章列表可访问。
- 注册新用户。
- 登录并取得 JWT。
- 使用 JWT 创建文章草稿。
- 发布文章。
- 再次查询公开文章列表，确认新发布文章可见。
- 访问公开文章详情，并按 `docs/redis.md` 的 Redis 阅读计数 smoke 步骤确认 `article:views:<article_id>` 递增。

常见排障入口：

- `docker compose logs nginx --tail=50`
- `docker compose logs app --tail=50`
- `docker compose logs db --tail=50`
- `docker compose logs redis --tail=50`

## Next Improvements

- 增加可复用 smoke checklist 或脚本。
- 明确更新部署流程和回滚思路。
- 补充数据备份和恢复策略。
- 区分仓库部署规则和服务器私有配置。

## Out Of Scope For Now

- Kubernetes
- 蓝绿或灰度发布
- 复杂自动发布平台
- 完整监控报警体系
