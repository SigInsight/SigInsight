# Trace Detail Related Metrics 设计

## 目标

在 Trace Detail 的 Related Signals 中恢复 Metrics 视图，让用户从一个 span 出发观察同一服务实例、Pod、Node 或 Host 在 span 发生前后的指标变化。

该能力必须复用轻量查询引擎和当前 V5 查询协议，不恢复旧版 InfraMetrics 的硬编码查询清单，也不引入新的 ClickHouse 专用查询路径。

## 关联原理

Trace 和 Metrics 是两类独立的 OpenTelemetry 信号。普通 metric point 通常不携带 `trace_id` 或 `span_id`，因此当前系统不能把任意一个指标点精确归因到某个 span。

当前可实现的是资源与时间相关性：

1. Trace span 的 `resources_string` 保存产生 span 的资源属性。
2. Collector 将 metric 的 resource attributes 保存到 `siginsight_metrics.metric_series.resource_attrs`，并同时合入 `labels`。
3. 系统从选中 span 提取稳定的资源身份，例如 `service.instance.id`、`k8s.pod.uid`、`host.id`、`k8s.pod.name`、`host.name` 和 `service.name`。
4. `/api/v5/metrics/stats` 在 span 附近的时间窗口内寻找同时具有这些资源属性值的活跃指标。
5. 前端为少量候选指标构造标准 V5 metric builder query，通过 `/api/v5/query_range` 获取时间序列。
6. 图表以 span 时间为观察中心，并显示本次关联使用的资源属性。

因此这里的“Related”含义是：这些 metric series 与 span 来自同一资源身份，并且时间上相邻。它不是 metric point 与 span 之间的因果证明。

## 与 Logs 关联的区别

| 信号 | 关联键 | 语义 |
| --- | --- | --- |
| Logs | `trace_id`，必要时再用 `span_id` | 精确 trace/span 关联 |
| Metrics | 资源身份 + 时间窗口 | 资源级或实例级相关 |
| Metrics Exemplars | `trace_id` / `span_id` | 精确指标样本关联，当前未存储 |

不得在 UI、接口或文档中把资源相关性描述为精确 span-to-metric 关联。

## 资源身份选择

资源属性按照唯一性和稳定性分层。系统选择当前 span 和 metric series 都可能具有的最强组合，不将不同层级混成一个宽泛的 OR 条件。

优先级如下：

1. `service.name + service.instance.id`
2. `service.name + k8s.pod.uid`
3. `service.name + k8s.namespace.name + k8s.pod.name`
4. `service.name + container.id`
5. `service.name + host.id`
6. `service.name + host.name`
7. `k8s.cluster.name + k8s.node.name`
8. `host.id`
9. `host.name`
10. `service.name`

前六级通常能定位一个运行实例；Node 级、standalone host 和 service-only 属于降级关联，UI 必须展示实际使用的键。没有任何可用身份时不发起 metric 查询，显示明确空状态。

所有属性必须保留 `resource` field context 和 `string` data type。字段名称相同但 context 不同不能视为同一个字段。

## 时间窗口

第一阶段采用选中 trace 起止时间外扩 30 分钟的窗口，并保证最小观察窗口为 60 分钟。这样既能展示 span 前后的趋势，又不会恢复旧实现固定前后各三小时的高成本查询。

时间窗口只用于相关性和展示，不参与资源身份降级。窗口内没有活跃 series 时显示无数据，不应自动扩大到任意历史范围。

## 候选发现

候选发现复用：

```text
POST /api/v5/metrics/stats
```

请求使用资源身份构造 Filter DSL，并按样本数排序。服务端已经负责：

- 只检查指定时间窗口内的原始 metric series；
- 通过 metric metadata 返回类型、单位和 temporality；
- 参数化并编译资源属性过滤条件；
- 返回匹配指标的样本数和 series 数。

Related Metrics 最多展示 6 个候选。候选选择遵循以下确定性规则：

1. 排除 SigInsight/Collector 自监控指标，除非选中 span 本身来自 Collector。
2. 同一 histogram family 优先保留 `.bucket`，避免同时展示 `.count`、`.sum`、`.min` 和 `.max`。
3. 优先展示延迟、错误、请求、CPU、内存、JVM、队列和连接池等诊断指标。
4. 其余候选按样本数降序、metric name 升序补足。

前端候选排序只处理服务端已经限制过的结果，不允许先拉取完整 metric catalog。

## V5 查询映射

每个候选使用一个标准 metric builder query：

| Metric 类型 | Time aggregation | Space aggregation |
| --- | --- | --- |
| Gauge | `avg` | `avg` |
| 非单调 Sum | `avg` | `avg` |
| 单调 Sum | `rate` | `sum` |
| Histogram `.bucket` | `count` | `p90` |

