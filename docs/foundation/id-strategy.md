# 统一 ID 策略规范

> 文档类型：Foundation  
> 状态：Active  
> 权威级别：Authoritative  
> 最后更新：2026-04-08

---

## 1. 权威说明

本文件定义 RCProtocol 的统一 ID 策略，覆盖以下问题：

- 各类核心资源应使用什么 ID 形式
- 为什么这样设计
- 哪些 ID 属于业务资源 ID，哪些属于内部技术 ID
- 生成规则、存储规则、接口规则、测试规则分别是什么
- 后续新增资源时应如何接入统一 ID 策略

凡涉及以下字段与新增同类字段的设计，均以本文件为准：

- `brand_id`
- `product_id`
- `asset_id`
- `batch_id`
- `session_id`

如后续代码实现、任务文档、脚本、seed 数据、测试 fixture 与本文件冲突，以本文件为准。

---

## 2. 目标

统一 ID 策略要同时满足以下目标：

1. **语义清晰**：从 ID 本身可以看出资源类型
2. **风格统一**：品牌、资产、批次、会话等核心资源采用一致规则
3. **易于审计与排障**：日志、审计、脚本、SQL 查询中一眼可区分资源类型
4. **便于 API 使用**：对外接口直接暴露稳定字符串 ID，无需调用方理解内部二进制类型
5. **兼容当前数据库现实**：当前主资源表大量使用 `TEXT PRIMARY KEY`
6. **避免类型漂移**：不能出现“数据库是字符串、代码里一会儿 UUID、一会儿普通字符串”的混乱状态
7. **支持未来扩展**：后续新增资源时可按同一模式接入

---

## 3. 总体策略

RCProtocol 采用 **分层 ID 策略**：

### 3.1 业务资源 ID

对外业务资源一律使用：

**`<prefix>_<ULID>`**

例如：

- `brand_01JQ...`
- `product_01JQ...`
- `asset_01JQ...`
- `batch_01JQ...`
- `session_01JQ...`

这类 ID 的特点：

- 是字符串
- 带资源类型前缀
- 后缀使用 ULID
- 用于 API、数据库主键、日志、脚本、审计输出、后台展示

### 3.2 内部技术 ID / 事件 ID

内部技术记录继续允许使用：

- `UUID`

适用对象：

- `trace_id`
- `event_id`
- `audit_event_id`
- `authority_id`
- `entanglement_id`
- 纯内部事件流 / 技术关联表主键

这类 ID 的特点：

- 不强调业务可读性
- 主要用于内部唯一性、事件追踪、技术记录
- 不要求与外部资源 ID 风格一致

### 3.3 不允许的混合状态

同一个字段不得同时处于以下混合状态：

- 数据库列是 `TEXT`
- 代码生成逻辑使用裸 `UUID`
- 路由和 DTO 又把它当普通业务字符串

尤其不允许：

- 主资源 ID 列是 `TEXT`，但随机塞入无前缀 UUID 文本
- 某些资源用 `prefix + ULID`，某些同类资源用手写自由字符串
- 新代码继续引入新的 ID 风格

---

## 4. 为什么采用 `prefix + ULID`

### 4.1 为什么不是统一改成 `Uuid`

不统一改成 `Uuid` 的原因：

1. 当前核心业务资源表大量是 `TEXT PRIMARY KEY`
2. 业务资源 ID 已直接暴露在 API 路由、JSON 响应、脚本、seed、测试数据中
3. `Uuid` 不表达资源类型，不利于排障与审计
4. 当前项目里 `brand_id` / `batch_id` 已经部分采用前缀字符串风格，继续统一比回退到裸 UUID 更合理
5. 对当前项目而言，主问题是**ID 体系一致性**，不是单点字段性能优化

### 4.2 为什么使用 ULID 而不是普通随机字符串

使用 ULID 的原因：

