# 轻量查询引擎路线图

状态：Accepted
最后更新：2026-07-31

## 总体依赖

```text
M0 -> M1 -> M2 -> M3 -> M4 -> M5 -> M6 -> M7 -> M8 -> M9 -> M10 -> M11 -> M12 -> M13
```

M2 的 Logs 和 Traces compiler 可以分子提交实现。M3 必须在 M2 建立的 Schema Catalog 和 Compiler contract 上完成，禁止形成另一套 Metrics 专用基础设施。

## M0：能力与兼容基线

目标：把“核心 UI 可以工作”转化为可执行的请求、响应、schema 和规模基线。

任务：

- 采集 Logs、Traces、Metrics、Meter、Services、Dashboard 和 Alert 的 V5 请求样本。
- 标记每个字段属于保留、迁移转换、明确拒绝中的哪一类。
- 固化 V5 响应 fixture 和前端 View Model 结果。
- 从 ClickHouse 25.5.6 导出核心表 schema fingerprint。
- 记录当前 query engine 与前端 Query Builder 的 production LOC、测试、消费者和最大文件。
- 建立真实 ClickHouse/SQLite/Collector/SigInsight 验证入口。

退出条件：

- 每个保留页面至少有一个成功 fixture 和一个错误 fixture。
- 功能矩阵不存在“待定但阻塞 IR”的项目。
- 当前 schema、模块规模和生产依赖可以由命令重复生成。

提交锚点：`docs(query): establish lightweight engine baseline`

## M1：Lite Query Core

目标：建立完全不依赖 ClickHouse 的查询语言核心。

任务：

- 定义 QueryRequest、signal spec、Filter AST、aggregation、formula 和 ResultType。
- 实现严格 JSON 解码、类型恢复、规范化和领域错误。
- 实现 capability validation 和 query budget。
- 实现 Logical Plan 与 planner contract。
- 建立从受支持 V5 请求子集到 Lite IR 的兼容转换。

退出条件：

- IR、校验和 planner 具有表驱动测试、边界测试和 fuzz 测试。
- 包中不包含 ClickHouse 表名或 SQL 生成。
- 所有排除能力都有稳定的 unsupported error。

提交锚点：`feat(query): introduce lightweight query IR and planner`

## M2：Logs 与 Traces Compiler

目标：使用共同的 Catalog/Compiler contract 完成核心日志和链路查询。

任务：

- 实现版本化 Schema Catalog 和 schema fingerprint 检查。
- 实现参数化 filter、group、order、pagination 编译基础设施。
- 实现 Logs raw/time-series/scalar compiler。
- 实现 Traces raw/trace/time-series/scalar compiler。
- 明确 trace 归并、排序、cursor 和时间范围语义。
- 建立 golden SQL、恶意输入、真实 ClickHouse 测试。

退出条件：

- ClickHouse 25.5.6 执行所有核心查询矩阵。
- SQL 值参数不通过字符串拼接进入语句。
- Trace 列表能够打开现有 Trace Detail。

提交锚点：`feat(query): compile lightweight log and trace plans`

## M3：Metrics 与 Meter Compiler

目标：实现 gauge、sum、histogram 和 Meter 的常用查询。

任务：

- 实现 metric type/temporality 元数据解析。
- 实现 Gauge、Sum、Histogram 操作矩阵。
- 实现显式 time aggregation 和 space aggregation 计划。
- 实现 Meter source 与 metric discovery。
- 实现 histogram quantile、rate/increase 和 fill gap 语义测试。
- 使用真实 OTLP 数据和 Cost Meter 查询验证。

退出条件：

- Metrics Explorer 和 Cost Meter 的目标 fixture 全部通过。
- 不使用通用 secondary aggregation 或任意 aggregation expression。
- 不支持的 metric 类型和操作组合返回明确错误。

提交锚点：`feat(query): compile lightweight metrics and meter plans`

## M4：执行、结果与基本告警

目标：形成可由 API 和 rule evaluator 使用的完整后端链路。

