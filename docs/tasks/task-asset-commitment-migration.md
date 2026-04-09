# Task: AssetCommitment 迁移落地

> **关联 Spec**: [spec-asset-commitment.md](../specs/spec-asset-commitment.md)  
> **状态**: ⬜ 待开始  
> **优先级**: P0  
> **预计工期**: 3-5 天

---

## 任务目标

将 `AssetCommitment` 从文档对象落地为可并行写入、可被后续承诺与验真流程引用的工程对象。

本任务不要求立刻替换现有 `assets` 模型，而是完成以下桥接目标：

1. 新增 `asset_commitments` 持久化结构
2. 在激活链路中稳定生成 `asset_commitment_id`
3. 在 `assets` 与审计链路中建立桥接字段
4. 为后续 `Brand Attestation`、`Platform Attestation`、`Verification V2` 提供稳定引用对象

---

## Phase 1: 数据模型与 Migration（0.5 ~ 1 天）

### Task 1.1: 新增 `asset_commitments` 表
- [ ] 新增 migration，创建 `asset_commitments`
- [ ] 字段至少包括：
  - [ ] `commitment_id`
  - [ ] `payload_version`
  - [ ] `brand_id`
  - [ ] `asset_uid`
  - [ ] `chip_binding`
  - [ ] `epoch`
  - [ ] `metadata_hash`
  - [ ] `canonical_payload`
  - [ ] `created_at`
- [ ] 增加索引：
  - [ ] `brand_id + asset_uid + epoch`
  - [ ] `asset_uid`

**验收标准**:
- [ ] migration 可执行
- [ ] 表结构与 spec 一致
- [ ] 索引创建成功

---

### Task 1.2: 为 `assets` 增加桥接字段
- [ ] 新增 `assets.asset_commitment_id`
- [ ] 允许第一阶段为 `NULL`
- [ ] 为后续激活路径回填提供空间

**验收标准**:
- [ ] 旧数据不被破坏
- [ ] 新字段可被激活流程写入

---

### Task 1.3: 为审计表预留桥接能力
- [ ] 评估现有审计表是否需要新增 `asset_commitment_id`
- [ ] 至少保证关键激活审计事件可记录 `asset_commitment_id`

**验收标准**:
- [ ] 审计链路有明确桥接策略
- [ ] 不出现“承诺已落地但审计完全不可追踪”的情况

---

## Phase 2: Rust 领域结构与生成逻辑（1 天）

### Task 2.1: 定义 Rust 结构体
- [ ] 新增 `AssetCommitmentPayloadV1`
- [ ] 新增 `AssetCommitmentRecord`
- [ ] 如有必要，新增 `AssetCommitmentId(String)` 强类型包装

**建议位置**:
- `rust/rc-api/src/domain/asset_commitment.rs`
- 或 `rust/rc-common/` 中的共享协议对象模块

**验收标准**:
- [ ] 结构体字段与 spec 对齐
- [ ] 可被路由、db、测试共同引用

---

### Task 2.2: 实现规范化与哈希逻辑
- [ ] 实现 payload 规范化
- [ ] 实现 `metadata_hash` 生成函数
- [ ] 实现 `chip_binding` 生成函数
- [ ] 实现 `asset_commitment_id = sha256(canonical_payload)`

**验收标准**:
- [ ] 相同输入稳定生成相同 `asset_commitment_id`
- [ ] 关键字段变化会生成不同结果
- [ ] 单元测试覆盖规范化与哈希稳定性

---

### Task 2.3: 实现 DB 访问层
- [ ] 新增 `insert_asset_commitment()`
- [ ] 新增 `fetch_asset_commitment_by_id()`
- [ ] 新增 `fetch_asset_commitment_by_uid_epoch()` 或等价索引读取函数

**验收标准**:
- [ ] 编译通过
- [ ] 插入与查询测试通过
- [ ] 错误语义清晰

---

## Phase 3: 激活链路接入（1 ~ 1.5 天）

### Task 3.1: 接入激活主路径
- [ ] 在激活流程中确定 `brand_id / uid / epoch / external_product_*`
- [ ] 生成 `metadata_hash`
- [ ] 生成 `chip_binding`
- [ ] 生成 `AssetCommitment`
- [ ] 在同一事务中写入：
  - [ ] 现有 `assets`
  - [ ] 新 `asset_commitments`
  - [ ] `assets.asset_commitment_id`

**涉及路径**:
- `rust/rc-api/src/routes/protocol.rs`
- 或现有激活 handler 所在文件

**验收标准**:
- [ ] 激活成功后必有 `asset_commitment_id`
- [ ] 事务失败不会出现半成功写入

---

### Task 3.2: 激活响应扩展
- [ ] 在激活响应 DTO 中追加 `asset_commitment_id`
- [ ] 保持现有 `asset_id` 返回不被破坏

**验收标准**:
- [ ] 当前前端/调用方兼容
- [ ] 新调试或联调脚本可拿到 `asset_commitment_id`

---

## Phase 4: 读路径与审计增强（0.5 ~ 1 天）

### Task 4.1: 详情接口返回桥接信息
- [ ] 资产详情接口可返回 `asset_commitment_id`
- [ ] 内部调试接口可返回 `canonical_payload` 或摘要信息

**验收标准**:
- [ ] 研发联调时可确认 commitment 已生成

---

### Task 4.2: 审计事件增强
- [ ] 激活成功事件记录 `asset_commitment_id`
- [ ] 关键协议写操作开始支持记录 commitment 上下文

**验收标准**:
- [ ] 后续排查可从审计事件追到 commitment

---

## Phase 5: 测试与验证（0.5 ~ 1 天）

### Task 5.1: 单元测试
- [ ] 测试 canonical payload 稳定序列化
- [ ] 测试 `metadata_hash` 稳定性
- [ ] 测试 `chip_binding` 生成逻辑
- [ ] 测试 commitment hash 结果稳定

### Task 5.2: 集成测试
- [ ] 激活成功后断言 `asset_commitment_id` 已生成
- [ ] 重复相同输入不产生语义冲突
- [ ] 关键字段变化时生成不同 commitment

### Task 5.3: 联调脚本
- [ ] 扩展现有激活联调脚本，打印 `asset_commitment_id`

**验收标准**:
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 联调脚本可见结果

---

## 涉及文件建议

### Migration
- `rust/rc-api/migrations/*_create_asset_commitments.sql`
- `rust/rc-api/migrations/*_add_asset_commitment_id_to_assets.sql`

### Rust
- `rust/rc-api/src/db/assets.rs`
- `rust/rc-api/src/db/asset_commitments.rs`
- `rust/rc-api/src/routes/protocol.rs`
- `rust/rc-api/src/routes/assets.rs`
- `rust/rc-api/tests/activation_integration.rs`

---

## 关键验收标准（DoD）

- [ ] 激活链路稳定生成 `asset_commitment_id`
- [ ] `assets` 与 `asset_commitments` 完成桥接
- [ ] 现有 MVP 主链路不被破坏
- [ ] 后续承诺与验真 V2 可以直接引用该对象

---

## 非目标

本任务当前不包含：

- Brand Attestation 落地
- Platform Attestation 落地
- `/verify/v2` 完整实现
- 用 `AssetCommitment` 彻底替换 `asset_id`
