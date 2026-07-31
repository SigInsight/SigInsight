# M0：能力与兼容基线

状态：In progress  
起始分支：`feature/lightweight-query-engine`  
起始提交：`e5a3ae5`

## 问题与目标

“支持核心 Logs、Traces、Metrics、Meter UI”目前仍是自然语言要求。M0 要将它转化为固定的请求 fixture、响应 fixture、ClickHouse schema 和模块规模，使后续实现能够判断兼容、退化和删除是否安全。

## 范围

- `/api/v5/query_range` 的核心消费路径。
- Logs Explorer、Traces Explorer、Metrics Explorer、Cost Meter。
- time-series、table、value 和 trace/list view。
- Services 对 V5 query range 的内部调用。
- 基本 threshold alert 的查询准备、预览和评估。
- Collector 创建并写入的 Logs、Traces、Metrics、Meter 表。

## 非目标

- 本阶段不实现 Lite IR 或 compiler。
- 本阶段不修改 API、ClickHouse schema 或 Collector 写入。
- 本阶段不删除现有查询能力。
- 不将 Trace Detail 等专用 store 纳入 query range。

## 已确认的当前入口

- 后端 V5 总调度：`pkg/querier/querier.go`。
- V5 请求模型：`pkg/types/querybuildertypes/querybuildertypesv5`。
- 前端请求准备：`frontend/src/api/v5/queryRange/prepareQueryRangePayloadV5.ts`。
- 前端响应转换：`frontend/src/api/v5/queryRange/normalizeQueryRangeResponse.ts`。
- Threshold rule：`pkg/query-service/rules/threshold_rule.go`。
- Services 内部查询：`pkg/modules/services/implservices/module.go`。
- Meter 分支：V5 metric query 的 `source=meter`。

## 基线产物

计划增加：

- 可机器读取的能力矩阵。
- 各核心页面的 V5 请求/响应 fixture。
- ClickHouse 25.5.6 schema fingerprint。
- 前后端 query 模块规模与生产消费者报告。
- 可重复执行的 Collector/SigInsight 协作验证脚本入口。

## 初始能力分类

| 能力 | 分类 | 说明 |
| --- | --- | --- |
| builder query | 迁移 | 转换为 signal-specific Lite IR |
| simple arithmetic formula | 保留 | 限定为 `+ - * /` |
| trace operator | 拒绝 | 新 UI 隐藏，兼容层返回 unsupported |
| builder join | 删除候选 | 当前没有完整执行能力 |
| builder sub-query | 删除候选 | 不暴露为公共 IR |
| ClickHouse SQL | 拒绝 | 不进入新 UI 或 Lite IR |
| secondary aggregation | 拒绝 | Metrics 使用专用两阶段计划 |
| limitBy | 拒绝 | 保留全局 limit 和 cursor |
| arbitrary having | 拒绝 | 替换为类型化 aggregation predicate |
| function chain | 拒绝 | 只保留结果规则 fillZero |

## 测试计划

- 从真实前端操作捕获并去敏的请求 fixture。
- 对现有前端 payload/normalizer 建立快照或结构断言。
- 在 `clickhouse/clickhouse-server:25.5.6` 上运行 Collector schema migration。
- 写入最小 Logs、Traces、Gauge、Sum、Histogram 和 Meter 数据集。
- 使用认证 API 验证所有目标场景。
- 查询 `system.query_log` 记录表、列、read rows、read bytes 和异常。

## 验收矩阵

| 场景 | 请求 fixture | 响应 fixture | 真实 CH | 前端展示 |
| --- | --- | --- | --- | --- |
| Logs raw/list | Pending | Pending | Pending | Pending |
| Logs time series | Pending | Pending | Pending | Pending |
| Traces list/trace | Pending | Pending | Pending | Pending |
| Traces time series | Pending | Pending | Pending | Pending |
| Gauge | Pending | Pending | Pending | Pending |
| Sum rate/increase | Pending | Pending | Pending | Pending |
| Histogram quantile | Pending | Pending | Pending | Pending |
| Meter | Pending | Pending | Pending | Pending |
| Services | Pending | Pending | Pending | Pending |
| Threshold alert | Pending | Pending | Pending | Pending |

## 退出条件

- 上述场景均有成功 fixture 和至少一个相关错误 fixture。
- schema fingerprint 与生成命令已提交。
- 模块规模和生产依赖报告可以重复生成。
- 未决事项不再阻塞 Lite IR 的字段和类型定义。

## 残余风险

- 保存的 Dashboard 和 Alert 可能包含高级能力，需要统计后决定迁移提示方式。
- Live Logs 是否共用新 executor 尚未决定，但不阻塞第一版。
- 当前 Metrics metadata 和 Meter discovery 仍可能包含查询引擎之外的专用依赖。

