# M10：退役不可达的 Legacy 编辑渲染树

状态：Complete
前置条件：M8 完成 Lite-only V5 执行边界；M6 已将公共 QueryBuilder 外壳切换为 Lite UI

## 问题与目标

`frontend/src/components/QueryBuilder/QueryBuilder.tsx` 现在只有两个渲染结果：可表达的
查询进入 `LiteQueryBuilder`，不可表达的保存查询显示明确替换提示。旧
`container/QueryBuilder/components/Query` 渲染树因此没有入口，但它的 Query、Formula、
函数链、Having、聚合和数据源组件仍保留在源码中。

M10 删除这棵不可达树，降低前端生产代码和测试负担。它不把仍在运行的 Query Builder
状态模型误判为死代码。

## 删除范围

- 旧 Query/Formula 编辑组件及其私有组件：函数链、数据源切换、聚合、group by、having、
  reduce、附加筛选、列表标记和对应样式/测试。
- 旧 Formula 的 limit/order/having 子组件。
- 只被旧 Query props 使用的 `QueryProps` 类型入口。

## 明确保留

- `providers/QueryBuilder`、`hooks/queryBuilder`、V5 DTO 和 payload mapper：它们仍支撑
  URL/保存查询兼容、Explorer 请求与 Lite UI 状态。
- `QueryBuilderSearch`、`QueryBuilderSearchV2`、`MetricNameSelector`、`BuilderUnitsFilter`、
  `OrderByFilter`：详情页、funnel、Metrics Inspect、Live Logs 或 Explorer toolbar 仍直接
  使用。
- `components/QueryBuilder/utils` 和 `QuerySearch`：仍被日志详情、metrics filter 和请求转换
  使用。

## 验证计划

- production import 扫描确认删除路径在树外无引用。
- `yarn tsc --noEmit --pretty false`。
- `yarn jest src/features/lite-query/LiteQueryBuilder.test.tsx --runInBand`。
- 前端 production build。

## 实现结果

- 删除 Query、Formula、函数链、聚合、group by、having、reduce、附加筛选、数据源切换及
  相应样式和测试，共 71 个文件、3,881 行。
- `types/common/operations.types.ts` 直接声明旧组件唯一需要的 `index` / `query` 参数，
  不再反向依赖已删除的渲染组件类型。
- `filters/index.ts` 只保留仍有生产消费者的 `BuilderUnitsFilter`、`MetricNameSelector` 和
  `OrderByFilter`。

## 验证结果

- 删除路径的 production import 扫描为零。
- `yarn tsc --noEmit --pretty false` 通过。
- `yarn jest src/features/lite-query/LiteQueryBuilder.test.tsx --runInBand` 通过（3 tests）。
- `yarn build` 通过；仅有既存的动态/静态混合导入提示。

## 残余风险

该删除不减少共享 V5 状态模型的复杂度。后续若要继续缩减 Provider/DTO，必须先逐页面
迁移 Explorer、Metric Inspect、告警和保存查询转换，不能把它们与不可达渲染树混为一谈。
