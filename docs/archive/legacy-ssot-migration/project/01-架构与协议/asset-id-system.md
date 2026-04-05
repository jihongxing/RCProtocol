# RC-Protocol 资产 ID 体系设计规范

> 文档编号：RC-ARCH-AID-001
> 版本：v1.1（审计修订版）
> 状态：草案
> 日期：2026-03-14
> 密级：机密
> 审计基线：rc-kms `001_init.sql`、rc-api `001_api_tables.sql`、
>           rc-core `entanglement.rs`/`descriptor.rs`/`kdf.rs`/`hbm.rs`

---

## 一、设计目标

资产 ID 是 RC-Protocol 最核心的基础设施——它不仅是数据库主键，更是**全球奢侈品资产的身份证体系**。

资产 ID 必须同时满足五项约束：

| 约束 | 说明 |
|------|------|
| 全球唯一 | 任意两件资产的 ID 永远不会碰撞 |
| 不可伪造 | 包含校验位，无法通过猜测或篡改生成合法 ID |
| 可解析 | 从 ID 本身即可提取品牌、品类、批次等结构化信息 |
| 品牌自治 | 品牌方在注册后自主管理产品型号和序列号空间 |
| 全球规模 | 支持未来数十亿件资产的编址空间 |

因此，资产 ID 不是简单的 UUID，也不是裸 NFC UID，而是一个**结构化分段标识符**。

---

## 二、ID 分段结构

### 2.1 总体格式

```
RC-{Brand}-{Product}-{Serial}-{Check}
```

示例：

```
RC-HERM-BB27SP-A1B2C3-7K
```

### 2.2 各字段定义

| 字段 | 长度 | 编码 | 说明 |
|------|------|------|------|
| System | 2 字符 | 固定 `RC` | 协议主权标识（Regalis Clavis） |
| Brand | 4 字符 | 大写字母+数字 | 品牌唯一代码，品牌注册时分配 |
| Product | 6 字符 | 字母+数字 | 产品型号编码，含品类码 + 年份 + 季节 |
| Serial | 6 字符 | Crockford Base32 | 单件序列号，编码 `u32` 值（覆盖约 42.9 亿件/品牌·产品） |
| Check | 2 字符 | CRC-8 双字符 hex | 校验位，基于完整 ID 前缀计算 |

总长度：**24 字符**（含 4 个分隔符 `-`），固定长度，适合二维码、NFC NDEF 和全球系统通用。

> **审计修正 [v1.1]**：
> - 原设计将 Batch 和 Serial 合并为 `u32 × u32` 并声称 Base32 编码后 6–10 字符。
>   实际 `u32 × u32 = 8 字节`，Crockford Base32 编码需要 13 字符，与 "6–10" 不符。
> - 修正方案：**取消独立 Batch 字段**，将批次信息下沉到 `product_registry` 表的元数据中。
>   Serial 字段仅编码单个 `u32`（4 字节 → Base32 = 7 字符，截断填充后 6 字符足够覆盖 42.9 亿件）。
> - 原设计使用 Verhoeff 算法生成 2 字符校验位，但 Verhoeff 是单位校验算法（仅产生 1 位数字）。
>   修正为 **CRC-8**（1 字节 → 2 字符 hex），可检测所有单字节错误和大多数突发错误。

### 2.3 字段详细设计

#### System（系统标识）— `RC`

- 固定为 `RC`，标识资产属于 Regalis Clavis 协议网络
- 未来若协议成为行业标准，此字段等价于 GS1 的 Application Identifier
- 预留扩展：若出现联盟链/子网络，可扩展为 `RC-{RegionCode}`（如 `RC-EU`、`RC-AP`）

#### Brand（品牌代码）— 4 字符

- 品牌在 `rc-kms` 注册时获得唯一 4 字符代码
- 4 字符提供 `36^4 = 1,679,616` 个命名空间
- 示例映射：

| 品牌 | 代码 |
|------|------|
| Hermès | `HERM` |
| Louis Vuitton | `LVTN` |
| Chanel | `CHNL` |
| Prada | `PRDA` |
| Rolex | `ROLX` |

> **审计修正 [v1.1]**：
> 现有 `brand_configs.brand_id` 是自由格式字符串（如 `"BRAND01"`、`"000000"`），
> 没有 4 字符固定长度约束。引入 Asset ID 体系时需要：
> - 在 `brand_configs` 表新增 `brand_code CHAR(4) UNIQUE` 字段
> - `brand_id` 保留为内部标识（用于密钥派生），`brand_code` 作为 Asset ID 中的品牌段
> - 两者通过 `brand_configs` 表关联，避免破坏现有 KDF 链

#### Product（产品型号）— 6 字符

推荐编码结构：`[Category 2字符][Year 2字符][Season 2字符]`

| 子段 | 长度 | 说明 | 示例 |
|------|------|------|------|
| Category | 2 字符 | 品类码（字母） | `BB`=手袋, `WC`=腕表, `JW`=珠宝 |
| Year | 2 字符 | 年份后两位 | `27`=2027 |
| Season | 2 字符 | 季节/系列 | `SP`=春, `FW`=秋冬, `SS`=春夏, `CR`=早春 |

