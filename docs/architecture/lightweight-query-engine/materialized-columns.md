# ClickHouse 物化列在遥测查询引擎中的应用：术语、机制与工程现状

状态：Accepted（作为术语参考文档）
最后更新：2026-07-31
适用范围：Lightweight Query Engine 文档集；与 ADR-011、M9 配套阅读

## 摘要

本仓库的遥测存储以半结构化方式承载 OpenTelemetry 数据：高频属性以键值对形式
保存在 Map 类型列中，查询需在每一行的 Map 内做哈希查找，难以利用列式存储的压缩、
类型化与索引优势。为此，存储层通过"从 Map 提取为独立列"的物化机制建立高频属性的
快速访问路径。本文阐述相关 ClickHouse 术语、物化列的创建与发现机制、当前仓库中
物化列的分布现状，以及轻量查询引擎对该机制的显式化改造（M9 / ADR-011）的设计考量。

---

## 1. 背景与问题

遥测数据（Logs、Traces、Metrics 的 resource/span attribute）具有两个显著特征：

1. **半结构化**：属性集合在写入时才能确定，无法在编译期预知全部字段；
2. **倾斜分布**：少数高频属性（如 HTTP 路由、RPC 方法、服务名）承载了绝大部分
   查询过滤与分组需求，而长尾属性极少被查询。

为同时满足"任意属性可查"与"高频属性快查"，存储层采用 Map 列 + 物化列的两级结构。
本文讨论的物化列即第二级：将高频属性从 Map 中提取为独立的强类型列。

## 2. 核心术语

### 2.1 Map 列

`Map(K, V)` 是 ClickHouse 的一种复合类型，单行内可容纳任意数量的键值对。例如某
日志行 `attributes` 列的值可能为 `{'http.method':'GET','http.route':'/api/v1','env':'prod'}`。

- 优点：schema 灵活，新属性无需改表；
- 缺点：过滤需逐行哈希查找；不同类型键值混存导致压缩率低；无法建立列级索引。

### 2.2 列定义形态

| 形态 | 写入行为 | 存储 | 说明 |
| --- | --- | --- | --- |
| 普通列 | INSERT 显式提供 | 落盘 | 基准形态 |
| `DEFAULT expr` 列 | 未提供时由表达式计算 | 落盘 | 计算结果持久化，之后等同于普通数据 |
| `MATERIALIZED expr` 列 | 禁止显式提供，恒由表达式计算 | 落盘 | 与 DEFAULT 的区别仅在于禁止写入 |
| `ALIAS expr` 列 | 恒由表达式计算 | 不落盘 | 查询时即时计算 |

### 2.3 物化列（本仓库语境）

本仓库所称"materialized column"实际指 **`DEFAULT` 表达式列**，其表达式从对应
Map 列提取单一键。例如：

```sql
ALTER TABLE <traces_index_table>
  ADD COLUMN `attribute_string_http$$route` String
  DEFAULT attributes_string['http.route'];
```

- **新数据**：写入时由 DEFAULT 表达式自动填充；
- **存量数据**：后加列对历史行无值，需 `ALTER TABLE ... MATERIALIZE COLUMN ...`
  将表达式结果"落实"为已存储数据——"物化"一词源于此操作；
- **术语澄清**：此处与 ClickHouse 关键字 `MATERIALIZED` 含义不同，仅表达"由
  表达式派生并持久化"的语义。解析代码亦以"DEFAULT 表达式非空"作为物化列的
  判定条件。

### 2.4 命名约定与 exists 列

物化列名遵循统一模板（源码 `FieldKeyToMaterializedColumnName`）：

```text
<context>_<datatype>_<key，其中 "." 替换为 "$$">
```

示例：

```text
resource_string_service$$name        -- resource 上下文、string 类型、键 service.name
attribute_string_http$$route         -- attribute 上下文、string 类型、键 http.route
```

每个物化列伴随一个布尔"存在性"列（后缀 `_exists`），如
`attribute_string_http$$route_exists`。其必要性在于：Map 查询对缺失键返回类型默认
值（字符串为 `''`），无法区分"值为空"与"键不存在"；存在性列消除了这一语义歧义，
使 `EXISTS` / `NOT EXISTS` 过滤可快速求值。

## 3. 物化列机制

### 3.1 创建机制：手动指定

本仓库代码中不存在自动创建物化列的逻辑。物化列仅有两个来源：

1. **迁移内建**：采集端仓库（schema 的唯一所有者）在 migration 中硬编码创建一组
   经过人工挑选的高频属性列；
2. **运行时手工扩展**：运维人员通过 `ALTER TABLE ... ADD COLUMN ... DEFAULT ...`
   按需添加（含历史版本的管理端"属性提升"操作）。

两种来源均属"手动指定"；系统本身不根据数据规模或查询频率自动提升属性。

### 3.2 发现机制：自动

查询引擎不维护物化列的静态清单，而是通过解析 `SHOW CREATE TABLE` 自动发现：

```text
SHOW CREATE TABLE <traces_index_table>
  -> AST 解析每个列定义
  -> 判定条件：列名符合 <context>_<type>_ 前缀 且 存在 DEFAULT 表达式且表达式形如 map['key']
  -> 生成字段描述 {Name, Context, DataType, Materialized: true}
```

该发现结果同时服务于：

- **字段候选**：前端过滤器的键/值自动补全列表；
- **编译决策**：旧编译器的字段映射器在 `Materialized == true` 时自动将查询定向
  到物化列，未物化的键回退到 Map 访问。

即"创建手动、发现自动、使用自动"：新增物化列后无需修改任何查询代码即可生效。

### 3.3 查询路径差异

同一语义字段（如 `http.route`）存在两条物理路径：

