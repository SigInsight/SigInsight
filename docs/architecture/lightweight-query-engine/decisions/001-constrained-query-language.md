# ADR-001：采用受约束的类型化查询语言

状态：Proposed  
日期：2026-07-31  
关联里程碑：M0、M1

## 背景

现有 Query Range v5 使用一个通用查询结构承载 Logs、Traces、Metrics、Meter、公式、Trace Operator、任意聚合表达式、secondary aggregation、Having 和函数链。它的表达能力超过 SigInsight 核心遥测 UI 的实际需要，并导致前端选项、协议校验、planner、SQL compiler 和 postprocess 之间形成大量组合状态。

项目目标是形成自主设计、可维护的查询引擎，而不是保留全部能力后更换实现方式。

## 决策

新引擎采用受约束的类型化查询语言：

- 公共查询结构只保留 name、filter、group、order、limit 和 cursor。
- Logs、Traces、Metrics/Meter 使用独立 spec 和 aggregation 类型。
- Filter 使用 AST，不接受 SQL 字符串。
- Metrics 的两阶段聚合是 `MetricPlan` 的专用语义。
- 公式只支持命名查询间的 `+`、`-`、`*`、`/`。
- 不支持 Trace Operator、join、sub-query、任意 SQL、通用 secondary aggregation、通用 limitBy、任意 Having 和函数链。

## 备选方案

### 继续裁剪现有 QueryBuilderQuery

不采用。大量泛型字段仍然存在于协议和前端状态中，无法从结构上阻止无效组合，也很难证明实现已经独立于原系统。

### 完整实现现有 V5 能力

不采用。实现成本和测试组合与轻量化目标冲突，且多个能力没有核心 UI 需求。

### 每个页面直接定义专用 API

不采用。短期简单，但会重复实现时间范围、过滤、分组、结果和告警逻辑，无法形成统一查询核心。

## 影响

正面影响：

- IR 状态空间更小，可以建立完整能力矩阵。
- 前端能够根据 signal/type 只显示合法操作。
- Compiler 不再解析用户聚合 SQL 表达式。
- 不支持能力可以在协议边界稳定拒绝。

能力损失：

- 无法查询跨 span 父子/祖先关系。
- 无法执行任意嵌套、join、分组内 Top-N 和复杂聚合后过滤。
- 无法在查询后端执行 anomaly、EWMA、median 等变换。
- 一部分旧 Dashboard 和 Alert 不能自动迁移。

## 迁移与回滚

迁移期保留旧引擎并双跑受支持请求。保存查询先经过能力分类：可转换的生成 Lite IR，不可转换的明确显示不支持。新引擎成为默认前不删除旧实现，因此可以按请求路由回滚。

## 验证

- M0 统计真实保存查询和核心页面请求所使用的能力。
- M1 对全部合法/非法组合建立表驱动和 fuzz 测试。
- M5 证明新 UI 不能构造非法组合。
- M6 对目标场景执行新旧结果比对。

## 后续动作

- 完成机器可读的 capability matrix。
- 在 M0 结束时根据真实使用数据重新审核公式与 `timeShift` 边界。

