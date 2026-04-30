# Issue: 确认 agent instructions，完成 Alpha 阶段开发前收口工作

## User Facing Issue

### Status

已完成并通过 PR #10 合并。

本任务完成后：

- 公共长期规则已经沉淀到 `AGENTS.md`。
- 项目说明已经同步到 `README.md`、`ROADMAP.md` 和 `docs/`。
- agent 使用的任务草稿已经同步到 `docs/issues/`。
- 用户新建 issue 的入口已经同步到 `.github/ISSUE_TEMPLATE/`。
- `.codex/` 仍作为本地个人 guidance，不随仓库提交。

下一步从 `001-align-deployment-docs.md` 开始推进。

### Background

项目新增 agent instructions，需要对 agent instructions 做 review 和修改，目标是对齐项目版本和阶段需求，并使 agent instructions 可持续自主更新。
需要做一次 Alpha 阶段开发前收口提交，保证下一阶段开发仓库干净、路线清晰。

### Tasks

- [x] 将 `AGENTS.md`、`.codex/*.md` 对齐当前项目进度和阶段需求。
- [x] 将 `docs/*.md` 对齐当前项目进度和阶段需求。
- [x] 将 `ROADMAP.md` 对齐当前项目进度和阶段需求。
- [x] 适当修改 agent instructions，让 agent instructions 可以自动同步项目开发进度。
- [x] 适当修改 agent instructions，在每个 issue 结束后，主动询问用户是否更新 issues。
- [x] 将 `README.md` 对齐当前项目进度和阶段需求。
- [x] 明确新增 agent instructions、`docs/issues/` 和 `.github/ISSUE_TEMPLATE/` 可以提交。
- [x] 做一次收口提交。

### Acceptance Criteria

- 新开发者可以根据项目中的 agent instructions，同步同样的 agent 辅助学习开发工作流。
- agent instructions 和 issues 可动态更新维护，减少人工阶段整理。
- 保证提交完成后工作区干净，可正式进入 Post-MVP Alpha 阶段开发。

## AI Agent Notes

- 这是文档任务，除非发现明显配置不一致，否则不要改业务代码。
- `.codex/` 是本地个人 guidance，默认不提交；公共长期规则应提炼到 `AGENTS.md`。
