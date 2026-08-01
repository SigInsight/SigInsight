# ADR-009: Server-Advertised Lightweight Rollout Capability

Status: Superseded by ADR-013
日期：2026-07-31
关联里程碑：M7

## 背景

M6 使用 `VITE_LIGHTWEIGHT_QUERY_EDITOR_ENABLED` 控制实验性 UI。它是构建期
变量，无法说明用户当前连接的 SigInsight 后端是否启用了 lightweight engine；
前端资源与后端可以独立部署，因而这不是可靠的运行时协商方式。

## 决策

本 ADR 记录 M7 的运行时协商机制，已由 ADR-013 取代。M8 删除 rollout feature flag 和
legacy UI 后，Lite-only QueryBuilder 始终使用同一 V5 契约；不支持的保存状态显示迁移入口。

在已认证的 `/api/v5/features/ui` 返回 `lightweight_query_engine`。只有运行中
的 `querier.lightweight_engine_enabled` 为真时该 feature 才 active。前端从
`AppContext` 消费这个 flag，并且仍以 Lite capability predicate 判断当前 V5
查询状态是否可表达。

该 flag 不改变 V5 query request，不将 Lite 实现细节暴露给匿名请求，也不表示
所有保存查询均可迁移。缺失 flag 的旧后端被当作未启用，以维持渐进部署的安全性。

## 后果

- 移除“前后端必须同时配置两个环境变量”的部署约束；运行时后端是唯一权威。
- 高级保存查询继续使用 legacy UI/engine，避免无损性不明的自动迁移。
- 仅 feature active 并不等于可删除 legacy；M7 真实 Collector 双跑、query log
  及页面回归仍是默认切换和 M8 删除的前置条件。
