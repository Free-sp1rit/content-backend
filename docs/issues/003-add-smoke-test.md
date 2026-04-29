# Issue: 增加部署 smoke 验证清单或脚本

## User Facing Issue

### Background

当前项目已经可以通过 Docker Compose 运行，但部署完成后的验证仍偏手工。需要补充一套最小 smoke 验证，确认服务、数据库和 Redis 依赖下的主链路可用。

### Tasks

- [ ] 设计 smoke 验证覆盖范围：`/healthz`、公开列表、注册、登录、创建文章、发布文章、公开查询。
- [ ] 决定采用文档 checklist、shell 脚本，或二者结合。
- [ ] 如果新增脚本，确保不写入真实 token、密码或服务器 IP。
- [ ] 在 README 或 `docs/deployment.md` 中说明如何运行 smoke 验证。
- [ ] 记录失败时应优先检查的组件：nginx、app、PostgreSQL、Redis、环境变量。

### Acceptance Criteria

- 部署后可以用一套明确步骤验证主链路。
- smoke 验证不会依赖仓库中的真实密钥或固定服务器 IP。
- smoke 验证能证明创建并发布的文章可以从公开列表查询到。
- `go test ./...` 保持通过。

## AI Agent Notes

- 如果添加脚本，优先考虑 `scripts/smoke.sh`，并使用环境变量接收 `BASE_URL`。
- 生成临时 email 时可以用时间戳，避免重复注册冲突。
- 脚本只做 smoke，不做完整测试框架。
- 注意 shell 脚本可移植性，不引入重依赖。
