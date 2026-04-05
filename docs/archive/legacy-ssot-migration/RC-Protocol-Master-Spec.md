# RC-Protocol Master Spec — 代码权威技术规范

> **本文档为 RC-Protocol 代码事实的单一权威来源（Single Source of Truth）。**
> 所有章节内容均基于代码实现提取，当文档与代码存在冲突时，以代码为准。
> 其他文档（`RC-Protocol-Specification.md`、`sovereignty-architecture.md`）应引用本文档作为权威参考。
>
> 最后更新：2025-07

---

## 一、协议概述

### 项目定位

RC-Protocol（Regalis Clavis Protocol，"王者之钥"协议）是面向高价值实物资产的数字主权协议。通过 NFC 芯片（NTAG 424 DNA）+ 密码学（AES-128 CMAC / HMAC-SHA256）+ 14 态状态机实现"人-物-权"三位一体的资产主权管理。

### 核心价值

- **防伪验真**：基于 NTAG 424 DNA 芯片的 SUN（Secure Unique NFC）动态消息，每次 NFC 扫描生成唯一的 CMAC 签名，不可伪造、不可重放
- **数字主权**：资产所有权通过 14 态状态机精确管理，支持锁定、释放、转让、争议冻结与恢复
- **全生命周期**：从工厂盲扫入库（PreMinted）到消费/传承/销毁（终态），覆盖资产完整生命周期

### 三方解耦

| 角色 | 职责 | 代码映射 |
|------|------|----------|
| 品牌方（Brand） | 资产注册、纠缠激活、合法销售 | `ActorRole::Brand` |
| 平台方（Platform） | 系统管理、密钥托管、管理员覆盖 | `ActorRole::Platform` |
| 消费者（Consumer） | 验真扫描、主权持有、C2C 转让 | `ActorRole::Consumer` |
| 工厂（Factory） | 芯片盲扫入库、UID 登记 | `ActorRole::Factory` |
| 审核员（Moderator） | 安全标记、争议冻结与恢复 | `ActorRole::Moderator` |

---

## 二、KDF 密钥派生体系

> **代码权威**：`rc-core/src/crypto/kdf.rs`

### 派生链总览

```
Root_Key (32B, HMAC-SHA256 密钥)
  │
  ├─ Brand_Key = HMAC-SHA256(Root_Key, Brand_ID || System_ID)     → 32 字节
  │    │
  │    ├─ K_chip = HMAC-SHA256(Brand_Key, UID || Epoch_LE)[..16]  → 16 字节（AES-128）
  │    │
  │    └─ K_honey = HMAC-SHA256(Brand_Key, Serial_BE)             → 32 字节（全量，不截断）
```

### 派生公式

#### Brand_Key（品牌密钥，32 字节）

```
Brand_Key = HMAC-SHA256(Root_Key, Brand_ID || System_ID)
```

- `Root_Key`：32 字节根密钥，`HmacKey` 类型，`Zeroize + ZeroizeOnDrop`
- `Brand_ID`：品牌标识字节序列（不可为空，否则返回 `CryptoError::InvalidKeyMaterial`）
- `System_ID`：系统标识字节序列（可为空，向后兼容旧版本）
- 输出：32 字节 `HmacKey`

#### K_chip（芯片终端密钥，16 字节 AES-128）

```
K_chip = HMAC-SHA256(Brand_Key, UID || Epoch_LE)[..16]
```

- `Brand_Key`：32 字节品牌密钥
- `UID`：7 字节芯片 UID
- `Epoch_LE`：4 字节小端序 epoch 值（密钥轮换代数）
- 输出：HMAC-SHA256 结果**截断前 16 字节**，封装为 `KeyHandle`（内含 `AesKey`）
- 中间 HMAC 输出和消息缓冲区在截断后立即 zeroize

#### K_honey（蜜獾标记密钥，32 字节 HMAC-SHA256）

```
K_honey = HMAC-SHA256(Brand_Key, Serial_BE)
```

