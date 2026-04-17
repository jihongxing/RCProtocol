# Stage 5 异常流矩阵

> 文档类型：Ops  
> 适用阶段：Stage 5 MVP 可交付闭环  
> 最后更新：2026-04-10

---

## 1. 目的

本矩阵用于固定 Stage 5 当前必须覆盖的 P0 异常场景，明确：

- 触发条件
- 期望结果
- 恢复方式
- 对应自动化入口

---

## 2. P0 异常流矩阵

| 场景 | 触发条件 | 期望错误 / 结果 | 恢复方式 | 自动化入口 |
|------|----------|------------------|----------|------------|
| 激活前置状态不合法 | 对非 `Unassigned` 资产调用 `POST /assets/:id/activate` | `400 Bad Request` 或状态转换失败 | 修正资产状态后重试 | `activation_integration.rs` |
| 验真认证失败 | `cmac` 篡改 | `verification_status = authentication_failed` | 使用真实/正确认证参数重试 | `verify_integration.rs` |
| replay suspected | 同一 UID 以更小或相同 CTR 重复验真 | `verification_status = replay_suspected` / risk flag 包含 `replay_suspected` | 使用更大 CTR 或执行 CTR 校准 | `verify_integration.rs` |
| 承诺缺失 / incomplete_attestation | 标签认证通过，但缺少品牌或平台承诺 | `/verify/v2` 返回 `incomplete_attestation` | 补齐承诺对象或修复激活链路 | `verify_integration.rs` |
| API Key 无效 / 已撤销 | 使用旧 API Key 或不存在 Key 调用品牌接口 | `401 Unauthorized` | 使用最新有效 Key | `brand_registration_integration.rs`, `test-brand-registration.*` |
| Brand 越权 | 品牌访问其他品牌资源 | `403 Forbidden` | 使用本品牌资源或平台身份 | `brand_registration_integration.rs`, `authorization_integration.rs` |
| 售出冲突 | 对不允许售出的状态重复执行 `legal-sell` | `400 Bad Request` / 状态冲突 | 校验资产状态，避免重复售出 | `protocol_write_flow_integration.rs` |
| 过户冲突 | 被拒绝的 transfer 再 confirm，或参与方不匹配 | `409 Conflict` / `403 Forbidden` | 重建 transfer 或由正确参与方操作 | `transfer_actions_integration.rs` |
| 冻结 / 恢复限制行为 | 资产被冻结后继续执行受限动作 | `restricted` 或状态转换失败 | 先 recover 再继续 | `protocol_write_flow_integration.rs`, `verify_integration.rs` |
| Redis 不可用 | Redis 未启动或连接失败 | 服务仍可在 `DirectPg` 下工作 | 切换 `RC_API_FALLBACK_STRATEGY=DirectPg` 或恢复 Redis | `stage-5-mvp-runbook.md` |

---

## 3. 演示说明要求

对外演示时至少要能解释以下三种差异：

### 3.1 `authentication_failed`
表示标签动态认证失败，不成立为真实标签。

### 3.2 `replay_suspected`
表示标签认证参数可能被重放，需要视为风险态。

### 3.3 `incomplete_attestation`
表示标签真实性成立，但承诺完整性未成立，不能偷换成终局真品成立。

---

## 4. 关联自动化入口

- `rust/rc-api/tests/verify_integration.rs`
- `rust/rc-api/tests/brand_registration_integration.rs`
- `rust/rc-api/tests/transfer_actions_integration.rs`
- `rust/rc-api/tests/protocol_write_flow_integration.rs`
- `scripts/stage5-error-matrix.sh`
- `scripts/stage5-error-matrix.ps1`
