# Task: Spec-06 统一 ID 策略落地与主资源 ID 改造

> **关联 Spec**: [spec-06-id-unification.md](../specs/spec-06-id-unification.md)  
> **关联 Foundation**: [id-strategy.md](../foundation/id-strategy.md)  
> **状态**: ✅ 已完成  
> **优先级**: P1  
> **预计工期**: 3-5 天  
> **开始时间**: 2026-04-08  
> **完成时间**: 2026-04-08

---

## 任务目标

将 `docs/foundation/id-strategy.md` 和 `docs/specs/spec-06-id-unification.md` 落地为可执行代码约束，完成以下工作：

1. 建立统一 ID 生成入口
2. 收敛 `brand_id / product_id / asset_id / batch_id / session_id` 的生成策略
3. 修正 `asset_id` 的主路径漂移问题
4. 统一 DTO、路由、seed、fixture 的 ID 风格
5. 冻结后续新增代码继续引入新的 ID 风格

---

## Phase 1: 统一生成入口（0.5 ~ 1 天）

### Task 1.1: 创建统一 ID 生成模块
- [x] 新增统一 ID 生成文件
  - [x] `rust/rc-common/src/ids.rs`
  - [ ] `rust/rc-api/src/id.rs`（未采用，改用 `rc-common` 统一承载）
- [x] 提供以下函数：
  - [x] `generate_brand_id()`
  - [x] `generate_product_id()`
  - [x] `generate_asset_id()`
  - [x] `generate_batch_id()`
  - [x] `generate_session_id()`
- [x] 统一生成规则：`<prefix>_<ULID>`
- [x] 编写单元测试，验证前缀、长度、格式

**验收标准**:
- [x] 5 类主资源 ID 都能通过统一入口生成
- [x] 所有函数输出 `String`
- [x] 不在业务 handler 内手写格式拼接

---

### Task 1.2: 迁移已有生成逻辑到统一入口
- [x] 将 `generate_brand_id()` 从 `auth/api_key.rs` 迁移/重定向到统一入口
- [x] 将 `batch_id` 的 `format!("batch_{}", ulid::Ulid::new())` 改为统一入口
- [x] 清理重复生成逻辑

**涉及文件**:
- `rust/rc-api/src/auth/api_key.rs`
- `rust/rc-api/src/db/batches.rs`
- 新增的统一 ID 模块

**验收标准**:
- [x] `brand_id` 与 `batch_id` 均由统一入口生成
- [x] 不再存在重复实现

---

## Phase 2: 修复主路径中的 `asset_id` 漂移（1 天）

### Task 2.1: 替换 `asset_id` 的 UUID 生成逻辑
- [x] 将资产创建主路径中的 `Uuid::new_v4()` 改为 `generate_asset_id()`
- [x] 明确 `assets.asset_id` 的正式格式为 `asset_<ULID>`
- [x] 检查其他新增资产路径，避免继续写入裸 UUID 文本

**涉及文件**:
- `rust/rc-api/src/routes/protocol.rs`
- 其他创建资产的路径（如有）

**验收标准**:
- [x] 新创建资产的 `asset_id` 均为 `asset_<ULID>`
- [x] 主业务路径中不再出现裸 UUID 业务资源 ID

---

### Task 2.2: 审查 `asset_id` 相关 DTO 与路由参数
- [x] 确认 `asset_id` 在路由层继续使用 `String`
- [x] 确认 `AssetDetail`、`AssetStateEvent`、响应结构体中的 `asset_id` 均为字符串
- [x] 清理可能残留的 `Uuid` 类型误用

**涉及文件**:
- `rust/rc-api/src/db/assets.rs`
- `rust/rc-api/src/routes/assets.rs`
- `rust/rc-api/src/routes/protocol.rs`
- 其他 `asset_id` 相关 DTO

**验收标准**:
- [x] 对外 DTO / 路由参数统一为 `String`
- [x] 不再出现“写入用 UUID、返回再转字符串”的混乱语义

---

## Phase 3: 规范 `product_id` 与 `session_id`（0.5 ~ 1 天）

### Task 3.1: 明确 `product_id` 的新增生成策略
- [x] 为 `product_id` 增加统一生成函数
- [x] 记录并替换后续新增产品路径中的自由字符串逻辑（当前写路径已废弃，因此以统一入口与测试/文档约束落地）
- [x] 保持历史只读兼容

**涉及文件**:
- 统一 ID 模块
- `rust/rc-api/src/db/products.rs`
- 未来产品创建路径（若存在）

**验收标准**:
- [x] 后续新增 `product_id` 有唯一合法生成方式：`product_<ULID>`
- [x] 历史数据不被破坏

---

### Task 3.2: 明确 `session_id` 的新增生成策略
- [x] 为 `session_id` 增加统一生成函数
- [x] 对 factory session 新增路径预留统一生成规范
- [x] 清理后续示例代码中的自由字符串倾向

**涉及文件**:
- 统一 ID 模块
- `rust/rc-api/migrations/20250101000001_init_batches_sessions.sql`
- session 相关业务代码（若后续补齐）

**验收标准**:
- [x] 后续新增 `session_id` 的唯一合法格式为 `session_<ULID>`

---

## Phase 4: 收敛 Seed / Fixture / Script（0.5 ~ 1 天）