示例：`BB27SP` = 2027 春季手袋系列

> **审计修正 [v1.1]**：
> `SovereignDescriptor.category` 是 6 位数值（0–63），而 Product 的 Category 子段是
> 2 字符字母码（如 `BB`、`WC`）。两者之间需要一张**品类映射表**：
>
> | 字母码 | 数值码 | 品类 |
> |--------|--------|------|
> | `BB` | `0x02` | 手袋 (Bags) |
> | `WC` | `0x03` | 腕表 (Watches) |
> | `JW` | `0x04` | 珠宝 (Jewelry) |
> | `WN` | `0x05` | 名酒 (Wine) |
> | `SH` | `0x06` | 鞋履 (Shoes) |
> | ... | ... | ... |
>
> 此映射表应定义在 `rc-common` 中作为常量，确保芯片内 Descriptor 的 `category`
> 字段与 Asset ID 的 Product Category 子段始终一致。

#### Serial（序列号）— 6 字符 Crockford Base32

- 内部存储为 `u32` 序列号（每品牌·产品组合下唯一）
- 显示时使用 **Crockford Base32** 编码（排除 `I/L/O/U` 防误读）
- `u32` 最大值 4,294,967,295，覆盖单品牌·单产品下约 42.9 亿件资产
- 6 字符 Base32 可表示 `32^6 = 1,073,741,824`（约 10.7 亿），对 MVP 阶段绰绰有余
- 若未来需要更大空间，可扩展至 7 字符（`32^7 ≈ 343 亿`）

> **审计修正 [v1.1]**：
> 原设计将 Batch + Serial 合并为 `u32 × u32`，但批次信息更适合作为
> `product_registry` 表的元数据（用于召回管理和质量追溯），
> 而非编码进 Asset ID 的显示格式中。这样 Asset ID 更紧凑，
> 批次查询通过数据库关联完成。

#### Check（校验位）— 2 字符

- 采用 **CRC-8**（多项式 `0x07`，即 CRC-8/SMBUS），对完整 ID 前缀计算
- CRC-8 产生 1 字节结果，以 2 字符 hex 表示
- 可检测所有单字节错误和大多数突发错误
- 基于完整 ID 字符串（不含 Check 本身和分隔符）计算

> **审计修正 [v1.1]**：
> 原设计使用 Verhoeff 算法，但 Verhoeff 仅产生 1 位十进制校验位，
> 无法填充 2 字符。CRC-8 天然产生 1 字节 = 2 hex 字符，且实现简单、
> 无外部依赖，符合 `rc-common` 零依赖约束。

---

## 三、三层 ID 体系

RC-Protocol 中实际存在三种不同层级的标识符，各司其职：

```
┌─────────────────────────────────────────────────┐
│  Layer 3: Ownership ID（所有权 ID）              │
│  记录资产归属：Owner Wallet / Account            │
│  转让时变更，资产 ID 不变                         │
├─────────────────────────────────────────────────┤
│  Layer 2: Asset ID（资产 ID）                    │
│  RC-HERM-BB27SP-A1B2C3-7K                       │
│  资产的全球唯一逻辑身份，终身不变                  │
├─────────────────────────────────────────────────┤
│  Layer 1: Chip ID（芯片 ID）                     │
│  NFC UID: 04:A3:1B:2C:3D:4E:5F (7 bytes)       │
│  物理芯片的硬件标识，不可更改                      │
└─────────────────────────────────────────────────┘
```

| 层级 | 标识符 | 来源 | 可变性 | 存储位置 |
|------|--------|------|--------|----------|
| L1 芯片 ID | NFC UID (7 bytes) | 芯片出厂烧录 | 不可变 | 芯片硬件 |
| L2 资产 ID | `RC-XXXX-XXXXXX-XXXXXX-XX` | 纠缠激活时生成 | 不可变 | 数据库 |
| L3 所有权 ID | Owner/Wallet/Account ID | 转让流程 | 随转让变更 | 数据库（加密层） |

关键绑定关系：

```
Asset ID ──绑定──→ NFC UID（纠缠激活时建立，不可解除）
Asset ID ──指向──→ Owner ID（转让时变更指向）
Asset ID ──关联──→ SKU ID（纠缠时通过 EntanglementRequest.sku_id 建立）
```

> **审计修正 [v1.1]**：
> 原文档说 Asset ID 在 `PreMinted` 阶段生成，但这与现有流程矛盾：
> - 现有流程：芯片出厂 → 盲扫入库（`FactoryLogged`，此时仅有 UID，无品牌信息）
>   → 纠缠激活（此时才绑定 `sku_id` 和品牌信息）
> - 盲扫协议的核心设计就是**工厂不知道商品信息**，因此 Asset ID 不可能在盲扫前生成
> - 修正：**Asset ID 在纠缠激活时生成**，此时品牌方选择 SKU，系统自动分配序列号并
>   计算 Asset ID。盲扫阶段仅以 NFC UID 作为临时标识。

