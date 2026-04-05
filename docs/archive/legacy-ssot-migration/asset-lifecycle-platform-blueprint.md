# 通用资产全生命周期管控平台 — 项目储备蓝图

> 项目代号：ALP (Asset Lifecycle Platform)
> 定位：行业无关的实物资产全生命周期数字管控基础系统
> 来源：从 RC-Protocol（NFC 母卡+子标签方案）中提炼的通用架构
> 状态：项目储备，待启动

---

## 一、核心理念

将 RC-Protocol 中"NFC 母卡纠缠子标签"的特定场景抽象为通用的"授权设备绑定受控资产"模型。
系统不绑定任何特定硬件（NFC/RFID/QR/BLE）或行业（奢侈品/酒类/艺术品/工业设备），
通过 trait 抽象和插件化设计，支持任意物理标识技术和业务场景。

**一句话定义**：给任何实物资产一个不可伪造的数字身份，并严格管控从计划到报废的全流程。

---

## 二、从 RC-Protocol 到 ALP 的抽象映射

| RC-Protocol 概念 | ALP 通用概念 | 说明 |
|---|---|---|
| NFC 母卡 (Mother Card) | 授权设备 (Authority Device) | 拥有绑定权限的物理设备，可以是 NFC 卡、BLE 信标、HSM 令牌等 |
| NFC 子标签 (Child Tag) | 资产标识 (Asset Tag) | 附着在实物上的唯一标识，可以是 NFC 标签、RFID、防伪二维码等 |
| 母子纠缠 (Entanglement) | 授权绑定 (Authorization Binding) | 授权设备与资产标识建立加密绑定关系 |
| SUN 消息 (NTAG 424 DNA) | 设备认证消息 (Device Auth Message) | 可插拔的设备认证协议接口 |
| Brand (品牌方) | Tenant (租户) | 多租户隔离，每个租户独立密钥树 |
| CMAC 验证 | 认证验证 (Auth Verification) | 可插拔的密码学验证引擎 |
| Sovereign Descriptor (4B) | 资产元数据 (Asset Metadata) | 可扩展的资产属性模型 |
| V-Value 估值 | 价值评估插件 (Valuation Plugin) | 可选的估值引擎接口 |
| Transport_Key | 出厂密钥 (Factory Key) | 资产出厂时的初始认证密钥 |
| Brand_Key | 租户密钥 (Tenant Key) | 绑定后的租户专属密钥 |

---

## 三、资产全生命周期状态机（14 态模型）

直接复用 RC-Protocol 验证过的 14 态状态机，这是经过属性测试验证的通用模型：

```
┌─────────────────────────────────────────────────────────────────┐
│                     资产全生命周期状态流转                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  PLANNED(0) ──→ REGISTERED(1) ──→ UNASSIGNED(10)               │
│    计划中          已登记             待分配                      │
│                                       │                         │
│                              ROTATING_KEYS(11) ←── 密钥翻转瞬态  │
│                                       │                         │
│                            BINDING_PENDING(12) ←── 绑定写入瞬态  │
│                                       │                         │
│                               ACTIVATED(2) ──→ RELEASED(3)      │
│                                 已激活           已发布/售出      │
│                                                    │            │
│                                              TRANSFERRED(4)     │
│                                                 已转让           │
│                                                    │            │
│                              ┌─────────┬──────────┤            │
│                              ▼         ▼          ▼            │
│                         CONSUMED(5) LEGACY(6) DESTRUCTED(9)    │
│                           已消耗      遗珍       已销毁          │
│                                                                 │
│  安全终态：TAMPERED(7) 已篡改 │ COMPROMISED(8) 已泄露            │
│  冻结态：  DISPUTED(13) 争议冻结（可逆）                          │
│                                                                 │
│  任何非终态 ──→ DISPUTED(13) ──→ 原状态（解冻）或 COMPROMISED    │
└─────────────────────────────────────────────────────────────────┘
```