- `Brand_Key`：32 字节品牌密钥
- `Serial_BE`：4 字节大端序序列号
- 输出：**全量 32 字节**（`Zeroizing<[u8; 32]>`），**不做截断**
- 用于 HBM（Honey Badger Marker）计算：`HBM = Truncate(HMAC-SHA256(K_honey, UID || CTR_LE), 4)`

### 安全约束

- `HmacKey` 和 `KeyHandle` 均实现 `Zeroize + ZeroizeOnDrop`，禁止 `Clone`
- `Debug` 输出为 `[REDACTED]`，禁止泄露密钥字节
- 中间密钥（Brand_Key）在派生出下级密钥后应立即清零
- 所有密钥比较使用 `subtle::ConstantTimeEq`，禁止 `==` 运算符


---

## 三、14 态资产状态机

> **代码权威**：`rc-common/src/states.rs`（`AssetStateConst` 枚举）、`rc-core/src/state_machine/states.rs`（`AssetState` 枚举 + `validate_recovery`）

### 完整状态枚举表

| 状态码 | 枚举名 | 常量名 | 分类 | 说明 |
|--------|--------|--------|------|------|
| 0 | PreMinted | PRE_MINTED | Virgin | 预铸造，芯片 UID 已录入但密钥全零 |
| 1 | FactoryLogged | FACTORY_LOGGED | Virgin | 工厂已登记，盲扫入库完成 |
| 2 | Activated | ACTIVATED | Enlightened | 已激活，ChangeKey 已执行，品牌已绑定 |
| 3 | LegallySold | LEGALLY_SOLD | Tangled | 合法售出，子母标签逻辑锚定 |
| 4 | Transferred | TRANSFERRED | Tangled | 已转让，所有权已变更 |
| 5 | Consumed | CONSUMED | 终态 | 已消费，不可逆 |
| 6 | Legacy | LEGACY | 终态 | 传承，不可逆 |
| 7 | Tampered | TAMPERED | 终态 | 被篡改，安全标记 |
| 8 | Compromised | COMPROMISED | 终态 | 已泄露，安全标记 |
| 9 | Destructed | DESTRUCTED | 终态 | 已销毁，不可逆 |
| 10 | Unassigned | UNASSIGNED | Virgin | 未分配，等待品牌认领 |
| 11 | RotatingKeys | ROTATING_KEYS | Enlightened | 密钥轮换中（瞬态），ChangeKey 执行期间 |
| 12 | EntangledPending | ENTANGLED_PENDING | Enlightened | 纠缠待确认（瞬态），子母标签绑定中 |
| 13 | Disputed | DISPUTED | 冻结态 | 争议冻结（可逆），冻结期间禁止一切流转 |

### 状态分类

- **终态**（Terminal）：`Consumed(5)`, `Legacy(6)`, `Tampered(7)`, `Compromised(8)`, `Destructed(9)` — 进入后不可再转换
- **瞬态**（Transient）：`RotatingKeys(11)`, `EntangledPending(12)` — 短暂中间状态，正常流程中自动离开
- **冻结态**（Frozen）：`Disputed(13)` — 冻结期间禁止一切流转操作，仅 Moderator 可解冻

### 主权三阶段映射（业务心智模型）

| 主权阶段 | 底层状态 | 密钥状态 |
|----------|----------|----------|
| **Virgin（初始态）** | PreMinted(0), FactoryLogged(1), Unassigned(10) | 全零出厂密钥 |
| **Enlightened（赋灵态）** | RotatingKeys(11), EntangledPending(12), Activated(2) | 私有密钥已注入 |
| **Tangled（纠缠态）** | LegallySold(3), Transferred(4) | 子母纠缠完成，所有权确立 |

终态和冻结态不属于三阶段抽象，它们是跨阶段的安全机制。

---

## 四、角色权限矩阵

> **代码权威**：`rc-core/src/state_machine/transitions.rs::check_permission`

### 5 角色完整权限矩阵

