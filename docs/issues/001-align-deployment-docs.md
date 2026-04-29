# Issue: 补充部署 runbook 与环境验证说明

## User Facing Issue

### Background

项目已经具备 Docker Compose 自托管链路，README 和基础部署文档已记录 `nginx -> app -> PostgreSQL/Redis` 的当前结构。下一步需要把部署说明从“能启动”推进到“能更新、能排障、能验证”，避免真实服务器维护时只靠临时记忆。

### Tasks

- [ ] 检查 `README.md`、`docs/deployment.md` 和 Compose 配置是否仍一致。
- [ ] 补充部署更新流程，例如拉取代码、重新构建、重启服务和检查日志。
- [ ] 补充回滚思路，说明代码回滚、镜像重建和数据 volume 的边界。
- [ ] 补充 PostgreSQL / Redis 数据备份和恢复的最小说明或后续 issue。
- [ ] 明确真实环境文件不得提交，示例文件可以提交。

### Acceptance Criteria

- 新机器可根据 README 和示例环境变量启动完整 Compose 链路。
- 文档能说明如何更新、检查和初步排障 `nginx`、`app`、PostgreSQL、Redis。
- 文档能区分仓库规则、服务器私有配置和真实数据。
- `go test ./...` 保持通过。

## AI Agent Notes

- 优先读取 `README.md`、`docker-compose.yml`、`app.env.example`、`.env.example`、`docs/deployment.md`。
- 不提交真实 `.env`、`app.env`、`db.env`、`.env.compose`。
- 如果修改启动说明，应同步检查健康检查、端口和 Redis 地址描述。
- 这是文档任务，除非发现明显配置不一致，否则不要改业务代码。
