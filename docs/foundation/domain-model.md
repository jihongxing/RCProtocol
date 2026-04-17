# 领域模型与术语

> 文档类型：Foundation  
> 状态：Active  
> 权威级别：Authoritative  
> 最后更新：2026-04-18（补充权利结构 V1 与高价值资产凭证分层）

---

## 1. 术语规则

本文件是 RCProtocol 的统一术语表与核心对象定义源。

其他文档引用以下术语时，含义必须与本文件保持一致，不得重新发明同义词或并行概念。

凡涉及容易混淆的概念，应优先参考 `protocol-definition.md`。本文件更偏向“对象建模”，不是对所有术语重新哲学化解释。

---

## 2. 核心对象

### 2.1 Asset

被 RCProtocol 管理的实物资产。它是状态机、权属流转、安全校验、审计与展示的中心对象。

当前 MVP 中，Asset 的工程表示仍然明显依赖数据库记录；终局协议中，Asset 的真相表达应逐步收敛到 `AssetCommitment + 协议事件 + 承诺证明`。

当前常见字段包括：

- `asset_id`：当前工程内唯一标识
- `uid`：子标签 UID
- `brand_id`：所属品牌
- `external_product_id`：品牌方 SKU ID（外部映射）
- `external_product_name`：品牌方 SKU 名称（可选，用于展示）
- `external_product_url`：品牌方 SKU 详情页（可选，用于跳转）
- `current_state`：当前状态机状态
- `owner_id`：当前合法持有者的当前投影表示

注意：

- `asset_id` 是当前工程主键，不应被直接等同为终局协议真相对象
- `owner_id` 当前是关键字段，但终局应逐步降级为投影视图

### 2.2 Tag

附着于资产上的物理标识与认证载体。当前实现基线为 `NTAG 424 DNA`。

Tag 提供：

- UID
- 计数器（CTR）
- 动态认证消息
- 芯片级安全边界

Tag 是协议的硬件锚点，不是单纯的可扫码 ID。

### 2.3 Authority Device / Mother Tag

用于发起授权、激活、处置或绑定确认的授权载体。

Authority Device 有两种形态：

**物理形态（Physical NFC）**：
- 基于 `NTAG 424 DNA` 的物理 NFC 芯片
- 安全锚点为硬件 `AES-128 CMAC`
- 适用于高价值商品

**虚拟形态（默认）**：
- `VIRTUAL_QR`
- `VIRTUAL_APP`
- `VIRTUAL_BIOMETRIC`（未来扩展）

当前默认以虚拟母卡为主。虚拟母卡的安全锚点不是物理芯片，而是：

- 账号身份
- 生物识别
- 虚拟凭证

历史文档中的“母卡 / 母标签 / Authority Device / Key”在当前体系下统一收敛到该对象。

### 2.4 Child Tag / Asset Tag

附着于实物上的资产标签。当前统一理解为：**资产本体所依附的物理身份载体**。

历史别名包括：

- 子标签
- Crown
- Asset Tag

### 2.5 Authorization Binding

历史文档中常称“母子纠缠”。当前统一定义为：

> 授权载体与资产标签之间建立的、受协议约束的授权绑定关系。

该关系用于：

- 激活资产
- 约束处置权
- 支撑近场或多因子交互
- 建立资产主权前提
- 支撑过户时的授权校验

“纠缠”保留为业务叙事词；“授权绑定”是工程与规范用词。

### 2.6 Blind Scan / Blind Entry

工厂端或生产端只登记 UID 与最小必要生产记录，不接触品牌定义与商品详情的流程。

### 2.7 Activation

将待分配资产转化为协议内已激活资产的过程，当前通常涉及：

- 密钥翻转或密钥准备
- 绑定建立（生成虚拟母卡或关联物理母卡）
- 品牌认领
- 外部 SKU 映射绑定
- 状态推进

当前 MVP 中，激活仍主要表现为平台系统中的状态变化与记录写入；终局中，激活应升级为围绕 `AssetCommitment` 生成承诺的协议动作。

### 2.8 Ownership

资产在协议中的当前合法持有关系。

Ownership 不是单纯“谁拿着实物”，而是协议确认的合法持有状态。

当前 MVP 中，`owner_id` 是主要承载字段；终局中，Ownership 应由：

- 协议状态机
- 授权证明
- 转移事件
- 必要的承诺规则

共同决定。

### 2.8.1 权利结构 V1

为避免把“持有”“使用”“处置”“品牌参与”混成一句话，当前建议先把 RCProtocol 的权利结构收敛为 V1 四层：

- `Ownership`：谁是协议中的当前合法 owner
- `ControlRight`：谁有权发起转让、冻结、恢复、终态推进等关键处置动作
- `UsageProof`：谁能够证明自己当前持有或接触实物，并完成近场验证或现场交互
- `Brand Governance Right`：品牌方在激活、合法售出、异常治理、分账与品牌声明中的保留治理权

这四层不是四套平行状态机，而是同一资产在协议中的四类不同事实。

