# Task: Verification V2 实现与阶段排期

> **关联 Spec**: [spec-verification-v2.md](../specs/spec-verification-v2.md)  
> **依赖任务**: [task-asset-commitment-migration.md](./task-asset-commitment-migration.md), [task-attestation-flows.md](./task-attestation-flows.md)  
> **状态**: ✅ 已完成  
> **优先级**: P0  
> **预计工期**: 3-4 天

---

## 任务目标

实现不破坏现有 `/verify` 的 `Verification V2`，让系统可以结构化地区分：

1. 标签真实性
2. 承诺完整性
3. 协议状态结论
4. 最终验真结论

同时输出一份清晰的阶段排期，确保 V2 能以灰度方式落地，而不是直接打断当前 MVP 验真链路。

---

## 前置依赖

在开始本任务前，必须已完成：

- [x] `AssetCommitment` 已落地
- [x] `Brand Attestation` 已落地
- [x] `Platform Attestation` 已落地
- [x] 当前 V1 验真链路已有稳定集成测试

---

## Phase 1: 响应模型与错误语义（0.5 天）

### Task 1.1: 定义 V2 响应 DTO
- [x] 新增 `VerificationV2Response`
- [x] 新增：
  - [x] `tag_authentication`
  - [x] `attestation_status`
  - [x] `protocol_state`
  - [x] `verification_status`

### Task 1.2: 定义错误码与状态枚举
- [x] `ASSET_COMMITMENT_NOT_FOUND`
- [x] `BRAND_ATTESTATION_MISSING`
- [x] `BRAND_ATTESTATION_INVALID`
- [x] `PLATFORM_ATTESTATION_MISSING`
- [x] `PLATFORM_ATTESTATION_INVALID`
- [x] `REPLAY_SUSPECTED`
- [x] `TAG_AUTHENTICATION_FAILED`

**验收标准**:
- [x] DTO 与 spec 一致
- [x] 错误码不与现有 V1 语义混淆

---

## Phase 2: `/verify/v2` 路由实现（1 ~ 1.5 天）

### Task 2.1: 复用当前标签动态认证链路
- [x] 复用现有 `uid / ctr / cmac` 解析与校验逻辑
- [x] 复用现有 `CTR` replay 检查
- [x] 不重复造一套标签认证逻辑

### Task 2.2: 接入 `AssetCommitment` 定位
- [x] 通过 `uid + epoch` 或桥接关系定位 `asset_commitment_id`
- [x] 若找不到，返回 `ASSET_COMMITMENT_NOT_FOUND`

### Task 2.3: 接入 Brand / Platform Attestation 校验
- [x] 校验品牌承诺是否存在且有效
- [x] 校验平台承诺是否存在且有效

### Task 2.4: 接入状态与风险判断
- [x] 读取当前状态
- [x] 结合风控标记输出 `restricted / authentic / incomplete_attestation`

**验收标准**:
- [x] `/verify/v2` 独立可用
- [x] 不影响 `/verify` 现有行为

---

## Phase 3: 审计与观测（0.5 天）

### Task 3.1: V2 验真事件审计增强
- [x] 记录 `verification_version = v2`
- [x] 记录 `asset_commitment_id`
- [x] 记录品牌与平台承诺状态
- [x] 记录最终结论

### Task 3.2: 调试与日志增强
- [x] 增加结构化日志字段
- [x] 便于排查“标签真实但承诺不完整”的场景

**验收标准**:
- [x] 审计可区分 V1 与 V2
- [x] 问题排查时可看到承诺状态细节

---

## Phase 4: 测试与联调（0.5 ~ 1 天）

### Task 4.1: 单元测试
- [x] 结论判定矩阵测试：
  - [x] 标签通过 + 双承诺有效 + 正常状态 = `authentic`
  - [x] 标签通过 + 承诺缺失 = `incomplete_attestation`
  - [x] 标签通过 + 状态受限 = `restricted`
  - [x] 标签失败 = `authentication_failed`

### Task 4.2: 集成测试
- [x] `/verify` 保持原行为
- [x] `/verify/v2` 返回结构化新结果
- [x] 至少覆盖 4 类核心结果

### Task 4.3: 联调脚本
- [x] 新增 `verify-v2` 联调脚本
- [x] 打印完整结构化响应

**验收标准**:
- [x] 测试覆盖核心结果矩阵
- [x] 脚本可直接用于演示 V1 / V2 区别

---

## Phase 5: 阶段排期（Implementation Sequencing）

### Stage A: 数据桥接先行
- [x] 先完成 `AssetCommitment`
- [x] 不碰前端与 `/verify`

### Stage B: 承诺对象落地
- [x] 完成品牌承诺与平台承诺
- [x] 激活响应开始暴露承诺状态

### Stage C: 验真 V2 灰度
- [x] 新增 `/verify/v2`
- [x] 仅供内部联调与协议演示使用

### Stage D: 前端与对外接口评估
- [x] 评估是否在 C 端增加 V2 结果解释
- [x] 评估品牌 API 是否暴露 V2 结果

### Stage E: 替换策略评估
- [x] 只有在 V2 足够稳定后，才评估是否让其成为默认验真路径

---

## 研发执行顺序建议

推荐执行顺序：

1. `task-asset-commitment-migration.md`
2. `task-attestation-flows.md`
3. 本任务 `/verify/v2` 落地
4. 前端与品牌接口灰度评估

不建议顺序：

- 先写 `/verify/v2`，再补承诺对象
- 先改前端展示，再补后端判定模型

---

## 涉及文件建议

### Rust
- `rust/rc-api/src/routes/verify.rs`
- `rust/rc-api/src/routes/assets.rs`
- `rust/rc-api/src/db/asset_commitments.rs`
- `rust/rc-api/src/db/brand_attestations.rs`
- `rust/rc-api/src/db/platform_attestations.rs`
- `rust/rc-api/tests/verify_integration.rs`

### Scripts
- `scripts/test-verify-v2.sh`
- `scripts/test-verify-v2.ps1`

---

## 关键验收标准（DoD）

- [x] `/verify/v2` 成功上线且不破坏 `/verify`
- [x] 能明确区分“标签真实”与“终局协议真品成立”
- [x] 能展示双承诺状态
- [x] 可作为 Stage 8~10 的协议演进演示入口

---

## 非目标

本任务当前不包含：

- C 端最终文案与 UI 全量改版
- 对外默认切换到 V2
- 多协议版本协商机制