| 角色 | 允许的状态转换 | 说明 |
|------|---------------|------|
| **Platform** | 任意合法转换（`check_permission` 始终返回 `true`） | 管理员覆盖，可触发任意转换 |
| **Factory** | PreMinted→FactoryLogged, FactoryLogged→Unassigned | 工厂盲扫入库流程 |
| **Brand** | Unassigned→RotatingKeys, RotatingKeys→EntangledPending, EntangledPending→Activated, Activated→LegallySold | 品牌纠缠激活与销售流程 |
| **Consumer** | LegallySold→{Transferred, Consumed, Legacy}, Transferred→{Transferred, Consumed, Legacy} | C2C 转让、消费、传承 |
| **Moderator** | any→{Tampered, Compromised, Disputed}（安全标记）, Disputed→any（恢复/解冻） | 安全审核与争议处理 |

### 关键业务流程路径

#### 盲扫入库（Factory）
```
PreMinted(0) → FactoryLogged(1) → Unassigned(10)
```

#### 纠缠激活（Brand）
```
Unassigned(10) → RotatingKeys(11) → EntangledPending(12) → Activated(2)
```

#### 销售与转让
```
Activated(2) → LegallySold(3) → Transferred(4) → Transferred(4)（可多次转让）
                    │                    │
                    ├→ Consumed(5)       ├→ Consumed(5)
                    └→ Legacy(6)         └→ Legacy(6)
```

#### 争议冻结与恢复（Moderator）
```
any → Disputed(13) → previous_state（通过三重验证恢复）
```


---

## 五、SUN 消息规范

> **代码权威**：`rc-common/src/sun_spec.rs`

### Mode A — 明文兼容模式（MVP 默认）

UID 和 CTR 以明文 hex 编码嵌入 URL 查询参数。

#### URL 模板

```
https://rc-protocol.com/v?u={UID_HEX}&c={CTR_HEX}&m={CMAC_HEX}
```

#### 字段规格

| 参数 | 查询键 | 字节长度 | Hex 字符长度 | 编码 | 说明 |
|------|--------|----------|-------------|------|------|
| UID | `u` | 7 字节 | 14 字符 | 大端序 hex | 芯片唯一标识 |
| CTR | `c` | 3 字节 | 6 字符 | 大端序 hex | NFC 扫描计数器（SDM 镜像输出为大端序）|
| CMAC | `m` | 8 字节 | 16 字符 | hex | AES-128 CMAC 签名 |

#### CTR 约束

- 最大值：`CTR_MAX = 0x00FF_FFFF`（16,777,215），达到后芯片停止生成 SUN 消息
- 预警阈值：`CTR_WARNING_THRESHOLD = 0x00FF_F000`（约 99.6%），触发芯片更换预警

### Mode B — 加密隐私模式（扩展）

UID 和 CTR 通过 SDM（Secure Dynamic Messaging）加密传输，需 K1 密钥解密。

- SDM Session Vector (SV2) 前缀：`[0x3C, 0xC3, 0x00, 0x01, 0x00, 0x80]`
- SV2 总长度：16 字节
- 加密字段偏移量通过 NTAG 424 DNA 芯片的 SDM 配置寄存器设定
- 解密流程：使用 SV2 构造 AES-128 CBC-IV，以 K1 解密 UID 和 CTR 密文

> MVP 阶段默认使用 Mode A，Mode B 为隐私增强扩展，适用于需要隐藏 UID 的高安全场景。

### HBM（Honey Badger Marker）扩展参数

| 参数 | 查询键 | 字节长度 | Hex 字符长度 | 说明 |
|------|--------|----------|-------------|------|
| HBM | `h` | 4 字节 | 8 字符 | 蜜獾标记，HMAC-SHA256 截断前 4 字节 |

HBM 计算公式（代码权威：`rc-core/src/crypto/hbm.rs`）：

```
HBM = Truncate(HMAC-SHA256(K_honey, UID || CTR_LE), 4)
```

- `K_honey`：32 字节 HMAC 密钥（`Zeroizing<[u8; 32]>`）
- `UID`：7 字节芯片 UID
- `CTR_LE`：3 字节小端序计数器（取 `u32` 低 3 字节）
- 消息构造：`UID(7B) || CTR_LE(3B)` = 10 字节
- 碰撞概率：~1/2^32，对防伪场景足够

