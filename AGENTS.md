# Content Backend Project Guidance

本文件用于仓库级共享 guidance，只放适合随项目一起维护、适合提交到远程仓库的公共规则。

## Project Scope

这是一个使用 Go 实现的内容发布后端项目。

当前仓库协作应服务于以下长期目标：

- 保持接口行为清晰、稳定、可验证
- 保持 `repository` / `service` / `handler` / `main` 分层边界清楚
- 让核心内容发布链路能在真实运行环境中构建、启动、访问和排障
- 在 Redis、并发控制、部署能力进入项目时，优先保证行为正确和边界明确

## Tech Stack And Runtime Shape

- Go 版本以 `go.mod` 为准，当前为 Go 1.22.5
- HTTP 层使用标准库 `net/http` 和 `http.ServeMux`，不要在没有明确收益时引入 Web 框架
- PostgreSQL 是核心事实来源，Go 侧通过 `database/sql` + `github.com/lib/pq` 访问
- Redis 使用 `github.com/redis/go-redis/v9`，测试可使用 `github.com/go-redis/redismock/v9`
- 密码哈希使用 `golang.org/x/crypto`，缓存击穿合并使用 `golang.org/x/sync/singleflight`
- Compose 运行形态是 `host -> nginx -> app -> PostgreSQL/Redis`，其中 nginx 是宿主机入口，app 在内部网络监听 `8080`

## Current Phase Direction

项目已经越过第一版 MVP 闭环，进入 Post-MVP Alpha 阶段。当前公共工程方向是：

- 成熟化部署：让 Docker Compose、环境变量示例、健康检查、README 和 smoke 验证保持一致
- Redis 系统能力：围绕缓存、限流、会话或运行保护等真实场景接入，不做脱离业务的空连接
- 常见并发场景：优先处理状态流转、缓存一致性、重复请求、计数器和读写竞争
- 最小 Web 产品：在后端与运行能力稳定后再补，用于验收成果和前后端协作，不反向扩大后端范围

近期优先级按 `ROADMAP.md` 和 `docs/issues/` 推进：部署文档与 smoke 验证对齐、文章发布/编辑并发安全加固、PostgreSQL / Redis 集成验证、最小 Web 前端验收。

## Layer Boundaries

- `repository` 负责数据访问，不承载业务语义解释；可以提供事务、条件更新等数据层能力，但业务规则由 `service` 决定
- `service` 负责业务流程、权限判断、状态规则、缓存/限流语义和用例组织
- `handler` 负责 HTTP 输入输出、基础输入校验、认证上下文读取和错误映射
- `middleware` 负责横切请求控制，例如认证拦截、上下文注入
- `main` 负责配置加载、依赖装配、健康检查和应用启动
- 如果加入前端代码，前端只消费公开 API，不复制后端权限、状态流转和一致性规则

## Current API And Data Model

- 当前公开认证接口是 `POST /register`、`POST /login`
- 当前文章接口是 `GET /articles`、`POST /articles`、`GET /articles/{id}`、`POST /articles/publish`、`GET /me/articles`、`PUT /me/articles/{id}`
- 作者侧接口必须通过 `Authorization: Bearer <token>` 获取登录态；公开文章详情允许匿名访问，但显式携带无效 token 时应返回认证错误
- `users` 和 `articles` 表由 `migrations/001_init.sql` 初始化；文章状态当前只允许 `draft` 和 `published`
- 文章内容、作者归属和发布状态以 PostgreSQL 为准；Redis 阅读计数、缓存、限流和去重标记不应反向成为核心内容事实来源

## Error Boundaries

- 按错误来源分层，不按严重程度分层
- `handler` 现场处理 HTTP 输入错误，例如 method 不匹配、path 参数非法、JSON decode 失败和基础输入校验失败
- `middleware` 处理认证中间件错误，例如未登录、token 无效、token 过期和认证头缺失
- `handler` 通过 `statusFrom<X>ServiceError` / `write<X>ServiceError` 映射 service 返回的错误
- 未知且未映射的 service error 统一兜底为 `500`
- 不把 method 错误、decode 错误和 middleware 错误塞进 service error 映射函数

