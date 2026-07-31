# ADR-011：物化列显式目录契约

状态：Accepted
日期：2026-07-31
关联里程碑：M9

## 背景

ClickHouse schema 通过 migration 将高频 attribute 物化为独立物理列（如
`attribute_string_<key>`、`resource_string_<key>`），
以 DEFAULT 表达式从 Map 派生。旧引擎在编译每条查询时探测物化列并自动选择最快路径。

Lite 引擎的 Schema Catalog（ADR-003）坚持静态、确定性映射：
`pkg/litequery/catalog.go` 目前对 attribute 一律解析为 `attributes_string` 等
类型化 Map 列，完全不引用物化列。M8 完成 legacy 清零后，物化列的唯一潜在消费者
将只剩 Lite Catalog，此时"利用"还是"删除"必须显式决策。

## 决策

M9 采用"显式物化目录"：物化列不通过运行时逐查询探测进入查询路径，而是作为
版本控制的 Catalog manifest 声明。

- manifest 从 Collector v1 baseline DDL 提取，定义 semantic key -> physical value / exists
  column；它和 Lite 源码一起评审、版本化。
- Collector 的 migration integration test 冻结 `system.tables` / `system.columns`
  fingerprint；任何影响 manifest 的 schema 变更必须在同一跨仓库改动中更新 manifest、
  schema-baseline、golden SQL 和真实协作验证。
- `Catalog.Resolve` 解析优先级：物化目录命中 -> 物化列；未命中 -> 类型化 Map 列
  （维持现状路径）。
- 不提供运行时自动选择、不提供用户可见的物理路径选择，也不提供运行时配置开关。

## 备选方案

### 运行时逐查询自动探测（旧引擎方式）

不采用。同一查询在不同 schema 状态下编译出不同 SQL，golden SQL 测试、query log
审计和性能可预测性均被破坏，与 ADR-003 的静态契约冲突。

### 完全忽略物化列（维持纯 Map Catalog）

不采用。已存在的物化列由 migration 维护、占用存储并参与写入计算，若永不使用则
成为纯成本；M8 之后应显式选择利用或删除。

### 删除所有物化列

不在本 ADR 范围内。删除由真实使用数据驱动并需要独立 ADR；本 ADR 只决定"利用
已存在的物化列"，收益不足的低频列进入后续删除候选。

## 影响

- Catalog 需要版本化的物化列集合；Collector fingerprint 测试成为物化列新增/删除的
  跨仓库协作契约。
- manifest 命中的语义字段始终编译为物化列；未命中 manifest 的字段始终编译为 Map。
  缺少 manifest 列的 schema 不兼容，而不是运行时 Map fallback。
- 性能收益由真实查询的 read rows、read bytes 与延迟对比证明；收益不达标的物化列
  进入删除候选清单，由独立 ADR 处理。

## 迁移与回滚

- 回滚为删除相应 manifest 项并恢复 Map Catalog，作为一项可审查的代码变更。
- 不修改 Collector schema；物化列的新增/删除仍由 OtelCollector migration 驱动。

## 验证

- manifest 命中物化列和非 manifest Map 字段的 golden SQL 与真实 ClickHouse 25.5.6
  执行。
- Collector migration fingerprint 与 manifest 物理列名的协作验证。
- 同 fixture 的 Map vs 物化列查询延迟、read rows、read bytes 对比报告。

## 后续动作

- M9 完成后，依据使用数据决定低频物化列的删除（独立 ADR）。
