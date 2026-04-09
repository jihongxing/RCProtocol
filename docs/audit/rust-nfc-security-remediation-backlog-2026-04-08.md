# RCProtocol Rust 安全整改任务清单（排期版）

- 来源报告：`docs/audit/rust-nfc-security-audit-2026-04-08.md`
- 目的：把审计结论转成可分配、可排期、可验收的研发任务
- 范围：`rust/` 下全部子项目
- 日期：2026-04-08

---

## 一、排期建议总览

建议按 3 个阶段推进：

### Phase 1：立即封堵 `BLOCKER`
目标：先把最可能打穿 NFC / 动态认证主链路的问题堵住。

建议周期：**1~1.5 周**

### Phase 2：补强 `HIGH`
目标：把品牌边界、账号级授权、重放处置、统一审计补齐。

建议周期：**1.5~2 周**

### Phase 3：补完工程闭环与测试资产
目标：把 provision / readback / reset / fixture / runbook 全部工程化。

建议周期：**1~2 周**

---

## 二、任务清单（按优先级）

---

## P0 / BLOCKER

### TASK-001：修复 `K_chip_mother` 独立派生域

- 优先级：`P0`
- 风险等级：`BLOCKER`
- 涉及项目：
  - `rust/rc-crypto`
  - `rust/rc-kms`
- 责任建议：`密码学 / 协议核心`

#### 背景
当前母卡密钥派生没有按安全模型建立独立域，存在 child/mother 密钥空间混淆风险。

#### 目标
实现与文档一致的独立派生：
- `K_chip_child = HMAC-SHA256(Brand_Key, UID || Epoch_LE)[..16]`
- `K_chip_mother = HMAC-SHA256(Brand_Key, UID_or_authority_uid || Epoch_LE || "MOTHER")[..16]`

#### 具体任务
- 在 `rc-crypto::kdf` 增加独立 `derive_mother_key`
- 在 `rc-kms` 中改用新的 mother 派生逻辑
- 禁止 `derive_mother_key` 复用 child 派生路径
- 明确 child / mother / honey 三个 domain 的接口与注释

#### 验收标准
- 同品牌、同 UID、同 epoch：`child key != mother key`
- 不同 brand 派生结果不同
- 不同 epoch 派生结果不同
- 现有 child key 测试全部通过
- 新增 mother key 测试通过

#### 依赖
- 无

---

### TASK-002：修复 `authority_uid` 被截断为 7 字节的问题

- 优先级：`P0`
- 风险等级：`BLOCKER`
- 涉及项目：
  - `rust/rc-kms`
  - `rust/rc-api`
- 责任建议：`密码学 / API`

#### 背景
当前虚拟母卡 `authority_uid` 被截断/补零到 7 字节参与派生，这会造成 authority 标识语义失真，并带来碰撞风险。

#### 目标
让 `authority_uid` 完整参与 mother key 派生，不再压缩为 7 字节伪 UID。

#### 具体任务
- 重构 `derive_mother_key(brand_id, authority_uid, epoch)` 输入模型
- 对物理母卡和虚拟母卡分别定义输入域
- 移除截断/补零逻辑
- 增加碰撞测试样例

#### 验收标准
- 两个前 7 字节相同、完整值不同的 `authority_uid`，派生结果必须不同
- virtual mother card 认证流程仍可正常跑通
- 不影响 physical mother card 路径

#### 依赖
- 依赖 `TASK-001`

---

### TASK-003：给 `transfer` 子标签校验补 CTR 防重放

- 优先级：`P0`
- 风险等级：`BLOCKER`
- 涉及项目：
  - `rust/rc-api`
  - 可能补测试到 `rust/rc-test-helpers`
- 责任建议：`API / 协议安全`

#### 背景
当前过户链路对子标签只校验 CMAC，没有严格校验 `new_ctr > last_ctr`，存在历史动态消息重放风险。

#### 目标
使过户链路与 public verify 一样，对 child tag 强制执行 CTR 单调递增和原子更新。

#### 具体任务
- 在 `transfer.rs` 的 `verify_child_tag()` 中读取 child asset 历史 CTR
- 拒绝 `new_ctr <= last_ctr`
- 成功后执行原子更新
- 为 replay 冲突返回明确风险语义
- 增加并发场景测试

#### 验收标准
- 正常递增 CTR 的过户可通过
- 相同 CTR 重放必须失败
- 回滚 CTR 必须失败
- 并发重复提交只允许一个成功

#### 依赖
- 无

---

## P1 / HIGH

### TASK-004：让 `transfer` 回到 `rc-core` 统一状态推进主路径

- 优先级：`P1`
- 风险等级：`HIGH`
- 涉及项目：
  - `rust/rc-api`
  - `rust/rc-core`
- 责任建议：`协议核心 / API`

#### 背景
当前 `transfer` 直接写 `assets` 和审计表，绕开统一的 `apply_action + persist_action` 主路径。

#### 目标
统一状态推进、幂等、owner 更新、审计、wallet snapshot、webhook 逻辑。

