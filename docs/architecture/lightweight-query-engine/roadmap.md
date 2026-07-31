# 轻量查询引擎路线图

状态：Accepted
最后更新：2026-07-31

## 总体依赖

```text
M0 -> M1 -> M2 -> M3 -> M4 -> M5 -> M6 -> M7
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

## M5：轻量前端查询 UI

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

## M6：协作验证与切换

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

## M7：删除与量化收敛

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

## 阶段提交准则

每个生产代码提交至少满足一项：

1. 交付一个已写入阶段文档的可验收能力。
2. 抽取并解除一个明确的旧代码删除阻塞。
3. 实际删除旧实现或无入口能力。
4. 增加能证明后续删除安全的测试或验证脚本。

纯文件移动、只增加抽象但不形成能力、没有删除计划的拆包不进入该重构分支。
