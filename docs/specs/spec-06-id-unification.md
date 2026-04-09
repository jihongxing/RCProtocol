# Spec-06: 统一 ID 策略落地与主资源 ID 改造

> 文档类型：Technical Specification  
> 状态：Implemented  
> 优先级：P1  
> 依赖：`docs/foundation/id-strategy.md`  
> 预计工期：3-5 天  
> 负责人：已完成  
> 最后更新：2026-04-08

---

## 1. 背景与目标

### 1.1 背景

RCProtocol 曾长期存在多种 ID 风格并存的现实：

- `brand_id` 已使用 `brand_<ULID>`
- `batch_id` 已基本使用 `batch_<ULID>`
- `asset_id` 的数据库列为 `TEXT`，但部分创建路径直接写入裸 `UUID` 文本
- `product_id`、`session_id` 曾存在历史自由字符串和 demo 风格值
- seed / fixture / 脚本 / 路由参数 / DTO 层对同类资源的 ID 语义并不完全统一

这会带来以下问题：

1. 同类资源 ID 风格不统一，增加 API、日志、审计、脚本排障的认知成本
2. 主资源 ID 存在“数据库是字符串、实现里却按 UUID 生成”的语义漂移
3. 后续新功能容易继续引入新的 ID 风格，形成长期技术债
4. Foundation 文档已定义统一策略，但需要真正落地到代码、测试与 migration

### 1.2 目标

本 Spec 的目标是把 `docs/foundation/id-strategy.md` 转成明确的工程落地方案，并完成以下事情：

1. 冻结新的 ID 漂移继续扩散
2. 给主资源 ID 建立统一生成入口
3. 收敛 `brand_id / product_id / asset_id / batch_id / session_id` 的实现方式
4. 统一 DTO、路由、数据库写入和测试数据的 ID 风格
5. 明确历史兼容边界与迁移顺序
6. 为后续类型强化与数据库约束提供可增量落地的基础

### 1.3 非目标

本 Spec 不包含以下内容：

- 不将所有主资源 ID 切换为 PostgreSQL 原生 `uuid`
- 不强制一次性迁移全部历史 demo 数据
- 不改动内部事件类主键（如 `event_id`, `authority_id`, `entanglement_id`）的 UUID 策略
- 不在本次任务中解决业务规则问题（状态机、鉴权、激活链路等）

---

## 2. 设计结论

### 2.1 总体原则

依据 `docs/foundation/id-strategy.md`，RCProtocol 采用分层 ID 策略：

#### A. 业务资源 ID

统一采用：

**`<prefix>_<ULID>`**

适用范围：

- `brand_id`
- `product_id`
- `asset_id`
- `batch_id`
- `session_id`

#### B. 内部技术 ID

继续采用：

- `UUID`

适用范围：

- `trace_id`
- `event_id`
- `audit_event_id`
- `authority_id`
- `entanglement_id`
- 纯内部事件流 / 技术关联表主键

### 2.2 本次改造的核心结论

1. **`brand_id` 保持 `brand_<ULID>`**
2. **`batch_id` 保持 `batch_<ULID>`**
3. **`asset_id` 已收敛**：主写入路径不再使用裸 UUID 文本，统一改为 `asset_<ULID>`
4. **`product_id` 已规范**：统一生成能力已补齐，后续新增使用 `product_<ULID>`
5. **`session_id` 已规范**：统一生成能力已补齐，后续新增使用 `session_<ULID>`

### 2.3 统一生成入口

后续所有业务资源 ID 统一通过公共层生成。

已落地位置：

- `rust/rc-common/src/ids.rs`

当前已提供：

- `generate_brand_id()`
- `generate_product_id()`
- `generate_asset_id()`
- `generate_batch_id()`
- `generate_session_id()`

并额外提供强类型包装：

- `BrandId`
- `ProductId`
- `AssetId`
- `BatchId`
- `SessionId`

---

## 3. 资源级详细要求

### 3.1 `brand_id`

#### 目标格式
- `brand_<ULID>`

#### 当前状态
- 已符合规范
- 已统一接入公共生成入口

#### 已落地结果
- `auth/api_key.rs` 中的 `generate_brand_id()` 已重定向到 `rc_common::ids::generate_brand_id()`
- 测试与 fallback 场景的新样例已改用规范化 `brand_id`

#### 兼容性
- 历史 demo 数据如 `brand-demo` 仍保留兼容，但已明确标记为 legacy sample，不再作为新增模板

---

### 3.2 `product_id`

#### 目标格式
- `product_<ULID>`

#### 当前状态
- 新增统一生成函数已落地
- 测试中新增产品样例已可使用规范化 `product_id`
- 历史只读与 legacy sample 仍保留兼容

#### 已落地结果
- 公共生成入口已提供 `generate_product_id()`
- 测试 helper 已提供 `generate_test_product_id()`
- `verify_integration.rs` 的固定测试上下文已改为规范化品牌/产品初始化，不再依赖 `product-demo-001`