这种分层设计确保：**资产转让时只需转移所有权 ID，资产 ID 和芯片 ID 保持不变**，
符合 GDPR/PDPA 的数据最小化原则。

### 3.1 Asset ID 与 SKU ID 的关系

> **审计新增 [v1.1]**

现有代码中 `sku_id` 是纠缠流程的核心字段（见 `EntanglementRequest.sku_id`、
`EntanglementRecord.sku_id`、`asset_records.sku_id`）。Asset ID 与 SKU ID 的关系：

```
SKU ID（品类标识，一对多）
  │  例如："SKU-CHANEL-CF-MEDIUM-BLACK"
  │  一个 SKU 对应同款的所有单品
  │
  └──→ Asset ID（单品标识，一对一）
       例如："RC-CHNL-BB27SP-A1B2C3-7K"
       每件单品有唯一的 Asset ID
```

- `sku_id` 描述"这是什么产品"（品类维度）
- `asset_id` 描述"这是哪一件产品"（单品维度）
- 纠缠激活时：品牌方选择 `sku_id` → 系统根据 `sku_id` 对应的品牌和产品信息
  自动生成 `asset_id` → 写入 `asset_records`

---

## 四、On-Chip 与 On-Cloud 的影子映射

### 4.1 核心矛盾

- 完整 Asset ID 长度 24 字符
- NTAG 424 DNA 芯片 File 01 仅分配 **4 字节** 给 `SovereignDescriptor`

### 4.2 解决方案：影子 ID 映射

```
┌──────────────────┐         ┌──────────────────────────────┐
│   On-Chip (4B)   │         │       On-Cloud (Full)        │
│                  │         │                              │
│ SovereignDesc:   │◄═══════►│ Asset ID: RC-HERM-BB27SP-... │
│  category (6bit) │  映射   │ NFC UID:  04:A3:1B:...      │
│  brand_series    │  纽带   │ Brand:    HERM               │
│  msrp (2B)       │         │ Product:  BB27SP              │
│                  │         │ Serial:   A1B2C3              │
│ + NFC UID (7B)   │         │ SKU ID:   SKU-HERM-...       │
└──────────────────┘         └──────────────────────────────┘
```

映射纽带：**NFC UID** 是芯片与云端的唯一桥梁。

- 芯片内：存储 4 字节 Descriptor（含暗哨魔数 `0b10`），作为品类/品牌的冗余校验
- 云端/App：数据库存储完整 Asset ID，通过 `uid` 主键关联
- 验证流程：SUN 消息携带 UID → 后端通过 UID 查找 `asset_records` →
  获取完整 Asset ID + Descriptor → 解码 Descriptor 交叉验证品类/品牌一致性

> **审计修正 [v1.1]**：
> 原文档说"NFC UID + SovereignDescriptor 在后端唯一指向全量 Asset ID"。
> 实际上 NFC UID 本身就是 `asset_records` 表的主键，已经唯一。
> Descriptor 的作用是**交叉验证**（检测芯片内数据是否被篡改），
> 而非映射纽带。纽带就是 UID 本身。

### 4.3 Descriptor 与 Asset ID 的字段对齐

| Descriptor 字段 | 字节 | 对应 Asset ID 字段 | 对齐方式 |
|-----------------|------|-------------------|----------|
| `category` (低 6 位) | 1B | Product 的 Category 子段 | 通过品类映射表（见 2.3 节） |
| `brand_series` | 1B | Brand 代码的系列映射 | 通过 `brand_configs` 表关联 |
| `msrp` (大端序) | 2B | `product_registry.msrp` | 注册时写入 |

高 2 位 `Descriptor_Magic` (`0b10`) 作为协议锚定暗哨，不参与 Asset ID 映射。

---

## 五、Asset ID 与密钥体系的绑定

### 5.1 密钥派生层级（现有实现）

```
Root_Key (32B, HSM)
  └─ Brand_Key = HMAC-SHA256(Root_Key, Brand_ID || System_ID)
       └─ K_chip = Truncate16(HMAC-SHA256(Brand_Key, UID || Epoch_LE))
```

代码位置：`rc-core/src/crypto/kdf.rs`

### 5.2 Asset ID 参与 K_honey 派生（待实现）

蜜獾字符（HBM）的密钥 `K_honey` 应将 Asset ID 的序列号部分纳入派生输入，
实现**一物一密**：

```
K_honey = HMAC-SHA256(Brand_Key, Asset_ID_Serial_Bytes)
```

其中 `Asset_ID_Serial_Bytes` 为 Asset ID 中 Serial 段解码后的 `u32` 小端序字节。

