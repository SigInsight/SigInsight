# Lightweight Query Engine

本文档集描述 SigInsight 自主设计的轻量化遥测查询引擎，以及与其配套的前端查询界面。

项目目标不是继续裁剪现有 SigNoz Query Range v5 实现，而是在保留必要 HTTP 兼容边界的前提下，建立一条受约束、类型安全、可验证的查询链，并在完成真实协作验证后替换旧实现。

## 文档入口

- [总体设计与功能边界](design.md)
- [任务节点与里程碑](roadmap.md)
- [ClickHouse schema 基线](schema-baseline.md)
- [物化列术语与机制](materialized-columns.md)
- [论文章节底稿：架构设计与边界防御](thesis-chapter-architecture.md)
- [阶段工程文档规范](milestones/README.md)
- [M0：能力与兼容基线](milestones/m0-capability-baseline.md)
- [M1：Lite Query Core](milestones/m1-lite-query-core.md)
- [M2：Logs/Traces Schema Catalog 与 Compiler](milestones/m2-log-trace-compiler.md)
- [M3：Metrics/Meter Compiler](milestones/m3-metrics-meter-compiler.md)
- [M4：Executor、Result 与基本 Formula](milestones/m4-executor-result.md)
- [M5：V5 兼容桥与受控 API 接入](milestones/m5-v5-compatibility-bridge.md)
- [M6：轻量前端查询体验](milestones/m6-frontend-query-experience.md)
- [M7：协作验证与受控切换](milestones/m7-collaboration-rollout.md)
- [M8：Legacy 删除准备与量化收敛](milestones/m8-deletion-readiness.md)
- [M9：物化列加速查询](milestones/m9-materialized-column-acceleration.md)
- [M10：退役不可达的 Legacy 编辑渲染树](milestones/m10-legacy-editor-render-tree.md)
- [M11：清理无调用的 Legacy QueryBuilder Helper](milestones/m11-unused-builder-helpers.md)
- [M12：QueryBuilderSearchV3 与 Trace Funnel 过滤子集](milestones/m12-query-builder-v3.md)
- [架构决策记录规范](decisions/README.md)
- [ADR 模板](decisions/000-template.md)
- [ADR-001：采用受约束的类型化查询语言](decisions/001-constrained-query-language.md)
- [ADR-002：兼容 V5 HTTP 边界而非全部查询能力](decisions/002-v5-compatibility-boundary.md)
- [ADR-003：使用 Schema Catalog 隔离查询语义与 ClickHouse schema](decisions/003-schema-catalog-contract.md)
- [ADR-004：Compiler 输出参数化 Statement](decisions/004-parameterized-statement-contract.md)
- [ADR-005：Metrics/Meter 数据源与双阶段聚合契约](decisions/005-metrics-source-and-aggregation-contract.md)
- [ADR-006：Executor 与结果扫描边界](decisions/006-execution-and-result-boundary.md)
- [ADR-007：V5 兼容桥的受控回退](decisions/007-v5-bridge-controlled-fallback.md)
- [ADR-008：Lite 前端状态桥](decisions/008-lite-frontend-state-bridge.md)
- [ADR-009：服务端发布的 rollout capability](decisions/009-server-advertised-rollout-capability.md)
- [ADR-010：专用读取 API 先于 Legacy V5 删除](decisions/010-specialized-readers-before-legacy-removal.md)
- [ADR-011：物化列显式目录契约](decisions/011-materialized-column-catalog-contract.md)
- [ADR-012：在 V5 边界退役 Trace Operator](decisions/012-retire-trace-operator-at-v5-boundary.md)
- [ADR-013：拒绝不支持的 V5 能力](decisions/013-reject-unsupported-v5-capabilities.md)
- [ADR-014：字段元数据消歧契约](decisions/014-field-metadata-disambiguation.md)
- [ADR-015：语义 Gauge 与物理 Metric Series 对齐](decisions/015-semantic-gauge-physical-series.md)
- [ADR-016：查询正确性不变量与结果预算](decisions/016-query-correctness-invariants.md)
- [ADR-017：OTel Entrypoint、V5 Identity 与结果截断协议](decisions/017-otel-entrypoint-and-result-limit.md)
- [ADR-018：QueryBuilderSearchV3 与 Trace Funnel 过滤边界](decisions/018-query-builder-v3-and-funnel-filter-boundary.md)

## 文档状态