## Redis And Runtime State

- Redis 具体实现应通过接口接入业务代码，避免 `service` 主流程直接散落 Redis 命令细节
- 每个 Redis 使用场景都要明确 key 设计、TTL、失效策略、失败策略和测试方式
- 限流、计数器、锁或防击穿逻辑应优先使用 Redis 原子命令或 Lua 脚本
- 缓存应以正确性优先于命中率；状态变更后必须考虑失效、重建和并发读写边界
- Redis 不应替代 PostgreSQL 作为核心事实来源，除非该数据本身就是短期运行态

## Concurrency And Consistency

- 不把“先查再改”的读写序列默认视为并发安全；状态流转应优先考虑条件更新、事务、唯一约束或锁
- 对可重复请求要明确是否幂等，例如重复发布、重复登录失败、重复创建或重复提交
- 涉及文章发布、编辑、缓存失效、登录限流等链路时，应补充边界测试或最小并发验证
- 设计并发方案时先写清楚要保护的不变量，再选择数据库事务、条件 SQL、Redis 原子操作或应用层协调

## Database And Migration Rules

- `migrations/` 保存数据库初始化和迁移 SQL；修改表结构、索引或约束时必须同步检查 repository、service 测试、README 和部署说明
- 当前 Compose 通过把 `migrations/` 挂载到 `/docker-entrypoint-initdb.d` 初始化新 PostgreSQL volume；已有 volume 不会自动重跑初始化 SQL
- 修改已有数据结构时，要明确新库初始化路径和旧库升级路径，不能只验证空库启动
- 文章状态、唯一约束、外键和索引属于业务不变量的一部分；并发安全改造应优先考虑条件 SQL、事务或数据库约束

## Common Commands

- 本地测试：`gofmt -l .`、`go test ./...`
- 本地启动：按 `.env.example` 手动导出环境变量后执行 `go run ./cmd/server`
- 初始化本地数据库：`psql -U <your_user> -d <your_database> -f migrations/001_init.sql`
- 首次 Compose 配置：从 `.env.compose.example`、`app.env.example`、`db.env.example` 复制出本地真实环境文件
- Compose 启动：`docker compose --env-file .env.compose up --build`，后台运行加 `-d`
- Compose 检查和排障：`docker compose --env-file .env.compose config`、`docker compose ps`、`docker compose logs app --tail=50`、`docker compose logs nginx --tail=50`、`docker compose logs db --tail=50`、`docker compose logs redis --tail=50`
- 健康检查：`curl http://127.0.0.1:8080/healthz`
- 清理 Compose 服务时区分 `docker compose down` 和会删除数据卷的 `docker compose down -v`

## Deployment Rules

- 真实环境文件、密钥、密码、token、服务器 IP 和本机代理配置不得提交
- `.env*.example`、`app.env.example`、`db.env.example`、`docker-compose.yml`、`Dockerfile`、`deploy/` 和 README 必须与实际启动链路同步更新
- 部署相关修改应能在新机器上复现；只属于个人服务器的临时配置不进入仓库
- 健康检查和 smoke 验证应覆盖至少 `/healthz`、注册、登录、创建文章、发布文章、公开查询

## Testing And Verification

- 提交前保持 `gofmt -l .` 无输出，并保持 `go test ./...` 通过
- CI 当前执行格式检查和 `go test ./...`，本地验证口径应与 CI 保持一致
- 修改 service 业务规则时优先补 service 单测；修改 HTTP 行为时补 handler/middleware 测试
- 修改 repository、migration、Docker Compose 或 Redis 真实交互时，优先补可运行的集成验证或清晰的手工验证步骤
- 修改 Docker Compose、Dockerfile、环境变量或 nginx 配置时，至少执行 Compose 配置检查，并按风险补充启动或 smoke 验证
- 修改 Redis 场景时，验收必须说明 key、TTL、原子性、失效策略、失败降级和真实 Redis 验证方式
- 修改公开 API 行为时，验收必须覆盖状态码、响应体、认证要求和错误映射
- README、示例环境变量和部署说明属于可验证交付的一部分，不能长期落后于代码

