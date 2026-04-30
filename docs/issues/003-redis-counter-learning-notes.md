# Issue: 整理 Redis 原子计数器学习笔记

## Background

文章阅读计数 Redis 原型已经完成，但用户明确要求笔记部分后续再做。本 issue 用于单独整理这次学习内容，避免阻塞代码收口，同时保留学习闭环。

## Goals

- 总结 Redis 原子计数器的基础概念和适用场景。
- 解释本项目为什么使用 `INCR` 记录阅读次数。
- 对比查询缓存、登录限流和阅读计数三类 Redis 场景。
- 记录当前实现的边界：不设置 TTL、不去重、不落库、不返回 API。

## Tasks

- [ ] 结合用户自己的理解，整理并纠正关键概念。
- [ ] 在 `.notes/` 下新增或更新 Redis 原子计数器学习笔记。
- [ ] 说明 `article:views:<article_id>` key 设计、原子性、失败策略和 PostgreSQL 边界。
- [ ] 关联代码位置：service 接口、Redis counter 实现、依赖装配和测试。
- [ ] 标注后续可学习方向：批量落库、短期去重、排行榜或趋势榜。

## Non Goals

- 不修改业务代码。
- 不新增 Redis 功能。
- 不把笔记内容塞进 `AGENTS.md`；只有长期项目规则才迁移到 agent instructions。

## Acceptance Criteria

- `.notes/` 中有一份可复习的 Redis 原子计数器笔记。
- 笔记能解释这次实现和 Redis 设计思想之间的关系。
- 用户提出的理解已被评价、纠正或融入笔记。

## AI Agent Notes

- 本任务需要等待用户愿意进入笔记整理阶段后再执行。
- 如果用户给出自己的理解，先评价准确性，再整理成笔记。
- 笔记是学习资料，不替代 `docs/redis.md` 的项目运行说明。