### 状态分类

| 分类 | 状态 | 说明 |
|------|------|------|
| 正常态 | PLANNED, REGISTERED, UNASSIGNED, ACTIVATED, RELEASED, TRANSFERRED | 资产正常生命周期流转 |
| 瞬态 | ROTATING_KEYS, BINDING_PENDING | 毫秒级中间状态，系统自动推进 |
| 终态 | CONSUMED, LEGACY, DESTRUCTED, TAMPERED, COMPROMISED | 不可逆，进入后永久锁定 |
| 冻结态 | DISPUTED | 可逆冻结，需三重验证解冻 |

### 状态转换权限矩阵

| 角色 | 允许的转换 |
|------|-----------|
| Platform（平台管理员） | 任意合法转换（管理员覆盖） |
| Factory（工厂/生产方） | PLANNED→REGISTERED, REGISTERED→UNASSIGNED |
| Tenant（租户/品牌方） | UNASSIGNED→ROTATING_KEYS→BINDING_PENDING→ACTIVATED→RELEASED |
| Consumer（消费者/持有者） | RELEASED→TRANSFERRED, TRANSFERRED→{TRANSFERRED,CONSUMED,LEGACY} |
| Moderator（安全审核员） | any→{TAMPERED,COMPROMISED,DISPUTED}, DISPUTED→原状态 |

---

## 四、系统架构（四层分离）

```
┌─────────────────────────────────────────────────────────┐
│  L3: alp-api（业务 API 层）                              │
│  ─ Axum Web 框架，RESTful API                           │
│  ─ JWT 认证 + RBAC 权限                                 │
│  ─ 速率限制、输入验证                                    │
│  ─ 依赖: alp-core + alp-kms + alp-common               │
├─────────────────────────────────────────────────────────┤
│  L2: alp-kms（密钥管理服务层）                           │
│  ─ HSM 抽象接口（软件/硬件 HSM 可切换）                  │
│  ─ 密钥派生链 + Brand_Key 缓存                          │
│  ─ CMAC 验证流水线                                      │
│  ─ 盲扫入库、审计日志                                    │
│  ─ 依赖: alp-core + alp-common                         │
├─────────────────────────────────────────────────────────┤
│  L1: alp-core（加密核心库）                              │
│  ─ 密码学原语（AES-CMAC、KDF、常量时间比较）             │
│  ─ 状态机引擎 + 转换验证                                │
│  ─ 熔断器（频率限制、地理围栏、计数器防重放）             │
│  ─ 可插拔设备认证协议 trait                              │
│  ─ 依赖: alp-common                                    │
├─────────────────────────────────────────────────────────┤
│  L0: alp-common（纯数据定义层）                          │
│  ─ 状态枚举、错误码、消息模板                            │
│  ─ 类型定义、常量                                       │
│  ─ 零上游依赖                                           │
└─────────────────────────────────────────────────────────┘
```

### 依赖规则

- 严格单向依赖，禁止循环：L0 ← L1 ← L2 ← L3
- 同层 crate 禁止互相依赖
- L0 是绝对孤立的纯数据 crate，不包含任何业务逻辑、IO 或异步代码

---

## 五、角色权限模型

### 五种核心角色

| 角色 | RC-Protocol 对应 | 职责 |
|------|------------------|------|
| Platform | Platform | 平台超级管理员，可覆盖任意合法操作 |
| Factory | Factory | 生产方，负责资产登记入库（盲扫） |
| Tenant | Brand | 租户/品牌方，负责资产绑定激活和发布 |
| Consumer | Consumer | 终端持有者，负责流转、消费、传承 |
| Moderator | Moderator | 安全审核员，负责冻结/解冻和安全事件处理 |

### 认证方式

- API 端点：JWT Bearer Token（含 `sub`、`role`、`tenant_id`）
- 公开端点（验真查询）：IP 级速率限制，无需认证
- 工厂端点：独立授权凭证（MVP 阶段为 API Key，后续升级为 mTLS）

