# M15：基础告警编辑器功能边界与交互设计

状态：Completed；10.1-10.6 已完成

最后更新：2026-08-05

## 1. 目标

将现有 Alert Builder 收敛为面向轻量查询引擎的基础阈值告警编辑器。编辑器继续支持
Logs、Traces、Metrics、Exceptions 和 Meter 场景需要的查询、累计评估、多个数据查询与简单
算术/布尔公式，同时删除复杂频率节奏和日历/自定义调度。累计窗口保留显式时区，默认使用
创建者浏览器的本地时区。

本阶段首先固定产品和协议边界。没有被本文列入的能力不应继续以隐藏字段、兼容分支或禁用
控件的形式保留。

## 2. 已确认决策

### 2.1 保留

- `threshold_rule`，不增加 anomaly、预测或复合规则类型。
- Metrics、Logs、Traces、Exceptions，以及 Metrics 下的 Meter source。
- 多个命名数据查询，以及有类型的算术/比较/布尔公式。
- Rolling 和 Cumulative 两种评估窗口。
- Cumulative 的显式 IANA 时区；新规则默认取创建者浏览器的本地时区。
- 数值阈值、多级严重性、静态标签、静态描述和通知渠道。
- No Data 告警、最小数据点数和按 group-by 生成告警实例。
- 查询预览、规则测试、创建、编辑、启停、状态历史和基本通知。
- 新 Alert JSON 只接受 `schemaVersion: "v3alpha1"`；旧规则允许直接删除，不做版本转换。

### 2.2 删除

- 自定义 cadence、RRULE、指定开始时间、星期选择和重复日程。
- hourly/daily/weekly/monthly `CumulativeSchedule` 及其 minute/hour/day/weekday 参数。
- 日历月累计；月份不是固定时长，保留它会重新引入日历边界和日期异常处理。
- evaluation delay。当前前端只有被注释的未完成功能，不纳入新编辑器。
- anomaly、季节性、EWMA、预测、任意函数链、PromQL、ClickHouse SQL、join、sub-query、
  Trace Operator 和通用 Having。
- 标签/描述模板、规则级 renotify、高级通知模板和旧 Alert JSON 兼容。

## 3. 对“多查询条件”的精确定义

当前实现只支持数值公式，再选择一个输出序列执行阈值判断。目标设计将 Formula IR 扩展为
有类型的数值/布尔表达式：多个数据查询既可以组成数值公式，也可以直接组成布尔条件公式，
最终仍由一个查询或公式输出驱动告警状态。

```text
Query A ─┐
         ├─ Formula F1: A > 10 AND B < 5 ── Boolean policy ── Alert instances
Query B ─┘
```

支持：

- Alert UI 最多 4 个独立数据查询和 4 个公式；共享 Lite 引擎仍保留 8 个独立查询的全局预算。
- 查询名称使用 `A`、`B`、`C`、`D`；公式使用 `F1` 至 `F4`。
- 数值表达式包含查询/数值公式引用、十进制常量、括号、`+ - * /` 和受限内联函数。
- 比较表达式支持 `>`、`>=`、`<`、`<=`、`=`、`!=`，比较两侧必须是单位兼容的数值表达式。
- 布尔表达式支持比较表达式、布尔公式引用、括号和 `NOT`、`AND`、`OR`。
- 公式依赖必须无环；输入必须具有相同时间桶和兼容的分组列。
- 阈值的 `selectedQueryName` 可以指向数据查询或公式。
- 数值输出继续配置数值阈值；布尔输出直接配置 `Last`、`At least once` 或
  `All the time`，不再追加“F1 > 0”这样的伪数值阈值。

不支持：

- 字符串比较、字段访问或把 Filter DSL 嵌入 Formula。
- 隐式 number/bool 转换，例如把 0/1 自动当作 false/true。
- 链式比较，例如 `A < B < C`；必须写为 `A < B AND B < C`。
- 跨信号 join，或在同一规则中混合 Logs、Traces、Metrics 数据源。

公式优先级固定为：括号、算术乘除、算术加减、比较、`NOT`、`AND`、`OR`。`AND`、`OR`、
`NOT` 大小写不敏感，序列化和回显统一为大写；不接受 `&&`、`||`、`!`、`==` 等别名。
每个时间桶和 group-by identity 分别求值。