#### 兼容性
- 历史 `product-demo-001` 可继续作为 legacy sample 保留在兼容 seed 中

---

### 3.3 `asset_id`

#### 目标格式
- `asset_<ULID>`

#### 当前状态
- 新资产主创建路径已改为统一格式
- 对外 DTO / 路由参数仍以 `String` 暴露

#### 已落地结果
- `routes/protocol.rs` 中 blind scan 写入路径已从裸 UUID 改为 `generate_asset_id()`
- `BlindScanResponse.asset_id` 已统一为 `String`
- 集成测试与 helper 中新增资产样例已改用规范化 `asset_id`

#### 兼容性
- 历史 `asset-main-001`、UUID 字符串型 `asset_id` 可继续读取
- 但不得再作为新增模板或新写入主路径

#### 优先级
- 本次改造中已完成的最高优先级字段

---

### 3.4 `batch_id`

#### 目标格式
- `batch_<ULID>`

#### 当前状态
- 已符合规范
- 已统一接入公共生成入口

#### 已落地结果
- `db/batches.rs` 中的内联拼接已迁移为 `generate_batch_id()`

#### 兼容性
- 历史 `batch-demo-001` 保留兼容，仅作为 legacy sample

---

### 3.5 `session_id`

#### 目标格式
- `session_<ULID>`

#### 当前状态
- 统一生成函数已提供
- 后续新增 session 已有明确合法格式

#### 已落地结果
- 公共生成入口已提供 `generate_session_id()`
- 测试 helper 已提供 `generate_test_session_id()`

#### 兼容性
- 历史 `session-demo-001` 保留兼容，仅作为 legacy sample

---

## 4. 数据库与存储规则

### 4.1 数据库列类型

当前阶段，以下字段统一保留为：

- `TEXT`

适用字段：

- `brands.brand_id`
- `products.product_id`
- `assets.asset_id`
- `batches.batch_id`
- `factory_sessions.session_id`

### 4.2 本次任务不做的事

本任务未将上述列切换为 PostgreSQL 原生 `uuid`，原因如下：

1. 业务资源 ID 已带前缀，不适合原生 `uuid` 类型
2. 当前系统已大量将这些资源 ID 暴露为字符串
3. 本次目标是风格统一，不是数据库主键大迁移

### 4.3 已落地增强

本次已新增前缀约束 migration：

- `rust/rc-api/migrations/20250101000021_add_prefixed_id_constraints.sql`

新增约束：

- `brands.brand_id LIKE 'brand_%'`
- `products.product_id LIKE 'product_%'`
- `assets.asset_id LIKE 'asset_%'`
- `batches.batch_id LIKE 'batch_%'`
- `factory_sessions.session_id LIKE 'session_%'`

约束采用：

- `NOT VALID`

这样可以：

- 不阻塞历史 legacy 数据继续被读取
- 保护后续新写入 / 更新数据符合前缀规则
- 为后续历史数据清洗完成后执行 `VALIDATE CONSTRAINT` 留好路径

---

## 5. API、DTO 与代码规范

### 5.1 路由参数

业务资源 ID 在路由参数中统一使用：

- `String`

### 5.2 DTO 字段

业务资源 ID 在请求/响应结构体中统一使用：

- `String`
- `Option<String>`

### 5.3 已落地代码约束

当前已明确禁止以下实现方式：

1. 在主业务 handler 内手写 `format!(...)` 拼业务资源 ID
2. 使用 `Uuid::new_v4().to_string()` 直接生成业务资源 ID
3. 在新增测试与样例中继续引入自由字符串风格主资源 ID

### 5.4 已落地强类型能力

公共层已增加：

- `BrandId`
- `ProductId`
- `AssetId`
- `BatchId`
- `SessionId`

这些类型具备：

- 前缀校验
- ULID 后缀校验
- `TryFrom<String>` / `TryFrom<&str>`
- `Display`
- `AsRef<str>`

当前主业务路径仍以 `String` 为主，强类型包装作为公共增强能力已先行落地，后续可逐步把业务边界切到这些类型上。

---

## 6. 历史兼容策略

### 6.1 兼容原则

本次改造采用：

- **新增从严**
- **历史兼容**

即：

- 后续所有新增主资源 ID 必须符合本 Spec
- 历史存量数据允许继续读取与查询
- 不要求本次任务一次性改写全部历史数据

### 6.2 历史兼容数据示例

以下值仍视为历史兼容数据：

- `brand-demo`
- `product-demo-001`
- `asset-main-001`
- `batch-demo-001`
- `session-demo-001`
- 已落库的 UUID 文本型 `asset_id`

### 6.3 已收敛结果

已完成的收敛包括：

