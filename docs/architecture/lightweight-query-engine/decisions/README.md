# 架构决策记录

本目录保存 Lightweight Query Engine 的 Architecture Decision Records（ADR）。

## 何时需要 ADR

- 改变公共 API 或兼容策略。
- 改变 Lite IR 的核心结构或能力边界。
- 改变 schema catalog、ClickHouse 表、列、projection 或物化视图。
- 改变 cursor、时间范围、聚合、缺失值或告警语义。
- 引入新的跨仓库协议或依赖。
- 推翻已经接受的架构决定。

## 状态

- `Proposed`：等待审核。
- `Accepted`：允许实现。
- `Superseded`：被新的 ADR 替代，保留历史。
- `Rejected`：评估后不采用，保留理由。

文件名使用三位序号和短标题，例如：

```text
001-v5-compatibility-boundary.md
002-schema-catalog-contract.md
```

新 ADR 从 [000-template.md](000-template.md) 复制，并在相关阶段工程文档中建立双向链接。