Formula 的名称仍由查询行单独保存。界面中的 F1 输入框填写 `A > 10 AND B < 5`，而不是把
`F1 =` 也写进表达式。

### 3.1 协议与执行边界

该能力不是现有公式解析器已经支持的语法。当前 Go tokenizer/evaluator 和前端
`FormulaExpressionEditor` 都只处理浮点数与 `+ - * /`。实现时必须使用项目自有的小型 typed
parser，不恢复 `govaluate`，并整体扩展类型边界：

- Formula AST 节点携带 `number` 或 `bool` 静态类型。
- Lite validator 在执行前完成引用类型、比较单位和操作符校验。
- Executor 返回数值或布尔列；V5 result schema 明确标记 bool，不能用 0/1 冒充。
- Alert evaluator 对数值结果进入 Numeric Threshold，对布尔结果进入 Boolean Policy。
- Alert Preview 将布尔序列渲染为 true/false 状态带，而不是数值折线。
- Formula IR 和 V5 Result 对 number/bool 一视同仁；第一阶段仅 Alert Editor 开放布尔构造，
  其他 Lite Query UI 仍只允许数值公式。

这仍然复用同一个 `/api/v5/query_range` 请求和 Formula IR，不另建 Alert 专用表达式 parser。

### 3.2 单位与 Missing 不变量

Formula 在执行前推导结果单位：

- `+`、`-`、比较、`min`、`max` 的两侧单位必须兼容；执行前转换到同一规范单位。
- 兼容单位相除得到无量纲，例如 duration/duration；不兼容的有单位值不能相除。
- 有单位值只能乘除无量纲常量或无量纲公式；不支持两个有单位值相乘。
- 比较表达式中的数值常量继承另一侧单位，例如 `latency > 500` 中 500 使用 latency 的单位。
- `clamp(x, low, high)` 的 low/high 常量继承 x 的单位；非字面量参数必须与 x 单位兼容。
- 公式 result unit 由 AST 推导，前端不能另填一个冲突的 result unit。

Missing 在进入 Formula 前先执行 aggregation-aware zero default：

- count、sum、rate 等加法/计数语义允许将缺失时间桶补为 0。
- avg、min、max、percentile 等统计语义保持 missing。
- 除零、函数定义域错误和仍然缺少的必要输入产生 missing，不产生 false、0 或无穷值。
- 布尔表达式严格传播 missing；`true OR missing` 仍为 missing，进入统一 No Data 策略，不能
  通过短路隐藏遥测中断。

### 3.3 内联函数边界

第一版只支持四个确定性、逐时间点执行且保持单位的数值函数：

| 函数 | 参数 | 单位规则 | 语义 |
| --- | --- | --- | --- |
| `abs(x)` | 1 个 number | 输出单位与 x 相同 | 绝对值 |
| `min(x, y)` | 2 个 number | x/y 单位必须兼容 | 取较小值 |
| `max(x, y)` | 2 个 number | x/y 单位必须兼容 | 取较大值 |
| `clamp(x, low, high)` | 3 个 number | low/high 必须与 x 单位兼容 | 将 x 限制在闭区间内 |

支持函数嵌套，例如 `clamp(abs(A - B), 0, 100)`。函数参数可以是查询引用、公式引用、常量
或数值子表达式。缺失参数值继续传播为 missing；函数不得自行填零。`clamp` 在 low > high 时
返回明确的公式错误，不能交换参数或静默修正。

公式 parser 使用显式 allowlist、固定参数个数和静态类型检查，不通过反射注册 Go `math`
函数，也不接受未知函数名。第一版不加入 sqrt/log/trigonometric/time 函数，避免单位维度、定义域
和非确定时间依赖进入基础告警。

### 3.4 Legacy 结果函数链评估

Aggregate、Formula function 和 Result transform 是三个不同阶段：

```text
ClickHouse rows
  -> Aggregate（count/sum/avg/rate/p95）
  -> Formula（A/B、abs、clamp、布尔条件）
  -> Result transform（EWMA、moving median、time shift 等）
  -> Alert condition / Visualization
```

Aggregate 决定 ClickHouse 如何从原始数据产生每个时间桶的值；结果函数在聚合结果返回后按
时间顺序修改序列。结果函数不是 Aggregate 下拉框中的选项，也不能共享同一个字段。