### 权限检查流程

```
请求 → JWT 解析 → 角色提取 → 状态转换权限矩阵校验 → 租户隔离校验 → 放行/拒绝
```

---

## 六、密钥体系（三级派生树）

### 密钥派生链

```
Root_Key（HSM 中存储，每个租户独立）
    │
    ├── system_id 盐（地理/逻辑隔离）
    │
    ▼
Tenant_Key = HMAC-SHA256(Root_Key, tenant_id || system_id)
    │
    ├── device_uid（设备唯一标识）
    ├── epoch（密钥轮换版本号）
    │
    ▼
Device_Key (K2) = HMAC-SHA256(Tenant_Key, uid || epoch)
    │
    ▼
用于 CMAC 计算、设备认证
```

### 密钥安全规范

| 规则 | 说明 |
|------|------|
| Zeroize | 所有密钥材料必须实现 `Zeroize` + `ZeroizeOnDrop`，用完即清零 |
| 常量时间比较 | 所有密码学值比较使用 `subtle::ConstantTimeEq`，禁止 `==` |
| 禁止序列化 | 密钥类型禁止 `Serialize`/`Deserialize`/`Debug`/`Clone` |
| 中间密钥清零 | `Tenant_Key` 派生出 `Device_Key` 后立即清零，不缓存原始字节 |
| Arc 包装 | 缓存的 `Tenant_Key` 使用 `Arc<HmacKey>` 避免 Clone 语义冲突 |

### Transport_Key（出厂密钥）

- 使用系统保留租户 ID（如 "000000"）派生
- 用于盲扫阶段的 CMAC 验证（芯片尚未绑定租户）
- 绑定激活后，密钥从 `Transport_Key` 翻转为 `Tenant_Key` 派生的 `Device_Key`

---

## 七、安全防御体系

### 7.1 阶梯式防御（Graduated Defense）

系统采用三级安全等级，在冻结（Disputed）之前提供缓冲层，避免误伤合法持有者：

| 安全等级 | 触发条件 | 响应 |
|----------|----------|------|
| Normal (0) | 默认状态 | 正常服务，返回完整信息 |
| Elevated (1) | 频率异常、熵异常 | 降级解析，资产状态不变但返回受限信息 |
| Critical (2) | 计数器回滚、地理围栏违规 | 冻结资产为 Disputed 状态 |

### 7.2 熔断器（Circuit Breaker）

五维异常检测引擎，每个维度独立运行：

| 检测维度 | 说明 | 严重级别 |
|----------|------|----------|
| 计数器回滚 | 收到的 CTR ≤ 上次记录值，疑似克隆/重放 | Critical |
| 频率限制 | 同一设备在时间窗口内扫描次数超限 | Elevated |
| 地理围栏 | 短时间内出现不可能的地理位移（Haversine 距离计算） | Critical |
| 纠缠熵异常 | 同一授权设备在时间窗口内绑定次数异常 | Elevated |
| 计数器溢出 | 设备计数器接近或达到物理上限（生命终结预警） | Critical/Elevated |

### 7.3 主权自愈（Sovereign Recovery）

Disputed 资产通过"人-物-权"三位一体验证解冻：

```
Disputed 资产
    │
    ├── ① 物理标签感应验证（object_verified）
    ├── ② 生物识别验证（biometric_verified）
    └── ③ 授权设备确认（authority_device_verified）
    │
    ▼ 三重验证全部通过
恢复到冻结前的原状态
```

### 7.4 日志安全

**绝对禁止出现在日志中的字段**：
- 任何密钥字节
- 完整的设备唯一标识
- 完整的 CMAC 值
- JWT Secret / API Key 明文

**脱敏规则**：设备标识仅显示前 2 字节 + 掩码（如 `04:A3:**:**:**:**:**`）

---

## 八、可插拔设备认证协议接口

### 核心 trait 抽象

