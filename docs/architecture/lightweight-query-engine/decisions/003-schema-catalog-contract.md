# ADR-003：使用 Schema Catalog 隔离查询语义与 ClickHouse schema

状态：Accepted
日期：2026-07-31
关联里程碑：M0、M2、M3、M6

## 背景

查询引擎必须将 OpenTelemetry 语义字段映射到 ClickHouse 表、列、Map、JSON path 或计算表达式。若 compiler 直接引用物理字段，Collector schema 调整会扩散到过滤、聚合、排序和多个 signal builder，也难以证明删除列后所有读取仍然安全。

SigInsight 与 OtelCollector 位于不同仓库，不能依赖只在一侧维护的注释作为 schema 契约。

## 决策

建立版本化 Schema Catalog，作为 Lite Query IR 到物理 ClickHouse schema 的唯一映射层。

Catalog 至少声明：

- semantic field ID、signal、名称和值类型。
- 物理表、时间列、组织列和字段表达式。
- 可用过滤操作、是否可 select/group/order。
- schema version 和必需列 fingerprint。

Signal compiler 不得在 Catalog 之外维护字段到列的映射。跨仓库契约通过真实 ClickHouse migration、`system.columns` fingerprint、Collector 写入和 SigInsight API 查询共同验证。

## 备选方案

### Compiler 直接引用表和列

不采用。实现初期代码较少，但会重新建立当前希望消除的存储耦合。

### SigInsight 与 Collector 复制一份 schema 常量

不采用。两个副本容易漂移，编译通过不能证明运行时兼容。

### 独立共享 Go module

暂不采用。它会引入发布和版本协调成本，而且仍不能代替真实 migration 与查询验证。后续只有在 schema contract 被多个项目消费时再评估。

## 影响

- Compiler 输入更稳定，schema 优化集中在 Catalog 和 migration。
- Catalog 本身成为关键正确性边界，需要直接测试和审查。
- JSON body、Map attribute、materialized field 等差异可以隐藏在物理表达式中。
- schema 变化必须同时更新 fingerprint 和真实协作测试，增加少量变更成本但降低运行时缺列风险。

## 迁移与回滚

M2 首先为当前 schema 建立 Catalog，不修改表。任何后续表结构优化作为独立 migration 和 ADR 提交。在删除旧表或列前，保留上一版 Catalog/查询路径直到新 schema 协作验证通过。

## 验证

- ClickHouse 25.5.6 schema migration 测试。
- `system.columns` fingerprint 比对。
- 每个 semantic field 的映射和 capability 单元测试。
- 参数化 SQL golden test。
- Collector 写入后通过认证 API 执行 Logs、Traces、Metrics 和 Meter 查询。
- 检查 `system.query_log` 的未知表/列错误和扫描量。

## 后续动作

- M0 生成当前核心表 fingerprint。
- M2 确定 Catalog 的 Go API 和版本策略。
- 每个 schema 性能优化建立独立 ADR。
