# M14: 删除运行时兼容与不可达产品残留

状态：完成

## 决策

本阶段允许丢弃升级前保存的查询、告警 JSON 和 Dashboard 状态，不提供数据迁移、只读
回退或旧格式恢复。新的 Saved View 和 Alert 仍使用当前受支持的 Lite V5 子集。

## 删除边界

### 旧保存查询兼容

- 删除旧 Saved View V4 -> V5 SQL migration 和 `pkg/transition` 转换器。
- 删除前端对不受支持保存查询的替换提示；Lite Query Builder 只接受当前模型。
- 删除已退役 Trace Operator 的 DTO、校验和“解析后拒绝”路径。
- 保留 `/api/v5/query_range`、当前 Builder Query/Formula DTO、Saved View CRUD 和当前 URL
  查询状态。

### Dashboard 残留

- Dashboard 路由、后端模块和表已在 `7fe7f6f` 删除。
- 删除没有生产写入入口的 Dashboard variable store、layout/lock store 和相关无效分支。
- 将仍由 Home、APM、Alert Preview 使用的 panel 类型和渲染 helper 留在 visualization
  边界，不能按目录名误删。

### Alert 兼容与高级通知

- 删除旧 Alert JSON schema 反序列化、schema version 转换和旧类型分派。
- 删除 renotification 条件和高级通知模板编辑能力。
- 保留当前 threshold rule 的查询、评估、状态历史、通知渠道和基础描述。
- 旧规则数据允许删除；不为旧规则提供只读或兼容编辑器。

### Metrics 元数据编辑

- 删除 Metrics UI 中 description/unit/type/temporality/monotonicity 的编辑入口。
- 删除 UpdateMetricMetadata HTTP、module、SQL 写入、历史覆盖表读取和写后缓存更新。
- 保留元数据读取，因为 Summary、Metric Details、编译类型解析和自动补全仍消费它。

## 验收

1. 生产 import、路由和 handler 扫描无被删除能力引用。
2. 当前 Logs、Traces、Metrics、Meter、Saved View 和基础 Alert 可编译并通过直接测试。
3. 前端类型检查、测试与 production build 通过；Go 全量测试和 lint 通过。
4. ClickHouse 25.5.6/SQLite 真实协作脚本通过，查询日志无 SQL exception。
5. 每个提交记录删除前后生产与测试 LOC，不以纯移动作为收益。

## 进度

### 旧保存查询兼容

- V4 -> V5 Saved View 数据迁移、Trace Operator DTO/grammar、前端兼容状态与替换提示已删除。
- OpenAPI、生成的 TypeScript client、集成 fixture 和能力矩阵不再声明该 query type。
- 保留 Builder Query/Formula 的原有 DTO 测试，并增加旧类型无法解码的边界测试。
- 本阶段净删除约 2,800 行；精确数字以里程碑提交的 `git show --numstat` 为准。

### Dashboard 残留

- 删除无生产初始化入口的变量、layout、lock、column-width store 及其 38 个实现/测试文件。
- 删除只为 Dashboard 变量解析服务的 `/api/v5/substitute_vars` API 和前端调用；V5
  `query_range` 协议的通用 `variables` 字段仍保留。
- 删除 Dashboard 变量 drilldown、动态变量建议、变量替换标题和无效等待状态；context link
  的全局时间和行字段变量迁至通用 `useContextVariables`。
- 将活跃的 Widget 和 query-result 类型、V5 panel adapter 和 chart cursor sync 迁出
  Dashboard 命名空间，避免目录名继续掩盖生产依赖。

### Alert 兼容与高级通知

- 删除未被生产调用的 `pkg/transition` Alert/V4 查询转换器，以及 Alert JSON v1 自动补齐、
  schema 分支序列化和旧字段转换。
- Alert API 现在只接受 `schemaVersion: "v2alpha1"` 与 `version: "v5"`；旧
  `evalWindow`、`frequency`、`preferredChannels` 和 `renotify` 会明确返回 invalid input。
- 删除规则标签/注解的 Go template 展开和 Alert Builder 的模板提示、可配置重复通知、旧详情页
  与编辑页分支。描述只接受静态文本；运行时仍自动附加标准 `value`、`threshold` 注解。
- 保留阈值、评估窗口、缺失数据、最小点数、通知渠道和按 group-by 的通知分组。重通知使用
  Alertmanager 默认的 firing/no-data 4 小时间隔，不能按规则调整。

### Metrics 元数据编辑

- 删除 `POST /api/v5/metrics/{metric_name}/metadata`、其 OpenAPI/生成 client、模块写入、
  前端表单、临时单位的保存提示和对应测试 mock。
- `updated_metadata` 不再是读取优先级的一部分。查询服务和 Metrics Explorer 统一从采集的
  `time_series_v4` 读取元数据，并仅缓存该结果；旧人工 type/unit/description 覆盖允许丢失。
- Collector schema migration `siginsight_metrics/2001` 直接删除 `updated_metadata`。冻结的
  v1 baseline 保持不变，因而新库会在 baseline 后立即执行该清理迁移。
- Summary 和 Metric Details 的元数据显示、Lite compiler 类型解析、指标字段补全和临时 Y 轴
  显示单位保持可用；前端不再能够修改采集端声明的类型或单位。