ALP 的关键设计：将设备认证协议从业务逻辑中完全解耦，通过 trait 接口支持任意物理标识技术。

```rust
/// 设备认证消息（从物理设备读取的原始数据）
pub trait DeviceAuthMessage: Send + Sync {
    /// 设备唯一标识（字节数组）
    fn device_uid(&self) -> &[u8];
    /// 单调递增计数器（防重放）
    fn counter(&self) -> u32;
    /// 认证签名/CMAC（用于验证真伪）
    fn auth_signature(&self) -> &[u8];
}

/// 设备认证协议（可插拔实现）
#[async_trait]
pub trait DeviceAuthProtocol: Send + Sync {
    type Message: DeviceAuthMessage;
    type Error: std::error::Error;

    /// 从原始 URL/数据中解析设备认证消息
    fn parse_message(&self, raw: &str) -> Result<Self::Message, Self::Error>;

    /// 构造用于 CMAC/签名计算的数据向量
    fn construct_auth_vector(
        &self, uid: &[u8], counter: u32,
    ) -> Result<Vec<u8>, Self::Error>;

    /// 验证设备认证签名
    async fn verify(
        &self, message: &Self::Message, key: &[u8],
    ) -> Result<bool, Self::Error>;
}
```

### 已验证的实现（来自 RC-Protocol）

| 协议 | 设备类型 | 认证方式 | 状态 |
|------|----------|----------|------|
| NTAG 424 DNA SUN | NFC 标签 | AES-128 CMAC (SDM) | 已实现，经属性测试验证 |

### 未来可扩展的实现

| 协议 | 设备类型 | 认证方式 | 适用场景 |
|------|----------|----------|----------|
| RFID UHF Gen2v2 | RFID 标签 | AES-128 加密认证 | 仓储物流、供应链 |
| 防伪二维码 | 印刷二维码 | HMAC-SHA256 + 时间戳 | 快消品、票务 |
| BLE Beacon | 蓝牙信标 | ECDSA P-256 签名 | 高价值设备、车辆 |
| SE (Secure Element) | 嵌入式安全芯片 | RSA/ECC 证书链 | 工业设备、医疗器械 |

### 协议注册机制

```rust
/// 协议注册表（运行时动态注册）
pub struct ProtocolRegistry {
    protocols: HashMap<String, Arc<dyn DeviceAuthProtocol<...>>>,
}

impl ProtocolRegistry {
    /// 注册新的设备认证协议
    pub fn register(&mut self, name: &str, protocol: Arc<dyn DeviceAuthProtocol<...>>);

    /// 根据协议名称获取实现
    pub fn get(&self, name: &str) -> Option<Arc<dyn DeviceAuthProtocol<...>>>;
}
```

---

## 九、多租户隔离模型

### 隔离维度

| 维度 | 隔离方式 | 说明 |
|------|----------|------|
| 密钥隔离 | 每个租户独立 Root_Key | 不同租户的密钥派生树完全独立，互不可达 |
| 数据隔离 | tenant_id 字段 + 查询过滤 | 所有资产记录关联 tenant_id，API 层强制过滤 |
| 配额隔离 | 每个租户独立配额 | 绑定激活次数、API 调用频率独立计量 |
| 审计隔离 | 审计日志按 tenant_id 分区 | 租户只能查看自己的审计记录 |

### 系统保留租户

- 租户 ID `000000` 为系统保留，用于 Transport_Key 派生
- 该租户在数据库迁移中预插入，不可被外部注册
- 盲扫阶段的资产 `tenant_id` 为 NULL，绑定激活后填充实际租户 ID

### 租户生命周期

```
注册申请 → 审核通过 → 分配 Root_Key（HSM 生成）→ 配额初始化 → 正常运营
                                                              │
                                                    配额调整 / 密钥轮换
                                                              │
                                                    停用 → 数据归档
```

---

## 十、审计与合规

### 双轨日志架构