任务：

- 实现 executor、超时、取消、并发和执行统计。
- 实现 raw、trace、time-series、scalar 的 V5 Result 映射。
- 实现 aggregation-aware fillZero。
- 实现简单算术 formula evaluator。
- 将 threshold rule 接到新查询接口。
- 统一领域错误到 HTTP/alert 状态的映射。

退出条件：

- `/api/v5/query_range` 可通过受支持 fixture 完整执行。
- 单查询和公式告警可以预览、评估、触发和恢复。
- 请求取消不会留下后台查询或错误状态。

提交锚点：`feat(query): execute lightweight queries and threshold alerts`

## M5：V5 兼容桥与受控 API 接入

目标：将受支持的 V5 请求明确转换到 Lite IR，并以可回退方式接入现有 API。

任务：

- 实现 V5 request -> Lite IR 和 Lite result -> V5 response adapter。
- 将 ClickHouse driver row adapter 放在 `pkg/querier` 基础设施边界。
- 使用受控配置开关逐步使 `/api/v5/query_range` 走新引擎。
- 显式识别并拒绝适配器不支持的 V5 字段；迁移期由调用方回落旧引擎。
- 继续使用现有 metric metadata store 解析 type/temporality，但不将其依赖带入 core。
- 用 V5 DTO、golden SQL 和 ClickHouse 25.5.6 验证转换、执行和响应整形。

退出条件：

- 开关关闭时原 API 行为不变；开关开启时每个受支持请求只走 Lite engine。
- 适配器不会忽略不支持的字段、函数或格式选项。
- Logs、Traces、Metrics、Meter 和 formula 的最小 V5 fixture 可在真实环境返回前端可消费的响应。
- 兼容层和基础设施层具有直接单测，core 保持无 V5/ClickHouse driver 依赖。

提交锚点：`feat(query): bridge V5 requests to lightweight engine`

## M6：轻量前端查询 UI

目标：使用能力模型驱动新的查询编辑体验，不再让 UI 构造无效组合。

任务：

- 实现 signal、source、field、filter、aggregation、group/order/limit 编辑器。
- 为 Gauge、Sum、Histogram、Meter 提供类型相关操作选项。
- 实现简单 formula 和 threshold alert 编辑。
- 接入 Logs Explorer、Traces Explorer、Metrics Explorer、Cost Meter。
- 接入 time-series、table、value 和 Trace list visualization。
- 移除新页面中的 Trace Operator、raw SQL、Having DSL 和函数链入口。

退出条件：

- UI 生成的每一种请求都有协议测试。
- 浏览器回归覆盖成功、空结果、无效输入、后端不可达和超时。
- 新 UI 不 import 旧高级 Query Builder 控件。

提交锚点：`feat(frontend): add lightweight telemetry query experience`

## M7：协作验证与切换

目标：逐个消费者替换旧引擎，并用真实环境证明兼容性。

任务：

- 新旧引擎对同一 fixture 双跑，比较值、标签、时间桶和分页。
- 转换可兼容的保存查询；明确标记无法迁移的高级查询。
- 切换 Dashboard、Services、Explorer、Alert 消费者。
- 检查 ClickHouse query log 中的 SQL、错误、read rows 和 read bytes。
- 仅在证据支持时修改 projection、物化视图、表或列。
- 跨仓库验证 Collector 写入和 SigInsight 读取。

退出条件：

- 所有指定 API 的真实认证请求通过。
- query log 无未知表、未知列和非预期全表扫描。
- 生产代码默认使用新引擎，旧引擎只剩待删除引用。

提交锚点：`refactor(query): switch core consumers to lightweight engine`

## M8：删除与量化收敛

目标：删除被替代的实现，并形成可用于工程评估和论文的数据。

任务：

- 删除旧 signal compiler、通用 postprocess 和过渡 adapter。
- 删除复杂 Query DTO、Trace Operator、函数链和相关前端控件。
- 删除旧测试、mock、样式和不可达分支。
- 执行生产 import、路由、动态 import 和生成代码引用检查。
- 执行全量 Go、前端 build/test、Playwright 和真实协作验证。
- 输出旧/新 LOC、复杂度、测试覆盖、延迟和扫描量对比。

