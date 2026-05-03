# Issue: 为 Docker Compose 运行链路补最小 smoke 验证

## Status

实现完成，真实 Compose smoke 待验证。当前已新增 `scripts/smoke.sh`，并同步 README 与部署文档；本环境缺少 `jq` 和 Docker/Compose，暂不能执行完整 smoke。

## Background

当前项目已经具备 `nginx -> app -> PostgreSQL/Redis` 的 Docker Compose 运行链路，并完成注册、登录、创建文章、发布文章、公开查询、Redis 阅读计数和文章状态一致性等核心能力。

现有 README 和部署文档中已经有 smoke checklist，但主要依赖手工步骤。下一阶段需要把这些关键检查收敛成可复用脚本，让新机器、更新部署和本地验证都有稳定入口。

本任务采用“启动、构建、验证分离”的边界：脚本只验证已经运行的服务，不负责构建镜像、启动 Compose 或清理数据卷。

## Goals

- 新增最小可维护 smoke 脚本，验证 Compose 运行链路可用。
- 覆盖核心 HTTP 主链路、PostgreSQL 持久化链路和 Redis 运行态链路。
- 验证刚完成的文章状态一致性行为，例如重复发布和发布后编辑返回 `409`。
- 同步 README 和部署文档，让 smoke 验证成为部署/更新后的标准步骤。

## Tooling Decision

第一版脚本建议使用：

```text
bash + curl + jq + docker compose
```

选择原因：

- `bash` 适合组织轻量流程和失败退出。
- `curl` 是 HTTP smoke 验证的标准工具。
- `jq` 专门解析 JSON，比用 `sed`/正则解析响应更稳，也更符合常见工程实践。
- `docker compose` 用于检查服务状态，必要时通过 `redis` 服务读取 Redis key。

第一版不使用 Python、Go e2e 测试框架、Makefile 或 CI 集成测试，避免验证工具本身变重。

## Scope

- 新增 `scripts/smoke.sh`。
- 脚本支持通过第一个参数传入 `BASE_URL`，默认使用 `http://127.0.0.1:8080`。
- 脚本启动前检查必要依赖：`curl`、`jq`、`docker`。
- 脚本只做验证，不执行 `docker compose up --build`、`docker compose down` 或 `docker compose down -v`。
- 更新 `README.md`，说明 smoke 脚本运行方式和依赖。
- 更新 `docs/deployment.md`，把 smoke 脚本纳入部署/更新后验证步骤。

## Proposed Verification Flow

脚本建议覆盖：

1. `/healthz` 返回 `ok`。
2. 注册临时用户。
3. 登录并取得 JWT。
4. 创建文章草稿。
5. 发布文章。
6. 公开文章列表能看到新发布文章。
7. 公开文章详情可访问。
8. Redis `article:views:<article_id>` 会随详情访问递增。
9. 重复发布同一文章返回 `409 Conflict`。
10. 发布后编辑同一文章返回 `409 Conflict`。

推荐调用方式：

```bash
docker compose --env-file .env.compose up --build -d
scripts/smoke.sh http://127.0.0.1:8080
```

脚本输出应尽量清晰，例如：

```text
[ok] healthz
[ok] register user
[ok] login
[ok] create article
[ok] publish article
[ok] list published articles
[ok] get article detail
[ok] redis view counter
[ok] duplicate publish returns 409
[ok] update published article returns 409
```

## Tasks

- [x] 新增 `scripts/smoke.sh`，实现最小 smoke 流程。
- [x] 使用 `jq` 解析注册、登录、创建文章和公开查询响应。
- [x] 使用唯一 email 和文章标题，避免重复运行时冲突。
- [x] 验证 `/healthz`、注册、登录、创建文章、发布文章、公开列表和公开详情。
- [x] 验证 Redis 阅读计数 key 会递增。
- [x] 验证重复发布返回 `409`。
- [x] 验证发布后编辑返回 `409`。
- [x] 更新 `README.md` 的运行和 smoke 验证说明。
- [x] 更新 `docs/deployment.md` 的部署后验证说明和排障入口。
- [ ] 运行 `shellcheck`（如本机可用）或至少执行一次脚本手工验证。
- [x] 运行 `git diff --check` 和 `go test ./...`。

## Verification

- `bash -n scripts/smoke.sh`
- `scripts/smoke.sh http://127.0.0.1:8080` 依赖检查按预期失败：当前环境缺少 `jq`
- 未运行 `shellcheck`：当前环境未安装 `shellcheck`
- 未运行真实 Compose smoke：当前环境无法访问 Docker/Compose
- 已检查 `.github/ISSUE_TEMPLATE/`，当前模板无需同步修改

## Non Goals

- 不让 smoke 脚本自动启动、构建或关闭 Compose。
- 不引入完整 e2e 测试框架。
- 不把 smoke 验证接入 GitHub Actions。
- 不做复杂并发压测。
- 不做生产部署自动化、备份恢复或回滚策略。
- 不新增业务接口。
- 不做前端页面。

## Acceptance Criteria

- `scripts/smoke.sh http://127.0.0.1:8080` 能在已启动 Compose 环境中跑通。
- 脚本缺少依赖或某一步失败时，能给出清晰错误并非零退出。
- smoke 覆盖 `/healthz`、注册、登录、创建文章、发布文章、公开列表、公开详情。
- smoke 覆盖 Redis 阅读计数递增。
- smoke 覆盖重复发布返回 `409`。
- smoke 覆盖发布后编辑返回 `409`。
- README 和 `docs/deployment.md` 与脚本使用方式一致。
- `git diff --check` 通过。
- `go test ./...` 通过。

## AI Agent Notes

- 优先读取 `AGENTS.md`、`README.md`、`docs/deployment.md`、`docs/redis.md`、`docker-compose.yml` 和 `docs/issues/007-compose-smoke-verification.md`。
- 保持启动、构建、验证分离：脚本不负责 `docker compose up --build`。
- 第一版脚本依赖 `jq` 是有意选择，不要退回用 `sed` 解析 JSON。
- Redis key 验证可以通过 `docker compose --env-file .env.compose exec -T redis redis-cli GET "article:views:<article_id>"`。
- 如果本机没有正在运行的 Compose 环境，先说明未能完成真实 smoke，并给出手工验证命令；不要伪造通过结果。
- 收口时检查 `README.md`、`docs/deployment.md`、`ROADMAP.md`、`docs/issues/README.md` 和 `.github/ISSUE_TEMPLATE/` 是否需要同步。