> **审计修正 [v1.1]**：
> 现有 `compute_hbm` 函数（`rc-core/src/crypto/hbm.rs`）接受外部传入的
> `k_honey: &Zeroizing<[u8; 32]>`，函数本身不负责密钥派生。
> 要实现 Asset ID 参与 K_honey 派生，需要：
>
> 1. 在 `rc-core/src/crypto/kdf.rs` 中新增 `derive_honey_key` 函数
> 2. 注意 `Brand_Key`（`HmacKey`）实现了 `ZeroizeOnDrop` 且不可 `Clone`，
>    因此 `derive_honey_key` 必须在 `Brand_Key` 被清零前调用
> 3. 推荐在 `derive_full_chain` 的扩展版本中同时派生 `K_chip` 和 `K_honey`，
>    共享同一个 `Brand_Key` 实例
>
> 安全意义：
> - 即使攻击者获知某品牌的通用算法，每个 Asset ID 对应的 HBM 毒素都不同
> - 无法批量克隆
> - 但需注意：Asset ID 在纠缠激活时才生成，因此 K_honey 也只能在纠缠后派生

### 5.3 UID 与 Asset ID 的双重校验

验证流程中同时检查：

```
1. NFC UID → 查找 asset_records 记录
2. CMAC 验证 → 证明芯片物理真实性（K_chip 派生自 UID）
3. HBM 验证 → 证明 Asset ID 绑定关系未被篡改（K_honey 派生自 Asset ID Serial）
4. Descriptor 解码 → 交叉验证品类/品牌一致性
```

如果攻击者复制了芯片 UID 但 Asset ID 不匹配，系统会检测到：
**同一 UID 出现在不同地理位置**（CTR 回溯），触发 `SECURITY.clone_suspected` 审计事件。

---

## 六、Asset ID 生命周期

Asset ID 的生命周期与 `AssetStateConst`（14 态状态机）对齐：

> **审计修正 [v1.1]**：
> 原文档将 Asset ID 生成放在 `PreMinted` 阶段，与盲扫协议矛盾。
> 修正后的生命周期如下：

```
芯片出厂
  │  NFC UID 烧录，无任何业务数据
  ▼
盲扫入库 (Blind Log)                              ← FactoryLogged (1)
  │  工厂扫描 NFC UID 录入 asset_records
  │  此时仅有 uid + status + trace_id + batch_no
  │  无 brand_id、无 sku_id、无 Asset ID
  │  （盲扫协议核心：工厂不知道商品信息）
  ▼
纠缠激活 (Entangle + Activate)                    ← Activated (2)
  │  品牌方选择 sku_id → 系统生成 Asset ID
  │  写入 asset_records.asset_id
  │  写入 Descriptor 到芯片 File 01
  │  建立 Asset ID ↔ NFC UID 永久绑定
  │  建立 entanglement_records (m_uid ↔ c_uid ↔ sku_id)
  ▼
合法售出 (Legally Sold)                           ← LegallySold (3)
  │  消费者完成确权，Owner ID 首次写入
  ▼
流通生命周期
  │  Transferred (4) → Transferred (4) → ...
  │  Asset ID 不变，Owner ID 随转让变更
  ▼
终态 (Terminal States)
  ├─ Consumed (5)    — 物理使用殆尽，转入荣誉态
  ├─ Legacy (6)      — 物理损坏不可修复
  ├─ Destructed (9)  — 品牌方主动注销
  └─ CRITICAL:
     ├─ Tampered (7)    — 子标签物理回路断裂
     └─ Compromised (8) — 检测到克隆或异常攻击
        触发 SECURITY 审计事件，Asset ID 进入黑名单
```

关键时序约束：
- Asset ID 在 `FactoryLogged → Activated` 转换时生成（不是之前）
- `asset_records.asset_id` 字段在盲扫阶段为 `NULL`，纠缠激活时填充
- 终态资产的 Asset ID 永久保留（荣誉态需要），但标记为不可流通

---

## 七、数据库存储设计

### 7.1 现有 Schema 审计

> **审计新增 [v1.1]**

现有 `asset_records` 表（`rc-kms/migrations/001_init.sql`）：

```sql
CREATE TABLE IF NOT EXISTS asset_records (
    uid TEXT PRIMARY KEY NOT NULL,       -- NFC UID (已有)
    status TEXT NOT NULL,                -- 状态 (已有)
    batch_no TEXT,                       -- 批次号 (已有)
    trace_id TEXT NOT NULL,              -- 追踪 ID (已有)
    brand_id TEXT REFERENCES brand_configs(brand_id),  -- 品牌 ID (已有)
    sku_id TEXT,                         -- SKU ID (已有)
    owner_anchor TEXT,                   -- 所有者锚点 (已有)
    security_level INTEGER NOT NULL DEFAULT 0,
    ...
);
```

引入 Asset ID 体系需要的 schema 变更：

### 7.2 Migration 方案

> **审计修正 [v1.1]**：
> 原 v1.0 文档提出创建全新的 `assets` 表和 `brand_registry` 表，
> 与现有 `asset_records` 和 `brand_configs` 表严重冲突。
> 修正方案：**通过增量 migration 扩展现有表**，不破坏已有数据和外键关系。

#### Migration 1: `brand_configs` 新增 `brand_code` 字段（rc-kms）