| 文档 | 状态 | 说明 |
| --- | --- | --- |
| 总体设计与功能边界 | Proposed | 等待首次工程审核 |
| 任务节点与里程碑 | Proposed | 等待首次工程审核 |
| M0 能力与兼容基线 | Complete | 已固化请求样本、schema baseline、规模数据和真实协作验证入口 |
| M1 Lite Query Core | Complete | 独立类型化 IR、校验器、预算和公式依赖检查 |
| M2 Logs/Traces Compiler | Complete | 参数化 Catalog/Compiler，已在 ClickHouse 25.5.6 真实 schema 验证 |
| M3 Metrics/Meter Compiler | Complete | 双阶段聚合、counter rate/increase、explicit Histogram 和 Meter 已真实验证 |
| M4 Executor/Result | Complete | 可取消并发执行、动态 row 扫描和 arithmetic formula 已真实验证 |
| M5 V5 兼容桥 | Complete | 四类信号的 V5 转换、字段元数据解析、认证 API 与错误映射均已验证；真实回读由 M7 闭环 |
| M6 Lite Frontend | Complete | Lite 编辑器复用 V5 状态/协议，payload 清理与能力校验具有组件和契约测试；真实协作由 M7 闭环 |
| M7 协作验证与切换 | Complete | 当前 Collector 在 ClickHouse 25.5.6 的 Logs、Traces、Metrics、Meter 写入和认证 V5 读回均已验证；受控能力协商已完成 |
| M8 Legacy 删除准备与量化收敛 | Complete | legacy executor、editor、rollout flag 与过渡 compiler 已删除；V5 边界为 Lite-only capability error |
| M9 物化列加速查询 | Complete | Trace v1 静态 manifest、非 manifest Map、9 字段认证 API/query-log workload 与 Map-vs-column 基准已验证；当前无删列候选 |
| M10 Legacy 编辑渲染树 | Complete | 已删除不可达的 Query/Formula/函数/聚合渲染树，保留共享状态、DTO 与独立筛选控件 |
| M11 Legacy QueryBuilder Helper | Complete | 已删除无调用旧 helper，保留 metadata 与专用读取仍使用的 parser/field mapper |
| M12 QueryBuilderSearchV3 | Complete | V3 文本补全、字符串谓词、生产消费者迁移和旧编辑器删除已完成；全量 build/test 与真实浏览器查询已闭环 |
| 机器可读能力矩阵 | Accepted | `capability-matrix.json` 是后续协议和 UI 的约束来源 |

## 查询引擎边界：引擎之外的专用查询构建器

轻量查询引擎只接管四信号（Logs/Traces/Metrics/Meter）的面板查询（`/api/v5/query_range`）与
告警阈值读取（`pkg/query-service/rules/litequery_runner.go`）。其余遥测读取路径仍由一批"专用
查询构建器"承担：它们不经过 Lite 的 DSL → IR → compiler 链路，而是用固定 SQL 模板或参数化
语句直接读取 ClickHouse。这些构建器不是 Lite 的竞争者，而是
[ADR-010](decisions/010-specialized-readers-before-legacy-removal.md) 定义的"专用读取 API"
边界内的既有事实：Lite 按设计只服务面板聚合查询
（[ADR-013](decisions/013-reject-unsupported-v5-capabilities.md)），多步时序、点查、流式与
预聚合表读取均不在其能力矩阵内。历史上曾有一个大型专用 compiler——Trace Operator——负责
把 Traces 面板 DSL 编译为 ClickHouse SQL，已在
[ADR-012](decisions/012-retire-trace-operator-at-v5-boundary.md) 退役：其面板查询能力由 Lite
接管，跨 span 关系等能力被拒绝而非迁移，这确立了"能进矩阵的整合、进不了的显式拒绝"的决策
标准。

### 多步时序分析：专用模板 SQL

| 模块 | 读取内容 | 构建方式 | 规模 |
| --- | --- | --- | --- |
| Trace Funnel（`pkg/modules/tracefunnel`） | 多步漏斗的顺序匹配、转化率、步间延迟、慢/错 trace | 模板 SQL 生成器（`clickhouse_queries.go`）：WITH 定义每步，`minIf(timestamp, ...)` 求每步首现时间，`HAVING tN_time > tN-1_time` 判定时序 | 617 行模板 + 327 行编排 |

为什么不在引擎内：多步顺序匹配是跨 span 的时序语义，需要把"多个步骤"作为一个整体规划，
超出 Lite 单查询的能力矩阵（[ADR-018](decisions/018-query-builder-v3-and-funnel-filter-boundary.md)）。
边界策略是"过滤器统一、计算分离"：Funnel 的过滤子句已复用 V3/Lite 语法，多步计算保持专用。

### 固定聚合形状：单条参数化语句