现有仓库仍保留旧结果函数实现，但 Lite adapter 明确拒绝 query/formula `functions`。不应仅
恢复前端按钮绕过该边界。

| Legacy 函数 | 当前实现作用 | M15 决策 |
| --- | --- | --- |
| `absolute` | 每个点取绝对值 | 不恢复；与 `abs(x)` 重复 |
| `clampMin/Max` | 小于/大于边界的值替换为边界 | 不恢复；由内联 `clamp` 覆盖 |
| `cutOffMin/Max` | 边界外的值替换为 NaN | 不恢复；会改变 No Data 和阈值样本集合 |
| `log2/log10` | 对每个点取对数 | 暂缓；需先定义无量纲单位和非正数行为 |
| `runningDiff` | 当前点减前一点并删除首点 | 不恢复；counter reset 和缺口语义不正确，优先使用 rate/increase |
| `cumulativeSum` | 从查询范围起点开始累计，NaN 不增加总和 | 不恢复；结果依赖任意查询起点，且与 Cumulative evaluation 容易混淆 |
| `ewma3/5/7` | 因果指数移动平均，NaN 时保持上次状态 | 后续候选；需定义预热 lookback、alpha 和 missing 语义 |
| `median3/5/7` | 以当前点为中心做 3/5/7 点中位数，边缘点保持原值 | 不原样恢复；告警需要 trailing window，当前居中窗口不适合实时末端 |
| `timeShift` | 只把结果时间戳平移指定秒数 | 不恢复；正确实现必须由 planner 扩展读取区间并重新对齐 |
| `fillZero` | 在 start/end/step 网格中补 0 | 不恢复；与 V5 fill-gaps 重复并可能掩盖 No Data |
| `anomaly` | 当前开源实现是 no-op 占位 | 删除/拒绝，不作为能力 |

M15 不加入独立 Result transform UI。若基础告警稳定后仍需要降噪，优先单独设计 causal EWMA
或 trailing median，并明确 warm-up 数据预算、窗口单位和 No Data 行为；它们应放在名为
`Transform` 的独立区域，不能混入 Aggregate 下拉框。

## 4. 查询功能边界

告警只消费 `time_series` 或 `scalar` 结果，不对 raw/trace 列表做阈值判断。所有输入继续
经过 `POST /api/v5/query_range` 的 Lite V5 协议和能力校验。

| 数据源 | 基础告警查询能力 |
| --- | --- |
| Metrics Gauge | latest、avg、min、max |
| Metrics Sum | sum、rate、increase |
| Metrics Histogram | p50、p90、p95、p99 |
| Meter | count、sum、avg、rate、increase |
| Logs | count，以及受类型约束的 sum、avg、min、max |
| Traces | count、duration avg、p50、p90、p95、p99 |
| Exceptions | 复用 Traces source，只提供异常过滤后的 count/受支持聚合 |

每个查询保留：

- 一个受数据源约束的聚合。
- Lite Filter DSL 和结构化 Filter 的同一份状态。
- 最多 4 个 group-by 字段。
- 自动计算的 step；告警编辑器不暴露原始 step、limit、order、legend、field order。

## 5. 评估语义

### 5.1 Rolling

Rolling 表示每次评估最近一段固定时长的数据。

```json
{
  "kind": "rolling",
  "spec": {
    "evalWindow": "5m",
    "frequency": "1m"
  }
}
```

- `evalWindow` 提供 5m、10m、15m、30m、1h、3h、6h、12h、24h 预设，也允许输入正整数加
  `m`/`h` 的简单固定时长。
- `frequency` 只提供 30s、1m、5m、10m、15m 预设。
- 必须满足 `frequency <= evalWindow`；后端校验仍是最终权威。

### 5.2 Cumulative

Cumulative 表示从所选时区最近一个固定周期边界累计到当前评估时刻。时区决定“当前小时、
当前天、当前周”的边界，但用户不能指定任意开始时间或日历计划。

```json
{
  "kind": "cumulative",
  "spec": {
    "period": "1d",
    "frequency": "5m",
    "timezone": "Asia/Shanghai"
  }
}
```

- `period` 只支持 1h、1d、7d。
- 周期边界固定为所选时区的整点、00:00 和周一 00:00。
- `timezone` 必须是明确的 IANA 名称，例如 `Asia/Shanghai`，不能保存为 `local`、UTC offset
  或部署服务器时区。