### Descriptor 暗哨

嵌入 Sovereign Descriptor `category` 字节的静态协议锚定值：

- 高 2 位魔数：`DESCRIPTOR_MAGIC_BITS = 0b10`
- 高 2 位掩码：`DESCRIPTOR_MAGIC_MASK = 0xC0`
- 低 6 位品类掩码：`DESCRIPTOR_CATEGORY_MASK = 0x3F`（品类码 0–63）

工厂烧录后永久存在，解码时用于识别伪造芯片。


---

## 六、API 端点全量清单

> **代码权威**：`rc-api/src/api/mod.rs`（路由定义）

### 公开路由（无 JWT 认证）

Accept-Language 中间件自动解析请求语言偏好。

| HTTP 方法 | 路径 | 说明 | 限流 |
|-----------|------|------|------|
| GET | `/health` | 健康检查 | 无限制 |
| GET | `/v1/resolve/{sun_msg}` | SUN 消息解析（两步验证 Step 1） | IP 级 30 次/分钟 |
| GET | `/v1/asset/verified` | 验证令牌兑换（两步验证 Step 2，Bearer Token） | — |
| GET | `/v1/i18n/locales` | 可用语言列表 | — |
| GET | `/v1/i18n/{locale}` | 语言包获取 | — |
| GET | `/v1/attributes/translations` | 属性翻译 | — |
| GET | `/v1/attributes/codes` | 属性代码列表 | — |
| GET | `/v1/legal/statuses` | 法律状态列表 | — |
| GET | `/v1/legal/statuses/{code}/interpretations` | 法律解释 | — |
| GET | `/v1/legal/translations` | 法律翻译 | — |

### 工厂路由（JWT + Factory/Admin 角色）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| POST | `/v1/factory/blind-log` | 盲扫入库（PreMinted→FactoryLogged→Unassigned） |

### 品牌路由（JWT + Brand/Admin 角色）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| POST | `/v1/brand/register` | 品牌注册 |
| GET | `/v1/brand/quota` | 配额查询 |
| GET | `/v1/brand/assets` | 品牌资产列表 |
| POST | `/v1/brand/entangle-active` | 纠缠激活（Unassigned→...→Activated） |
| POST | `/v1/brand/{brand_id}/products` | 创建产品 |
| GET | `/v1/brand/{brand_id}/products/{product_id}` | 获取产品详情 |
| PUT | `/v1/brand/{brand_id}/products/{product_id}/translations/{locale}` | 更新产品翻译 |

### 资产路由（JWT 认证）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| GET | `/v1/asset/{uid}` | 查询资产信息 |
| POST | `/v1/asset/{uid}/consume` | 消费资产（→ Consumed 终态） |
| GET | `/v1/asset/{uid}/sovereignty` | 查询主权状态 |
| POST | `/v1/asset/{uid}/sovereignty/release` | 释放主权 |
| POST | `/v1/asset/{uid}/sovereignty/lock` | 锁定主权 |
| POST | `/v1/asset/{uid}/challenge/acknowledge` | 质询确认（SecurityLevel Elevated→Normal） |
| POST | `/v1/asset/{uid}/recovery/initiate` | 发起主权恢复 |
| POST | `/v1/asset/{uid}/recovery/{recovery_id}/verify-object` | 实物验证 |
| POST | `/v1/asset/{uid}/recovery/{recovery_id}/verify-biometric` | 生物识别验证 |
| POST | `/v1/asset/{uid}/recovery/{recovery_id}/verify-mother-card` | 母卡验证 |
| POST | `/v1/asset/{uid}/recovery/{recovery_id}/complete` | 完成恢复 |

### 转让路由（JWT + Brand/Admin/Consumer 角色）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| POST | `/v1/asset/transfer/initiate` | 发起转让 |
| POST | `/v1/asset/transfer/confirm` | 确认转让 |
| POST | `/v1/asset/transfer/cancel` | 取消转让 |