#### 具体任务
- 重构 `transfer_handler`
- 授权校验通过后调用 `rc-core::apply_action`
- 统一使用 `persist_action`
- 删除 transfer 中的重复直写逻辑

#### 验收标准
- transfer 成功后资产状态、owner、审计事件与统一主路径一致
- 幂等键重复提交行为与其他 asset action 一致
- transfer 的 webhook / wallet snapshot 行为不漂移

#### 依赖
- 建议依赖 `TASK-003`

---

### TASK-005：修复 API Key 哈希模型与配置意图不一致问题

- 优先级：`P1`
- 风险等级：`HIGH`
- 涉及项目：
  - `rust/rc-api`
- 责任建议：`API / 安全基础设施`

#### 背景
代码中存在 `RC_API_KEY_SECRET`，但实际哈希逻辑是裸 `SHA-256(api_key)`，与注释和设计意图不一致。

#### 目标
二选一：
1. 真正实现 `HMAC-SHA256(server_secret, api_key)`；或
2. 删除误导性 secret 配置与注释，明确当前就是裸 hash。

建议采用方案 1。

#### 具体任务
- 重构 `hash_api_key()`
- 统一注册、轮换、鉴权的哈希算法
- 设计旧数据迁移方案
- 增加兼容期策略（如需要）

#### 验收标准
- 新签发 API Key 可正常认证
- 老 key 的迁移策略明确且可执行
- 注释、配置、实现三者一致

#### 依赖
- 无

---

### TASK-006：补 physical mother card 注册的品牌边界校验

- 优先级：`P1`
- 风险等级：`HIGH`
- 涉及项目：
  - `rust/rc-api`
- 责任建议：`API`

#### 背景
当前 Brand 角色理论上可为其他品牌注册 mother card。

#### 目标
确保：
- `Platform` 可跨品牌
- `Brand` 只能操作自己的 `brand_id`

#### 具体任务
- 在 `register_physical_mother_card` 中增加 `actor.brand_id == req.brand_id` 校验
- 补对应测试

#### 验收标准
- Brand 跨品牌注册返回拒绝
- Platform 跨品牌注册仍可执行
- 本品牌注册不受影响

#### 依赖
- 无

---

### TASK-007：升级 replay 风险处置策略

- 优先级：`P1`
- 风险等级：`HIGH`
- 涉及项目：
  - `rust/rc-api`
  - `rust/rc-core`
- 责任建议：`协议安全 / 风控`

#### 背景
当前 public verify 遇到 replay 多为软提示，风险语义偏弱。

#### 目标
把 replay 从“提示”提升为“可配置阻断 / 可触发治理”的风险事件。

#### 具体任务
- 统一 replay 风险枚举与错误语义
- 支持配置：返回 restricted / 触发工单 / 触发冻结前检查
- 审计事件中明确标记 replay suspected

#### 验收标准
- replay 不再与正常 verified 响应语义混淆
- replay 事件可被检索、统计、追责

#### 依赖
- 建议依赖 `TASK-003`

---

### TASK-008：补齐虚拟母卡的账号级强认证能力

- 优先级：`P1`
- 风险等级：`HIGH`
- 涉及项目：
  - `rust/rc-api`
  - 可能联动 Go IAM / WebAuthn 侧
- 责任建议：`身份认证 / API`

#### 背景
当前虚拟母卡更像“长期 hash token 比对”，还不符合文档中的 WebAuthn + token 防伪要求。

#### 目标
让虚拟母卡具备真正的账号级强认证语义。

#### 具体任务
- 定义虚拟母卡 token 时效
- 接入 WebAuthn 断言验证结果
- 引入设备绑定或设备指纹摘要
- 明确恢复/轮换后的旧 token 作废策略

#### 验收标准
- 失效 token 不可通过
- 用户不匹配不可通过
- 已轮换/已恢复后的旧 token 不可继续通过
- 能输出结构化失败原因

#### 依赖
- 需要产品/身份系统配合

---

### TASK-009：把 provision / readback / reset 证据纳入接口与审计

- 优先级：`P1`
- 风险等级：`HIGH`
- 涉及项目：
  - `rust/rc-api`
  - 可能补 `rc-common`
- 责任建议：`硬件集成 / API / 审计`

#### 背景
当前激活、entangle、母卡注册等流程里，缺少硬件写入后的证据位。

#### 目标
任何 NFC 写入型动作，都能回答：写了什么、是否 readback、失败怎么复位。

#### 具体任务
- 在请求/审计模型中补字段：
  - `write_session_id`
  - `reader_id`
  - `provision_result`
  - `readback_verified`
  - `reset_plan_ref`
- 明确哪些接口必须带这些字段

#### 验收标准
- 关键写入流程的审计事件中能看到这些证据
- 写入失败后可追溯到对应 reset plan

#### 依赖
- 需要硬件流程定义配合

---

## P2 / MODERATE

### TASK-010：收紧 degraded verify 的使用边界

- 优先级：`P2`
- 风险等级：`MODERATE`
- 涉及项目：`rust/rc-api`
- 责任建议：`API / 风控`