- 新建规则默认使用 `Intl.DateTimeFormat().resolvedOptions().timeZone` 得到的浏览器本地
  时区，并把解析后的 IANA 名称写入规则；换浏览器或换服务器不会改变已有规则。
- 浏览器无法提供有效 IANA 时区时使用 UTC，并在表单中明确显示回退，不能静默处理。
- DST 切换日允许一天实际包含 23 或 25 小时，这是本地日累计的正确语义，必须由后端测试。
- 不支持 1 month、某月第几天、可选星期几或每天任意开始时间。
- `frequency` 使用与 Rolling 相同的简单预设，且必须小于 `period`。

这意味着当前后端 `CumulativeWindow{Schedule, Frequency, Timezone}` 需要在实现阶段替换为
`FixedCumulativeWindow{Period, Frequency, Timezone}`。项目已允许丢弃旧规则数据，不需要兼容
旧 schedule JSON。

### 5.3 阈值窗口归约

保留以下 Match Type：

- Last：最后一个点。
- On average：窗口平均值。
- In total：窗口求和。
- At least once：任意一个点满足。
- All the time：所有有效点都满足。

数值阈值和布尔公式统一使用 `>`、`>=`、`<`、`<=`、`=`、`!=`；outside-bounds 删除并通过
`A < low OR A > high` 表达。阈值必须是与查询结果单位兼容的数值。No Data 和最小数据点数
在归约前判定，不能通过不安全填零或 false 掩盖数据缺失。

## 6. 前端信息架构

编辑器使用一个连续页面和固定底部操作栏，不再用四个看似独立、实际共享状态的 Stepper。
信息从上到下只分为四个无嵌套区域。

### 6.1 基本信息

- 告警名称：必填，失焦和保存时校验重复/空值。
- 数据源：Metrics、Logs、Traces、Exceptions；Meter 在 Metrics 查询 source 中选择。
- 静态标签：键和值输入，键自动补全已有告警标签名；拒绝模板语法。

编辑已有规则时数据源不可直接切换。需要切换时提示“将清空查询和公式”，由用户明确确认。

### 6.2 查询与预览

- 上方是稳定高度的时间序列预览，支持选择预览时间范围和手动刷新。
- 下方按 A、B、C、D 展示查询行；每行只显示当前数据源允许的控件。
- `Add query` 和 `Add formula` 并列，达到预算后禁用并显示原因。
- 查询/公式行支持禁用、复制和删除；最后一个有效查询不可删除。
- 预览图例显示查询名、单位和 group-by 标签，点击可聚焦对应查询行。
- 未运行的修改显示“有未应用更改”，不会把旧预览误认为新查询结果。

### 6.3 条件与评估

使用可扫描的条件句，而不是把所有状态塞进一个长句：

```text
WHEN       [A ▾] [On average ▾]
IS         [Above ▾] [80] [%]
OVER       [Rolling ▾] [5 minutes ▾]
EVALUATE   [Every 1 minute ▾]
```

当选中布尔公式时，条件自动收敛为：

```text
WHEN       [F1 ▾] [At least once ▾] IS TRUE
OVER       [Cumulative ▾] [Current day ▾] [Asia/Shanghai ▾]
EVALUATE   [Every 5 minutes ▾]
```

布尔结果不显示数值操作符、阈值或单位。切换 Cumulative 后显示可搜索的 IANA 时区选择器，
默认标注“本地时区”；界面仍不出现开始时间、日期、星期选择或 RRULE 控件。

数值输出的严重性阈值作为同一条件下的行列表呈现，默认 Critical，可增加 Warning 和 Info。
每行包括严重性、目标值、可选恢复值和通知渠道；最多三个内置级别，不允许无限新增匿名
级别。布尔输出只有一个严重性和通知渠道，因为同一个 true/false 结果配置多个严重性会同时
触发；需要不同严重性条件时创建不同规则。

No Data 和最小数据点数放在“数据质量”折叠区。它们仍是基础正确性设置，不与 cadence 混在
“高级选项”中。

### 6.4 通知与保存

- 每个严重性选择一个或多个已有通知渠道。
- group-by 通知分组只允许从当前查询的 group-by 字段中选择。
- 描述为静态纯文本，不提供模板变量或函数。
- 底部固定操作：`Test rule`、`Save`、`Cancel`。
- Test 必须使用当前未保存草稿，并返回查询窗口、选中输出、归约值、阈值结果和 No Data 状态，
  不能只显示“成功/失败”。
