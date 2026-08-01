# M11：清理无调用的 Legacy QueryBuilder Helper

状态：Complete

## 边界

production import 与符号调用扫描确认下列 helper 没有包外调用：aggregation 注册表、CTE
拼接、字段 collision 调整、Having 重写和 filter 矛盾检测。它们曾为已删除的通用 V5
执行器服务。

仍保留 `FormatValueForContains`、`DataTypeCollisionHandledFieldName`、`PrepareWhereClause`、
`QueryStringToKeysSelectors` 与 `ToNanoSecs`：metadata、Metrics Explorer metadata 与专用
读取路径仍直接使用。

## 验证

- production 符号扫描没有删除 helper 的调用。
- `go test ./pkg/querybuilder ./pkg/telemetrylogs ./pkg/telemetrymetadata ./pkg/telemetrymetrics ./pkg/telemetrytraces`。

## 实现结果

- 删除 aggregation 注册表、CTE 拼接、字段 collision 调整、Having 重写和 filter 矛盾
  检测，以及这些 helper 的测试，共 7 个 Go 文件。
- 将 `time.go` 收紧到仍被 Metrics Explorer metadata 使用的 `ToNanoSecs`，删除旧 V5
  执行器的建议 step、metric 时间对齐和 reserved variable helper。
- 保留的 QueryBuilder 包缩减为 metadata/专用读取仍需要的 field collision、where clause
  parser、key selector、格式化与时间归一化。

## 验证结果

- 删除 helper 的 production 符号扫描为零。
- 直接依赖包测试通过。
- `go test ./pkg/...` 通过。