退出条件：

- 旧引擎和排除能力生产引用为零。
- 全量验证通过，两个仓库工作区干净。
- 最终报告能够说明功能损失、复杂度收益和性能变化。

提交锚点：`refactor(query): remove legacy query engine and report results`

## M9：物化列加速查询

目标：在保持 Catalog 静态确定性契约（ADR-003）的前提下，将已存在的物化列显式纳入
查询路径并量化收益，输出低频物化列删除候选清单。

任务：

- 从基线 DDL 提取物化列，生成与 schema fingerprint 绑定的显式物化目录（ADR-011）。
- `Catalog.Resolve` 支持"manifest 命中物化列、非 manifest 字段走 Map"的确定性解析。
- 覆盖 manifest 命中的物化列路径和非 manifest 字段的 Map 路径，并以真实 ClickHouse
  执行测试验证。
- 同 fixture 双路径性能对比（延迟、read rows、read bytes），输出删除候选或明确无候选。

退出条件：

- manifest 列与非 manifest 字段的核心查询矩阵全部通过。
- fingerprint 变化不能导致目录静默过期或部分加载；缺少 manifest 列的 schema 被视为
  不兼容。
- 对比报告记录 p50/p95 延迟、read rows、read bytes，并给出删除候选清单或明确无候选。

提交锚点：`perf(query): resolve materialized columns through explicit catalog`

## M10：退役不可达的 Legacy 编辑渲染树

目标：删除 Lite Query Builder 已替代、且没有生产入口的旧编辑器渲染组件，同时保留仍由
Explorer、详情页和保存查询兼容层使用的 DTO、状态、自动补全及独立筛选控件。

任务：

- 证明 `components/QueryBuilder/QueryBuilder.tsx` 不再渲染旧 Query/Formula/函数编辑组件。
- 删除旧 Query、Formula、聚合、Having、函数链、数据源切换及它们专属样式和测试。
- 将仅为旧 Query 组件存在的类型下沉到共享 operations 类型中。
- 以 production import 扫描、TypeScript 编译和 Lite Builder 测试验证删除。

退出条件：

- 旧编辑渲染树没有静态或路由入口。
- Provider、V5 DTO、metadata autocomplete 和独立筛选控件保持可编译、可测试。
- 删除清单与保留依赖在阶段文档中可审计。

提交锚点：`refactor(frontend): remove unreachable legacy query editor`

## M11：清理无调用的 Legacy QueryBuilder Helper

目标：在不触及仍服务于 metadata 和专用读取路径的 parser/field mapper 前提下，删除无
生产调用的旧 V5 helper。

退出条件：

- CTE、collision、Having、矛盾检测与旧 aggregation 注册表没有代码或测试残留。
- 保留的 field collision、where-clause parser、key selector 和时间归一化通过直接包测试。

提交锚点：`refactor(query): remove unused legacy builder helpers`

## M13：收敛 V2 图表边界与 Container 编排层

目标：在旧 uPlot 已删除的前提下，消除 `PanelWrapper`、`TimeSeriesView` 与
`PanelVisualization` 的重复适配和环依赖，保留 V2 单一渲染栈及核心图表能力。

设计、迁移顺序、删除门槛和验证矩阵见
[M13 阶段文档](milestones/m13-visualization-container-consolidation.md)。

提交锚点：`refactor(visualization): consolidate panel rendering boundary`

## 阶段提交准则

每个生产代码提交至少满足一项：

1. 交付一个已写入阶段文档的可验收能力。
2. 抽取并解除一个明确的旧代码删除阻塞。
3. 实际删除旧实现或无入口能力。
4. 增加能证明后续删除安全的测试或验证脚本。

纯文件移动、只增加抽象但不形成能力、没有删除计划的拆包不进入该重构分支。