### 市场路由（JWT 认证）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| GET | `/v1/market/v-value/{uid}` | V-Value 多维价值评估 |

### 营销路由（JWT + Brand 角色）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| GET | `/v1/marketing/segments` | 获取用户细分 |
| POST | `/v1/marketing/campaign/create` | 创建营销活动 |
| POST | `/v1/marketing/campaign/{campaign_id}/launch` | 启动营销活动 |
| GET | `/v1/marketing/campaign/{campaign_id}/stats` | 活动统计数据 |

### 用户偏好路由（JWT 认证）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| GET | `/v1/user/invite-preferences` | 获取邀请偏好 |
| PUT | `/v1/user/invite-preferences` | 更新邀请偏好 |

### 认证管理路由（JWT 认证）

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| POST | `/v1/auth/revoke` | 撤销 Token |


---

## 七、两步验证协议（Resolve → Verified）

> **代码权威**：`rc-api/src/api/resolve.rs`、`rc-api/src/api/verified.rs`

### 协议流程

```
┌──────────┐     Step 1: GET /v1/resolve/{sun_msg}     ┌──────────┐
│  H5 前端  │ ──────────────────────────────────────────→ │  rc-api  │
│          │ ←────────────────────────────────────────── │          │
│          │     ResolveResponse + Verification Token    │          │
│          │                                             │          │
│          │     Step 2: GET /v1/asset/verified           │          │
│          │     Authorization: Bearer {token}            │          │
│          │ ──────────────────────────────────────────→ │          │
│          │ ←────────────────────────────────────────── │          │
└──────────┘     完整业务数据（资产详情、V-Value 等）      └──────────┘
```

### Step 1：SUN 消息解析

**端点**：`GET /v1/resolve/{sun_msg}`（公开端点，IP 级速率限制 30 次/分钟）

解析 NFC 芯片生成的 SUN URL，委托 rc-kms `VerificationService` 执行 CMAC 验证。

### 8 种验证状态

| 状态 | 含义 | 是否生成 Token |
|------|------|---------------|
| `GENUINE` | 验证通过，正品 | ✅ 生成 Verification Token |
| `COUNTERFEIT` | CMAC 验证失败，疑似伪造 | ❌ |
| `REPLAY_ATTACK` | CTR 未递增，疑似重放攻击 | ❌ |
| `RATE_LIMITED` | IP 级速率限制触发 | ❌ |
| `GEO_FENCE_VIOLATION` | 地理围栏违规 | ❌ |
| `ASSET_DISPUTED` | 资产处于争议冻结状态 | ❌ |
| `SECURITY_VERIFICATION_PENDING` | 安全验证待确认（降级响应） | ❌ |
| `UNKNOWN` | 资产未注册或未知错误 | ❌ |

### Verification Token 机制

- **生成条件**：仅当验证状态为 `GENUINE` 且 UID 和 CTR 均存在时生成
- **TTL**：5 分钟
- **一次性使用**：通过 JTI（JWT ID）去重，防止 Token 重放
- **载荷**：包含 UID、CTR、验证状态

### SecurityVerificationPending 降级响应

当资产 SecurityLevel 为 Elevated 时，返回降级响应：
- `uid`：`None`（不泄露资产标识）
- `ctr`：`None`
- `verification_token`：`None`
- `security_level`：`"Elevated"`

### Step 2：验证令牌兑换

**端点**：`GET /v1/asset/verified`（公开端点，Bearer Token 认证）

凭 Step 1 获取的 Verification Token 获取完整业务数据，包括资产详情、主权状态、V-Value 估值等。Token 使用后即失效。

---

## 八、转让工作流

> **代码权威**：`rc-api/src/api/transfer.rs`

### 流程

```
initiate → confirm → (完成)
    │
    └→ cancel（取消）
```

### 端点与角色权限

| 端点 | 角色要求 | 说明 |
|------|---------|------|
| `POST /v1/asset/transfer/initiate` | Brand / Admin / Consumer | 发起转让请求 |
| `POST /v1/asset/transfer/confirm` | Brand / Admin / Consumer | 确认转让完成 |
| `POST /v1/asset/transfer/cancel` | Brand / Admin / Consumer | 取消转让请求 |

