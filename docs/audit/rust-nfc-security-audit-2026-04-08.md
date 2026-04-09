# RCProtocol Rust 全量安全审计报告（NFC-标签安全官）

- 审计范围：`rust/rc-common`、`rc-core`、`rc-crypto`、`rc-kms`、`rc-api`、`rc-test-helpers`
- 对照基线：
  - `docs/foundation/security-model.md`
  - `docs/foundation/state-machine.md`
  - `docs/foundation/api-and-service-boundaries.md`
  - `docs/engineering/hardware-and-ops-baseline.md`
  - `docs/engineering/technical-solution.md`
- 硬件基线：`NTAG 424 DNA` + `ACR122U`
- 日期：2026-04-08

---

## 总结论

### [安全判断]
当前 Rust 体系已经有状态机、KDF、CMAC、验真接口、母卡授权等主骨架，但从 NFC / 标签安全官视角看，**认证主链路还没有完全闭环**。问题集中在：

1. `K_chip_mother` 派生与安全模型文档不一致。
2. 过户链路对子标签只验 CMAC，不严控 CTR 单调递增。
3. 部分关键写操作绕过 `rc-core` 统一状态推进与统一审计。
4. API Key 安全实现与注释/配置意图不一致。
5. provision / readback / reset 的证据链还不够。

### [风险等级]
**总体：BLOCKER**

---

## 1. `rc-common`

### [安全判断]
属于系统集成 / 审计模型层，不直接做硬件认证。

### [标签 / 认证链路拆解]
这里定义了：`AssetState`、`AssetAction`、`ActorRole`、`AuditContext`、`AuditEvent`、`RcError`。

### 结论
- 优点：14 态与文档基本一致；`buyer_id` 已进入审计上下文，方向正确。
- 问题：审计结构还缺 NFC 关键证据，未直接覆盖：`uid`、`ctr`、`cmac_valid`、`key_epoch`、`authority_type`、`reader_id`、`readback_result`。

### [风险等级]
**MODERATE**

### [实验或修复建议]
把标签认证关键证据纳入结构化审计字段，而不是只留状态变化。

### [审计与留痕要求]
至少补齐：`uid`、`ctr`、`previous_ctr`、`cmac_valid`、`authority_type`、`key_epoch`、`reader/session id`。

---

## 2. `rc-core`

### [安全判断]
属于协议核心层，负责状态机与权限最终裁决。

### [标签 / 认证链路拆解]
当前已实现：状态推进、角色权限、冻结/恢复、LegalSell 的 `buyer_id` 约束。

### 结论
- 优点：终态/冻结态规则基本正确；Platform 执行业务动作要求 `approval_id`；状态机测试覆盖不错。
- 问题：NFC 安全前置条件没有被核心显式建模。像 `ActivateEntangle`、`ActivateConfirm`、`Transfer` 的安全性，仍靠 API 层自己拼。

### [风险等级]
**MODERATE**

### [实验或修复建议]
在核心层引入统一的认证前置检查摘要，例如：`uid_match`、`ctr_monotonic`、`cmac_valid`、`authority_verified`、`verification_mode`。

### [审计与留痕要求]
与标签认证有关的状态推进，不应只记动作名和状态变化，还要保留认证结论摘要。

---

## 3. `rc-crypto`

### [安全判断]
这是认证与密钥派生核心。

### [标签 / 认证链路拆解]
已实现：`HMAC-SHA256`、`AES-128 CMAC`、常量时间比较、`SUN Mode A` 验证、`Brand Key / Chip Key / Honey Key` 派生、敏感数据零化。

### 结论
- 优点：`SUN` 报文结构明确；CMAC 截断实现与注释一致；密钥零化、Debug 脱敏、常量时间比较做得不错。
- 严重问题：**没有独立的 `derive_mother_key`**。文档要求：
  - `K_chip_child = HMAC-SHA256(Brand_Key, UID || Epoch_LE)[..16]`
  - `K_chip_mother = HMAC-SHA256(Brand_Key, UID || Epoch_LE || "MOTHER")[..16]`
  但这里并未建立单独母卡派生域。

### [风险等级]
**BLOCKER**

### [实验或修复建议]
新增真正的 `derive_mother_key`，输入必须包含 `... || "MOTHER"`，并补充 child/mother 域隔离测试。

### [审计与留痕要求]
记录 `key_derivation_domain = child|mother|honey`，不要只记录“派生过”。

---

## 4. `rc-kms`

### [安全判断]
属于密钥体系实现问题，这里把 `rc-crypto` 的设计缺口放大成了实际调用风险。