- Test Rule 响应使用结构化 `evaluationPreview`：包含实际 start/end、frequency、输出类型、每个
  group identity 的最终值/状态、match policy、命中严重性、No Data 原因和查询 warning。

## 7. 补全设计

### 7.1 Filter 补全

Alert Editor 直接复用 `QueryBuilderSearchV3`，不新建第二套 Filter 编辑器。

- 字段候选来自 Lite capability catalog、遥测字段元数据和当前 metric/meter 维度。
- 展示名允许输入 `http.route`，插入值必须是明确上下文的
  `attribute.http.route`、`resource.service.name` 等规范字段。
- 根据字段类型补全合法操作符；bool 字段只建议 bool 合法操作，数值字段不建议 LIKE。
- 字符串值查询后端建议，bool 只建议 `true`/`false`，枚举字段展示可选值。
- 输入 `AND`/`OR` 后继续字段补全；混合 AND/OR、括号和不支持语法立即在输入框下方报错。
- 保持连续文本编辑体验，支持复制、粘贴、撤销和键盘选择，不把条件转换成可关闭的小标签。

### 7.2 聚合与字段补全

聚合不是自由文本函数调用，而是由数据类型驱动的选择器：

1. 先选择 metric/字段。
2. 读取 signal、metric type、temporality 和字段类型。
3. 只展示能力矩阵允许的 aggregation。

例如 Gauge 不显示 rate，Sum 不显示 p95，Histogram 只显示明确支持的分位数。候选项同时展示
函数名、返回单位和一句语义说明，避免用户靠试错理解函数。

### 7.3 Formula 补全

Formula 使用单行代码输入框，输入或按 `Ctrl+Space` 时按当前期望类型补全：

- 当前可引用的 A-D 和已经定义、无循环风险的 F1-F4。
- 每个候选显示结果类型、数据源、结果单位和 group-by 签名。
- 数值位置补全十进制常量、数值查询/公式和 `+ - * /`。
- 比较位置补全 `> >= < <= = !=`；布尔位置补全 `NOT AND OR` 和布尔公式。
- 数值操作数位置补全 `abs`、`min`、`max`、`clamp`；接受函数后插入完整括号与参数占位，
  并把光标放到第一个参数。
- 补全项必须根据类型过滤，不能在数值运算符后建议布尔公式。
- 非 Alert 场景不建议比较或布尔操作符；后端仍按同一 typed Formula IR 校验请求。

函数候选展示签名、参数说明、返回类型和单位规则；输入逗号后根据参数位置继续补全兼容查询、
公式和常量。`sum`、`avg`、`rate`、`p95` 属于数据查询的聚合选择器；`Last`、`On average`、
`In total` 属于阈值窗口归约。三者必须在 UI 上分区，避免同名函数在不同阶段产生歧义。

输入过程中执行增量校验：未知引用、循环引用、括号不配对、除数为字面量零、number/bool
类型错误、比较单位不兼容和 group-by 签名不兼容都显示在公式行内。错误公式不能运行预览、
测试或保存。

### 7.4 补全交互约定

- 键盘：上下键选择，Enter 接受，Esc 关闭，Tab 只在候选打开时接受补全。
- 补全不得截获 `Ctrl/Cmd+C`、`Ctrl/Cmd+V`、`Ctrl/Cmd+Z`。
- 异步字段/值建议显示 loading；请求失败时保留本地 catalog 候选，不清空用户文本。
- 自动补全只帮助构造合法输入，不在保存时静默改写不支持的表达式。

## 8. 前端状态与协议边界

新编辑器应使用一个领域草稿状态，而不是继续维护 `alertState`、`thresholdState`、
`advancedOptions`、`evaluationWindow` 和 `notificationSettings` 五组互相修正的 reducer。

```ts
type BasicAlertDraft = {
  identity: AlertIdentity;
  query: LiteCompositeQuery;
  condition: NumericThresholdCondition | BooleanCondition;
  evaluation: RollingEvaluation | FixedCumulativeEvaluation;
  dataQuality: DataQualityPolicy;
  notification: BasicNotificationPolicy;
};
```

UI 状态转换只发生一次：`BasicAlertDraft -> PostableRule`。预览、Test 和 Save 共用同一个
serializer 和同一套 validator，禁止分别拼装三种近似 payload。

