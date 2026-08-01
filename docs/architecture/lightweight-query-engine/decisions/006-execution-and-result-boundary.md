# ADR-006：Executor 与结果扫描边界

状态：Accepted
日期：2026-07-31
关联里程碑：M4、M5、M6

## 决策

Compiler 只返回参数化 `Statement`；Executor 负责为每条 statement 建立带 timeout 的
context、执行、扫描动态列、关闭 rows 和记录错误。Executor 通过函数式 Queryer 接收
ClickHouse client，因此 `pkg/litequery` 的领域模型和测试不依赖具体驱动。

扫描结果使用轻量内部模型 `QueryResult{Columns, Rows}`，列顺序由 Statement 的
`ResultColumn` contract 固定。V5 response 或 HTTP error 映射不进入该模型，留给 adapter。

简单 formula 在所有基础 query 完成并扫描后按 `(timestamp, group values)` 对齐，使用
`+ - * /` 的受限表达式求值。缺失的对齐输入以零参与计算并附 warning；除数为零返回
`NaN`，不让一个公式错误取消其他已完成 query。

请求取消和 timeout 直接传给 ClickHouse Queryer，Executor 不启动脱离 request context
的后台 goroutine。并发数由 Executor 配置限制，结果按原始 query 顺序返回。