| 轨道 | 工具 | 存储 | 用途 |
|------|------|------|------|
| 运行日志 | tracing + tracing-subscriber | stdout / 文件 | 开发调试、性能监控、错误排查 |
| 审计日志 | 数据库持久化 | audit_log 表 | 合规审计、纠纷举证、安全事件追溯 |

### 审计事件分类

| 类别 | 事件示例 | 严重级别 |
|------|----------|----------|
| ASSET | 盲扫入库、绑定激活、消费、销毁 | NORMAL |
| TRANSFER | 发起转让、确认、取消、过期 | NORMAL |
| SOVEREIGNTY | 冻结、解冻、自愈发起、自愈完成 | HIGH |
| SECURITY | 重放检测、克隆嫌疑、地理围栏违规、熔断触发 | CRITICAL |
| AUTH | 登录、登出、Token 撤销、API Key 管理 | NORMAL |
| ADMIN | 租户注册、配额调整、配置变更 | HIGH |

### 审计日志保留策略

- CRITICAL 事件：永久保留
- HIGH 事件：保留 3 年
- NORMAL 事件：保留 1 年
- 过期清理由后台定时任务执行，清理前导出到冷存储

---

## 十一、适用行业场景举例

| 行业 | 资产类型 | 设备认证方式 | 核心价值 |
|------|----------|-------------|----------|
| 奢侈品 | 手表、箱包、珠宝 | NFC (NTAG 424 DNA) | 防伪溯源、二手流转认证 |
| 酒类 | 高端白酒、红酒 | NFC 标签 + 防伪瓶盖 | 开瓶检测（CONSUMED 终态）、渠道防窜货 |
| 艺术品 | 画作、雕塑、限量版 | NFC + 防伪二维码 | 传承记录（LEGACY）、主权证明 |
| 工业设备 | 精密仪器、医疗器械 | SE 安全芯片 / BLE | 维保记录、合规审计、生命周期管理 |
| 供应链 | 零部件、原材料 | RFID UHF | 批量盲扫入库、全链路追踪 |
| 票务 | 演出票、会员卡 | 防伪二维码 | 一次性消费（CONSUMED）、转让限制 |
| 汽车 | 整车、核心零部件 | SE + BLE Beacon | 所有权转让、维修记录、召回追踪 |
| 农产品 | 有机食品、特产 | 防伪二维码 + NFC | 产地溯源、防伪验真 |

### 行业适配要点

- 不同行业的状态机可能需要裁剪（如票务不需要 LEGACY 状态）
- 通过配置文件定义每个租户可用的状态子集和转换规则
- 设备认证协议通过 trait 实现按需加载，不影响核心架构

---

## 十二、技术栈建议

### 后端

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 语言 | Rust (Edition 2021+) | 内存安全、零成本抽象、密码学生态成熟 |
| Web 框架 | Axum 0.7+ | 异步原生、Tower 中间件生态、类型安全 |
| 异步运行时 | Tokio 1.x | 事实标准 |
| 数据库 | SQLite (MVP) → PostgreSQL (生产) | sqlx 编译期 SQL 校验，feature flag 切换 |
| 密码学 | RustCrypto 生态 | aes、cmac、hmac、sha2、subtle、zeroize |
| 序列化 | serde + serde_json | Rust 生态标准 |
| 错误处理 | thiserror | 派生宏，零运行时开销 |
| 日志 | tracing + tracing-subscriber | 结构化日志，span 上下文传播 |
| 属性测试 | proptest | 密码学和状态机的正确性验证 |
| API 文档 | utoipa (OpenAPI 3.0) | 自动生成，与代码同步 |

### 前端（验真 H5 页面）

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 框架 | 轻量级 SPA（Vue/React 均可） | 扫码后展示验真结果 |
| 构建 | Vite | 快速构建，HMR |
| 样式 | Tailwind CSS | 原子化 CSS，适配多品牌主题 |

### 基础设施