### 关键机制

- **幂等性**：通过 `idempotency_key` 保证同一转让请求不会被重复处理
- **乐观锁**：资产状态变更使用 `WHERE state = expected_state` 防止并发冲突
- **Consumer 持有者校验**：Consumer 发起转让时验证其为资产当前持有者
- **状态转换**：LegallySold→Transferred 或 Transferred→Transferred（支持多次转让）


---

## 九、主权锁定协议

> **代码权威**：`rc-api/src/api/sovereignty.rs`

### 端点

| HTTP 方法 | 路径 | 说明 |
|-----------|------|------|
| GET | `/v1/asset/{uid}/sovereignty` | 查询主权状态（锁定/释放） |
| POST | `/v1/asset/{uid}/sovereignty/release` | 释放主权（解除锁定，允许转让） |
| POST | `/v1/asset/{uid}/sovereignty/lock` | 锁定主权（禁止转让，保护所有权） |

### 语义

- **锁定**：资产所有者主动锁定主权，锁定期间禁止转让操作
- **释放**：解除锁定状态，恢复正常流转能力
- 主权锁定是业务层约束，不改变状态机中的 `AssetState`

---

## 十、争议恢复协议

> **代码权威**：`rc-core/src/state_machine/states.rs::validate_recovery`、`rc-api/src/api/recovery.rs`

### 双层架构

争议恢复采用"状态机层开放 + 业务层约束"的双层设计：

#### 状态机层（rc-core）

`check_permission(Moderator, Disputed, to)` 对任意非终态 `to` 返回 `true`。状态机层不限制恢复目标，提供最大灵活性。

#### 业务层（rc-api）

`validate_recovery(record)` 施加严格约束：

1. **previous_state 非终态检查**：若 `RecoveryRecord.previous_state` 为终态（Consumed/Legacy/Tampered/Compromised/Destructed），直接返回 `Err(InvalidRecoveryState)`，拒绝恢复
2. **三重验证**：`object_verified` + `biometric_verified` + `mother_card_verified` 必须全部为 `true`
3. **恢复目标**：恢复目标状态 = `RecoveryRecord.previous_state`（冻结前的状态），不可自由选择

### 三重验证流程

```
POST /v1/asset/{uid}/recovery/initiate          → 创建 RecoveryRecord
POST /v1/asset/{uid}/recovery/{id}/verify-object     → 实物验证 ✓
POST /v1/asset/{uid}/recovery/{id}/verify-biometric  → 生物识别验证 ✓
POST /v1/asset/{uid}/recovery/{id}/verify-mother-card → 母卡验证 ✓
POST /v1/asset/{uid}/recovery/{id}/complete          → 完成恢复（Disputed → previous_state）
```

### 错误场景

| 场景 | 错误 | 说明 |
|------|------|------|
| previous_state 为终态 | `InvalidRecoveryState` | 终态不可恢复，优先于验证检查 |
| 任一验证未通过 | `RecoveryIncomplete { missing_steps }` | `missing_steps` 精确列出未通过的验证步骤名称 |


---

## 十一、深水区特性（Financial-Grade Deep-Water Features）

### V-Value 多维价值评估

> **代码权威**：`rc-api/src/api/market.rs`

**端点**：`GET /v1/market/v-value/{uid}`（JWT 认证）

为高价值资产提供多维度价值评估数据：

| 字段 | 类型 | 说明 |
|------|------|------|
| `msrp` | number | 官方建议零售价（Manufacturer's Suggested Retail Price） |
| `market_median` | number | 市场参考中位数 |
| `buyback_reference` | number | 回购参考价 |
| `currency` | string | 货币代码（USD/EUR/GBP/SGD/CNY/AED/JPY 等） |
| `liquidity_index` | float | 流动性指数（0.0–1.0） |
| `sample_size_sufficient` | bool | 样本量是否充足 |
| `disclaimer` | string | 免责声明 |

### Challenge/Acknowledge 主权质询

