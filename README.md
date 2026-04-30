# Content Backend

一个使用 Go 实现的内容发布后端项目。

当前已经实现的主链路包括：

- 用户注册
- 用户登录并签发 JWT
- 创建文章草稿
- 编辑自己的草稿
- 发布文章
- 查看我的文章列表
- 查看公开文章列表
- 查看公开文章详情

## Tech Stack

- Go
- PostgreSQL
- Redis
- 标准库 `net/http`
- `github.com/lib/pq`
- `github.com/redis/go-redis/v9`

## Project Structure

- `cmd/server`
  应用启动入口和依赖装配
- `internal/handler`
  HTTP 输入输出、错误映射
- `internal/service`
  业务流程、权限判断、状态规则
- `internal/repository`
  数据访问
- `internal/auth`
  JWT 生成与校验
- `internal/middleware`
  登录态中间件
- `internal/config`
  环境变量配置加载
- `migrations`
  数据库初始化 SQL
- `docs`
  架构、部署、Redis 和 issue 草稿等项目上下文

## Requirements

- Go 1.22.5
- PostgreSQL
- Redis（如果不使用 Compose，需要本地或远端 Redis）
- Docker / Docker Compose（如果使用 Compose 方式运行）

## Database Setup

先创建数据库，然后执行初始化 SQL：

```bash
psql -U <your_user> -d <your_database> -f migrations/001_init.sql
```

## Configuration

项目通过环境变量启动，示例配置见 [`.env.example`](./.env.example)。

注意：

- 当前项目不会自动加载 `.env` 文件
- `.env.example` 用于说明需要哪些配置项和示例值
- 本地运行前，需要先在终端中手动导出这些环境变量

常用配置项：

- `PORT`
- `READ_HEADER_TIMEOUT`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`
- `JWT_SECRET`
- `JWT_ISSUER`
- `JWT_TOKEN_TTL`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `LOGIN_RATE_LIMIT_EMAIL_MAX_FAILURES`
- `LOGIN_RATE_LIMIT_IP_MAX_FAILURES`
- `LOGIN_RATE_LIMIT_WINDOW`

如果你只是本地开发，可以先在终端里临时导出这些变量：

```bash
export PORT=8080
export READ_HEADER_TIMEOUT=5s

export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=content_dev
export DB_PASSWORD=devpass
export DB_NAME=content_backend
export DB_SSLMODE=disable

export JWT_SECRET=dev-secret
export JWT_ISSUER=content-backend
export JWT_TOKEN_TTL=24h

export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=
export REDIS_DB=0

export LOGIN_RATE_LIMIT_EMAIL_MAX_FAILURES=5
export LOGIN_RATE_LIMIT_IP_MAX_FAILURES=20
export LOGIN_RATE_LIMIT_WINDOW=10m
```

## Run

```bash
go run ./cmd/server
```

服务默认监听：

```text
http://127.0.0.1:8080
```

## Docker Compose

项目提供了最小可用的 Compose 运行配置，用于同时启动 nginx、应用、PostgreSQL 和 Redis。

首次使用时，可以先基于示例文件准备本地 Compose 配置：

```bash
cp .env.compose.example .env.compose
cp app.env.example app.env
cp db.env.example db.env
```

其中：

- `.env.compose` 负责 Compose 自己的参数，例如宿主机暴露端口
- `.env.compose` 也可以配置构建阶段使用的 Go module proxy，例如 `GOPROXY`
- `app.env` 负责应用容器运行时环境变量
- `db.env` 负责 PostgreSQL 容器运行时环境变量
- `nginx` 作为对外入口，负责把宿主机请求转发到内部 `app` 服务
- `redis` 负责登录失败限流、公开文章列表缓存、阅读计数原型和缓存击穿保护的运行态数据

然后启动整套服务：

```bash
docker compose --env-file .env.compose up --build
```

如果希望后台运行：

```bash
docker compose --env-file .env.compose up --build -d
```

Compose 环境默认也会暴露：

```text
http://127.0.0.1:8080
```

如果本机 `8080` 已被占用，可以只修改 `.env.compose` 中的 `APP_PORT`，例如：

```text
APP_PORT=18080
```

这时宿主机访问地址会变成：

```text
http://127.0.0.1:18080
```

常用命令：

```bash
docker compose ps
docker compose logs nginx --tail=50
docker compose logs app --tail=50
docker compose logs db --tail=50
docker compose logs redis --tail=50
docker compose down
docker compose down -v
```

应用还提供了一个最小健康检查接口：

```bash
curl http://127.0.0.1:8080/healthz
```

如果应用和数据库都可用，返回：

```text
ok
```

当前 `/healthz` 检查应用和 PostgreSQL；Redis 可通过 Compose healthcheck 和登录限流/公开列表 smoke 验证覆盖。

当前 Compose 链路是：

```text
host -> nginx -> app -> PostgreSQL
                 -> Redis
```

其中：

- `nginx` 负责对外暴露端口
- `app` 只在 Compose 内部网络中提供 `8080`
- `db` 只在 Compose 内部网络中提供 PostgreSQL 服务
- `redis` 只在 Compose 内部网络中提供 Redis 服务

部署或更新后的最小 smoke 验证应覆盖：

- `/healthz`
- 公开文章列表
- 注册
- 登录
- 创建文章
- 发布文章
- 再次查询公开文章列表，确认新发布文章可见
- 访问公开文章详情，并验证 Redis `article:views:<article_id>` 阅读计数会递增

## Test

```bash
go test ./...
```

## API Overview

### Auth

- `POST /register`
- `POST /login`

### Articles

- `GET /articles`
- `POST /articles`
- `GET /articles/{id}`
- `POST /articles/publish`
- `GET /me/articles`
- `PUT /me/articles/{id}`

其中需要登录态的作者侧接口包括：

- `POST /articles`
- `POST /articles/publish`
- `GET /me/articles`
- `PUT /me/articles/{id}`

这些接口需要携带：

```text
Authorization: Bearer <token>
```

## Current Status

当前项目已经完成第一版 MVP 闭环，进入 Post-MVP Alpha 阶段。下一阶段重点是部署成熟化、Redis 运行边界、并发一致性和最小 Web 验收。

当前已经支持：

- 基于环境变量的启动配置
- `nginx -> app -> PostgreSQL/Redis` 的 Compose 运行链路
- 应用健康检查 `/healthz`
- PostgreSQL 数据持久化
- Redis 登录失败 email/IP 限流
- 公开文章列表 Redis 缓存和 singleflight 防击穿
- 文章详情访问 Redis 阅读计数原型
- 登录限流 `Retry-After` 响应和可配置阈值

Alpha 阶段后续优先补充：

- README、部署文档和 smoke 验证脚本继续对齐
- Redis 场景的集成验证和失败边界说明
- 文章发布/编辑并发安全加固
- PostgreSQL / Redis 可复现集成验证
- 最小 Web 前端验收界面

## Project Guidance

- 仓库级长期规则见 [`AGENTS.md`](./AGENTS.md)。
- 架构、部署和 Redis 解释性上下文见 [`docs/`](./docs)。
- AGENT 使用的任务草稿见 [`docs/issues/`](./docs/issues)。
- GitHub 新建 Issue 模板见 [`.github/ISSUE_TEMPLATE/`](./.github/ISSUE_TEMPLATE)。