## Development Priorities

- 优先让核心业务链路、部署链路和运行保护形成闭环
- 修改代码时优先保持现有结构一致性
- 新增逻辑时优先补边界条件和可验证性
- 设计接口时优先区分业务动作，不把不同职责混成一个入口
- 不为了展示效果提前扩展花哨功能；前端只在后端阶段成果需要验收时进入

## Agent Instruction And Issue Maintenance

- `AGENTS.md` 是仓库级长期规则，适合提交；只放跨 issue、跨阶段仍然成立的公共协作规则
- `.codex/*.md` 和 `.notes/*.md` 是本地个人 guidance / 学习笔记，默认不提交；只有用户明确要求把其中内容公共化时，才提炼后迁移到 `AGENTS.md` 或 `docs/`
- `docs/` 放解释性上下文，例如架构、部署、Redis 运行状态；不要把单次任务 checklist 长期堆在解释文档里
- `docs/issues/` 放可复制到 GitHub Issues 的具体任务草稿和 AI Agent Notes，适合提交
- `.github/ISSUE_TEMPLATE/` 放 GitHub 新建 Issue 模板，适合提交；模板默认使用中文，方便与项目 issue 草稿保持一致
- `ROADMAP.md` 放公开阶段路线，`README.md` 放新开发者启动、配置、运行、测试和当前状态说明，二者都必须随实际项目阶段更新
- 每个 issue 收口前，Codex 应主动检查本次改动是否改变了项目阶段、分层规则、部署链路、Redis 场景、并发边界、测试方式、README 或路线图；若有变化，应在同一任务内同步更新相关公共文档，无需等待用户额外提示
- 每个 issue 收口时，Codex 应主动询问用户是否需要同步更新 issues；这里的“更新 issues”指同时检查 `docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否需要新增、删除、改名或改写
- issue 内容只描述单次任务；当某条 issue 经验沉淀为长期规则时，提炼进 `AGENTS.md`，而不是长期留在 issue 中充当规则

## Git Naming Conventions

- 使用 Conventional Commits 规范
- 一条提交只表达一类变更，避免把业务、重构、测试、文档和部署配置混在一起

## Git Workflow

- 开始新任务前先检查当前分支、工作区状态和远程同步状态，确认任务不是建立在过期或混乱的本地状态上
- 默认工作流是：同步 `main` -> 从最新 `main` 创建任务分支 -> 开发并小步提交 -> 本地验证 -> 整理 PR 描述 -> 由用户 push 并创建 PR
- 如果本地 `main` 落后远程，应先回到 `main` 并拉取远程更新，再创建任务分支
- 如果工作区存在未提交改动、未跟踪文件或分支分叉，不直接覆盖、删除、reset 或 checkout 掉这些改动；先判断是否属于当前任务，必要时请用户确认
- 每个任务使用独立分支，分支名应体现任务类型和主题，例如 `docs/align-deployment-docs`、`feat/add-smoke-test`、`fix/article-publish-race`
- 开发过程中按清晰断点提交；一次提交只表达一个意图，避免把业务、测试、文档、部署配置混在同一个提交里
- 提交前尽量只暂存相关文件，并完成与变更风险匹配的验证，例如 `gofmt`、`go test ./...`、Compose 配置检查或手工 smoke 验证
- 如果开发期间需要同步最新 `main`，优先选择团队能接受的方式；不擅自执行需要强推的 rebase/force push 流程
- 收口时必须给出变更摘要、验证结果、风险/限制和建议 PR 描述
- 默认不直接 push 或创建 PR，除非用户明确要求；由用户执行 push 和 PR 创建时，agent 负责提供可直接粘贴的 PR 标题和描述
- PR 描述应包含：改动内容、变更原因、测试方式、风险与影响、备注或后续任务

## Collaboration Notes

- 如果某项规则只属于个人学习节奏、本机习惯或临时实验，不应写在本文件中
- 个人 guidance 放在本地层，例如 `.codex/AGENTS.md`