```sql
-- Migration: 为 Asset ID 体系新增品牌代码字段
-- Purpose: brand_code 作为 Asset ID 中的 4 字符品牌段，与 brand_id（KDF 链标识）分离
-- Crate: rc-kms

ALTER TABLE brand_configs ADD COLUMN brand_code TEXT;
-- brand_code: 4 字符大写字母+数字，品牌注册时分配
-- 允许 NULL 以兼容已有记录（系统保留品牌 '000000' 无需 brand_code）

CREATE UNIQUE INDEX IF NOT EXISTS idx_brand_configs_brand_code
    ON brand_configs(brand_code) WHERE brand_code IS NOT NULL;
```

说明：
- `brand_id` 保留为主键和 KDF 派生输入（自由格式，如 `"BRAND01"`）
- `brand_code` 为 Asset ID 专用的 4 字符标准化代码（如 `"HERM"`）
- 两者通过 `brand_configs` 表行级关联
- 系统保留品牌 `'000000'` 的 `brand_code` 保持 `NULL`

#### Migration 2: `asset_records` 新增 `asset_id` 字段（rc-kms）

```sql
-- Migration: 为 Asset ID 体系新增资产 ID 字段
-- Purpose: 存储结构化 Asset ID，纠缠激活时填充
-- Crate: rc-kms

ALTER TABLE asset_records ADD COLUMN asset_id TEXT;
-- asset_id: 格式 RC-XXXX-XXXXXX-XXXXXX-XX，纠缠激活时生成
-- 盲扫阶段为 NULL（工厂不知道商品信息）

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_records_asset_id
    ON asset_records(asset_id) WHERE asset_id IS NOT NULL;
```

说明：
- `uid` 仍为主键（物理芯片标识，不可变）
- `asset_id` 为逻辑资产标识，纠缠激活时由系统生成并填充
- 盲扫阶段 `asset_id = NULL`，此时仅有 `uid` + `status` + `batch_no`
- `UNIQUE` 约束确保全局唯一性
- 现有 `batch_no` 字段保留，用于工厂批次追溯（与 Asset ID 中取消的 Batch 段互补）

#### Migration 3: 创建 `product_registry` 表（rc-kms）

```sql
-- Migration: 创建产品注册表
-- Purpose: 存储品牌方注册的产品型号信息，支撑 Asset ID 的 Product 段生成
-- Crate: rc-kms

CREATE TABLE IF NOT EXISTS product_registry (
    product_code TEXT NOT NULL,              -- 6 字符产品型号码（如 'BB27SP'）
    brand_id TEXT NOT NULL REFERENCES brand_configs(brand_id),
    category_alpha TEXT NOT NULL,            -- 2 字符品类字母码（如 'BB'）
    category_numeric INTEGER NOT NULL,       -- 6 位品类数值码（0-63，对齐 Descriptor）
    year_code TEXT NOT NULL,                 -- 2 字符年份码（如 '27'）
    season_code TEXT NOT NULL,               -- 2 字符季节码（如 'SP'）
    description TEXT,                        -- 产品描述
    msrp_default REAL,                       -- 默认 MSRP（用于 Descriptor 编码）
    msrp_currency TEXT DEFAULT 'USD',        -- MSRP 币种
    next_serial INTEGER NOT NULL DEFAULT 1,  -- 下一个可用序列号（自增）
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (product_code, brand_id)
);

CREATE INDEX IF NOT EXISTS idx_product_registry_brand
    ON product_registry(brand_id);
CREATE INDEX IF NOT EXISTS idx_product_registry_category
    ON product_registry(category_numeric);
```

说明：
- 复合主键 `(product_code, brand_id)` 确保同一品牌下产品型号唯一
- `category_alpha` 和 `category_numeric` 维护品类映射关系（对齐 Descriptor 的 6 位 category）
- `next_serial` 用于原子递增分配序列号，纠缠激活时 `UPDATE ... SET next_serial = next_serial + 1 RETURNING next_serial - 1`
- `msrp_default` 用于 Descriptor 编码时的 MSRP 字段填充
- 原 v1.0 的 `brand_registry` 表被取消，其功能由 `brand_configs` + `brand_code` 覆盖

#### Migration 4: 创建品类映射表（rc-kms）

```sql
-- Migration: 创建品类映射表
-- Purpose: 维护 Asset ID 品类字母码与 Descriptor 品类数值码的双向映射
-- Crate: rc-kms

CREATE TABLE IF NOT EXISTS category_mapping (
    alpha_code TEXT PRIMARY KEY NOT NULL,    -- 2 字符品类字母码（如 'BB'）
    numeric_code INTEGER NOT NULL UNIQUE,    -- 6 位品类数值码（0-63）
    name_en TEXT NOT NULL,                   -- 英文品类名
    name_zh TEXT NOT NULL,                   -- 中文品类名
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 预置品类映射
INSERT OR IGNORE INTO category_mapping (alpha_code, numeric_code, name_en, name_zh) VALUES
    ('BB', 2, 'Bags', '手袋'),
    ('WC', 3, 'Watches', '腕表'),
    ('JW', 4, 'Jewelry', '珠宝'),
    ('WN', 5, 'Wine', '名酒'),
    ('SH', 6, 'Shoes', '鞋履'),
    ('AC', 7, 'Accessories', '配饰'),
    ('PF', 8, 'Perfume', '香水'),
    ('CL', 9, 'Clothing', '服饰'),
    ('AR', 10, 'Art', '艺术品'),
    ('CR', 11, 'Cars', '名车');
```

