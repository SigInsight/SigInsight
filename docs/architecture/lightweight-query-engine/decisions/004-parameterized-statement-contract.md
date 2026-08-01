# ADR-004：Compiler 输出参数化 Statement

状态：Accepted
日期：2026-07-31
关联里程碑：M2、M3、M4

## 背景

轻量引擎需要将字段、Map key、JSON path、过滤值和时间范围转换为 ClickHouse SQL。若 compiler 将任意用户输入拼接进 statement，新的实现会复制旧 SQL 表达式模型的注入风险和难测性。

## 决策

Compiler 输出：

```go
type Statement struct {
    SQL      string
    Args     []any
    Columns  []ResultColumn
    Warnings []string
}
```

SQL 仅由 compiler 和 Schema Catalog 的静态模板构成。所有值，包括 Map key 和 JSON path，使用驱动参数绑定。结果列使用 compiler 生成的稳定 alias，语义列名保存在 `Columns` metadata 中。排序方向、表名和已知物理列仅能来自受限制 enum 或 Catalog，不能来自请求字符串。

## 备选方案

### 使用用户可输入的 SQL expression

不采用。它无法建立字段能力矩阵，也无法可靠保护查询边界。

### 继续使用旧 sqlbuilder 和字段 mapper

不采用。该路径仍依赖 V5 DTO、历史 metadata fallback 和 signal-specific 旧 package，无法构成独立引擎。

### Compiler 直接执行 ClickHouse 查询

不采用。执行、超时、观测与错误映射属于 M4 Executor，混入 compiler 会削弱 SQL golden test 的价值。

## 影响

- Statement 可以独立 snapshot 和驱动真实 ClickHouse 测试。
- Catalog 的静态 SQL 模板成为严格审查点。
- 含动态标识符的 ClickHouse 功能只能在存在安全绑定策略时加入。
- 后续 Executor 可统一记录 SQL、参数数目、扫描统计和 warning。

## 验证

- M2/M3 的 golden SQL 与 Args 测试。
- 恶意 field/map key/JSON path 输入测试。
- ClickHouse 25.5.6 对生成 statement 的执行测试。
