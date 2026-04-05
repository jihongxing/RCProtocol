# RCP DoD、联调回归矩阵与 GA 准入标准

## 1. DoD（完成定义）

## 1.1 功能DoD

- 每个 B 端模块都满足“三联单”：
  - 现有 Rust 映射点（endpoint/service）
  - 在位扩展点（归属到 `rc-api`/`rc-kms`/`rc-core`）
  - 可执行验收用例（接口级）
- Go 治理平面不重写任何核心密码学与协议逻辑。
- BConsole 每个写动作都具备审批流、幂等键和审计记录。

## 1.2 工程DoD

- 关键契约文档齐全：请求头、错误码、幂等语义、版本策略。
- 可观测性就绪：trace_id 贯穿 Go -> Rust -> Audit。
- 迁移与回滚演练至少各 1 次，记录可追溯。

## 1.3 安全DoD

- Delegation token 的 issue/verify/revoke/rotate 全链路可用。
- 高风险动作必须双重门控（审批 + 角色/策略校验）。
- 审计写入失败触发 fail-close，不允许静默成功。

## 2. 联调回归矩阵

### 2.1 主链路回归（必须全绿）

| 链路 | 接口序列 | 通过标准 |
| --- | --- | --- |
| 验证链路 | `resolve` -> `asset/verified` | SUN 解析成功，资产状态一致 |
| 发行链路 | `factory/batches` -> `factory/sessions` -> `blind-log` -> `entangle-active` | 入库与纠缠成功率达到门槛 |
| 信任链路 | `delegation issue/verify/revoke/rotate` | token 生命周期完整可验证 |
| 主权链路 | `sovereignty get/lock/release` | 状态迁移符合策略 |
| 转移链路 | `transfer initiate/confirm/cancel` | 幂等与冲突处理正确 |
| 恢复链路 | `recovery initiate -> verify* -> complete` | 失败可补偿，完成可审计 |
| 再生链路 | `rebirth initiate/authorize/complete` + `lineage` | 血缘链可追溯 |

### 2.2 治理链路回归（Go编排）

| 场景 | Go动作 | Rust验证点 | 判定 |
| --- | --- | --- | --- |
| 品牌发布 | 审批后发布品牌与SKU | `brand` `brand_products` | 状态与审计一致 |
| 批次执行 | 工位任务编排 | `factory_batch/session/blind-log` | 失败可重试且不重复写 |
| 策略发布 | 授权与风控策略应用 | `delegation` `geo-fence` `re-verify` | 生效版本正确 |
| 工单推进 | 转移/恢复工单推进 | `transfer` `recovery` `rebirth` | 阶段推进可回滚 |

### 2.3 兼容回归

- 旧客户端继续调用 `/v1/*` 不受影响。
- 新控制台通过 `/ops/v1/*` 调用时，结果与旧链路一致。
- 版本兼容窗口内，不出现强制升级阻断。

## 3. GA 准入门槛（可打勾）

## 3.1 集成覆盖率门槛

- [ ] B 端菜单 100% 具备 Rust 映射点。
- [ ] 在位扩展点全部归属清晰，无平行核心实现。
- [ ] Go 侧无密码学核心代码（KDF/签名/EV2）。

## 3.2 质量门槛

- [ ] 回归套件全绿（workspace tests + 关键集成测试）。
- [ ] 主链路 P95 达标（内部基线设定后固化）。
- [ ] 灰度阶段错误率不超过门槛（建议 <0.5%）。

## 3.3 安全门槛

- [ ] delegation 轮换演练通过且旧 token 在宽限期行为可控。
- [ ] 权限穿透测试通过（无越权读写）。
- [ ] 审计不可抵赖链完整（trace_id 可追到动作与审批）。

## 3.4 运维门槛

- [ ] Go治理层故障演练通过，Rust 核心可独立提供关键验证能力。
- [ ] 回滚SOP实测可用（<= 30 分钟恢复到稳定版本）。
- [ ] 监控告警覆盖接口错误、审计断链、幂等冲突、权限异常。

## 4. 发布建议（阶段化）

- `GA-1`（治理基础可用）：品牌/SKU/审批/审计上线。
- `GA-2`（发行闭环可用）：批次/会话/盲扫/纠缠全量上线。
- `GA-3`（全域可用）：策略中心、争议恢复、风控联动全量上线。

## 5. 退出条件（禁止GA）

- 任一主链路存在“不可回滚”的写入风险。
- 任一高风险动作缺少审批或审计。
- 出现 Go 平面替代 Rust 核心逻辑的实现偏移。