### [标签 / 认证链路拆解]
负责 Root Key 加载、Brand Key 缓存，以及 `derive_chip_key` / `derive_honey_key` / `derive_mother_key`。

### 结论
- 优点：Root Key 读取后的 zeroize、日志脱敏、缓存方向都合理。
- 严重问题 A：`SoftwareKms::derive_mother_key()` 最终还是走 `derive_chip_key()` 逻辑，不符合安全模型文档。
- 严重问题 B：`authority_uid` 被截断/补零成 7 字节再参与派生，会造成虚拟母卡语义退化为“伪 UID”，存在碰撞/歧义风险。
- 严重问题 C：接口说支持可变长度 `authority_uid`，实现却只吃 7 字节，接口语义与实现语义不一致。

### [风险等级]
**BLOCKER**

### [实验或修复建议]
- 为母卡派生单独实现 KDF，不要复用 child 路径。
- `authority_uid` 必须完整参与 HMAC 消息，禁止截断到 7 字节。
- 增加碰撞测试：不同 `authority_uid` 前 7 字节相同，但完整值不同，派生结果也必须不同。

### [审计与留痕要求]
记录：`brand_id`、`authority_uid` 摘要、`key_epoch`、`domain=physical_mother|virtual_mother`。

---

## 5. `rc-api`

### [安全判断]
这是系统集成、认证、标签状态问题最集中的项目，也是本次发现最多高风险项的地方。

### [标签 / 认证链路拆解]
承载：public verify、authority device 注册、资产状态推进、过户、JWT / API Key 认证、CTR cache、DB 持久化。

### 结论

#### 优点
- `/verify` full flow 已具备：UID 查资产、`K_chip` 派生、CMAC 校验、CTR 比较、验真事件落库。
- `authentication_failed` 时不返回资产详情，避免明显泄露。
- 物理母卡授权验证已有：UID match、CTR fail-fast、CMAC verify、`atomic_update_ctr`。

#### 严重问题 A：API Key 哈希模型不一致
注释和配置里存在 `RC_API_KEY_SECRET`，但实际 `hash_api_key()` 只是裸 `SHA-256(api_key)`，并未使用 server secret。会带来更强的离线撞库面，也制造伪安全感。

#### 严重问题 B：physical mother card 注册缺品牌边界校验
`Brand` 角色可提交任意 `req.brand_id`，没有强制 `actor.brand_id == req.brand_id`。存在跨品牌注册母卡设备风险。

#### 严重问题 C：过户链路对子标签缺少 CTR 防重放
`transfer.rs` 的 `verify_child_tag()` 只验 UID/CTR/CMAC 格式和 CMAC，有**没有**：
- 校验 `new_ctr > last_ctr`
- 原子更新 child tag CTR
这会让过户链路对历史动态消息重放缺少硬阻断。

#### 严重问题 D：`transfer` 绕过 `rc-core` 统一主路径
`transfer_handler()` 直接更新 `assets` 并手写审计事件，没有统一走 `apply_action + persist_action`。这会导致状态机、审计、wallet snapshot、webhook、幂等等规则漂移。

#### 严重问题 E：虚拟母卡授权还不满足文档定义的账号级强认证
当前只做 user_id 和 stored hash 对比，没有看到：WebAuthn 断言、token 时效、设备指纹绑定、恢复后旧 token 作废策略。

#### 问题 F：degraded verify 允许 UID-only 探测
虽然不返回资产详情，但会暴露“UID 是否存在”的弱探测面。若公网开放，应严限流并明示这不构成验真。

#### 问题 G：replay 处置偏软
`/verify` 遇到 CTR 回滚时打 `replay_suspected`，但仍可能返回资产信息。对高价值标签场景偏软。

#### 问题 H：`blind_scan_asset` 直接创建资产并写成 `FactoryLogged`
这让 blind scan 同时承担“创建资产 + 推状态”的双重语义，偏离真源边界。

#### 问题 I：缺 provision/readback/reset 证据位
激活、母卡注册、entangle 等接口没有显式携带写入会话、readback 成功与否、回滚/复位计划等证据。

### [风险等级]
**BLOCKER**

### [实验或修复建议]
1. 给 `transfer` 的 child tag 校验补齐 CTR 单调与原子更新。
2. 让 `transfer` 回到 `rc-core` 统一状态推进主路径。
3. API Key 若目标是 HMAC-SHA256，就按目标真正实现；否则删除误导性配置和注释。
4. 补 `register_physical_mother_card` 的品牌边界校验。
5. 提高 replay 事件处置强度，至少触发高风险审计和可配置阻断。
6. 把 provision / readback / reset 结果纳入接口字段和审计证据。