说明：
- `numeric_code` 范围 0–63，与 `SovereignDescriptor.category` 的 6 位编码对齐
- 0 和 1 保留（0 = 未分类，1 = 系统保留）
- 此表同时作为 `rc-common` 中品类常量的运行时数据源

> **审计修正 [v1.1]**：
> 原 v1.0 建议将品类映射定义为 `rc-common` 中的 Rust 常量。
> 考虑到品类可能随业务扩展动态增加，改为数据库表 + `rc-common` 中仅定义
> 已知品类的常量子集（编译期可用），运行时通过数据库查询完整映射。

### 7.3 查询模式

Asset ID 体系引入后的典型查询模式：

```sql
-- 通过 Asset ID 查找资产（消费者扫码后的二次查询）
SELECT ar.*, bc.brand_code, bc.name as brand_name
FROM asset_records ar
LEFT JOIN brand_configs bc ON ar.brand_id = bc.brand_id
WHERE ar.asset_id = ?;

-- 通过 NFC UID 查找资产（SUN 验证主路径，不变）
SELECT * FROM asset_records WHERE uid = ?;

-- 纠缠激活时生成 Asset ID 并写入
-- 1. 获取并递增序列号（原子操作）
UPDATE product_registry
SET next_serial = next_serial + 1, updated_at = datetime('now')
WHERE product_code = ? AND brand_id = ?
RETURNING next_serial - 1 AS serial;

-- 2. 写入 asset_id
UPDATE asset_records
SET asset_id = ?, status = 'ACTIVATED', brand_id = ?, sku_id = ?,
    updated_at = datetime('now')
WHERE uid = ? AND status = 'FACTORY_LOGGED';

-- 按品牌+产品查询资产列表
SELECT * FROM asset_records
WHERE asset_id LIKE 'RC-HERM-BB27SP-%';
```

---

## 八、rc-common 中的 Asset ID 类型定义

### 8.1 零依赖约束下的设计

> **审计修正 [v1.1]**：
> `rc-common` 是 L0 纯数据 crate，零上游 crate 依赖。
> `AssetId` 的 `FromStr` 实现需要 CRC-8 校验计算——这是否违反"零业务逻辑"约束？
>
> 结论：**CRC-8 是数据完整性校验，不是业务逻辑**。
> 类似于 `u8::from_str` 会检查数字范围，`AssetId::from_str` 检查校验位
> 属于类型自身的合法性验证。CRC-8 实现仅需约 20 行纯函数代码，
> 无外部依赖，可内联在 `rc-common` 中。

### 8.2 AssetId 结构体

```rust
// rc-common/src/asset_id.rs

/// 结构化资产 ID
///
/// 格式：`RC-{Brand}-{Product}-{Serial}-{Check}`
/// 示例：`RC-HERM-BB27SP-A1B2C3-7K`
///
/// 所有字段均为已验证的值，构造时保证合法性。
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct AssetId {
    /// 品牌代码（4 字符，大写字母+数字）
    pub brand_code: String,
    /// 产品型号码（6 字符）
    pub product_code: String,
    /// 序列号（u32 内部值，显示时 Crockford Base32 编码为 6 字符）
    pub serial: u32,
    /// CRC-8 校验值（1 字节）
    pub check: u8,
}
```

> **审计修正 [v1.1]**：
> - 原 v1.0 将 `brand_code` 定义为 `[u8; 4]`，但品牌代码是字母数字字符串（如 `"HERM"`），
>   不是原始字节。修正为 `String`（构造时验证长度和字符集）。
> - 原 v1.0 包含 `batch: u32` 字段，已取消（批次信息下沉到 `product_registry` 元数据）。
> - `check` 字段类型从 `String` 修正为 `u8`（CRC-8 原始值），
>   `Display` 实现时再格式化为 2 字符 hex。

### 8.3 核心 trait 实现

```rust
impl std::fmt::Display for AssetId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "RC-{}-{}-{}-{:02X}",
            self.brand_code,
            self.product_code,
            encode_crockford_base32(self.serial),  // 6 字符
            self.check,
        )
    }
}

impl std::str::FromStr for AssetId {
    type Err = AssetIdParseError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        // 1. 按 '-' 分割，验证段数 = 5
        // 2. 验证 System = "RC"
        // 3. 验证 Brand: 4 字符，[A-Z0-9]
        // 4. 验证 Product: 6 字符，[A-Za-z0-9]
        // 5. 解码 Serial: Crockford Base32 → u32
        // 6. 解码 Check: 2 字符 hex → u8
        // 7. 重新计算 CRC-8 并比对
        todo!()
    }
}
```

### 8.4 CRC-8 实现（内联，零依赖）

```rust
/// CRC-8/SMBUS 计算（多项式 0x07）
///
/// 对输入字节序列计算 CRC-8，用于 Asset ID 校验位。
/// 纯函数，无外部依赖，适合 rc-common 零依赖约束。
pub fn crc8_smbus(data: &[u8]) -> u8 {
    let mut crc: u8 = 0x00;
    for &byte in data {
        crc ^= byte;
        for _ in 0..8 {
            if crc & 0x80 != 0 {
                crc = (crc << 1) ^ 0x07;
            } else {
                crc <<= 1;
            }
        }
    }
    crc
}
```

