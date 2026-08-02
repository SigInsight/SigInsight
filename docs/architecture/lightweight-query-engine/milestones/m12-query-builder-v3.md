# M12：QueryBuilderSearchV3 与 Trace Funnel 过滤子集

状态：Complete
日期：2026-08-02

## 目标

- 建立与轻量引擎能力矩阵一致的文本查询构建器。
- 让字段补全插入确定的 telemetry context。
- 为 Trace Funnel 提供基础字符串过滤，而不引入跨 span planner。
- 删除 QueryBuilderSearch、QueryBuilderSearchV2、QuerySearch、ClientSideQBSearch 和临时 Lite
  文本框的重复实现。

## 实现边界

设计由 [ADR-018](../decisions/018-query-builder-v3-and-funnel-filter-boundary.md) 固化。V3 复用
`/api/v5/fields/keys`、`/api/v5/fields/values` 和 V5 `TagFilter`，后端增加受限且参数化的
LIKE/ILIKE/REGEXP/NOT CONTAINS 系列谓词。

## 验证与退出条件

- Go adapter/compiler 单测和 ClickHouse 25.5.6 集成测试通过。
- V3 completion、parser、scope 和主要页面测试通过。
- `rg` 无旧编辑器生产 import。
- TypeScript、lint、production build 通过。
- Logs、Traces、Metrics、Trace Detail 和基础 Alert 页面完成真实浏览器回归。

## 残余风险

- Trace Funnel 的多步顺序与转化计算仍属于专用 API，不由 Lite compiler 执行。
- 字段和值补全依赖 metadata API 的完整性；服务端未返回 context 的未知同名字段不会被 V3
  猜测，用户必须输入限定名。

## 完成结果

- Logs Explorer、Traces Explorer、Metrics Explorer、Alert Query Builder、Trace Detail、
  Exceptions、API Monitoring 和 Trace Funnel step 已迁移到 V3。
- 已删除 QueryBuilderSearch、QueryBuilderSearchV2、QuerySearch、ClientSideQBSearch 和临时 Lite
  editor；scope selector 与通用 filter helper 已按职责提取，无旧路径兼容层。
- 本里程碑约删除 7,779 行、增加 1,400 行，净减少约 6,380 行（Git rename detection
  口径，包含文档和测试）。

## 验证证据

- `go test ./...` 与 `golangci-lint run --timeout 10m` 通过。
- ClickHouse 25.5.6 integration compiler 测试通过。
- 前端 ESLint、TypeScript 和 production build 通过。
- 全量 Jest：285 suites、2,772 passed、10 skipped、0 failed。
- Playwright 实测 Logs、Traces、Metrics、Alert 主页面只渲染 V3；`http.route` 只补全为
  `attribute.http.route`。Trace `ILIKE`、Logs `NOT LIKE` 和 Trace Detail attribute filter 的
  真实 `/api/v5/query_range` 均返回 HTTP 200，未出现 invalid/unsupported lightweight query。