### 2.8.2 ControlRight

`ControlRight` 是对资产执行关键协议动作的处置权。

它不等于 Ownership，也不等于拿着实物本身。

当前 V1 中，`ControlRight` 主要由以下因素共同决定：

- 当前 Ownership
- Authority Device / Mother Tag 绑定关系
- 生物识别或账户身份
- 品牌或平台在特定动作中的治理权限

在当前实现中，`ControlRight` 已部分落地为：

- owner 发起 transfer 的资格检查
- 子标签动态认证
- 虚拟母卡 / 物理母卡授权校验

但它尚未被抽象为独立数据库对象或独立协议承诺对象。

### 2.8.3 UsageProof

`UsageProof` 是关于“谁当前持有实物并能完成现场交互”的证明层。

它的作用是：

- 支撑验真场景中的现场真实性
- 支撑过户场景中的近场操作
- 支撑未来使用权、借展、租赁等更复杂场景

必须明确：

- `UsageProof` 不是 Ownership
- `UsageProof` 也不是独立的 `UsageRight`
- 它是对当前物理占有或现场交互能力的证明，而不是长期法定权利本身

当前实现中，`UsageProof` 主要通过以下能力体现：

- 子标签动态认证
- 虚拟母卡凭证
- 物理母卡扫描
- 生物识别或账户身份确认

### 2.8.4 Brand Governance Right

`Brand Governance Right` 是品牌方在协议中的保留治理权。

它至少包括：

- 激活与外部 SKU 映射的声明权
- 合法售出前的业务确认权
- 异常争议中的协助治理权
- 流转确权费中的分账参与权

它不包括：

- 品牌方单方面重写 Ownership
- 品牌方单方面绕过协议状态机强行回收资产
- 品牌方单方面定义真品是否成立

### 2.8.5 V1 暂不独立建模的权利

以下权利被视为后续阶段再引入的扩展对象，当前不进入正式独立对象层：

- `UsageRight`：可被单独签发、转让、租赁的使用权
- `IncomeRight`：由借展、租赁、分润等行为产生的收益权
- `CollateralRight`：可用于质押、授信或清算的担保权

原因不是这些方向不重要，而是当前 MVP 仍应优先打穿：

- 确权
- 合法售出
- 入馆
- 过户
- 审计

只有这条主链路稳定后，再把使用权、收益权、质押权做成独立协议对象，复杂度才可控。

### 2.8.6 Ownership Credential Profile

针对不同价值带的资产，当前建议采用分层凭证策略来承载 Ownership 与 ControlRight：

- **低价值带资产**：默认以账户登录态、App Token、虚拟母卡凭证为主
- **高价值带资产**：可要求额外的物理 ownership card、实体母卡或更强的双因子共签凭证

当前建议可采用如下默认商务分层：

- `<= 50,000`：优先走账户凭证 + 虚拟母卡路径
- `> 50,000`：优先走实体卡 / 物理母卡 / 更强共签路径

但必须明确：

- 该阈值是**产品与风控策略默认值**，不是协议常量
- 最终阈值应按品类、品牌要求、售后风险和交易金额配置
- 不应把“标签本身”直接定义为使用权，也不应把“App Token 本身”直接定义为 Ownership

更准确的说法是：

- 标签是物理锚点与 `UsageProof` 的核心组成
- App Token / 实体卡 / 母卡是承载 Ownership 与 ControlRight 的不同凭证层

### 2.9 Transfer

协议内的权属变更流程。它要求状态、权限与安全校验条件全部满足，才可完成所有权更新。

当前数据库中的 `asset_transfers` 是重要工程记录；终局中，它应被理解为协议动作的审计与查询视图，而不是协议真相本体。

### 2.10 Dispute

资产因风险、争议、异常认证、审计问题等原因进入冻结审查状态的治理过程。

### 2.11 External Product Mapping

RCProtocol 不管理 SKU 的完整业务详情，而只管理“资产与外部 SKU 的映射关系”。

核心理念：

> 我们不是品牌的 ERP，我们是品牌 ERP 的防伪与协议插件。

当前常见映射字段：

- `external_product_id`
- `external_product_name`
- `external_product_url`

### 2.12 AssetCommitment

`AssetCommitment` 是 RCProtocol 终局协议应围绕收敛的统一承诺对象。

建议最小构成包括：

- `brand_id`
- `asset_uid` 或协议级唯一资产标识
- `chip_binding`
- `epoch`
- `metadata_hash`

可表达为：

```text
AssetCommitment = H(brand_id || asset_uid || chip_binding || epoch || metadata_hash)
```

它的意义在于：

- 让协议真相不再直接绑定数据库表主键
- 让品牌承诺与平台承诺可以指向同一对象
- 让验真流程能逐步摆脱“先查资产表是否存在”的逻辑依赖

当前该对象尚未正式落地到实现层，但已经成为后续协议演进的正式对象。

### 2.13 Brand Attestation

