# Lightweight Query Engine

本文档集描述 SigInsight 自主设计的轻量化遥测查询引擎，以及与其配套的前端查询界面。

项目目标不是继续裁剪现有 SigNoz Query Range v5 实现，而是在保留必要 HTTP 兼容边界的前提下，建立一条受约束、类型安全、可验证的查询链，并在完成真实协作验证后替换旧实现。

## 文档入口

- [总体设计与功能边界](design.md)
- [任务节点与里程碑](roadmap.md)
- [ClickHouse schema 基线](schema-baseline.md)
- [阶段工程文档规范](milestones/README.md)
- [M0：能力与兼容基线](milestones/m0-capability-baseline.md)
- [M1：Lite Query Core](milestones/m1-lite-query-core.md)
- [M2：Logs/Traces Schema Catalog 与 Compiler](milestones/m2-log-trace-compiler.md)
- [M3：Metrics/Meter Compiler](milestones/m3-metrics-meter-compiler.md)
- [M4：Executor、Result 与基本 Formula](milestones/m4-executor-result.md)
- [M5：V5 兼容桥与受控 API 接入](milestones/m5-v5-compatibility-bridge.md)
- [架构决策记录规范](decisions/README.md)
- [ADR 模板](decisions/000-template.md)
- [ADR-001：采用受约束的类型化查询语言](decisions/001-constrained-query-language.md)
- [ADR-002：兼容 V5 HTTP 边界而非全部查询能力](decisions/002-v5-compatibility-boundary.md)
- [ADR-003：使用 Schema Catalog 隔离查询语义与 ClickHouse schema](decisions/003-schema-catalog-contract.md)
- [ADR-004：Compiler 输出参数化 Statement](decisions/004-parameterized-statement-contract.md)
- [ADR-005：Metrics/Meter 数据源与双阶段聚合契约](decisions/005-metrics-source-and-aggregation-contract.md)
- [ADR-006：Executor 与结果扫描边界](decisions/006-execution-and-result-boundary.md)
- [ADR-007：V5 兼容桥的受控回退](decisions/007-v5-bridge-controlled-fallback.md)

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
| M5 V5 兼容桥 | Complete | 认证 API 已验证 Metrics、Meter 与 formula；Logs/Traces 的当前 Collector 数据回读留作 M7 切换门槛 |
| 机器可读能力矩阵 | Accepted | `capability-matrix.json` 是后续协议和 UI 的约束来源 |

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
- 宏观提交锚点采用 `M0` 至 `M7`，阶段内可以有可独立验证的子提交。
