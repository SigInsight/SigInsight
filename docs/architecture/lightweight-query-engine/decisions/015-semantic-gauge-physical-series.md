# ADR-015：语义 Gauge 与物理 Metric Series 对齐

状态：Accepted
日期：2026-08-01
关联里程碑：M3、M5、M8

## 背景

V5 metadata 将 `type=Sum, is_monotonic=false` 的 OTLP 指标作为 Gauge 暴露给 UI。这是
用户语义归一化，不是 ClickHouse 物理行的改写：`time_series_v4` 和 `samples_v4` 仍保存
原始 `Sum` 与原始 temporality。

若轻量编译器直接把语义 Gauge 编译为：

```sql
type = 'Gauge' AND temporality = 'Unspecified'
```

则非单调 Sum 永远没有匹配 series，API 返回成功但结果为空。

## 决策

1. Lite IR 保留 UI 使用的语义类型 Gauge。
2. Metrics series catalog 将语义 Gauge 解析为原生 `Gauge`，或
   `Sum AND is_monotonic=false`，且只读取 `__normalized=false` 的原始 series。
3. Gauge points 通过已筛选 series 的 fingerprint 关联，不再使用语义 temporality 过滤
   原始 points；Sum 和 Histogram 继续使用明确的物理 temporality。
4. adapter 在 metadata 将指标归一化为 Gauge 后清除 Sum temporality，避免产生无效的
   `Gauge + Cumulative/Delta` IR。

## 影响

- UI、告警和 API 继续看到稳定的 Gauge/Sum/Histogram 三类能力。
- ClickHouse schema 保留原始 OTLP 类型，无需为 UI 语义复制 points。
- series type 映射集中在 Metrics compiler，不泄漏到前端协议。

## 验证

- 单元测试验证 Gauge series 同时匹配原生 Gauge 与非单调 Sum，且 points 查询不附加错误
  temporality。
- ClickHouse 25.5.6 集成测试执行语义 Gauge 查询并要求返回正值。
- 认证 API 验证覆盖 metadata 类型恢复、label filter 与 groupBy 的组合。