品牌方对某个 `AssetCommitment` 给出的协议级承诺证明。

当前尚未落地为正式实现，但后续会成为品牌从“业务参与方”升级为“共同信任根之一”的关键机制。

### 2.14 Platform Attestation

平台方对某个 `AssetCommitment` 给出的共同承诺证明。

它不是平台数据库中的一行记录，而应是平台对同一承诺对象给出的不可抵赖声明。

---

## 3. 角色对象

### 3.1 Platform

平台方。当前负责：

- 系统治理
- 平台侧密钥托管边界
- 审计
- 风控与恢复
- API / BFF / 运维承载

必须明确：

- 当前 Platform 仍然承担较强的中心化承载职责
- 终局中 Platform 应降级为协议参与者之一，而不是绝对裁判

### 3.2 Brand

品牌方。当前负责：

- 资产激活
- 外部 SKU 映射
- 销售合法化流程
- 在自己的系统中完成审批后，通过 API 或 Web 后台调用协议执行层

当前 Brand 的主要身份是：**业务参与方**。

终局中，Brand 应升级为：**协议共同信任根之一**。

### 3.3 Factory

工厂或生产方。负责：

- 盲扫登记
- 生产入库阶段的最小记录

### 3.4 Consumer

消费者或合法持有者。负责：

- 验真
- 过户参与
- 消耗 / 传承类终态操作

### 3.5 Moderator

审核员 / 安全审核角色。负责：

- 冻结
- 恢复
- 篡改与失陷标记
- 争议处理

---

## 4. 技术对象

### 4.1 Root Key

系统根密钥。位于最高安全边界，用于派生品牌级密钥空间。

当前它由平台侧掌握，因此当前模型是平台主导的品牌隔离模型，而不是共同持密模型。

### 4.2 Brand Key

基于 `Root Key + Brand ID + System ID` 派生出的品牌级密钥。

注意：

- `Brand Key` 的存在不等于 Brand 已成为共同信任根
- 它只表明当前实现具备品牌隔离能力

### 4.3 Chip Key / K_chip

基于品牌密钥与标签 UID、epoch 等信息派生的芯片终端密钥，用于动态认证。

当前常见子类包括：

- `K_chip_child`
- `K_chip_mother`

### 4.4 K_honey

用于 HBM / 蜜标相关校验的 HMAC 密钥。

### 4.5 CTR

标签或认证消息中使用的计数器，用于防重放与异常检测。

### 4.6 CMAC / HMAC

RCProtocol 当前核心认证与派生中使用的两类密码学机制：

- `HMAC-SHA256`：用于派生
- `AES-128 CMAC`：用于动态认证消息验证

### 4.7 API Key

品牌方的 API 对接凭证。

它代表：

- 品牌获得平台接入能力
- 品牌可通过 API 调用协议执行层

它不代表：

- 品牌已经成为协议共同信任根
- 品牌已经完成协议承诺职责落地

### 4.8 Webhook

品牌方配置的事件回调地址。用于平台向品牌方推送关键事件通知。

---

## 5. 产品对象

### 5.1 Sovereign Vault

C 端资产馆 / 数字资产档案馆，用于展示持有资产、验证结果、状态与流转记录。

### 5.2 Sovereign Preview

用于在受控场景下分享资产预览、展示权属信息或引导交易前校验的受限预览能力。

### 5.3 Sovereign Console / BConsole

B 端治理后台。定位为：

> 品牌 ERP 的防伪与协议插件，而非品牌的 ERP 系统本体。

核心能力：

- 极简品牌注册与管理
- API Key 管理
- 资产列表与外部 SKU 映射
- 盲扫批次管理
- 激活操作
- 售出确认
- 基础审计查询

---

## 6. 状态术语

以下状态正式定义见 `state-machine.md`：

- `PreMinted`
- `FactoryLogged`
- `Unassigned`
- `RotatingKeys`
- `EntangledPending`
- `Activated`
- `LegallySold`
- `Transferred`
- `Consumed`
- `Legacy`
- `Tampered`
- `Compromised`
- `Destructed`
- `Disputed`

历史文档中的业务叙事词不作为正式状态枚举。

---

## 7. 文档术语清理结论

本次整理后，以下处理成立：

- “母子纠缠”保留为业务表述，但规范术语采用“授权绑定”
- “主权”是业务层总称，不替代状态、权限、密钥、所有权这些精确定义
- `asset_id` 是当前工程 ID，不等于终局协议承诺对象
- 品牌参与不等于品牌共同信任根
- 品牌隔离不等于品牌共同持密
- 数据库记录不等于终局协议真相
- `AssetCommitment`、`Brand Attestation`、`Platform Attestation` 已进入正式术语体系，后续实现必须围绕其收敛

---

## 8. 关联文档

- 协议术语：`protocol-definition.md`
- 当前差距：`protocol-gap-analysis.md`
- 终局架构：`target-protocol-architecture.md`
- 安全模型：`security-model.md`
- 服务边界：`api-and-service-boundaries.md`