目标 `PostableRule` 使用 `schemaVersion: "v3alpha1"`，其 condition 是明确的
`NumericThresholdCondition | BooleanCondition` 判别联合。`v2alpha1`、旧 cumulative schedule
和旧 condition JSON 不读取、不转换；部署迁移可以直接删除旧规则数据。

`POST /api/v5/testRule` 成功响应的 `data` 必须包含结构化评估预览：

```json
{
  "evaluationPreview": {
    "alertCount": 2,
    "state": "firing",
    "evaluatedAt": 1780000000000
  }
}
```

`state` 使用 `inactive|pending|firing|nodata|recovering|disabled`，`evaluatedAt` 为 Unix
milliseconds。Test 保留测试通知副作用；该预览只描述当前隔离评估，不能当作已保存规则的持久状态。

## 9. 验证与删除门槛

实现阶段至少覆盖：

1. Rolling 与带 IANA 时区的固定 Cumulative 边界、频率、DST 和跨周期行为。
2. A/B 查询与 F1/F2 的数值/布尔类型、函数参数/嵌套、canonical 语法、优先级、依赖、循环、
   单位转换、group-by 对齐、aggregation-aware zero default、missing 和除零。
3. Metrics、Meter、Logs、Traces、Exceptions 的可创建、测试、触发和编辑。
4. Last/average/total/once/all、No Data、最小点数和多严重性阈值。
5. Filter、aggregation、formula 补全的键盘、剪贴板、异步失败和非法输入行为。
6. 前端 payload 与 Go `PostableRule v3alpha1`、typed V5 result 和结构化 Test Rule 响应的契约
   fixture；明确拒绝 v2alpha1、操作符别名和类型不匹配。
7. ClickHouse 25.5.6、真实 SQLite、规则调度器和通知 webhook 的协作测试。

完成验证后删除：

- 当前 `EvaluationCadence`、custom/rrule 编辑器、任意开始时间和日历计算组件；保留精简的
  IANA 时区选择器。
- 前端 `AdvancedOptionsState.evaluationCadence`、`EvaluationWindowState.startingAt` 等过渡状态。
- 后端 `CumulativeSchedule`、`ScheduleType` 和任意日历计划；保留 IANA timezone 校验。
- `CreateAlertV2`/`EditAlertV2`/`FormAlertRules` 中被统一编辑器替代的包装与重复序列化逻辑。

提交准则沿用轻量查询引擎路线图：每个实现提交必须包含实际删除、解除删除阻塞的领域抽取，
或证明删除安全的测试。

## 10. 实施阶段与提交锚点

实现按以下宏观阶段推进。每个阶段独立提交，后一阶段不得通过临时兼容分支绕过前一阶段的
类型或协议约束。

### 10.1 Typed Formula Core

- 实现项目自有 tokenizer、parser、typed AST、canonical serializer 和依赖图校验。
- 固化 number/bool、单位推导、group-by 对齐、missing 传播和除零语义。
- 为语法优先级、非法别名、循环引用、单位冲突和 missing 真值表增加 Go 单元/模糊测试。

提交锚点：`feat(formula): add typed alert expression core`

完成记录：项目自有 parser、typed AST、canonical serializer、公式依赖图、number/bool 类型、
基础单位兼容/换算、series signature 对齐与严格 missing 传播已实现并由单元及 fuzz 测试覆盖。
四个内联函数、V5 bool result 和 Alert 协议仍属于 10.2 及后续阶段，不能据此向当前 V5 UI
开放布尔公式。

### 10.2 V5 Typed Result 与内联函数

- 将 Formula IR、executor 和 V5 Result 扩展为明确的 number/bool 判别类型。
- 实现 `abs`、`min`、`max`、`clamp`，并保持 arity、类型、单位和定义域校验一致。
- 保持非 Alert Lite UI 的数值公式边界，不恢复 legacy result function chain。

提交锚点：`feat(query): execute typed alert formulas`

完成记录：`abs`、`min`、`max`、`clamp` 已在同一 typed parser/evaluator 中实现；Lite Plan
保存已检查 Formula Program，执行器不再维护旧 float-only tokenizer/evaluator。`QueryResult` 和
V5 `time_series`/`scalar` 响应以 `valueType: number|bool` 明确结果类型；bool point 使用
`boolValue`，缺失 bool 不参与 fill-gaps，不能被伪造为 `false`。当前通用 Query Builder 仍只
开放数值 Formula；布尔 Formula 和函数补全仅会随 10.4 Basic Alert Editor 对 Alert UI 开放。