| 组件 | 建议 | 说明 |
|------|------|------|
| 容器化 | Docker + multi-stage build | Rust 编译产物为静态二进制 |
| CI/CD | GitHub Actions | cargo clippy → cargo test → cargo build --release |
| HSM | 软件 HSM (MVP) → 硬件 HSM (生产) | trait 抽象，无缝切换 |
| 监控 | Prometheus + Grafana | tracing 指标导出 |

---

## 十四、多品牌聚合接入层（HOAC Gateway）

> 来源：聚合协议与跨行清算层设计

### 14.1 核心逻辑：从中心化系统到分布式网关

ALP 不要求品牌方放弃自有数据库，而是提供 HOAC Gateway（适配器），实现"多品牌无感接入"。

**统一资产映射（Unified Mapping）**：定义 `Universal Luxury Schema` 标准。无论品牌方内部使用 `item_id` 还是 `product_code`，在 ALP 底座系统中统一映射为 `HOAC_Standard_ID`。

**联邦身份验证（Federated Identity）**：用户持有唯一的 `Master Vault ID`。当用户感应某品牌资产时，HOAC 自动识别设备特征，将请求分发至对应的 `Brand-Connector`，由品牌方 API 确认真伪后，在用户统一藏馆中生成"镜像子卡"。

### 14.2 品牌端三种接入方案

为实现品牌方"极简接入"，提供三个维度的兼容方案：

| 方案 | 适用场景 | 操作方式 | 技术逻辑 |
|------|----------|----------|----------|
| Light-Link（API 转发） | 已有成熟数字化系统的品牌 | 品牌方开放只读 `Verification_API` | ALP 充当前端展示层，权属变更由品牌方内部完成，ALP 同步状态 |
| HOAC Sidecar（侧车模式） | 有数字化意识但技术实力中等的品牌 | 部署标准容器模块（Rust 实现）到品牌方云端 | 模块自动处理授权绑定逻辑和加密，天然兼容全局清算中心 |
| Ghost-Tag（影子标签） | 暂不愿开放 API 的品牌，存量资产场景 | ALP 作为第三方鉴定机构，为存量资产发放 `HOAC_Certified_Tag` | 流量倒逼策略：当大量用户在平台交易时，品牌方为获取"确权税"主动寻求 API 接入 |

### 14.3 聚合藏馆 API 抽象层

在前端设计统一的 `AssetProvider` 接口，屏蔽底层品牌差异：

```rust
/// 资产提供者 trait（后端 Rust 等价抽象）
#[async_trait]
pub trait AssetProvider: Send + Sync {
    /// 验证资产真伪
    async fn verify(&self, device_data: &str) -> Result<bool, ProviderError>;
    /// 获取资产元数据（材质、价格等）
    async fn get_metadata(&self, asset_id: &str) -> Result<AssetDetails, ProviderError>;
    /// 发起确权转移
    async fn initiate_transfer(&self, asset_id: &str, target_user: &str) -> Result<bool, ProviderError>;
}
```

各品牌 Connector（`LVProvider`、`HermesProvider`、`ChanelProvider`）分别实现此接口。用户在统一藏馆中只看到资产卡片，底层 API 调度完全透明。

### 14.4 品牌特权 API（Brand Privileges）

为吸引品牌方接入，协议预留以下品牌特权：

| 特权 | 说明 |
|------|------|
| 分润自动分拨 | 每笔二手流转的"确权税"通过托管账户自动划拨给品牌方，无需人工对账 |
| 数据黑盒 | 品牌方可设置隐私边界：可见资产流转状态，但不可见买卖双方身份（法律纠纷除外） |
| 二次营销接口 | 品牌方可向持有 `#已确权` 标签的用户定向推送新品预留、私人活动邀请等 |

### 14.5 最终愿景：资产界的 Apple Wallet