1. 全局唯一性足够强
2. 字符串表示适合 API 和数据库 `TEXT` 列
3. 相比随机 UUID 文本，更适合按时间大致排序
4. 便于日志、审计、排障时快速阅读与定位

### 4.3 为什么要加前缀

加前缀的原因：

1. `brand_`、`asset_`、`batch_`、`session_` 一眼可区分资源类型
2. 避免不同资源 ID 混用时难以排查
3. 降低脚本、SQL、日志、人工排查中的认知成本
4. 让接口与审计输出更稳定、更具可解释性

---

## 5. 本项目核心资源 ID 规范

### 5.1 `brand_id`

#### 规定
- 类型：`String`
- 格式：`brand_<ULID>`
- 数据库列：`TEXT PRIMARY KEY`
- API：字符串

#### 说明
`brand_id` 已经部分符合本规范，应继续保持，不得再引入其他生成方式。

#### 示例
- `brand_01JQ9X8ABCDEF1234567890XYZ`

---

### 5.2 `product_id`

#### 规定
- 类型：`String`
- 格式：`product_<ULID>`
- 数据库列：`TEXT PRIMARY KEY`
- API：字符串

#### 说明
当前 `product_id` 仍存在历史自由字符串和 demo 值。后续所有新增产品数据必须统一改为本格式。

#### 示例
- `product_01JQ9X8ABCDEF1234567890XYZ`

---

### 5.3 `asset_id`

#### 规定
- 类型：`String`
- 格式：`asset_<ULID>`
- 数据库列：`TEXT PRIMARY KEY`
- API：字符串

#### 说明
`asset_id` 是当前最需要统一的字段。后续不得再使用裸 `UUID` 文本直接写入 `assets.asset_id`。

#### 示例
- `asset_01JQ9X8ABCDEF1234567890XYZ`

---

### 5.4 `batch_id`

#### 规定
- 类型：`String`
- 格式：`batch_<ULID>`
- 数据库列：`TEXT PRIMARY KEY`
- API：字符串

#### 说明
`batch_id` 当前已基本符合本规范，应继续保持，不应单独切换到 `Uuid`。

#### 示例
- `batch_01JQ9X8ABCDEF1234567890XYZ`

---

### 5.5 `session_id`

#### 规定
- 类型：`String`
- 格式：`session_<ULID>`
- 数据库列：`TEXT PRIMARY KEY`
- API：字符串

#### 说明
`session_id` 当前尚未完全规范化。后续新建会话时必须统一采用本格式。

#### 示例
- `session_01JQ9X8ABCDEF1234567890XYZ`

---

## 6. 生成策略规范

### 6.1 统一生成入口

后续所有业务资源 ID 必须通过统一的 ID 生成入口生成，禁止在业务代码中手写格式拼接或临时生成。

建议集中位置：

- `rust/rc-common/src/ids.rs`

或在过渡阶段先放在：

- `rust/rc-api/src/id.rs`

但长期应沉淀到公共层，避免每个 crate 自己造规则。

### 6.2 推荐接口

建议提供以下生成函数：

- `generate_brand_id()`
- `generate_product_id()`
- `generate_asset_id()`
- `generate_batch_id()`
- `generate_session_id()`

统一规则：

- 输出 `String`
- 格式必须为 `<prefix>_<ULID>`
- 前缀固定、小写、单数名词

### 6.3 禁止事项

禁止以下做法：

1. 在 handler / route / service 内直接 `format!("brand_{}", ...)`
2. 在业务路径中直接 `Uuid::new_v4().to_string()` 作为业务资源 ID
3. 继续引入 `demo-001`、`main-001` 这类新风格作为正式新增资源 ID
4. 不同资源使用不同大小写或不同分隔符
5. 让调用方提交任意自由字符串作为系统主资源 ID

---

## 7. 存储与数据库规范

### 7.1 数据库列类型

当前阶段，以下主资源 ID 列统一保留为：

- `TEXT`

适用字段：

- `brands.brand_id`
- `products.product_id`
- `assets.asset_id`
- `batches.batch_id`
- `factory_sessions.session_id`

