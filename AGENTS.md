# Content Backend Project Guidance

本文件用于仓库级共享 guidance，只放适合随项目一起维护、适合提交到远程仓库的公共规则。

## Project Scope

这是一个使用 Go 实现的内容发布后端项目。

当前仓库中的协作应优先服务于以下目标：

- 保持接口行为清晰、稳定
- 保持分层边界清楚
- 让核心业务链路可运行、可验证

## Layer Boundaries

- `repository` 负责数据访问，不承载业务语义解释
- `service` 负责业务流程、权限判断、状态规则和用例组织
- `handler` 负责 HTTP 输入输出和错误映射
- `main` 负责依赖装配和应用启动

## Development Priorities

- 优先补齐核心业务闭环，而不是过早扩展花哨功能
- 修改代码时优先保持现有结构一致性
- 新增逻辑时优先补边界条件和可验证性
- 设计接口时优先区分业务动作，不把不同职责混成一个入口

## Collaboration Notes

- 如果某项规则只属于个人学习节奏、本机习惯或临时实验，不应写在本文件中
- 个人 guidance 放在本地层，例如 `.codex/AGENTS.md`