### 8.5 Crockford Base32 编解码（内联，零依赖）

```rust
/// Crockford Base32 字符表（排除 I/L/O/U 防误读）
const CROCKFORD_ALPHABET: &[u8; 32] = b"0123456789ABCDEFGHJKMNPQRSTVWXYZ";

/// 将 u32 编码为 6 字符 Crockford Base32 字符串
///
/// 6 字符可表示 32^6 = 1,073,741,824（约 10.7 亿），
/// 覆盖 u32 范围的前 25%，对 MVP 阶段足够。
pub fn encode_crockford_base32(value: u32) -> String {
    let mut result = [b'0'; 6];
    let mut v = value;
    for i in (0..6).rev() {
        result[i] = CROCKFORD_ALPHABET[(v % 32) as usize];
        v /= 32;
    }
    // Safety: CROCKFORD_ALPHABET 仅含 ASCII 字符
    String::from_utf8(result.to_vec())
        .unwrap_or_else(|_| "000000".to_string())
}

/// 将 6 字符 Crockford Base32 字符串解码为 u32
pub fn decode_crockford_base32(s: &str) -> Result<u32, AssetIdParseError> {
    if s.len() != 6 {
        return Err(AssetIdParseError::InvalidSerialLength);
    }
    let mut value: u32 = 0;
    for &b in s.as_bytes() {
        let digit = crockford_char_to_value(b)?;
        value = value.checked_mul(32)
            .and_then(|v| v.checked_add(u32::from(digit)))
            .ok_or(AssetIdParseError::SerialOverflow)?;
    }
    Ok(value)
}
```

> **审计修正 [v1.1]**：
> `encode_crockford_base32` 中使用了 `unwrap_or_else` 而非 `unwrap()`，
> 符合 `rc-common` 的 `#![deny(clippy::unwrap_used)]` 约束。
> 实际上由于 `CROCKFORD_ALPHABET` 全为 ASCII，`from_utf8` 不可能失败，
> 但为满足 clippy 规则仍提供 fallback。

### 8.6 品类映射常量（编译期子集）

```rust
// rc-common/src/asset_id.rs

/// 已知品类映射（编译期常量子集）
///
/// 运行时完整映射通过 `category_mapping` 数据库表查询。
/// 此处仅定义 MVP 阶段的核心品类，供编译期校验和测试使用。
pub const KNOWN_CATEGORIES: &[(&str, u8)] = &[
    ("BB", 0x02),  // 手袋 Bags
    ("WC", 0x03),  // 腕表 Watches
    ("JW", 0x04),  // 珠宝 Jewelry
    ("WN", 0x05),  // 名酒 Wine
    ("SH", 0x06),  // 鞋履 Shoes
    ("AC", 0x07),  // 配饰 Accessories
    ("PF", 0x08),  // 香水 Perfume
    ("CL", 0x09),  // 服饰 Clothing
    ("AR", 0x0A),  // 艺术品 Art
    ("CR", 0x0B),  // 名车 Cars
];
```

---

## 九、合规与隐私设计

### 9.1 GDPR/PDPA 合规

- Asset ID **不包含任何用户个人信息**（PII-free）
- 所有权信息（Owner ID）存储在独立加密层，与 Asset ID 物理隔离
- 资产转让时仅变更 Owner ID 指向，Asset ID 保持不变
- 数据最小化：Asset ID 仅编码资产属性（品牌、品类、序列号），不编码地理位置、购买者或价格

### 9.2 数据隔离架构

```
┌─────────────────────────────────┐
│  公开层 (Public Layer)           │
│  Asset ID: RC-HERM-BB27SP-...   │  ← 可公开展示
│  品牌、品类、序列号              │
├─────────────────────────────────┤
│  受控层 (Controlled Layer)       │
│  NFC UID: 04:A3:**:**:**:**:** │  ← 脱敏后可用于审计
│  Descriptor: [0x82, 0x01, ...]  │
│  SKU ID、批次号                  │
├─────────────────────────────────┤
│  加密层 (Encrypted Layer)        │
│  Owner ID / Wallet Address      │  ← 仅所有者和授权方可访问
│  转让历史、生物特征锚点          │
└─────────────────────────────────┘
```

### 9.3 司法管辖区适配

Asset ID 本身不受司法管辖区限制（纯资产标识），但关联的所有权数据遵循：

| 地区 | 法规 | 影响范围 |
|------|------|----------|
| EU | GDPR | Owner ID 加密存储，用户有权要求删除所有权关联 |
| Singapore/SEA | PDPA | 隐私声明通知，合理目的说明 |
| UAE | DIFC | 数据保护通知 |
| Default | — | 最小化隐私声明 |

---

## 十、未来扩展

### 10.1 地区/工厂码扩展

未来可在 Asset ID 中插入地区码和工厂码：

