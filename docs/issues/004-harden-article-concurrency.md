# Issue: 加固文章发布/编辑并发场景

## User Facing Issue

### Background

文章发布和编辑是内容平台的核心状态流转。当前实现已经能完成主链路，但需要针对并发读写和重复请求补充更清晰的保护，避免出现发布后仍被编辑、重复状态更新或缓存失效边界不清楚的问题。

### Tasks

- [ ] 梳理文章状态不变量，例如只有草稿可编辑、发布后不可编辑、重复发布应幂等。
- [ ] 检查当前 service/repository 是否存在“先查再改”的并发风险。
- [ ] 选择合适方案，例如条件更新、事务或更明确的 repository 方法。
- [ ] 补充 service 测试或 repository 级验证。
- [ ] 确认发布成功后公开列表缓存会被正确失效。

### Acceptance Criteria

- 至少一个文章状态并发风险被明确识别并处理。
- 方案说明能解释保护的不变量和选择该方案的原因。
- 重复发布、发布后编辑等边界有测试或可复现验证。
- `go test ./...` 保持通过。

## AI Agent Notes

- 优先读取 `internal/service/article_service.go`、`internal/repository/article_repository.go`、`internal/service/article_service_test.go`。
- 不要把业务判断下沉到 repository；repository 可提供条件更新能力，service 决定业务语义。
- 如果改 SQL，需要考虑 `RowsAffected` 和 `sql.ErrNoRows` 的映射。
- 缓存失效只应在状态真实变化后执行，避免把失败更新误认为成功。