#### 目标
避免 UID-only 模式被误解成验真接口或被用于探测。

#### 具体任务
- 明确 degraded 模式仅用于弱场景
- 增加限流 / 白名单 / 运营开关
- 响应文案中明确“不构成真实性验证”

#### 验收标准
- degraded 模式不会输出让调用方误认为“标签真实”的结果
- 接口限流策略清晰

---

### TASK-011：把 NFC 认证前置条件提升为核心模型

- 优先级：`P2`
- 风险等级：`MODERATE`
- 涉及项目：
  - `rust/rc-core`
  - `rust/rc-common`
- 责任建议：`协议核心`

#### 目标
用统一对象表达认证前置条件，避免分散在 handler 里各自实现。

#### 建议字段
- `uid_match`
- `ctr_monotonic`
- `cmac_valid`
- `authority_verified`
- `device_status_ok`
- `verification_mode`

#### 验收标准
- 关键动作能消费统一认证前置对象
- handler 自拼规则减少

---

### TASK-012：补齐 NFC 结构化审计字段

- 优先级：`P2`
- 风险等级：`MODERATE`
- 涉及项目：
  - `rust/rc-common`
  - `rust/rc-api`
- 责任建议：`审计 / API`

#### 目标
让任意一次标签认证都可追责。

#### 具体任务
在审计或 verification event 中补齐：
- `uid`
- `ctr`
- `previous_ctr`
- `cmac_valid`
- `authority_type`
- `key_epoch`
- `reader_id`
- `verification_mode`

#### 验收标准
- 日志/审计查询能直接还原关键认证事实

---

### TASK-013：梳理 blind scan 的创建与推状态职责

- 优先级：`P2`
- 风险等级：`MODERATE`
- 涉及项目：`rust/rc-api`
- 责任建议：`API / 协议核心`

#### 目标
避免 blind scan 同时承担“创建资产 + 推进状态”的混合语义。

#### 具体任务
- 评估 blind scan 是否应拆成：
  - 资产登记
  - 状态推进
- 保证状态迁移与 Foundation 文档一致

#### 验收标准
- blind scan 的前态和后态语义清晰
- 不再依赖“插入即视为已经从 PreMinted 推进”这种隐式逻辑

---

## 三、测试与验收专项任务

### TASK-014：补齐 NFC 安全 fixture 与并发测试

- 优先级：`P1`
- 涉及项目：`rust/rc-test-helpers`
- 责任建议：`测试 / 协议安全`

#### 目标
让以下问题都能稳定复现：
- CTR 回滚
- 同 CTR 重放
- CMAC 错误向量
- authority_uid 碰撞
- virtual authority 过期 token
- physical mother concurrent CTR update
- provision 中断 / readback 不一致

#### 验收标准
- 新增集成测试可覆盖上述核心场景
- CI 可稳定跑过

---

### TASK-015：输出硬件 runbook 与恢复手册映射表

- 优先级：`P2`
- 涉及范围：文档 + 硬件流程
- 责任建议：`硬件集成 / 运维 / 测试`

#### 目标
形成面向 `NTAG 424 DNA + ACR122U` 的统一执行清单：
- provision
- readback
- reset to baseline
- 批量误写防护
- 失败排查

#### 验收标准
- 任一写卡实验都能找到对应恢复路径
- 关键步骤可映射到 API / 审计字段

---

## 四、建议的人员分工

### 1. 密码学 / 协议核心
负责：
- `TASK-001`
- `TASK-002`
- `TASK-011`

### 2. API / 后端
负责：
- `TASK-003`
- `TASK-004`
- `TASK-005`
- `TASK-006`
- `TASK-007`
- `TASK-010`
- `TASK-013`

### 3. 身份认证 / 账号安全
负责：
- `TASK-008`

### 4. 硬件集成 / 测试 / 运维
负责：
- `TASK-009`
- `TASK-014`
- `TASK-015`

---

## 五、推荐排期方案

### Sprint A（先封堵）
- `TASK-001`
- `TASK-002`
- `TASK-003`
- `TASK-006`

### Sprint B（统一主链路）
- `TASK-004`
- `TASK-005`
- `TASK-007`
- `TASK-012`
- `TASK-014`

### Sprint C（补强账号级与硬件闭环）
- `TASK-008`
- `TASK-009`
- `TASK-010`
- `TASK-011`
- `TASK-013`
- `TASK-015`

---

## 六、排期时的硬性门槛

以下 3 项建议设为上线前硬门槛：

1. `TASK-001` 完成：母卡派生域独立
2. `TASK-003` 完成：过户 child-tag CTR 防重放落地
3. `TASK-006` 完成：品牌边界不可越权注册 mother card

如果这 3 项没完成，不建议对外宣称 NFC 安全链路已闭环。

---

## 七、给管理层的简版结论

如果要一句话总结排期重点：

> 先修密钥域隔离和过户反重放，再收紧品牌边界与统一状态真源，最后补账号级强认证和硬件写入证据链。
