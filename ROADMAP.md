# Content Backend Roadmap

本路线图用于描述 Post-MVP Alpha 阶段的公开推进方向。具体任务草稿放在 `docs/issues/`，GitHub 新建 Issue 时使用 `.github/ISSUE_TEMPLATE/`。

## Current Baseline

项目已经完成第一版内容发布后端 MVP，并具备以下能力：

- 用户注册、登录、JWT 认证中间件
- 创建文章草稿、编辑草稿、发布文章
- 查看公开文章、查看公开文章详情、查看我的文章
- 环境变量配置、Dockerfile、Docker Compose、nginx 反向代理
- PostgreSQL 数据持久化
- Redis 登录失败 email/IP 限流、公开文章列表缓存、singleflight 防击穿
- Redis 文章阅读计数原型、登录用户阅读去重原型和 Lua 原子化去重计数
- 反向代理后的可信客户端 IP 解析，用于 IP 登录限流
- `Retry-After` 登录限流响应和可配置限流阈值
- `gofmt` + `go test ./...` 最小 CI

## Alpha Direction

Alpha 阶段目标不是继续堆接口，而是把项目推进到更稳定的工程化阶段：

1. 成熟化部署：让项目能被稳定构建、启动、更新、排障和验证。
2. 深化 Redis：围绕真实业务场景理解 key、TTL、原子操作、缓存一致性和失败降级。
3. 学习并发场景：围绕内容发布平台常见的状态流转、重复请求、缓存击穿和读写竞争做专题改造。
4. 最小 Web 验收：阶段末补一个可用前端，用来验证后端成果和前后端协作。

## Suggested Order

```text
Alpha 开发前文档与 agent instructions 收口
-> Redis 原子计数器原型
-> Redis 运行验证和边界测试
-> Redis 登录用户阅读去重原型
-> Redis 阅读计数原子化与验证收口
-> 部署文档与 smoke 验证对齐
-> 文章发布/编辑并发安全改造
-> PostgreSQL / Redis 集成验证
-> 最小 Web 前端验收
```

收口提交后，后续开发按学习节奏生成或调整小 issue 推进。

## Issue Drafts

- `docs/issues/000-Tie-up-loose-ends-before-next-phase.md`
- `docs/issues/000-next-phase-roadmap.md`
- `docs/issues/001-redis-atomic-counter-integration`
- `docs/issues/002-redis-counter-runtime-verification.md`
- `docs/issues/003-redis-counter-learning-notes.md`
- `docs/issues/004-redis-authenticated-view-dedup.md`
- `docs/issues/005-redis-view-counter-atomicity-tie-up.md`

## Issue Maintenance

每个 issue 完成后，需要检查：

- `AGENTS.md` 是否需要沉淀新的长期规则。
- `README.md`、`docs/`、`ROADMAP.md` 是否仍匹配代码和部署链路。
- `docs/issues/` 是否需要新增、关闭、改名或拆分任务草稿。
- `.github/ISSUE_TEMPLATE/` 是否需要同步模板，让用户新建 issue 时仍有清晰入口。

## Non Goals For This Phase

- 不做 Kubernetes、蓝绿/灰度发布或复杂自动发布平台
- 不做完整风控系统、推荐系统、搜索系统或富文本编辑器
- 不把前端作为深入学习重点
- 不在部署和运行能力稳定前扩展大量新业务接口