### 7.2 为什么当前不强制改数据库为 `uuid`

当前不强制切到 PostgreSQL 原生 `uuid` 的原因：

1. 现有表与接口已经广泛使用字符串 ID
2. 目标格式本身带前缀，不适合直接映射为原生 `uuid`
3. 当前更重要的是统一风格，而不是做主键类型大迁移

### 7.3 数据库约束建议

后续可逐步补充以下约束：

- `CHECK (brand_id LIKE 'brand_%')`
- `CHECK (product_id LIKE 'product_%')`
- `CHECK (asset_id LIKE 'asset_%')`
- `CHECK (batch_id LIKE 'batch_%')`
- `CHECK (session_id LIKE 'session_%')`

注意：

- 约束上线前需先评估存量历史数据
- 对存量非规范数据应先迁移再加约束

---

## 8. API 与 DTO 规范

### 8.1 对外接口

所有业务资源 ID 在 HTTP API 中统一按字符串处理。

例如：

- `GET /brands/:brand_id`
- `GET /assets/:asset_id`
- `GET /batches/:batch_id`
- `GET /sessions/:session_id`

### 8.2 Rust DTO

业务资源 ID 在 DTO / Request / Response 层统一使用：

- `String`

例如：

- `pub brand_id: String`
- `pub product_id: String`
- `pub asset_id: String`
- `pub batch_id: Option<String>`
- `pub session_id: String`

### 8.3 不允许的情况

不允许以下不一致：

- 路由参数是 `String`，生成逻辑却是裸 UUID
- 某些响应返回 `Uuid`，某些响应返回字符串形式的同类业务资源 ID
- 同类资源在不同 DTO 中使用不同类型

---

## 9. 审计、日志与脚本规范

### 9.1 审计日志

审计日志中输出业务资源 ID 时，应直接输出规范化字符串 ID，不应临时转换成其他形式。

例如：

- `asset_...`
- `batch_...`
- `brand_...`

### 9.2 运维脚本

脚本、测试脚本、seed、fixture 中涉及主资源 ID 时，应优先遵守正式格式。

允许例外：

- 历史 demo 数据
- 一次性迁移脚本

但新增脚本与新增测试数据，应统一采用正式前缀格式。

### 9.3 调试输出

调试输出中应尽量保留完整业务资源 ID，避免只输出裸后缀，防止排障时无法区分资源类型。

---

## 10. 测试与 Seed 数据规范

### 10.1 新增测试数据

后续新增测试 fixture / seed / integration test 数据时，应优先使用规范化 ID，例如：

- `brand_<ULID>`
- `product_<ULID>`
- `asset_<ULID>`
- `batch_<ULID>`
- `session_<ULID>`

### 10.2 历史数据兼容

现有历史 demo 数据如：

- `brand-demo`
- `product-demo-001`
- `asset-main-001`
- `batch-demo-001`
- `session-demo-001`

可以在过渡期保留，但必须视为**历史兼容数据**，不能再作为新代码模板。

### 10.3 测试断言

测试不应只断言“是任意字符串”，而应断言：

- 前缀正确
- 长度合理
- 生成函数返回的格式一致

---

## 11. 当前项目涉及的主要文件

以下文件会直接受到本规范影响，后续代码任务应以这些位置为重点：

### 11.1 数据库 Migration

- `rust/rc-api/migrations/20250101000000_init_brands_products.sql`
- `rust/rc-api/migrations/20250101000001_init_batches_sessions.sql`
- `rust/rc-api/migrations/20250101000002_init_assets.sql`
- `rust/rc-api/migrations/20250101000017_add_batch_id_to_assets.sql`

### 11.2 业务资源生成与认证相关

- `rust/rc-api/src/auth/api_key.rs`
- `rust/rc-api/src/db/batches.rs`
- `rust/rc-api/src/routes/brand.rs`
- `rust/rc-api/src/routes/protocol.rs`