### 10.3 Alert v3alpha1 领域模型

- 引入 `NumericThresholdCondition | BooleanCondition` 和固定 Rolling/Cumulative evaluation。
- 实现显式 IANA timezone、1h/1d/7d 累计边界及结构化 `evaluationPreview`。
- 删除 v2alpha1 读取/转换、旧 cumulative schedule、复杂 cadence、renotify 和模板协议残留。

提交锚点：`refactor(alerts): establish v3alpha1 rule contract`

完成记录：对外 Alert JSON 已收敛为仅接受 `schemaVersion: "v3alpha1"` 的判别联合。
`condition.kind` 明确区分 `numeric` 与 `boolean`：数值条件使用受限归约、比较运算符和一至三个
严重性阈值；布尔条件只允许 `last`、`at_least_once`、`all_the_time` 三种策略。No Data 使用可保留
秒级精度的 `noDataFor`，不再经旧的分钟整数截断。

`cumulative` 仅接受 `{period: 1h|1d|7d, frequency, timezone}`，以 IANA 时区的整点、当地午夜和
周一午夜计算边界，并有 UTC、上海和纽约 DST 边界测试。旧 `schedule`、`unit`、`thresholds`、
`v2alpha1`、renotify 和模板字段均在 JSON 解码阶段明确拒绝。规则 evaluator 仍通过一个未序列化的
内部适配器复用既有状态机；该适配器不是旧协议兼容层，也不会接受或输出旧 JSON。

### 10.4 统一 Basic Alert Editor

- 以单一 `BasicAlertDraft` 替换分裂的 Stepper/reducer 状态。
- Preview、Test、Save 共用 serializer/validator；实现数据源约束、数值/布尔条件和时区交互。
- 复用 QueryBuilderSearchV3，并实现查询、函数、操作符、类型、单位和参数位置感知的 Formula
  补全。

提交锚点：`refactor(frontend): consolidate basic alert editor`

完成记录：`BasicAlertEditor` 用一个 `BasicAlertDraft` 和一个 serializer 覆盖新建、编辑和 Test。
它只发出 v3 JSON，限制为 4 个数据查询与 4 个公式；数值/布尔条件、固定累计窗口、IANA timezone、
No Data、最小点数、静态标签和 group-by 都由同一份 validator 约束。Alert 公式补全开放比较、
布尔运算和 `abs/min/max/clamp`，且不截获复制、粘贴或撤销快捷键。

Exceptions 不再携带旧 `error_index_v2` 原生 SQL：默认值是 `traces` Lite 查询加
`has_error = true` Filter。这样保留异常告警语义，同时不恢复原生 SQL 能力。

### 10.5 Legacy UI 与调度实现删除

- 删除被统一编辑器替代的 Create/Edit 包装、calendar/RRULE/custom cadence 控件和重复样式测试。
- 删除后端任意日历计划、旧状态转换和只服务旧 JSON 的 helper。
- 用代码量、生产 import 和路由审计证明旧树没有残余入口。

提交锚点：`refactor(alerts): remove legacy builder and schedules`

完成记录：已删除 `CreateAlertV2`、`EditAlertV2`、`FormAlertRules`、旧 create/test/update hooks、
v2 rule JSON 类型和原生 SQL 默认告警。列表编辑只传递 rule id，由编辑页重新读取服务端 v3 rule；
复制会明确拒绝非 v3 规则，不再进行隐式转换。详情页标题直接消费 v3 读取模型，历史、启停和删除
不再依赖旧编辑器 context。

### 10.6 协作验证闭环

- 运行 Go、前端类型、lint、组件和契约测试。
- 使用 ClickHouse 25.5.6、真实 SQLite、规则调度器和 webhook 覆盖五类数据源。
- 使用浏览器覆盖创建、补全、预览、Test、编辑、触发、No Data、DST 和历史页面，并记录
  可重复执行的验证命令及结果。

提交锚点：`test(alerts): verify basic editor collaboration`

完成记录（2026-08-05）：

