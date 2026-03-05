# Test 目录说明

该目录用于集中存放测试相关文件（例如集成测试、测试数据、测试脚本）。

建议结构：

- `test/integration/`：集成测试
- `test/fixtures/`：测试数据
- `test/scripts/`：测试辅助脚本

说明：

- 当前仓库中的部分 `*_test.go` 仍在业务包目录下（如 `internal/service`、`pkg/logger`），因为这些测试依赖包内未导出函数与状态。
- 在 Go 项目中，这类测试通常需要与被测代码同目录，直接移动到独立目录会导致编译失败。
