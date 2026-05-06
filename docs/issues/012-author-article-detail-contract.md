# Issue: 补齐作者侧文章详情与草稿编辑契约

## Status

已完成。当前任务补齐了最小 Web 前端验收后暴露出的第一个 API 缺口：作者侧现在可以通过后端稳定读取自己的草稿详情，前端编辑 draft 时不再依赖浏览器本地缓存正文作为事实来源。

本任务已新增 `GET /me/articles/{id}`，并让前端编辑链路以 PostgreSQL 返回的文章详情为事实来源。

## Background

`web/` 最小前端已经覆盖注册、登录、公开文章、作者文章、创建、编辑、发布和删除的主流程。

实现过程中发现一个明确边界问题：

- `GET /articles/{id}` 是公开详情接口，只允许读取 `published` 文章。
- `GET /me/articles` 只能返回作者文章摘要，不返回 `content`。
- 实现前没有 `GET /me/articles/{id}`，作者无法通过 API 读取自己的 draft 正文。
- 前端为了完成最小验收，曾只能用 localStorage 保存本浏览器创建或编辑过的草稿正文。

localStorage 可以保留为表单草稿保护，但不应长期承担文章正文事实来源。本阶段应把作者侧详情能力补到后端，再调整前端编辑链路。

## Goals

- 新增作者侧文章详情接口：`GET /me/articles/{id}`。
- 作者可以读取自己未删除的 `draft` 和 `published` 文章详情。
- 非作者不能读取文章详情。
- 不存在或已逻辑删除文章返回 not found 语义。
- 公开详情接口 `GET /articles/{id}` 继续只返回已发布文章。
- 作者侧详情读取不增加公开阅读计数。
- 前端编辑 draft 时从 `GET /me/articles/{id}` 加载正文，不再把 localStorage 当作业务事实来源。
- 保持后端权限、状态和删除规则在 service 层收口，前端只做交互保护和错误展示。

## Invariants

- PostgreSQL 是文章标题、正文、作者归属、发布状态和删除状态的事实来源。
- `GET /articles/{id}` 是公开阅读接口，只暴露 `published` 且未删除文章。
- `GET /me/articles/{id}` 是作者侧详情接口，必须登录。
- 只有作者能读取自己的作者侧文章详情。
- 作者侧详情允许读取 `draft` 和 `published`，但不允许读取已删除文章。
- 作者侧详情读取不应触发 Redis 阅读计数或公开文章列表缓存变更。
- 前端不能复制后端权限、状态流转和一致性规则作为事实来源。

## Scope

### Backend

- 调整 `internal/service/article_service.go`
  - 在 service interface/实现中新增作者侧详情用例，例如 `GetMyArticle(ctx, articleID, currentUserID)`。
  - 复用 repository 的未删除文章查询能力。
  - service 负责解释 not found、permission denied 等业务语义。
- 调整 `internal/handler/article_handler.go`
  - 在 `articleService` interface 中加入作者侧详情方法。
  - 新增 `GetMyArticle` handler，解析 `/me/articles/{id}` 并返回 detail response。
- 调整 `cmd/server/main.go`
  - 让 `/me/articles/{id}` 同时支持 `GET`、`PUT`、`DELETE`。
- 调整 `internal/handler/article_errors.go`
  - 如现有错误映射已覆盖，则保持稳定；必要时补测试确认。
- 视实现需要检查 `internal/repository/article_repository.go`
  - 如果现有 `GetByID` 足够表达“未删除文章详情”，不新增 repository 方法。
  - 如果新增 repository 方法，repository 只表达数据访问条件，不承载业务语义解释。

### Frontend

- 调整 `web/src/api.ts`
  - 新增 `getMyArticle(token, id)`，请求 `GET /me/articles/{id}`。
- 调整 `web/src/App.tsx`
  - 编辑作者文章时优先从后端加载详情。
  - localStorage 只保留为未保存表单草稿保护或临时降级提示，不再作为业务事实来源。
- 视实现需要调整 `web/src/types.ts`。

### Tests And Docs

- 补充 `internal/service/article_service_test.go`。
- 补充 `internal/handler/article_handler_test.go`。
- 必要时补充 `internal/repository/article_repository_integration_test.go`。
- 更新 `README.md` 的 API 或前端说明。
- 必要时更新 `docs/architecture.md` 的前后端边界说明。
- 更新 `ROADMAP.md` 和 `docs/issues/README.md`。

## Proposed API Contract

```text
GET /me/articles/{id}
Authorization: Bearer <token>
```

成功响应使用现有文章详情结构：

```json
{
  "id": 1,
  "author_id": 10,
  "title": "draft title",
  "content": "draft content",
  "state": "draft",
  "created_at": "2026-05-06T00:00:00Z",
  "updated_at": "2026-05-06T00:00:00Z"
}
```

错误语义：

- 未登录或 token 无效：`401 Unauthorized`
- 路径 id 非法：`400 Bad Request`
- 非作者访问：`403 Forbidden`
- 文章不存在或已删除：`404 Not Found`
- 未映射内部错误：`500 Internal Server Error`

## Proposed Backend Design

service 层新增作者侧详情用例：

```go
func (s *ArticleService) GetMyArticle(ctx context.Context, articleID, currentUserID int64) (model.Article, error) {
    article, err := s.articleRepo.GetByID(ctx, articleID)
    if errors.Is(err, sql.ErrNoRows) {
        return model.Article{}, ErrArticleNotFound
    }
    if err != nil {
        return model.Article{}, err
    }
    if article.AuthorID != currentUserID {
        return model.Article{}, ErrPermissionDenied
    }

    return article, nil
}
```

