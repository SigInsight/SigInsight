# ADR-007：V5 兼容桥的受控回退

状态：Accepted
日期：2026-07-31
关联里程碑：M5、M6、M7

## 背景

`/api/v5/query_range` 是现有页面、保存查询和告警共同使用的公共边界，但 V5 DTO 的能力明显
大于轻量引擎的既定功能范围。直接将开关打开并尝试“尽力转换”会造成字段、函数或格式选项被
静默忽略；直接拒绝全部旧高级请求会阻断尚未迁移的页面。

## 决策

在迁移期采用两级结果：

1. adapter 对每个 V5 字段做完整 capability 检查。无法表达的合法 V5 feature 返回
   `UnsupportedError`，绝不产生部分 Lite IR。
2. 仅在内部 `lightweight_engine_enabled` 开关开启时尝试 adapter。`UnsupportedError` 使请求
   回落旧 Querier；适配成功后的 validation、ClickHouse execution 和 response mapping 错误则
   直接返回，不能伪装为“旧引擎成功”。

该开关不是 HTTP 协议字段，也不向用户暴露。M6 的新 UI 只能生成 Lite capability matrix 中的
请求；M7 在双跑证据足够后逐个移除 legacy 回退，M8 才删除旧实现。

## 后果

- 当前生产行为默认不变，且开关试运行不会改变不支持请求的语义。
- 每个被排除 feature 都有可测试的 adapter 失败点，可作为 UI 隐藏入口和最终删除旧路径的证据。
- 迁移期存在两套 engine，不能把 fallback 视为完成替换；必须以真实 API 双跑和 query log 证明
  每个核心消费者已无需 legacy。
- metrics metadata 仍是 V5 bridge 的输入，但 Lite core 不依赖 metadata store，也不复制旧
  aggregate-table selection 策略。
