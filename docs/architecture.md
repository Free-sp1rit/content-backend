# Architecture

本文档记录当前项目架构边界，供开发、Review 和后续任务拆分参考。当前阶段为 Post-MVP Alpha，重点是部署成熟化、Redis 运行边界、并发一致性和最小 Web 验收。

## Runtime Shape

当前运行链路：

```text
client -> nginx -> app -> PostgreSQL
                       -> Redis
```

- `nginx`：对外入口，转发请求到应用容器。
- `app`：Go HTTP 服务，负责认证、文章用例、缓存和限流编排。
- `PostgreSQL`：核心事实来源，保存用户和文章数据。
- `Redis`：运行态能力，当前用于登录失败限流、公开文章列表缓存和防击穿辅助。

## Code Layers

- `cmd/server`：应用入口，负责配置加载、依赖装配、路由注册和启动。
- `internal/config`：环境变量解析和默认值管理。
- `internal/handler`：HTTP 输入输出、基础输入校验和错误映射。
- `internal/middleware`：认证拦截和请求上下文注入。
- `internal/service`：业务流程、权限判断、状态规则、缓存和限流语义。
- `internal/repository`：数据库访问，不承载业务语义解释。
- `internal/auth`：token 签发与校验。
- `internal/model`：核心数据结构和业务常量。

## Design Rules

- PostgreSQL 是核心业务数据的事实来源。
- Redis 只保存适合短期运行态的数据，例如缓存、计数器、限流状态。
- 业务规则优先放在 `service`，HTTP 细节优先放在 `handler`。
- 数据库事务、条件更新和唯一约束可以放在 repository 能力中，但是否使用这些能力由 service 用例决定。
- 新增接口时要先明确业务动作，不把多个职责混进一个入口。
- agent instructions 和 issue 草稿属于工程协作资产；长期规则进入 `AGENTS.md`，解释性上下文进入 `docs/`，单次任务进入 `docs/issues/`。

## Current Risk Areas

- 文章发布和编辑目前存在典型的“先查再改”形态，后续需要针对并发读写补强。
- Redis 缓存和限流已接入，并已记录当前 key、TTL、失效和失败降级策略；后续需要补真实 Redis/PostgreSQL 集成验证。
- 部署链路已可用，README 和部署文档应继续随 Compose、环境变量和 smoke 验证同步维护。
- 当前还缺少可复用 smoke 脚本，部署验证主要依赖文档 checklist。
