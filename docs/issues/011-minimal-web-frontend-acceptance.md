# Issue: 增加最小 Web 前端验收界面

## Status

已完成。当前任务已在仓库内 `web/` 子项目中实现一个 Vite + React + TypeScript 最小前端，用来从浏览器侧验收现有内容发布后端主链路。

## Background

项目后端已经完成第一版 MVP，并在 Post-MVP Alpha 阶段补齐了多项工程化能力：

- 文章发布/编辑状态一致性
- 文章逻辑删除
- PostgreSQL migration 与 repository 真实 SQL 验证
- Redis 公开列表缓存一致性验证
- Docker Compose smoke 验证

当前已经准备好前端环境：

```text
web/
```

技术栈选择：

```text
Vite + React + TypeScript
```

本阶段目标不是做完整产品 UI，而是用一个可运行、可构建、可手工验收的最小 Web 界面验证后端 API 是否足够顺畅。

## Goals

- 在 `web/` 中保留并整理 Vite + React + TypeScript 前端子项目。
- 通过浏览器完成注册、登录、查看公开文章、查看详情、创建草稿、编辑草稿、发布文章、删除文章。
- 使用 Vite dev server proxy 访问后端，避免第一版就引入 CORS 改造。
- 前端只消费公开 API，不复制后端权限和状态一致性规则。
- 提供清晰的本地运行和构建命令。
- 让最小 Web 页面成为后端阶段成果的人工验收入口。

## Decisions

- 前端放在当前仓库 `web/` 子目录，不新开独立仓库。
- 第一版使用 Vite dev server 运行，不接入 Docker Compose / nginx 静态部署。
- 第一版不引入 UI 组件库、路由库或复杂状态管理库。
- 第一版使用 React state / context 管理登录态和当前视图。
- 第一版 token 可保存在 `localStorage`，用于个人学习项目的最小验收；不扩展 refresh token 或会话管理。
- 第一版通过 Vite proxy 转发 API 请求到后端，例如 `/register`、`/login`、`/articles`、`/me/articles`。
- 第一版重点是功能验收和接口体验，不追求完整视觉设计系统。

## Scope

- 整理并提交 `web/` Vite + React + TypeScript 子项目。
- 更新 `web/vite.config.ts`，增加后端 API proxy。
- 替换默认 Vite 示例页面，构建最小内容发布验收界面。
- 新增或整理前端 API client，例如 `web/src/api.ts`。
- 新增或整理前端类型定义，例如 `web/src/types.ts`。
- 实现基础登录态：
  - 注册
  - 登录
  - 保存 token
  - 注销
  - 认证请求自动携带 `Authorization: Bearer <token>`
- 实现公开侧视图：
  - 公开文章列表
  - 公开文章详情
  - 公开详情允许匿名访问
  - 显示基础加载、空列表和错误状态
- 实现作者侧视图：
  - 查看我的文章
  - 创建草稿
  - 编辑 draft 文章
  - 发布 draft 文章
  - 删除文章
  - 对 published 文章禁用或隐藏编辑动作
- 更新 `README.md`，补充前端本地开发、构建和后端启动要求。
- 必要时更新 `docs/architecture.md`，说明前端只消费公开 API，不承载后端业务规则。
- 必要时更新 `ROADMAP.md`、`docs/issues/README.md` 和本 issue 状态。

## User Flows

### Anonymous visitor

- 打开前端首页。
- 查看公开文章列表。
- 点击文章查看公开详情。
- 未登录时看不到作者侧操作入口，或作者侧操作入口提示需要登录。

### Author

- 注册账号。
- 登录账号。
- 创建文章草稿。
- 在“我的文章”中看到草稿。
- 编辑草稿。
- 发布草稿。
- 在公开文章列表中看到已发布文章。
- 查看公开文章详情。
- 删除文章。
- 删除后公开列表和公开详情不再显示该文章。

### Error and boundary behavior

- 登录失败显示错误。
- 未登录执行作者侧动作时提示需要登录。
- 发布已发布文章或编辑已发布文章时，前端能展示后端返回的冲突错误，或在 UI 上避免发起明显无效动作。
- 删除后再次访问详情显示 not found 类提示。

## Proposed UI Shape

第一版建议使用单页应用，不引入路由库：

- 顶部区域：应用标题、登录状态、登录/注销入口。
- 左侧或顶部 tabs：
  - 公开文章
  - 我的文章
  - 新建文章
- 主区域：
  - 公开列表和详情
  - 我的文章列表
  - 文章编辑表单
- 错误和加载状态保持轻量、明确。

视觉风格建议：

- 偏后台/内容管理工具，不做营销落地页。
- 信息密度适中，适合反复执行验收动作。
- 不使用大面积装饰性 hero、渐变背景或复杂动效。

## API Coverage

前端至少覆盖这些接口：

```text
POST /register
POST /login
GET /articles
GET /articles/{id}
POST /articles
POST /articles/publish
GET /me/articles
PUT /me/articles/{id}
DELETE /me/articles/{id}
```

