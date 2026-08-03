# ADR-019：分页探测与结果截断语义分离

状态：Accepted
日期：2026-08-03
关联里程碑：M12 correctness hardening

## 背景

ADR-017 为了判断 `limit` 行之后是否仍有数据，规定 SQL 查询 `limit + 1`。实现将额外一行
统一解释为 `Truncated=true` 和 `result_limit_reached`，导致每个非末页的 Logs/Traces 查询都
显示“结果被截断”。前端同时用本页行数猜测能否翻页，无法区分“总数刚好等于 page size”和
“确实存在下一页”。

分页是用户主动选择的数据窗口；截断是系统无法返回完整语义结果的部分成功状态。两者不能共享
同一个布尔值或 warning。

## 决策

1. `LIMIT pageSize + 1` 继续作为分页探测，不执行总量 `COUNT(*)`。
2. Compiler 的 `Statement.Pagination` 只用于 raw/trace 页，包含 `limit` 和 `offset`。
   Executor 裁掉探测行并返回 `PageInfo {limit, offset, returned, hasNextPage}`，不设置 warning。
3. `Statement.ResultLimit` 只用于 scalar/grouped aggregate 等不可翻页的结果上限。额外行表示
   结果语义确实不完整，Executor 设置 `Truncated=true` 并返回 `result_limit_reached`。
4. `pageInfo` 属于每个 V5 `RawData`，不能放在 response 全局 `meta`；一个请求中的多个 query
   可以有不同页大小和下一页状态。
5. 前端分页控件必须消费 `hasNextPage`，不能通过 `returned == pageSize` 猜测。未提供 pageInfo
   的旧响应暂时保留行数回退，但 Lite V5 正常响应必须提供 pageInfo。
6. Trace Detail 的分批过滤依据 `hasNextPage` 继续。只有累计达到 10,000 个 span 且
   `hasNextPage=true` 时，才生成最终截断 warning。
7. 单页请求超过 `MaxRawLimit` 属于无结果的预算错误，返回 HTTP error；它不是部分成功 warning。
   Executor/ClickHouse 扫描预算中止同样保持错误语义，除非未来明确实现可验证的部分结果协议。

## 备选方案

- 仅在前端隐藏 `result_limit_reached`：拒绝。它保留错误的服务端契约，并会破坏 Trace Detail
  的分页终止条件。
- 对每次查询执行 `COUNT(*)`：拒绝。UI 只需要是否存在下一页，高基数 ClickHouse 查询不应为
  精确总数承担额外全量扫描。
- 继续使用 `Truncated`，增加 warning 抑制参数：拒绝。同一个状态仍有两种相反语义。

## 影响

- `pagination.limit=10` 且存在第 11 行时，响应包含十行和 `hasNextPage=true`，不再显示 warning。
- 总结果恰好十行时 `hasNextPage=false`，Next 被正确禁用，不会进入空白页。
- 聚合 Top-N 和 Trace Detail 10,000 总量预算仍能明确提示真正的数据截断。
- V5/OpenAPI 增加向后兼容的可选 `RawData.pageInfo`；请求结构不变。

## 迁移与回滚

后端先发布 pageInfo，前端在字段存在时优先使用；字段缺失时暂时回退到旧行数判断。回滚后端不会
使旧前端失败，新前端也能继续使用回退逻辑。禁止恢复“分页即截断”的 warning。

## 验证

- Executor 单测分别覆盖分页探测无 warning 与 aggregate result cap 有 warning。
- Adapter/normalize 单测覆盖 `RawData.pageInfo` 往返。
- Controls 测试覆盖满页有/无下一页以及空末页返回上一页。
- Trace Detail 测试覆盖多页读取和 10,000 安全上限。
- ClickHouse 25.5.6 集成测试验证 `LIMIT + 1` 被解释为分页而非截断。
