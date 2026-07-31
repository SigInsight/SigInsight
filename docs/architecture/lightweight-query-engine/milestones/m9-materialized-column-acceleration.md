# M9：物化列加速查询

状态：Complete
关联 ADR：011
前置条件：M8 完成（legacy 引用清零）、ADR-011 审核通过

## 问题与目标

Collector migration 建立的高频 attribute 物化列（`attribute_string_<key>` 等）在 Lite
引擎中未被利用：Catalog 一律解析为类型化 Map 列，物化列只产生存储与写入
成本。M9 在保持 Catalog 静态确定性契约（ADR-003）的前提下，把物化列作为显式声明纳入
解析优先级，用真实查询数据证明收益，并输出低频列的删除候选清单。

## 范围

- `pkg/litequery` Catalog 的物化目录层：基线探测、fingerprint 绑定、解析优先级。
- Traces `span_index_v3` 的 v1 baseline resource/attribute 物化列。Logs `logs_v2`
  没有内建物化列，继续走 Map，不能把运维手工列隐式纳入查询。
- manifest 命中物化列和非 manifest Map 路径的 golden SQL 与真实执行测试。
- 同 fixture 的 Map vs 物化列性能对比（延迟、read rows、read bytes）。

## 非目标

- 不做运行时逐查询自动探测或隐式 fallback（ADR-011 明确拒绝）。
- 不改变 Lite IR、V5 协议边界或前端体验。
- 不修改 Collector schema、不新增或删除物化列（删除是独立 ADR 议题）。
- 不为物化目录引入用户可见配置或 UI。

## 当前代码与依赖

- `pkg/litequery/catalog.go`：`resolveTypedMapField` / `resolveMapField` 目前是
  attribute 的唯一物理路径。
- `pkg/telemetrymetadata/stmt_parse.go`：旧 metadata 侧解析 `SHOW CREATE TABLE`
  DEFAULT 表达式识别物化列的逻辑可参考，但 M9 不依赖 metadata store。
- Collector v1 DDL：9 个 Trace string 的命名约定
  （`<context>_string_<key>`，`.` 替换为 `$$`）及对应 `_exists` 列。
- schema fingerprint：`schema-baseline.md` 冻结的 v1 fingerprint 是目录失效校验的锚点。

## 设计

```text
受版本控制的 Catalog manifest {semantic key -> physical value / exists column}
  -> Collector migration fingerprint 跨仓库校验

Catalog.Resolve
  |-- 物化目录命中 -> 物化列（快路径）
  `-- 未命中       -> 类型化 Map 列（attributes_string 等，现状路径）
```

- manifest 是静态 Go 常量，避免同一二进制因运行时 schema 改变 SQL。
- 物理列变更必须同一变更更新 manifest 和 Collector fingerprint 验证；不允许运行时
  部分加载或静默回退。
- 解析优先级只存在于 Catalog 内部，IR、plan 和 Statement 对外语义不变。

## API/IR/schema 变化

- 无公共 API、IR 或协议变化；`Statement` 结构不变。
- Catalog 内部新增物化目录版本与 fingerprint 关联；`capability-matrix.json` 不变化。

## 迁移与回滚

- 回滚是移除 manifest 项并恢复纯 Map Catalog 的可审查代码变更，无需数据迁移。
- 不触碰 Collector schema；物化列的新增/删除仍由 OtelCollector migration 驱动，
  并通过 ADR-003 fingerprint 协作验证衔接。

## 测试计划

- manifest 解析优先级、非 manifest Map 路径和 trusted identifier 单元测试。
- manifest 列与非 manifest 字段的 Logs/Traces raw、time-series、scalar golden SQL 与 Args 测试。
- ClickHouse 25.5.6 + 当前 Collector migration：认证 API 查询与 query-log 物理列断言。
- 回归：全量 Go、前端 build/test、query log 无未知列错误。

## 验收矩阵

| 场景 | manifest 路径 | 非 manifest 路径 | 证据 |
| --- | --- | --- | --- |
| Traces manifest resource/attribute 过滤 | 物化列路径 | — | golden SQL + 真实执行 |
| Traces 非 manifest resource/attribute | — | Map 路径 | golden SQL + 真实执行 |
| 高频 key 查询延迟/扫描量 | 期望优于 Map | 基线 | 对比报告 |
| manifest 列缺失 | schema 不兼容 | — | Collector fingerprint 跨仓库验证 |

## 实现结果

Trace 9 项 manifest、value/exists SQL 路径和非 manifest Map 路径的单元测试已实现；
`tests/integration/scripts/run-materialized-catalog-integration.sh` 已通过 Collector
migration、认证 API 和 query-log 物理列断言。认证 API workload 将 9 个查询拆为两个
请求（Lite query budget 为每请求最多 8 个），逐项覆盖所有 manifest 列并从
`system.query_log` 验证物理路径。可选 10k 合成基准验证 Map 与物化列的延迟和读取字节
对比。

## 删除内容

- 当前 workload 的删除候选：**无**。全部 9 个 manifest 列均由认证 API/query-log
  workload 覆盖，不能据此证明任一列是低频列。
- 不删除物化列；未来若获得覆盖完整业务周期的生产 query log，可重新形成独立 ADR 的
  删除候选清单。

## 度量变化

- `M9_MATERIALIZED_BENCHMARK_ROWS=10000` 可选执行 Map vs 物化列 p50/p95 延迟、
  `read_rows`、`read_bytes` 采样；合成数据仅用于物理路径诊断，生产删除决策仍需
  真实 workload 的相同报告。
- 记录 Catalog 新增行数与测试文件数。

## 残余风险与后续任务

- 物化列集合与写入负载相关，收益会随时间变化；生产 workload 的删除评估必须另行
  形成 ADR 和跨仓库 migration 验证。