认证接口需要：

```text
Authorization: Bearer <token>
```

## Tasks

- [x] 阅读 `AGENTS.md`、`README.md`、`docs/architecture.md`、`docs/deployment.md`、`internal/handler/article_handler.go`、`internal/handler/article_responses.go`、`internal/handler/auth_handler.go` 和 `web/package.json`。
- [x] 确认 `web/` Vite + React + TypeScript scaffold 可提交，且 `node_modules`、`dist` 被忽略。
- [x] 配置 `web/vite.config.ts` 的 API proxy，默认转发到 `http://127.0.0.1:8080`。
- [x] 新增前端 API client 和类型定义。
- [x] 实现注册、登录、注销和 token 保存。
- [x] 实现公开文章列表和详情。
- [x] 实现我的文章列表。
- [x] 实现创建草稿、编辑草稿、发布文章、删除文章。
- [x] 实现基础加载态、空状态和错误提示。
- [x] 更新 `README.md` 的前端开发说明。
- [x] 必要时更新 `docs/architecture.md` 和 `ROADMAP.md`。
- [x] 运行后端验证：`gofmt -l .`、`go test ./...`。
- [x] 运行前端验证：`cd web && npm run lint && npm run build`。
- [x] 如本地后端可用，启动 Vite dev server 并完成一次手工浏览器验收。
- [x] 收口前检查 `docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否需要同步。

## Verification

- `gofmt -l .`
- `go test ./...`
- `cd web && npm run lint`
- `cd web && npm run build`
- `cd web && npm run dev -- --host 127.0.0.1`
- `scripts/smoke.sh http://127.0.0.1:5173`

本地还通过 Vite dev server proxy 完成了最小链路验收：

- 注册用户。
- 登录并保存 token。
- 创建草稿。
- 查看我的文章。
- 编辑 draft 草稿。
- 发布文章。
- 在公开列表和公开详情中查看已发布文章。
- 删除文章。
- 删除后公开详情返回 not found 类提示。

## Notes From Implementation

- 当前后端没有作者侧草稿详情接口，`GET /articles/{id}` 会隐藏 draft；前端第一版用 localStorage 保存本浏览器创建/编辑过的草稿正文，辅助完成最小编辑验收。
- 对于历史草稿，前端只能从 `GET /me/articles` 取得标题和状态，编辑时会提示需要重新填写正文。
- 这个限制不阻塞最小 Web 验收，但后续如果要产品化作者后台，建议增加作者侧文章详情接口，例如 `GET /me/articles/{id}`。

## Non Goals

- 不新增后端 API。
- 不修改文章状态模型。
- 不实现下架、恢复、审核、评论、点赞、搜索或富文本编辑器。
- 不引入 Redux、Zustand、React Router、UI 组件库或 CSS 框架，除非实现中发现明显必要。
- 不把前端接入 Docker Compose / nginx 生产静态部署。
- 不做 SSR、SEO、权限系统或复杂会话刷新。
- 不把后端权限判断、状态流转和一致性规则复制到前端。

## Acceptance Criteria

- `web/` 子项目可以被提交，且不包含 `node_modules` 或 `dist`。
- `gofmt -l .` 无输出。
- `go test ./...` 通过。
- `cd web && npm run lint` 通过。
- `cd web && npm run build` 通过。
- `cd web && npm run dev -- --host 127.0.0.1` 可以启动。
- 浏览器中可以注册、登录、注销。
- 登录后可以创建草稿、查看我的文章、编辑 draft、发布文章、删除文章。
- 公开列表可以看到已发布文章。
- 公开详情可以查看已发布文章。
- 删除文章后公开列表和公开详情不再显示该文章。
- 未登录或 token 缺失时，作者侧操作有清晰提示。
- README 有前端启动、构建和后端依赖说明。
- 收口前检查 `README.md`、`docs/architecture.md`、`ROADMAP.md`、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否因本任务变旧。

## AI Agent Notes

- 当前用户已经在本地准备了 `web/` 环境，验证过：
  - `node -v` -> `v22.22.1`
  - `npm -v` -> `11.13.0`
  - `cd web && npm run build`
  - `cd web && npm run lint`
  - `cd web && npm run dev -- --host 127.0.0.1`
- `web/node_modules/` 和 `web/dist/` 不应提交。
- 前端请求建议使用相对路径，例如 `fetch("/articles")`，由 Vite proxy 转发到后端。
- 不要为了前端第一版改后端 CORS；优先使用 Vite proxy。
- 前端只做 UI 可用性保护，不把后端权限和状态规则复制成事实来源。
- 如果发现后端 API 对前端明显不友好，先记录问题；除非阻塞最小验收，不在本 issue 中扩大后端改造。
- 如果需要测试真实链路，先确认后端服务或 Compose 已经在 `http://127.0.0.1:8080` 可访问。