- [run-basic-alert-source-collaboration.sh](../../../../tests/integration/scripts/run-basic-alert-source-collaboration.sh)
  会启动当前工作区编译出的源后端，使用新的临时 SQLite、`19091` 指标端口和配置的 ClickHouse。
  它不读取或写入现有 SigInsight SQLite，结束时关闭源进程并删除临时目录。
- 在 ClickHouse `25.5.6` 和运行中的 Collector 数据上，该脚本验证了认证后的 `query_range`：Metrics、
  Logs、Traces 以及 `has_error = true` Exceptions；随后对四个信号分别执行 `testRule`，验证每次都
  返回结构化 `evaluationPreview`。它创建只监听 `127.0.0.1` 的临时 webhook channel，并用一个确定
  无数据的 Logs 规则验证 `nodata` preview、实际 webhook 回执和通知渠道配置。Metrics 规则还完成了
  create、read、edit、history timeline read-back、delete。
- 可复现命令（从仓库根目录执行，`8080` 必须空闲）：

  ```bash
  SIGINSIGHT_TELEMETRYSTORE_CLICKHOUSE_DSN=tcp://<clickhouse-host>:9000 \
    ./tests/integration/scripts/run-basic-alert-source-collaboration.sh
  ```

  默认指标为正在运行 Collector 产生的
  `http.server.request.duration.sum`；其他环境可以用
  `SIGINSIGHT_M15_METRIC_NAME=<cumulative-sum-metric>` 覆盖。实际输出为：

  ```text
  basic alert source collaboration passed
    ClickHouse: configured telemetry store
    SQLite: fresh temporary database and migrations
    Query API: metrics, logs, traces, exceptions
    Alert API: four-signal previews, no-data state, local webhook, history read-back; metrics create, read, edit, delete
  ```

- 浏览器覆盖在 Vite 源前端指向同一源后端时执行：新建 Metrics v3 规则、选择 Sum 指标、配置阈值和
  一次性本地 webhook、保存并在 `finally` 中删除规则。该 Playwright 用例默认跳过，只有显式提供
  真实数据环境时运行，避免 CI 依赖外部遥测：

  ```bash
  VITE_FRONTEND_API_ENDPOINT=http://127.0.0.1:8080 yarn dev --host 127.0.0.1 --port 3302

  LOGIN_USERNAME=<temporary-admin-email> \
  LOGIN_PASSWORD=<temporary-admin-password> \
  BASIC_ALERT_E2E_METRIC=http.server.request.duration.sum \
  BASIC_ALERT_E2E_CHANNEL=<disposable-local-webhook-channel> \
  SIGINSIGHT_E2E_BASE_URL=http://127.0.0.1:3302 \
  SIGINSIGHT_E2E_API_BASE_URL=http://127.0.0.1:8080 \
  yarn --cwd frontend playwright test e2e/tests/alerts/basic-alert-editor.spec.ts \
    --project=chromium --workers=1 --timeout=30000 --reporter=list
  ```

  本次执行结果为 `2 passed`。通知渠道使用只绑定 `127.0.0.1` 的一次性 webhook；测试规则随后被删除，
  不会向外部系统发送通知。

- 本次真实保存还发现并修复了一个协议不变量：关闭 No Data 告警时，前端以前仍会发送 `noDataFor: "5m"`，
  而后端正确拒绝这种组合。serializer 现在只在 `alertOnNoData=true` 时发送该字段；读取时仍以编辑器默认值
  补齐缺失的 UI 状态。该行为有前端单测，并由上述真实创建/编辑路径覆盖。

- 补充的 Go 回归测试从 v3 JSON 边界构造 `30s` No Data 规则，并验证空聚合序列会触发一个 `nodata`
  preview、带 `nodata=true` 与 `testalert=true` 标签的测试通知，以及跨多个严重性渠道的稳定去重路由。
  该测试同时固定了三项正确性约束：空 series 不是有效数据；`labels` 可以在 API 请求中省略；No Data
  是规则级状态，必须向规则全部已配置的通知渠道发送，而不是因为没有 threshold label 而丢弃渠道。
- `testRule` 的通知回调现在是仅用于该 API 的同步错误边界：Alertmanager 未就绪、缺失 receiver 或
  webhook 投递失败会返回内部错误，不再记录日志后谎报“notification sent”。常规调度仍保持原有的
  best-effort 通知语义；回归测试同时验证这两条边界没有混淆。