```
RC-HERM-FR1-BB27SP-A1B2C3-7K
         ^^^
         地区/工厂码（3 字符）
```

扩展后总长度约 28 字符，仍在二维码和 NFC NDEF 的舒适范围内。

### 10.2 跨协议互操作

- Asset ID 的 `RC` 前缀可作为协议识别符，支持与其他资产确权网络互操作
- 未来可定义 `RC-{SubNetwork}` 格式（如 `RC-EU`、`RC-AP`）支持联盟链场景

### 10.3 编址空间规划

| 维度 | 当前容量 | 扩展方案 |
|------|----------|----------|
| 品牌数 | 36^4 ≈ 168 万 | 扩展至 5 字符（36^5 ≈ 6048 万） |
| 产品型号/品牌 | 理论无限（6 字符自由编码） | 无需扩展 |
| 序列号/品牌·产品 | 32^6 ≈ 10.7 亿 | 扩展至 7 字符（32^7 ≈ 343 亿） |
| 总资产数 | 168 万 × ∞ × 10.7 亿 | 远超数十亿目标 |

### 10.4 GS1 标准对标

Asset ID 体系的设计理念对标 GS1 全球商品条码体系：

| GS1 概念 | RC-Protocol 对应 |
|----------|------------------|
| GS1 Company Prefix | Brand Code (4 字符) |
| Item Reference | Product Code (6 字符) |
| Serial Number | Serial (Base32 编码) |
| Check Digit | CRC-8 (2 字符 hex) |
| Application Identifier | System Prefix `RC` |

---

## 附录 A：术语表

| 术语 | 定义 |
|------|------|
| Asset ID | 结构化资产标识符，格式 `RC-XXXX-XXXXXX-XXXXXX-XX` |
| Brand Code | 4 字符品牌代码，Asset ID 的品牌段 |
| Brand ID | `brand_configs.brand_id`，KDF 密钥派生链的输入标识 |
| Chip ID / NFC UID | 7 字节芯片硬件标识，`asset_records` 表主键 |
| Crockford Base32 | 排除 I/L/O/U 的 Base32 编码变体，用于序列号显示 |
| CRC-8/SMBUS | 多项式 0x07 的 8 位循环冗余校验，用于 Asset ID 校验位 |
| Descriptor | 4 字节 `SovereignDescriptor`，芯片内存储的品类/品牌/MSRP 压缩码 |
| HBM | 蜜獾字符 (Honey Badger Marker)，4 字节防伪标记 |
| K_honey | HBM 派生密钥，`HMAC-SHA256(Brand_Key, Asset_ID_Serial_Bytes)` |
| Owner ID | 所有权标识，记录资产当前归属，转让时变更 |
| Product Code | 6 字符产品型号码，含品类+年份+季节 |
| Serial | 序列号，u32 内部值，Crockford Base32 编码显示 |
| Shadow Mapping | 影子映射，On-Chip 4 字节 Descriptor 与 On-Cloud 完整 Asset ID 的对应关系 |
| SKU ID | 品牌方定义的库存单位标识，一个 SKU 对应多个 Asset ID |

## 附录 B：代码位置映射

| 组件 | 文件路径 | 说明 |
|------|----------|------|
| AssetId 类型定义 | `rc-common/src/asset_id.rs`（待创建） | 结构体、Display、FromStr、CRC-8 |
| 品类常量 | `rc-common/src/asset_id.rs`（待创建） | `KNOWN_CATEGORIES` 编译期子集 |
| SovereignDescriptor | `rc-core/src/protocol/nfc/descriptor.rs` | 4 字节芯片内编解码 |
| KDF 密钥派生 | `rc-core/src/crypto/kdf.rs` | Root→Brand→Chip 三级派生 |
| K_honey 派生 | `rc-core/src/crypto/kdf.rs`（待扩展） | `derive_honey_key` 函数 |
| HBM 计算 | `rc-core/src/crypto/hbm.rs` | `compute_hbm` 函数 |
| 纠缠流程 | `rc-core/src/protocol/nfc/entanglement.rs` | Asset ID 生成触发点 |
| brand_configs 表 | `rc-kms/migrations/001_init.sql` | 新增 `brand_code` 字段 |
| asset_records 表 | `rc-kms/migrations/001_init.sql` | 新增 `asset_id` 字段 |
| product_registry 表 | `rc-kms/migrations/`（待创建） | 产品注册与序列号管理 |
| category_mapping 表 | `rc-kms/migrations/`（待创建） | 品类字母码↔数值码映射 |
| transfer_records 表 | `rc-api/migrations/001_api_tables.sql` | 转让记录（无需修改） |
| sovereignty_records 表 | `rc-api/migrations/001_api_tables.sql` | 主权记录（无需修改） |

---

> **文档版本历史**
>
> | 版本 | 日期 | 变更说明 |
> |------|------|----------|
> | v1.0 | 2026-03-14 | 初始设计框架 |
> | v1.1 | 2026-03-14 | 审计修订版：对齐现有代码库，修正 10 项关键问题（详见各节 `审计修正 [v1.1]` 标注） |