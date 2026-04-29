# Issue: Post-MVP Alpha 工程化推进

## User Facing Issue

### Background

当前项目已完成第一版内容发布后端 MVP，并继续推进了一部分运行能力：

- 注册、登录、JWT 认证中间件
- 创建、编辑、发布、公开查询、作者侧文章列表
- `go test ./...` 单元测试基线和最小 CI
- Docker Compose 自托管链路：`nginx -> app -> PostgreSQL/Redis`
- Redis 已接入：登录失败 email/IP 限流、公开文章列表缓存、singleflight 防击穿
- 登录限流已支持可信代理后的客户端 IP、可配置阈值和 `Retry-After`

Post-MVP Alpha 阶段重点不再是继续堆接口，而是把项目推进到“可部署、可维护、能解释常见系统设计取舍，并最终可通过最小 Web 前端验收”的状态。

### Goals

- 成熟化部署：补齐自托管运行、更新、排障和 smoke 验证能力。
- 深化 Redis：明确 key 设计、TTL、失效策略、失败降级和适用边界。
- 落地并发场景：围绕内容发布平台常见的状态流转、缓存一致性、重复请求和读写竞争做专题改造。
- 阶段末补最小 Web 前端：用于验收后端成果和学习前后端协作，前端实现不作为深入学习重点。

### Task Breakdown

- [x] 完成 Alpha 开发前文档与 agent instructions 收口。
- [ ] 补充部署 runbook 与环境验证说明。
- [ ] 补充 Redis 运行验证和边界测试。
- [ ] 增加部署 smoke 验证清单或脚本。
- [ ] 加固文章发布/编辑并发场景。
- [ ] 阶段末增加最小 Web 前端验收界面。

### Non Goals

- 不做 Kubernetes、蓝绿/灰度发布或复杂自动发布平台。
- 不做完整风控系统、推荐系统、搜索系统或富文本编辑器。
- 不把前端做成学习重点；前端只服务于阶段成果验收。
- 不在部署和运行能力稳定前扩展大量新业务接口。

### Acceptance Criteria

- `gofmt` 和 `go test ./...` 稳定通过，CI 保持绿色。
- 新机器可根据 README 和示例环境变量启动完整 Compose 链路。
- smoke 验证能证明 `client -> nginx -> app -> PostgreSQL/Redis` 主链路可用。
- Redis 场景能说明为什么适合 Redis、key 如何设计、TTL/失效如何处理、Redis 不可用时系统如何表现。
- 至少一个并发场景有清晰的不变量、方案说明、测试或手工验证记录。
- 最小 Web 前端能够跑通注册/登录/文章发布/公开查看闭环。

### Suggested Labels

- `roadmap`
- `backend`
- `deployment`
- `redis`
- `learning`

## AI Agent Notes

- 这个 issue 适合作为 umbrella issue，不建议一次性在一个 PR 中完成全部内容。
- `000-Tie-up-loose-ends-before-next-phase.md` 已完成；推荐继续按 `001` 到 `005` 的任务草稿拆分 PR。
- 每个子任务都要保持 `go test ./...` 可通过。
- 不要为了最小 Web 前端提前扩大后端业务范围。
- 若发现 README、docs 和代码不一致，优先修正文档或开独立 issue，不把无关重构混入当前任务。
- 每个 issue 收口时，主动检查是否需要同步 `AGENTS.md`、`README.md`、`ROADMAP.md`、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/`。