> **代码权威**：`rc-api/src/api/challenge.rs`

**端点**：`POST /v1/asset/{uid}/challenge/acknowledge`（JWT 认证）

持有者确认安全警报，将资产 SecurityLevel 从 Elevated 回退为 Normal。

#### 前置条件

- 资产 `SecurityLevel` 必须为 `Elevated`（降级状态）
- 存在待确认的质询记录（由系统在检测到异常时自动创建）

#### SecurityLevel 枚举

| 值 | 名称 | 说明 |
|----|------|------|
| 0 | Normal | 正常状态 |
| 1 | Elevated | 降级状态，触发 SecurityVerificationPending |
| 2 | Critical | 严重状态，可能触发熔断 |

#### 流程

1. 系统检测到安全异常（如 CTR 异常、地理围栏违规）→ SecurityLevel 升至 Elevated
2. Resolve 端点返回 `SECURITY_VERIFICATION_PENDING` 降级响应
3. 持有者通过 App 调用 Challenge/Acknowledge 确认安全警报
4. SecurityLevel 回退为 Normal，恢复正常验证流程


---

## 十二、代码-文档映射表

| 架构概念 | 代码位置 | 状态 |
|----------|----------|------|
| KDF 密钥派生 | `rc-core/src/crypto/kdf.rs` | ✅ 已实现 |
| AES-CMAC 计算 | `rc-core/src/crypto/aes_cmac.rs` | ✅ 已实现 |
| HBM 蜜獾标记 | `rc-core/src/crypto/hbm.rs` | ✅ 已实现 |
| 14 态状态机（rc-common） | `rc-common/src/states.rs` | ✅ 已实现 |
| 14 态状态机（rc-core） | `rc-core/src/state_machine/states.rs` | ✅ 已实现 |
| 权限矩阵 | `rc-core/src/state_machine/transitions.rs` | ✅ 已实现 |
| 熔断器 | `rc-core/src/state_machine/circuit_breaker.rs` | ✅ 已实现 |
| SUN 消息常量 | `rc-common/src/sun_spec.rs` | ✅ 已实现 |
| SUN 消息解析 | `rc-api/src/api/resolve.rs` | ✅ 已实现 |
| 两步验证（Verified） | `rc-api/src/api/verified.rs` | ✅ 已实现 |
| 转让工作流 | `rc-api/src/api/transfer.rs` | ✅ 已实现 |
| 主权锁定 | `rc-api/src/api/sovereignty.rs` | ✅ 已实现 |
| 争议恢复 | `rc-api/src/api/recovery.rs` | ✅ 已实现 |
| V-Value 估值 | `rc-api/src/api/market.rs` | ✅ 已实现 |
| Challenge/Acknowledge | `rc-api/src/api/challenge.rs` | ✅ 已实现 |
| JWT 认证 | `rc-api/src/auth/` | ✅ 已实现 |
| 速率限制 | `rc-api/src/middleware/rate_limit.rs` | ✅ 已实现 |
| 审计日志 | `rc-api/src/stores/audit.rs` | ✅ 已实现 |
| 品牌注册与产品管理 | `rc-api/src/api/brand.rs`、`rc-api/src/api/brand_products.rs` | ✅ 已实现 |
| 盲扫入库 | `rc-api/src/api/blind_log.rs` | ✅ 已实现 |
| 纠缠激活 | `rc-api/src/api/entangle.rs` | ✅ 已实现 |
| 国际化 | `rc-api/src/api/i18n.rs` | ✅ 已实现 |
| 属性翻译 | `rc-api/src/api/attributes.rs` | ✅ 已实现 |
| 法律状态 | `rc-api/src/api/legal.rs` | ✅ 已实现 |
| 营销活动 | `rc-api/src/api/marketing.rs` | ✅ 已实现 |
| 用户偏好 | `rc-api/src/api/user_preferences.rs` | ✅ 已实现 |
| Token 撤销 | `rc-api/src/api/revoke.rs` | ✅ 已实现 |
| 资产消费 | `rc-api/src/api/consume.rs` | ✅ 已实现 |
