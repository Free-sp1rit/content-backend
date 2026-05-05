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
- 发布前查询公开文章列表，预热公开列表缓存。
- 发布文章。
- 再次查询公开文章列表，确认缓存失效后新发布文章可见。
- 访问公开文章详情，并按 `docs/redis.md` 的 Redis 阅读计数 smoke 步骤确认 `article:views:<article_id>` 递增。
- 重复发布同一篇文章返回 `409`。
- 发布后编辑同一篇文章返回 `409`。
- 删除前再次查询公开文章列表，预热包含该文章的公开列表缓存。
- 删除文章返回 `204`。
- 删除后公开详情返回 `404`，公开列表不再包含该文章，确认缓存失效后已删除文章不可见。
- 重复删除返回 `404`。

仓库提供了可复用 smoke 脚本：

```bash
scripts/smoke.sh http://127.0.0.1:8080
```

脚本只验证已经运行的 Compose 服务，不负责启动、构建或清理环境。推荐部署或更新后按以下顺序执行：

```bash
docker compose --env-file .env.compose up --build -d
docker compose --env-file .env.compose ps
scripts/smoke.sh http://127.0.0.1:8080
```

如果宿主机端口不是 `8080`，把实际访问地址传给脚本。脚本依赖 `curl`、`jq` 和 Docker Compose；其中 `jq` 用于解析 API JSON 响应。

## Database Migration Notes

当前 Compose 会把 `migrations/` 挂载到 PostgreSQL 的 `/docker-entrypoint-initdb.d`。新 volume 首次初始化时会按文件名顺序执行 SQL，例如 `001_init.sql` 后执行 `002_add_article_deleted_at.sql`。

已有 PostgreSQL volume 不会自动重跑这些初始化脚本。升级到支持文章逻辑删除的版本时，需要对已有数据库手动执行新增 migration：

```bash
psql -U <your_user> -d <your_database> -f migrations/002_add_article_deleted_at.sql
```

如果需要验证 migration 与 repository 真实 SQL 行为，使用一次性测试库或专用测试库运行 opt-in 集成测试：

```bash
CONTENT_BACKEND_TEST_DATABASE_DSN='postgres://user:password@127.0.0.1:5432/content_backend_test?sslmode=disable' go test ./internal/repository -run Integration -count=1
```

该测试会重建目标数据库的 `public` schema；测试库名必须包含 `test`，不要指向开发库或生产库。

当前 Compose 配置没有把 PostgreSQL 端口暴露到宿主机。如果需要在 Compose 网络内验证，可以使用一次性 Go 容器：

```bash
docker compose --env-file .env.compose exec -T db dropdb --if-exists -U content_dev content_backend_test
docker compose --env-file .env.compose exec -T db createdb -U content_dev content_backend_test
docker run --rm --network content-backend_default -v "$PWD":/app -w /app \
  -e CONTENT_BACKEND_TEST_DATABASE_DSN='postgres://content_dev:devpass@db:5432/content_backend_test?sslmode=disable' \
  golang:1.22.5 go test ./internal/repository -run Integration -count=1
```

常见排障入口：

- `docker compose logs nginx --tail=50`
- `docker compose logs app --tail=50`
- `docker compose logs db --tail=50`
- `docker compose logs redis --tail=50`

## Next Improvements

- 明确更新部署流程和回滚思路。
- 补充数据备份和恢复策略。
- 区分仓库部署规则和服务器私有配置。

## Out Of Scope For Now

- Kubernetes
- 蓝绿或灰度发布
- 复杂自动发布平台
- 完整监控报警体系
