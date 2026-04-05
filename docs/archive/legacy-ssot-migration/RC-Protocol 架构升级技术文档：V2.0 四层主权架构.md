这份文档将指导你完成 **RC-Protocol (RCP)** 从三层到四层架构的深度演进。这次升级的核心在于引入**“逻辑账户层”**，实现物、权、人的彻底解耦。

---

# 🚀 RC-Protocol 架构升级技术文档：V2.0 四层主权架构

## 1. 架构全景对比

|**层级**|**名称**|**核心标识 (ID)**|**物理/逻辑属性**|**升级职责**|
|---|---|---|---|---|
|**L1**|**物体层 (Object)**|`OID` (Chip UID)|物理唯一性|负责 NFC 硬件交互、SDM 动态校验、离线防伪。|
|**L2**|**资产层 (Asset)**|`AID` (Asset UUID)|品牌元数据|负责映射 MKS 系统的 SKU、生产批次、数字化双生数据。|
|**L3**|**账户层 (Owner)**|`WID` (Wallet ID)|逻辑归属|**[新增]** 负责资产的集合管理、权限锁定、质押状态维护。|
|**L4**|**认证层 (Auth)**|`KID` (Key/Bio ID)|意志验证|**[新增]** 负责 FaceID、设备指纹、母卡私钥的签名验证。|

---

## 2. 核心数据模型变更 (Schema Refactoring)

## 2.1 资产层 (L2) - `rc_assets`

不再直接关联用户信息，只记录物体与资产定义的绑定。



```SQL
CREATE TABLE rc_assets (
    aid UUID PRIMARY KEY,         -- 资产唯一标识
    oid VARCHAR(14) UNIQUE,       -- 对应 L1 的芯片 UID
    mks_sku_id VARCHAR(64),       -- MKS 系统中的物料编号
    status INT,                   -- 状态：0-未激活, 1-流通中, 2-锁定(质押)
    mint_at TIMESTAMP,            -- 点睛时间
    parent_oid VARCHAR(14)        -- 物理纠缠的母标签 UID
);
```


## 2.2 账户层 (L3) - `rc_owners` (核心升级)

作为“资产容器”，它屏蔽了底层认证的复杂性。



```SQL
CREATE TABLE rc_owners (
    wid UUID PRIMARY KEY,         -- 逻辑钱包/账户 ID
    display_name VARCHAR(64),     -- 藏馆名称 (如 "老朋友的数字保险柜")
    avatar_url TEXT,
    created_at TIMESTAMP
);

-- 资产归属关系表 (实现 L2 到 L3 的绑定)
CREATE TABLE rc_asset_ownership (
    wid UUID,
    aid UUID,
    acquired_at TIMESTAMP,
    PRIMARY KEY (wid, aid)
);
```


## 2.3 认证层 (L4) - `rc_auth_methods`

支持“一人多机”或“一号多证”的灵活验证。



```SQL
CREATE TABLE rc_auth_methods (
    kid UUID PRIMARY KEY,
    wid UUID,                     -- 关联到账户层
    auth_type VARCHAR(20),        -- BIO(生物), DEV(设备), NFC(母卡)
    public_key TEXT,              -- 用于校验签名的公钥/特征哈希
    last_used_at TIMESTAMP
);
```


---

## 3. 核心业务流演进

## 3.1 资产查询流 (由浅入深)

1. **匿名访问**：扫码 → 提取 `OID` → 查询 `L2` → 展示资产 3D 模型与品牌故事（不展示归属信息）。
    
2. **鉴权访问**：用户触发“管理” → 调用 `L4` 认证 → 验证成功后通过 `KID` 找到 `WID` → 展示该账户下的私密信息、行情与质押入口。
    

## 3.2 资产转让流 (原子操作)

资产转让不再是“人传人”，而是 **“账户间所有权变更”**。

1. **强认证触发**：调用 `L4` 的最高等级认证（生物识别 + 母卡 NFC 纠缠签名）。
    
2. **所有权变更**：在事务中将 `aid` 从 `wid_A` 移至 `wid_B`。
    
3. **状态同步**：更新 `rc_assets` 的流转历史记录。
    

---

## 4. 后端 API 模块调整清单

## 4.1 `rc-kms` 升级

- **任务**：实现从 `Auth-Key` 到 `Owner-Session` 的转换逻辑。
    
- **逻辑**：KMS 不再直接管理资产，而是管理“认证权限”。
    

## 4.2 `rc-api` 新增端点

- `GET /v1/assets/{oid}`：基础资产探测（匿名）。
    
- `POST /v1/account/auth`：执行 L4 认证，换取 L3 的操作令牌 (JWT)。
    
- `GET /v1/vault/list`：查询当前 `wid` 下的所有资产。
    

---

---

## 5. 质押金融与使用权经济扩展

> 详细设计参见 `docs/系统质押模式升级.md`

### 5.1 质押相关 Schema 扩展