| 模块 | 读取内容 | 构建方式 |
| --- | --- | --- |
| Span Percentile（`pkg/modules/spanpercentile`） | 单服务单操作在时间窗内的 p50/p90/p99 与百分位位置 | 固定聚合 SQL + `clickhouse.Named` 参数绑定；代码注释明确"one fixed aggregate shape, so it reads the trace index directly instead of building a V5 scalar request" |
| Services / Dependency Graph（`pkg/query-service/app/traces/servicestore`） | top-level operations、服务依赖图（p50–p99、callRate、errorRate） | 预聚合表直读 + `quantilesMergeState` 聚合状态函数 |
| Metric Metadata（`pkg/query-service/app/metricmetadatastore`） | metric 的 type/temporality/unit 等元数据 | 元数据表 `argMax` 聚合 |
| Metrics Explorer（`pkg/query-service/app/metricsexplorerstore`） | tag keys/values 搜索、相关指标、inspect 明细 | `JSONExtract` 枚举与专门的相关度算法 |

为什么不在引擎内：这些读取形状固定（输入强类型、输出固定行），参数化模板比 DSL 编译链路更
简单、更可审计。Lite 的 DSL → IR → planner 是为"任意查询"付费的能力，对固定形状是过度设计；
其中 Dependency Graph 依赖聚合状态函数（`quantilesMergeState`），不在 Lite 表达式语义内。

### 点查 / 游标 / 流式读取

| 模块 | 读取内容 | 构建方式 |
| --- | --- | --- |
| Trace Detail（`pkg/query-service/app/traces/tracedetailstore`） | 单 trace 的 waterfall/flamegraph 全量 span 字段 | `trace_id` + `ts_bucket_start` 范围点查，直接寻址 span 主表 |
| Exceptions（`pkg/query-service/app/traces/exceptionstore`） | 错误分组列表、errorID 上下翻页 | `groupID`/`errorID`/`timestamp` 游标查询 |
| Live Logs（`pkg/livelogs`） | 日志实时尾随流 | SSE 长连接 + 每批 500 行轮询 + `(timestamp, id)` 严格游标；**部分复用 Lite**：过滤表达式经 `liteadapter.ParseFilter` 解析，游标类型为 `litequery.RawLogCursor` |
| Rule State History（`pkg/query-service/app/rulestatehistorystore`） | 告警状态变更统计与时间线 | 规则状态表直读（非遥测表） |
| Retention（`pkg/query-service/app/retentionstore`） | TTL 设置与查询 | DDL 管理（`ALTER TABLE MODIFY TTL`），不是查询 |

为什么不在引擎内：这些路径的语义不是"面板聚合"——单点寻址（trace_id）、游标推进（exceptions）、
持续输出（SSE）都不属于查询编译的范畴；Rule State History 与 Retention 读的是规则/配置表而
非遥测表，与 Schema Catalog 无关。Live Logs 展示了边界策略的正面案例：**能复用的部分（过滤
器解析）复用 Lite，执行形态保持专用**。

### 不整合的统一原因

1. **能力矩阵外**：多步时序、点查、流式、预聚合表读取均不在 Lite 的 DSL 能力矩阵内（ADR-010、
   ADR-013、ADR-018）。
2. **读取的表不同**：多数专用 reader 读预聚合表/专用表/元数据表（`top_level_operations`、
   `dependency_graph`、规则状态表、错误索引表、trace summary 表），Schema Catalog 只覆盖四
   信号主表。
3. **固定形状 vs 任意查询**：固定输入/输出用参数化模板更简单；DSL 编译链路的复杂度只对"任意
   查询"值得。
4. **执行形态不同**：SSE 流式、DDL、单点寻址不是查询编译的形态。
5. **历史事实**：这些模块从上游引入时即为独立实现（Funnel 的模板 SQL 经 git 验证与上游逐字
   一致，仅替换表名）；轻量引擎的迁移策略是"先专用读取、后统一"（ADR-010），Lite 只接管
   query_range 面板路径。

## 变更规则

1. 功能边界、API 兼容策略、IR、存储映射或迁移策略发生变化时，必须先更新设计文档或新增 ADR。
2. 每个里程碑开始前必须建立阶段工程文档；没有设计、测试计划和退出条件，不开始该阶段的生产实现。
3. 每个里程碑结束时，阶段文档必须包含实现结果、验证证据、已删除内容、残余风险和规模变化。
4. 代码与文档在同一个里程碑提交中保持一致；禁止只在提交说明中记录关键设计。
5. 跨 SigInsight 与 OtelCollector 的 schema 或写入协议变化，两个仓库都要记录对应提交和协作验证结果。
6. `capability-matrix.json` 的每次变化都必须同步更新对应 ADR、阶段文档和测试。

## 分支与提交

- SigInsight 工作分支：`feature/lightweight-query-engine`
- OtelCollector 在首次发生 schema、迁移或写入修改时创建同名分支。
- 宏观提交锚点采用 `M0` 至 `M9`，阶段内可以有可独立验证的子提交。