### 11.3 DTO / 查询 / 路由参数

- `rust/rc-api/src/db/assets.rs`
- `rust/rc-api/src/routes/assets.rs`
- `rust/rc-api/src/routes/batch.rs`
- `rust/rc-api/src/db/products.rs`
- `rust/rc-api/src/db/brands.rs`

### 11.4 Seed / Fixtures / Tests

- `rust/rc-api/src/seed.rs`
- `rust/rc-test-helpers/src/fixtures.rs`
- 各类集成测试、脚本、测试数据生成逻辑

---

## 12. 当前代码需要重点收敛的地方

### 12.1 `brand_id`

现状：
- 已基本采用 `brand_<ULID>`

要求：
- 继续保持
- 不得新增其他风格生成逻辑

### 12.2 `product_id`

现状：
- 仍存在历史自由字符串

要求：
- 后续新增产品统一收敛为 `product_<ULID>`

### 12.3 `asset_id`

现状：
- 数据库列是 `TEXT`
- 某些创建路径直接使用 `Uuid::new_v4()` 写入

要求：
- 后续统一改为 `asset_<ULID>`
- 禁止再把裸 UUID 文本当成正式业务资源 ID 写入 `assets.asset_id`

### 12.4 `batch_id`

现状：
- 已基本采用 `batch_<ULID>`

要求：
- 保持不变
- 不改成 `Uuid`

### 12.5 `session_id`

现状：
- 还未完全规范化

要求：
- 后续统一改为 `session_<ULID>`

---

## 13. 后续新增资源如何接入统一策略

当新增新的业务资源时，必须先决定它属于以下哪一类：

### 13.1 如果它是业务资源

例如：

- `workorder_id`
- `delegation_id`
- `webhook_id`
- `policy_id`

则应默认采用：

- `<prefix>_<ULID>`
- 类型：`String`
- DB：`TEXT`

### 13.2 如果它是纯内部事件 / 技术记录

例如：

- 事件流水
- 审计事件主键
- 临时内部关联表主键

则可以继续采用：

- `UUID`

### 13.3 决策规则

判断标准：

如果这个 ID 会长期出现在以下任一场景，应优先视为业务资源 ID：

- API 路由参数
- JSON 响应
- 后台页面展示
- 审计报告
- 运维脚本
- 人工排障

否则可以视为内部技术 ID。

---

## 14. 建议的落地步骤

### Phase 1：冻结新风格扩散

立即执行：

1. 停止新增裸 UUID 作为业务资源 ID 的写法
2. 停止新增自由格式的 demo 风格正式 ID
3. 新增主资源统一按前缀 + ULID 设计

### Phase 2：统一生成入口

建议后续代码任务实现：

1. 建立统一 ID 生成模块
2. 把 `brand_id` / `batch_id` 的现有生成逻辑迁入统一模块
3. 新增 `product_id` / `asset_id` / `session_id` 的统一生成函数

### Phase 3：修正重点漂移点

建议优先修：

1. `asset_id` 的 UUID 写入路径
2. `session_id` 的生成缺失
3. `product_id` 的历史自由生成逻辑

### Phase 4：历史数据与约束收敛

在不影响当前业务的前提下逐步推进：

1. 新增数据全部采用规范格式
2. 存量历史 demo 数据保留兼容
3. 评估完成后再增加数据库 `CHECK` 约束

---

## 15. 最终结论

RCProtocol 的统一 ID 策略应明确为：

- **业务资源 ID：`prefix + ULID` 字符串**
- **内部技术 ID：UUID**

对于当前项目的核心资源，正式规定如下：

- `brand_id` → `brand_<ULID>`
- `product_id` → `product_<ULID>`
- `asset_id` → `asset_<ULID>`
- `batch_id` → `batch_<ULID>`
- `session_id` → `session_<ULID>`

后续所有新增代码、路由、DTO、seed、fixture、脚本、任务文档，都必须服从这一统一规则，不再引入新的 ID 风格。