说明：

- `repository.GetByID` 当前已经带 `deleted_at IS NULL` 条件，可隐藏已逻辑删除文章。
- 作者侧详情不检查 `state == published`，因为作者需要读取 draft。
- 作者侧详情不调用 `incrementArticleViewCount`。
- 公开详情 `GetArticle` 的行为不改变。

## Proposed Frontend Design

- 点击 draft 的“编辑”时，调用 `getMyArticle(token, id)` 获取正文。
- 点击 published 的“详情”仍可使用公开详情接口，或视 UI 需要使用作者详情接口；不要改变公开详情语义。
- localStorage 可以继续保存当前编辑表单的未提交内容，但页面加载时应优先相信后端返回。
- 如果 `GET /me/articles/{id}` 返回 `403` / `404`，前端展示现有状态码映射提示并刷新作者文章列表。

## Tasks

- [x] 阅读 `AGENTS.md`、`README.md`、`docs/architecture.md`、`internal/service/article_service.go`、`internal/handler/article_handler.go`、`cmd/server/main.go`、`web/src/api.ts` 和 `web/src/App.tsx`。
- [x] 在 service 层新增作者侧文章详情用例。
- [x] 在 handler 层新增 `GET /me/articles/{id}` 处理逻辑。
- [x] 在 `cmd/server/main.go` 中把 `/me/articles/{id}` 路由扩展为 `GET` / `PUT` / `DELETE`。
- [x] 补充 service 测试：作者读取 draft、作者读取 published、非作者 forbidden、不存在 not found、已删除 not found。
- [x] 补充 handler 测试：method、非法 id、未登录、成功响应、service 错误映射。
- [x] 更新前端 API client，增加 `getMyArticle`。
- [x] 调整前端编辑链路，从作者详情接口加载正文。
- [x] 移除或降级历史 draft 依赖 localStorage 正文的提示逻辑。
- [x] 更新 README 或 docs 中的 API/前端说明。
- [x] 运行 `gofmt -l .`、`git diff --check` 和 `go test ./...`。
- [x] 运行 `cd web && npm run lint` 和 `cd web && npm run build`。
- [ ] 如本地后端可用，完成一次前端手工验收：创建 draft -> 刷新页面 -> 从“我的文章”进入编辑 -> 正文来自后端。

## Verification

- `gofmt -l .`
- `git diff --check`
- `go test ./...`
- `cd web && npm run lint`
- `cd web && npm run build`

## Notes From Implementation

- 后端新增 `ArticleService.GetMyArticle`，复用 repository 的 `GetByID` 查询。由于 `GetByID` 已带 `deleted_at IS NULL`，已删除文章会统一解释为 not found。
- `GET /me/articles/{id}` 返回现有 `ArticleDetailResponse`，允许作者读取自己的 `draft` 和 `published`。
- 作者侧详情不会调用阅读计数器，也不会触碰公开文章列表缓存。
- 前端新增 `getMyArticle`，点击 draft 编辑时先从后端加载详情正文。
- 旧的 localStorage draft content 辅助逻辑已移除，避免把浏览器缓存误当作文章正文事实来源。
- 本次未扩展全局 JSON 错误响应体；前端仍按现有 HTTP 状态码展示错误。

## Non Goals

- 不新增文章状态。
- 不实现下架、恢复、审核、版本历史或富文本编辑器。
- 不改变 `GET /articles/{id}` 的公开文章语义。
- 不让作者侧详情触发阅读计数。
- 不引入 React Router、状态管理库、UI 组件库或端到端测试框架。
- 不做全局错误响应体改造；本阶段只稳定当前接口的 HTTP 状态语义。
- 不把前端接入 Docker Compose / nginx 静态部署。

## Acceptance Criteria

- `GET /me/articles/{id}` 需要登录。
- 作者可以读取自己的 draft 文章详情，响应包含 `content`。
- 作者可以读取自己的 published 文章详情。
- 非作者读取返回 `403 Forbidden`。
- 不存在或已逻辑删除文章返回 `404 Not Found`。
- 公开详情 `GET /articles/{id}` 仍然不返回 draft。
- 作者侧详情读取不增加 Redis 阅读计数。
- 前端编辑 draft 时从后端加载正文，刷新页面后仍可编辑历史草稿。
- `gofmt -l .` 无输出。
- `git diff --check` 通过。
- `go test ./...` 通过。
- `cd web && npm run lint` 通过。
- `cd web && npm run build` 通过。
- README 或 docs 已说明新增作者侧详情接口或前端编辑链路变化。
- 收口前检查 `README.md`、`docs/architecture.md`、`ROADMAP.md`、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 是否因本任务变旧。

## AI Agent Notes

- 优先保持现有分层：repository 只给数据访问能力，service 解释权限和状态语义，handler 负责 HTTP 输入输出和错误映射。
- 当前 `internal/repository/article_repository.go` 的 `GetByID` 已包含 `deleted_at IS NULL`，大概率可以直接复用。
- 不要为了作者侧详情修改公开详情接口；公开详情仍承担阅读计数入口。
- 实现前 `cmd/server/main.go` 的 `/me/articles/{id}` 只分发 `PUT` 和 `DELETE`，本任务已补 `GET`。
- 实现前 `web/src/App.tsx` 存在 localStorage draft content 辅助逻辑，本任务已移除这条业务事实来源。
- 如果发现当前错误响应只有状态码而没有 JSON body，不要在本 issue 中扩大成全局错误码系统；可另开后续 issue。