| 路径 | SQL 形态 | 特征 |
| --- | --- | --- |
| 物化列 | `attribute_string_http$$route = ?` | 类型化、可压缩、可索引，写入有额外成本 |
| Map | `attributes_string['http.route'] = ?` | 通用、零维护，逐行哈希查找 |

### 3.4 性能原理

独立列相对 Map 的优势：

1. **压缩**：高频键的取值重复度高，独立列可达到极高压缩比；
2. **类型化**：强类型列（含 LowCardinality 字典编码）的扫描与比较远快于 Map 内
   字符串混存；
3. **可索引**：可配合 PREWHERE、跳数索引等机制，Map 查询不具备；
4. **省 CPU**：免去逐行哈希查找。

代价：每列占用存储空间、每次写入需计算 DEFAULT 表达式。因此仅高频属性值得物化。

## 4. 项目现状

### 4.1 Traces 侧：内建物化列

Traces 索引表的字段映射中硬编码了 9 组物化列（各配 1 个 exists 列）：

| 上下文 | 语义键 | 物化列名 |
| --- | --- | --- |
| resource | service.name | `resource_string_service$$name` |
| attribute | http.route | `attribute_string_http$$route` |
| attribute | messaging.system | `attribute_string_messaging$$system` |
| attribute | messaging.operation | `attribute_string_messaging$$operation` |
| attribute | db.system | `attribute_string_db$$system` |
| attribute | rpc.system | `attribute_string_rpc$$system` |
| attribute | rpc.service | `attribute_string_rpc$$service` |
| attribute | rpc.method | `attribute_string_rpc$$method` |
| attribute | peer.service | `attribute_string_peer$$service` |

这些键覆盖服务、HTTP、RPC、消息、数据库与对端服务等查询高频维度，属人工挑选。

### 4.2 Logs 侧：无内建物化列

Logs 主表的字段映射未硬编码任何物化列，属性查询全部走 Map 路径。若存在 Logs
物化列，则来自运行时手工扩展，并经由 3.2 节机制自动被发现。

### 4.3 自动与手动对照

| 环节 | 机制 |
| --- | --- |
| 创建（Traces 内建 9 组） | 手动指定（migration 硬编码） |
| 创建（其余） | 手动指定（运行时 ALTER） |
| 创建（自动） | 不存在 |
| 发现（识别物化列） | 自动（SHOW CREATE TABLE 解析） |
| 使用（查询定向） | 自动（旧编译器按 Materialized 标记切换） |
| 权威校验 | `SHOW CREATE TABLE` / `system.columns` 的 `default_expression` 非空 |

### 4.4 轻量查询引擎的取舍

轻量引擎（`litequery`）坚持静态确定性 Catalog（ADR-003）：语义字段到物理表达式
的映射在编译期固定。M9 前 attribute 一律解析为类型化 Map 列；M9 后仅 Collector v1
baseline 明确声明的 9 个 Trace string 字段走物化列。其动机是：

- 同一查询在不同 schema 状态下编译出不同 SQL，将破坏 golden SQL 测试、query log
  审计与性能可预测性；
- "每次编译现扫现选"是旧引擎的不确定来源，与轻量引擎的价值主张冲突。

其他字段继续走 Map；物化列不会因运行时发现而改变 Lite SQL。

## 5. M9 与 ADR-011：显式目录方案

针对上述矛盾，M9（物化列加速查询）与 ADR-011（物化列显式目录契约）提出"显式
物化目录"：

```text
受版本控制的 Catalog manifest {semantic key -> physical value / exists column}
  -> Collector migration fingerprint 跨仓库校验

Catalog.Resolve
  |-- 目录命中 -> 物化列（快路径）
  `-- 未命中   -> 类型化 Map 列（现状路径，确定性回退）
```

要点：

1. **静态声明而非运行时探测**：manifest 是源码的一部分，同一二进制总生成同一 SQL；
2. **fingerprint 守卫**：Collector migration 测试冻结 schema fingerprint，物理列变化
   必须显式更新 manifest、golden SQL 和协作验证；
3. **解析优先级仅在 Catalog 内部**：IR、Logical Plan 与 Statement 对外语义不变；
4. **不做自动提升**：新增/删除物化列仍由采集端 migration 驱动；"按数据自动提升"
   属于另一独立议题，不在 M9 范围；
5. **收益量化**：同 fixture 下物化列与 Map 双路径的延迟、read rows、read bytes
   对比报告是验收依据；收益不足的低频列进入删除候选清单，由后续独立 ADR 处理。

## 6. 结论

物化列是本仓库遥测存储中"灵活性"（Map）与"性能"（独立列）之间的一种平衡机制：
创建由人工/迁移显式指定，旧引擎以运行时发现自动使用。轻量查询引擎为换取确定性，
放弃运行时自动选择，改以 M9 的静态 manifest + fingerprint 协作验证方式重新纳入物化列。
该改造不改变公共协议与查询语义，仅改变 Catalog 内部的物理路径解析，并以真实查询
数据作为保留或删除的最终裁决依据。

## 附：术语速查

| 术语 | 含义 |
| --- | --- |
| Map 列 | 单行内键值对集合，schema 灵活、查询慢 |
| DEFAULT 表达式列 | 写入时由表达式计算并落盘的列 |
| MATERIALIZE COLUMN | 将 DEFAULT/MATERIALIZED 表达式结果回填存量数据的命令 |
| 物化列 | 本仓库对"从 Map 提取的 DEFAULT 列"的称呼 |
| exists 列 | 布尔型伴随列，表示键是否存在，消除 Map 缺键歧义 |
| fingerprint | schema 的结构指纹，目录/契约的失效校验锚点 |
