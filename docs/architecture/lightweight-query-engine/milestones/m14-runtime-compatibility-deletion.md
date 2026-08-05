# M14: 删除运行时兼容与不可达产品残留

状态：进行中

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
- 删除 UpdateMetricMetadata HTTP、module、SQL 写入和写后缓存更新。
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