### Task 4.1: 规范新增 seed / fixture 的 ID 风格
- [x] 清点现有 seed / fixture 中的历史 demo 风格 ID
- [x] 保留历史兼容数据，但明确标注为 legacy sample
- [x] 新增测试数据统一使用规范 ID

**涉及文件**:
- `rust/rc-api/src/seed.rs`
- `rust/rc-test-helpers/src/fixtures.rs`
- 相关测试脚本

**验收标准**:
- [x] 新增样例不再默认使用 `*-demo-*` / `*-001` 作为正式风格
- [x] 历史数据保留但不再充当模板

---

### Task 4.2: 更新测试断言与脚本假设
- [x] 测试不再只断言“是任意字符串”
- [x] 对新增主资源 ID 断言其前缀正确
- [x] 测试脚本、集成测试示例遵循统一格式

**验收标准**:
- [x] 关键测试能断言资源 ID 格式
- [x] 不再新增与规范冲突的示例

---

## Phase 5: 可选增强（0.5 ~ 1 天，可后置）

### Task 5.1: 评估增加数据库约束
- [x] 评估历史数据是否允许增加 `CHECK` 约束
- [x] 补 migration：
  - [x] `brand_id LIKE 'brand_%'`
  - [x] `product_id LIKE 'product_%'`
  - [x] `asset_id LIKE 'asset_%'`
  - [x] `batch_id LIKE 'batch_%'`
  - [x] `session_id LIKE 'session_%'`
- [x] 使用 `NOT VALID` 方式增加前缀约束，避免阻塞当前 legacy 数据读取，同时保护后续新数据

**验收标准**:
- [x] 约束方案有明确评估结果
- [x] 已执行为 migration，不悬空

---

### Task 5.2: 评估资源 ID 强类型包装
- [x] 引入：
  - [x] `BrandId(String)`
  - [x] `ProductId(String)`
  - [x] `AssetId(String)`
  - [x] `BatchId(String)`
  - [x] `SessionId(String)`
- [x] 为强类型包装增加验证、显示与转换实现
- [x] 保持当前业务主路径仍以 `String` 为主，强类型作为公共层增强能力先行落地

**验收标准**:
- [x] 有明确结论，不悬空

---

## 涉及文件清单

### 已修改
- [x] `docs/foundation/id-strategy.md`
- [x] `docs/specs/spec-06-id-unification.md`
- [x] `rust/rc-api/src/auth/api_key.rs`
- [x] `rust/rc-api/src/db/batches.rs`
- [x] `rust/rc-api/src/routes/protocol.rs`
- [x] `rust/rc-api/src/seed.rs`
- [x] `rust/rc-test-helpers/src/fixtures.rs`
- [x] `rust/rc-api/tests/activation_integration.rs`
- [x] `rust/rc-api/tests/verify_integration.rs`
- [x] `rust/rc-api/tests/authorization_integration.rs`
- [x] `rust/rc-api/src/auth/middleware.rs`
- [x] `rust/rc-test-helpers/tests/integration.rs`

### 已新增
- [x] `rust/rc-common/src/ids.rs`
- [x] `rust/rc-api/migrations/20250101000021_add_prefixed_id_constraints.sql`

### 已补齐依赖与导出
- [x] `rust/rc-common/src/lib.rs`
- [x] `rust/rc-common/Cargo.toml`
- [x] `rust/rc-test-helpers/Cargo.toml`

---

## 关键验收标准（DoD）

以下条件全部满足，视为本任务完成：

- [x] 已存在统一 ID 生成入口
- [x] `brand_id / product_id / asset_id / batch_id / session_id` 均有明确合法生成规则
- [x] `asset_id` 主写入路径不再使用裸 UUID 文本
- [x] `brand_id` 与 `batch_id` 已从散落逻辑迁入统一入口
- [x] DTO / 路由参数仍统一按字符串处理
- [x] 新增 seed / fixture / 测试数据遵守规范风格
- [x] 文档、任务、代码三者对同一 ID 规则描述一致
- [x] 可选增强中的数据库前缀约束与强类型包装已落地

---

## 风险提醒

### 风险 1：历史 demo 数据仍存在
已明确视为 legacy sample，仅用于兼容，不再作为新代码模板。

### 风险 2：仓库其他非本任务模块仍可能存在独立编译问题
例如与审批流相关的既有不一致，不属于本任务范围，但不影响本任务的 ID 策略落地。

### 风险 3：数据库前缀约束尚未 `VALIDATE`
当前采用 `NOT VALID` 保护新数据，不阻塞旧数据。待历史数据清洗完成后可再 `VALIDATE CONSTRAINT`。

---

## 推荐后续动作

### 推荐立即执行
- 执行最新 migration，落库前缀约束
- 在后续新增资源路径优先使用 `rc_common::ids` 中的强类型包装或统一生成器

### 推荐后续增强
- 在业务边界逐步引入 `BrandId` / `AssetId` 等强类型，减少字符串误传
- 清洗 legacy demo 数据后，对前缀约束执行 `VALIDATE CONSTRAINT`

---

## 一句话总结

> 本任务已完成：主资源 ID 已统一到“前缀可读字符串 + ULID”策略，`asset_id` 主路径漂移已修复，测试与样例已收敛，可选增强也已落地。