### [审计与留痕要求]
必须补齐：
- public verify：`uid`、`ctr`、`previous_ctr`、`cmac_valid`、`verification_mode`、`trace_id`、客户端来源
- authority verify：`authority_type`、`authority_uid` 摘要、`last_known_ctr`、原子更新结果
- transfer：子标签验真结果、母卡授权结果、old/new owner、replay 标志
- 写入实验：`write_session_id`、`reader_id`、`readback_verified`、`reset_plan_ref`

---

## 6. `rc-test-helpers`

### [安全判断]
属于测试基础设施层，不直接认证，但会影响安全问题能否稳定复现。

### [标签 / 认证链路拆解]
提供临时数据库、fixture seed、authority / entanglement 测试数据。

### 结论
- 优点：适合集成测试状态机、品牌边界、授权绑定。
- 问题：缺少 NFC 安全场景化 fixture，例如：CTR 回滚、CMAC 错误向量、authority_uid 碰撞、provision 中断、readback 不一致、并发重放。

### [风险等级]
**LOW ~ MODERATE**

### [实验或修复建议]
补一套专门的 NFC 安全 fixture 和并发测试样例，尤其覆盖 replay、authority collision、physical mother concurrent CTR update。

### [审计与留痕要求]
测试产物应能稳定输出：原始 `uid/ctr/cmac`、预期结果、预期风险标志、是否更新 CTR。

---

## 跨项目问题清单

### P0 / BLOCKER
1. 母卡 KDF 不符合安全模型：`K_chip_mother` 域未独立。
2. `authority_uid` 被压缩成 7 字节，破坏虚拟母卡标识完整性。
3. 过户链路对子标签缺少 CTR 防重放。

### P1 / HIGH
4. `transfer` 绕过核心统一状态推进。
5. API Key 哈希模型与意图不一致。
6. Brand 可为其他品牌注册 physical mother card。
7. 虚拟母卡缺少 WebAuthn / token 时效 / 设备绑定。
8. replay 事件处置偏软。
9. 写入/实验缺少 provision-readback-reset 证据。

### P2 / MODERATE
10. UID-only degraded verify 存在探测面。
11. 协议核心缺统一认证前置条件模型。
12. 审计结构对 NFC 证据覆盖不足。
13. blind scan 的创建与推状态职责有漂移风险。

---

## 整改优先级

### 第一阶段：立即封堵
1. 修 `derive_mother_key` 域隔离。
2. 让 `authority_uid` 完整参与母卡 KDF。
3. 给 `transfer` child-tag 校验补 CTR 单调与原子更新。
4. 补 physical mother card 注册的品牌边界校验。
5. 统一 `transfer` 走 `rc-core` 主路径。

### 第二阶段：认证链路补强
6. 提升 replay 风险处置强度。
7. 收紧 degraded verify 使用边界和限流策略。
8. 重做虚拟母卡的账号级强认证模型。
9. 统一结构化审计字段。

### 第三阶段：硬件闭环
10. 面向 `NTAG 424 DNA + ACR122U` 建立 provision / readback / reset / 批量误写防护 / 失败排查的统一 runbook，并把关键结果映射到接口与审计字段。

---

## 最终判断

### [安全判断]
当前代码已经明确拒绝“只看 UID 就算验真”的大部分粗糙做法，但还**不能宣称链路已经站稳**。尤其以下四句话现在都不能说：

- “母卡和子卡密钥已经完全隔离”
- “CTR 回滚风险已经在所有关键动作里挡住”
- “虚拟母卡已经达到账号级强认证”
- “NFC 写入失败后可以完整追责和恢复”

### [风险等级]
**BLOCKER**

### [实验或修复建议]
如果只做两个最高优先级修复：

1. 先修 `mother key domain separation`
2. 再修 `transfer child-tag CTR replay protection`

这两项不修，近场安全链路都不算真正闭环。

### [审计与留痕要求]
下一版必须让任何一次“标签认证 / 母卡授权 / 写入实验”都能回答：

1. 哪张标签？
2. 当时 CTR 是多少、上一次是多少？
3. CMAC 是否通过？
4. 用的是 child 还是 mother 的哪条派生域？
5. 是物理母卡还是虚拟母卡？
6. 读卡器/会话是谁？
7. 写入后有没有 readback？
8. 失败后如何复位、如何追责？