每个 query 必须具有正数 `stepInterval`。Step 由时间窗口按目标约 120 个点计算，并限制在 10 秒到 5 分钟之间。内部时间结构必须使用带单位的 `startMs`、`endMs`、`stepSeconds` 字段；Metrics Stats 边界传毫秒，旧 `GetMetricQueryRange` 适配边界传秒并由其序列化为毫秒，禁止复用无单位的时间变量。

资源过滤必须同时用于候选发现和 `/api/v5/query_range`，防止发现阶段与图表阶段查询到不同的 series 集合。

## 前端交互

Related Signals 提供 `Logs` 和 `Metrics` 两个标签：

- Logs 保持当前精确关联行为。
- Metrics 标签仅在 span 至少具有一个可用资源身份时显示。
- Metrics 顶部显示 `Related by ...`，列出实际关联键和值。
- 每张图显示 metric name、单位、加载状态、错误状态和空状态。
- 单位由纵轴和 tooltip 负责格式化，卡片标题不得直接展示 `s`、`By` 等裸 UCUM 缩写；标题区显示实际查询语义 `P90`、`Rate` 或 `Average`。
- 每张卡片必须为标题栏和图表区分别分配高度，完整显示纵轴、横轴、legend 和 tooltip，不得继承 Explorer 页面使用的 `50vh` 图表高度。
- 提供 `Open in Metrics Explorer`，携带相同查询和绝对时间窗口。
- 图表使用 `PanelVisualization` / uPlotV2，不允许恢复 `components/Uplot` 或 `lib/uPlotLib`。

单个指标失败不能让其他图表消失。候选发现失败应显示可重试错误，而不是退回不带资源过滤的全局指标。

## 性能与安全边界

- 候选发现上限：30；最终图表上限：6。
- 同时执行的 query range 请求应由一个 composite V5 request 承载，避免 N 个独立 HTTP 请求。
- 查询必须包含时间范围、metric name 和资源过滤。
- 不允许使用 `trace_id` 或 `span_id` 作为普通 metric label，以免制造无界基数。
- 不允许直接拼接 SQL；所有过滤必须进入 V5 Filter DSL 和轻量 IR。
- 服务端返回截断 warning 时必须通过现有 warning UI 展示。

## Exemplar 后续边界

精确 span-to-metric 关联需要单独的 Exemplar 里程碑：

1. Collector Metrics Exporter 保存 OTLP exemplar 的 timestamp、value、trace ID、span ID 和 metric series fingerprint。
2. ClickHouse 增加独立 exemplar 表及 retention 策略。
3. 查询层增加按 trace/span ID 获取 exemplar 的只读接口。
4. Metrics 图表支持 exemplar 标记和跳转。

该里程碑不能通过给所有 metric point 增加 trace label 来替代，也不属于本次资源相关性恢复范围。

## 验证标准

实现完成必须覆盖：

1. 资源身份优先级和降级的纯函数测试。
2. Filter DSL 转义、field context 和 data type 测试。
3. Gauge、Sum、非单调 Sum、Histogram 的 V5 query 映射测试。
4. histogram family 去重和候选排序测试。
5. Metrics 标签显示、切换、空状态、发现失败和查询失败测试。
6. 前端类型检查、lint、相关 Jest 测试和 production build。
7. 使用本机 ClickHouse 25.5.6 的真实 span/resource 数据调用 metrics stats 与 query range，确认返回匹配 series。

## 真实协作验证记录

2026-08-18 使用本机 `clickhouse/clickhouse-server:25.5.6`、当前工作树后端和当前前端完成验证：

- Trace 资源身份：`service.name=matreeg_biz`、`service.instance.id=1e14d567-5a5e-4ac3-afcc-6da987fc550e`；
- Metrics Stats 返回 30 个受限候选，并返回 Histogram/Sum/Gauge 的 `temporality` 和 `isMonotonic`；
- 一个包含 6 个 builder query 的 composite `/api/v5/query_range` 请求返回 HTTP 200；
- 6 个候选分别渲染为 6 张 uPlotV2 图表，无 `invalid_input`、浏览器异常或截断 warning；
- 单独验证 `http.server.request.duration.bucket` 返回 1 条 series 和 30 个点。

验证中发现并修复了时间单位边界：Trace Detail 的绝对时间是毫秒，`GetMetricQueryRange` 的兼容入口参数是秒。若将毫秒直接传给该入口，V5 payload 会再次乘以 1000，导致 metadata 查询落在错误时间范围并报告 metric type 不存在。该约束现由带单位字段和纯函数测试固定。
