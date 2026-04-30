# Issue Drafts

本目录保存可复制到 GitHub Issues 的任务草稿，也保存给 AI Agent 的实施提示。这里的文件属于仓库公共协作资产，可以提交。

用途区分：

- `.github/ISSUE_TEMPLATE/`：GitHub 识别的新建 Issue 模板，默认使用中文。
- `docs/issues/`：项目维护者和 AI Agent 使用的具体任务草稿。
- GitHub Issues 页面：真正的任务看板。

建议流程：

```text
从 docs/issues 选择任务草稿
-> 复制用户可读部分到 GitHub Issue
-> 根据 AI Agent Notes 让 Codex 实施
-> PR 合并后关闭 GitHub Issue
```

任务草稿可以包含实现提示，但真正公开给协作者时应保留清晰、可验收的用户视角描述。

## Current Queue

- 已完成：`000-Tie-up-loose-ends-before-next-phase.md`
- 已完成：`001-redis-atomic-counter-integration`
- 已完成：`002-redis-counter-runtime-verification.md`
- 已完成：`004-redis-authenticated-view-dedup.md`
- Umbrella：`000-next-phase-roadmap.md`
- 后续学习笔记：`003-redis-counter-learning-notes.md`
- 当前收口：Redis 登录用户阅读去重实现，等待提交
- 后续任务：根据下一阶段学习推进结果继续生成或调整

## Maintenance Rules

- 每个文件只描述一个具体任务或一个 umbrella issue。
- 长期规则不要只留在 issue 中；沉淀后迁移到 `AGENTS.md`。
- 架构、部署、Redis 等解释性背景不要塞进 issue；沉淀后迁移到 `docs/`。
- 每个 issue 收口时，Codex 应主动询问是否同步更新本目录和 `.github/ISSUE_TEMPLATE/`。
- 如果新增任务草稿，应同步检查 `ROADMAP.md` 的 Issue Drafts 列表。
