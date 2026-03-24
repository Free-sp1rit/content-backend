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
- 标准库 `net/http`
- `github.com/lib/pq`

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

## Requirements

- Go 1.22.5
- PostgreSQL

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
```

## Run

```bash
go run ./cmd/server
```

服务默认监听：

```text
http://127.0.0.1:8080
```

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

当前项目仍处在 MVP 阶段，已经具备核心内容发布链路，并已支持基于环境变量的启动配置。

当前仍未系统补齐：

- 自动化测试
- 更完整的配置工程化能力
- 更细致的错误响应与运行保护能力