```SQL
-- 质押记录表
CREATE TABLE rc_pledges (
    pledge_id UUID PRIMARY KEY,
    wid UUID NOT NULL,                -- 质押人账户
    aid UUID NOT NULL,                -- 质押资产
    collateral_ratio DECIMAL(5,4),    -- 质押率 (0.0000-1.0000)
    ltv_current DECIMAL(5,4),         -- 当前 LTV
    market_value DECIMAL(18,2),       -- 质押时市场价值
    loan_amount DECIMAL(18,2),        -- 贷款金额
    status INT,                       -- 0-正常, 1-预警, 2-冻结, 3-违约, 4-已结清
    created_at TIMESTAMP,
    expires_at TIMESTAMP
);

-- 使用权令牌表 (L3.5 许可层)
CREATE TABLE rc_usage_tokens (
    ut_id UUID PRIMARY KEY,
    aid UUID NOT NULL,                -- 关联资产
    owner_wid UUID NOT NULL,          -- 所有者 WID
    user_wid UUID NOT NULL,           -- 使用者 WID
    permission_level VARCHAR(20),     -- DISPLAY / EXHIBIT / FULL_USE
    time_start TIMESTAMP,
    time_end TIMESTAMP,
    deposit_amount DECIMAL(18,2),     -- 保证金
    status INT,                       -- 0-有效, 1-过期, 2-吊销
    created_at TIMESTAMP
);

-- 清算记录表
CREATE TABLE rc_liquidations (
    liquidation_id UUID PRIMARY KEY,
    pledge_id UUID NOT NULL,
    wid UUID NOT NULL,
    trigger_reason VARCHAR(50),       -- MARGIN_CALL / DEFAULT / MANUAL
    status INT,                       -- 0-预警, 1-锁定, 2-公示, 3-已清算
    created_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- 全局黑名单
CREATE TABLE rc_blacklist (
    wid UUID PRIMARY KEY,
    reason VARCHAR(100),
    created_at TIMESTAMP
);
```

### 5.2 L3.5 许可层架构

```
┌─────────────────────────────────────────────────────────┐
│  L4: 认证层 (Auth)     KID — FaceID / 设备指纹 / 母卡   │
├─────────────────────────────────────────────────────────┤
│  L3.5: 许可层 (Permission)  UT — Usage Token            │  ← 新增
│  ─ 使用权代币化 (AID + Time_Range + Permission_Level)    │
│  ─ 阶梯式质押（Owner LTV 70% / User LTV 20%）          │
│  ─ 保证金自动结算                                        │
├─────────────────────────────────────────────────────────┤
│  L3: 账户层 (Owner)    WID — 逻辑钱包/藏馆               │
├─────────────────────────────────────────────────────────┤
│  L2: 资产层 (Asset)    AID — 品牌元数据                   │
├─────────────────────────────────────────────────────────┤
│  L1: 物体层 (Object)   OID — 芯片 UID                    │
└─────────────────────────────────────────────────────────┘
```

### 5.3 清算管理器状态机

```
Normal → Warning（价值波动接近阈值）
  ├─ 用户追加质押 → Normal
  └─ 超时未响应 → Locked（违约锁定 L3 账户）
       └─ 进入清算 → Defaulted（L2 全局标记违约，WID 列入黑名单）
```

### 5.4 风控引擎核心职责

需在 `rc-api` 中新增 `Risk-Control-Engine` 模块：

1. 实时估值：对接二级市场 API，动态计算资产公允价值
2. LTV 监控：持续监控质押率，触发 Margin Call
3. 账户信用评分：基于 VCV（藏馆信用价值）的综合授信管理
4. 自动平仓：保护期过期后的强制清算逻辑
5. 黑名单管理：违约用户的全局标记与传播

---

## 6. 开发者建议：如何逐步实施？

1. **第一步 (Database)**：根据上述 Schema 在数据库中建立 `rc_owners` 和 `rc_auth_methods` 表，并把现有的测试 UID 归属到第一个测试 `WID` 中。
    
2. **第二步 (Algorithm)**：将你的 `verify_sun_cmac` 算法保持在 `L1` 层。校验通过后，返回 `aid`，而不是直接返回用户信息。
    
3. **第三步 (Frontend)**：在“个人藏馆” App/H5 中，建立一个 `AuthService` 模块，专门负责收集 L4 的指纹或签名。
    
4. **第四步 (Deployment)**：利用你的 `haball.cc` 域名，通过子域名路由实现隔离：
    
    - `r.haball.cc` → 指向 L2（资产解析）。
        
    - `v.haball.cc` → 指向 L3/L4（账户与认证中心）。
        

---

## 💡 架构师点评

**“四层架构”最大的价值在于它赋予了 RCP 协议处理“复杂社会关系”的能力。** 比如：当一个资产被质押时，你依然是它的 **Owner (L3)**，但由于 **Auth (L4)** 层的权限被临时托管给质押平台，你无法行使转让权。这种灵活性是三层架构无法企及的。