如同 Apple Wallet 聚合了所有银行的信用卡，ALP 聚合所有品牌的实物资产。用户无需关心后台接入了多少品牌系统，只需知道：**进入 Vault 的资产即为真实的、可质押的、可跨境变现的全球资产。**

---

## 十五、与 RC-Protocol 的关系

### 可直接复用的模块

| RC-Protocol 模块 | 复用方式 | 说明 |
|------------------|----------|------|
| 14 态状态机 | 直接移植 | 经属性测试验证，通用性强 |
| 权限矩阵 (5 角色) | 重命名后移植 | Brand→Tenant，其余逻辑不变 |
| 密钥派生链 (KDF) | 直接复用 | Root→Tenant→Device 三级结构通用 |
| AES-CMAC 实现 | 直接复用 | 密码学原语与业务无关 |
| 熔断器 (Circuit Breaker) | 直接复用 | 五维检测引擎通用 |
| 阶梯式安全等级 | 直接复用 | Normal/Elevated/Critical 三级通用 |
| 主权自愈流程 | 直接复用 | 三重验证解冻逻辑通用 |
| 错误码体系 | 重新编号，保留分类结构 | 分类方式（Crypto/Protocol/State/Auth）通用 |
| 审计日志架构 | 直接复用 | 双轨分离 + 事件分类通用 |
| 脱敏工具函数 | 直接复用 | mask_uid 等安全工具通用 |

### 需要重新设计的模块

| 模块 | 原因 |
|------|------|
| SUN 消息解析 | NTAG 424 DNA 专用格式，需抽象为 trait |
| SDM 向量构造 | NFC SDM 专用，需抽象为 DeviceAuthProtocol |
| APDU 命令集 | NFC 专用，移入具体协议实现 |
| NFC 设备交互 | 硬件专用，移入具体协议实现 |
| H5 前端 | 需要适配多行业 UI 主题 |

### 建议的开发路径

```
Phase 1: 核心框架搭建
  ├── 从 RC-Protocol 提取通用模块（状态机、KDF、熔断器）
  ├── 定义 DeviceAuthProtocol trait 接口
  ├── 搭建 alp-common / alp-core 基础 crate
  └── 属性测试迁移验证

Phase 2: 第一个协议实现
  ├── 将 NTAG 424 DNA SUN 实现为 DeviceAuthProtocol 的第一个插件
  ├── 搭建 alp-kms 密钥管理服务
  ├── 搭建 alp-api 业务 API
  └── 端到端集成测试

Phase 3: 多协议扩展
  ├── 实现 RFID / 二维码 / BLE 协议插件
  ├── 多租户管理控制台
  └── 行业定制化配置

Phase 4: 生产化
  ├── PostgreSQL 迁移
  ├── 硬件 HSM 集成
  ├── 监控告警体系
  └── 合规认证
```

---

## 附录 A：术语对照表

| 英文术语 | 中文 | 说明 |
|----------|------|------|
| ALP | 资产全生命周期管控平台 | Asset Lifecycle Platform |
| Authority Device | 授权设备 | 拥有绑定权限的物理设备 |
| Asset Tag | 资产标识 | 附着在实物上的唯一标识 |
| Authorization Binding | 授权绑定 | 授权设备与资产标识建立加密绑定 |
| Device Auth Message | 设备认证消息 | 从物理设备读取的认证数据 |
| KDF | 密钥派生函数 | Key Derivation Function |
| Circuit Breaker | 熔断器 | 异常检测与自动防御引擎 |
| Graduated Defense | 阶梯式防御 | Normal→Elevated→Critical 三级响应 |
| Sovereign Recovery | 主权自愈 | 三重验证解冻流程 |
| Transport Key | 出厂密钥 | 资产绑定前的初始认证密钥 |
| Tenant Key | 租户密钥 | 绑定后的租户专属密钥 |
| Zeroize | 内存清零 | 密钥材料用完后立即清零 |
| Constant-Time Comparison | 常量时间比较 | 防止计时侧信道攻击 |