- `activation_integration.rs` 已改为规范化 `brand_id / asset_id`
- `authorization_integration.rs` 已改为规范化 `brand_id / asset_id`
- `verify_integration.rs` 的固定测试上下文已彻底改为规范化 `brand_id / product_id / asset_id`
- `auth/middleware.rs` 的 fallback 测试已改为规范化 `brand_id`
- `rc-test-helpers/tests/integration.rs` 中新增可组合测试样例已改为规范化 ID

当前保留旧风格的地方仅剩显式 legacy sample / demo seed，用于兼容与演示，不再作为新增模板。

---

## 7. 涉及的主要文件

### 7.1 Foundation / 规范文档
- `docs/foundation/id-strategy.md`

### 7.2 Spec / Task 文档
- `docs/specs/spec-06-id-unification.md`
- `docs/tasks/task-spec-06-id-unification.md`

### 7.3 统一生成与类型包装
- `rust/rc-common/src/ids.rs`
- `rust/rc-common/src/lib.rs`
- `rust/rc-common/Cargo.toml`

### 7.4 主业务路径
- `rust/rc-api/src/auth/api_key.rs`
- `rust/rc-api/src/db/batches.rs`
- `rust/rc-api/src/routes/protocol.rs`

### 7.5 Seed / Fixtures / Tests
- `rust/rc-api/src/seed.rs`
- `rust/rc-test-helpers/src/fixtures.rs`
- `rust/rc-test-helpers/tests/integration.rs`
- `rust/rc-api/tests/activation_integration.rs`
- `rust/rc-api/tests/authorization_integration.rs`
- `rust/rc-api/tests/verify_integration.rs`
- `rust/rc-api/src/auth/middleware.rs`

### 7.6 数据库约束
- `rust/rc-api/migrations/20250101000021_add_prefixed_id_constraints.sql`

---

## 8. 实施结果

### Phase 1：冻结漂移与统一入口

**已完成**。

交付结果：

- 已建立 `rc-common::ids` 统一入口
- `brand_id` / `batch_id` 的散落逻辑已迁入统一入口
- 已补齐 `product_id` / `asset_id` / `session_id` 生成函数

### Phase 2：修正关键主路径

**已完成**。

交付结果：

- `asset_id` 的 blind scan 主路径已改为 `asset_<ULID>`
- 已消除主业务路径中的裸 UUID 业务资源 ID 生成

### Phase 3：收敛 DTO / Seed / Fixture

**已完成**。

交付结果：

- DTO / 路由参数继续统一使用字符串
- 测试 helper 已提供规范化测试 ID 生成函数
- 新增测试样例已收敛为规范 ID
- 历史 demo seed 已明确标注为 legacy sample

### Phase 4：数据库约束与强类型增强

**已完成**。

交付结果：

- 已增加 `NOT VALID` 的前缀约束 migration
- 已增加资源 ID 强类型包装

---

## 9. 验收标准

本 Spec 的最低验收标准如下：

1. [x] 统一 ID 生成入口存在，且可生成 5 类主资源 ID
2. [x] `asset_id` 新写入路径不再使用裸 UUID 文本
3. [x] `brand_id`、`batch_id` 继续维持 `prefix + ULID`
4. [x] 新增 `product_id`、`session_id` 具备明确规范与统一生成能力
5. [x] DTO、路由、主资源查询仍统一使用字符串形式
6. [x] 新增测试数据不再默认使用 demo 风格自由字符串
7. [x] 文档、任务、实现三者对同一资源 ID 规则描述一致
8. [x] 数据库前缀约束已落地为 migration
9. [x] 资源 ID 强类型包装已落地为公共层能力

---

## 10. 风险与注意事项

### 10.1 当前已知风险

当前最大的风险已从“规则缺失”转为“历史样例与业务兼容边界管理”。

### 10.2 注意事项

1. 不要把 legacy sample 当作新实现模板
2. 不要单独把某个主资源切到 `Uuid`，造成新的不一致
3. 前缀约束目前是 `NOT VALID`，后续清洗历史数据后应补 `VALIDATE CONSTRAINT`
4. 公共层已有强类型包装，但业务路径尚未全面切换；后续可按边界逐步推进

### 10.3 非本 Spec 范围的独立问题

仓库中仍可能存在与本 Spec 无关的既有编译问题，例如审批模块与当前 `ActorContext` / `Role` 定义不一致。这类问题不影响 Spec-06 的落地完成判定。

---

## 11. 最终结论

本次 ID 改造并不是“把所有字段统一成 `Uuid`”，而是：

- **统一业务资源 ID 风格**
- **清除主资源 ID 语义漂移**
- **建立后续新增代码必须遵守的唯一规则**
- **为数据库约束和强类型包装铺好路并完成首轮落地**

正式结论为：

- `brand_id` → `brand_<ULID>`
- `product_id` → `product_<ULID>`
- `asset_id` → `asset_<ULID>`
- `batch_id` → `batch_<ULID>`
- `session_id` → `session_<ULID>`

内部技术记录继续保留 UUID，不做混改。
