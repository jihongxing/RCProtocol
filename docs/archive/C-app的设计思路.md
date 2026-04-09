> **⚠️ 已归档**：本文档内容已于 2026-04-07 合并至权威基线文档，仅作历史参考。  
> 变更已写入：`domain-model.md`、`mvp-scope-and-cutline.md`、`product-system.md`  
> 重构计划：`docs/refactoring-plan.md`

# C-App 设计思路：高净值人群的数字身份证

> 从"防伪工具" → "高净值人群的数字身份证"——从工具型产品到身份型产品的跃迁。

---

## 核心定位

**不是：**

- ❌ 防伪查询工具（得物、只二的鉴定功能）
- ❌ 资产管理工具（记账 App）
- ❌ 社交平台（小红书、Instagram）

**而是：**

- ✅ 高净值人群的数字身份证（类比：美国运通黑卡）
- ✅ 实物资产的主权钱包（类比：比特币钱包）
- ✅ 消费品位的信用体系（类比：蚂蚁信用分）

---

## 产品命名建议

| 方案 | 名称 | 定位 | Slogan |
| --- | --- | --- | --- |
| A（推荐） | **Vault** | 你的私人金库，存放真实资产的数字主权 | Your Sovereign Vault |
| B | Sovereign | 资产主权，身份主权，品位主权 | Own Your Sovereignty |
| C | Regalis | 皇家级资产管理，高净值人群的专属空间 | The Royal Asset Vault |

---

## 产品架构

```
┌─────────────────────────────────────────┐
│ Layer 1: 验证层（Trust Layer）           │
│ - 扫码验真 / 动态认证 / 风险提示        │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ Layer 2: 资产层（Asset Layer）           │
│ - 数字馆（私人展示）/ 资产过户 / 资产估值 │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│ Layer 3: 身份层（Identity Layer）        │
│ - 主权分数 / 身份认证（黑卡级别）/ 权益解锁│
└─────────────────────────────────────────┘
```

---

## 一、验证层（Trust Layer）

### 1.1 扫码验真（入口）

**场景：**

- 用户在商场看到一个包，扫码验真
- 用户收到礼物，扫码验真
- 用户在二手市场买东西，扫码验真

**验真结果关键设计点：**

- ✅ 显示 Owner 的用户名和主权分数（建立信任）
- ✅ 如果是自己的资产，显示"这是您的资产"
- ✅ 如果不是自己的资产，显示"该资产归属于 XXX"

```
┌─────────────────────────────────────────┐
│          ✓ 认证通过                      │
│                                         │
│  [品牌 Logo]                             │
│  Hermès Birkin 30                       │
│  黑色 Togo 皮 2024 款                    │
│                                         │
│  资产编号：#RC-2024-001234              │
│  激活时间：2024-03-15                   │
│  验证次数：第 12 次                     │
│                                         │
│  该资产归属于                            │
│  @sophia_chen                           │
│  主权分数：9,850                         │
│                                         │
│  [查看资产详情]  [添加到我的馆藏]        │
└─────────────────────────────────────────┘
```

### 1.2 风险提示（安全）

**场景：** 检测到 CTR 回滚（重放攻击）、资产被冻结、资产处于争议状态

```
┌─────────────────────────────────────────┐
│          ⚠ 风险警告                      │
│                                         │
│  该标签可能存在安全风险：                │
│  • 检测到异常扫描记录                    │
│  • 该资产当前处于争议冻结状态            │
│                                         │
│  建议：谨慎购买 / 联系品牌方 / 联系客服  │
│                                         │
│  [联系客服]  [返回]                     │
└─────────────────────────────────────────┘
```

---

## 二、资产层（Asset Layer）

### 2.1 数字馆（核心功能）

> 定位：不是"资产列表"，而是"私人博物馆"

**首页关键设计点：**

- ✅ 主权分数放在最显眼位置（核心指标）
- ✅ 资产总值实时更新（基于市场估值）
- ✅ 全球排名（激发攀比心理）
- ✅ 卡片式展示（类似 Instagram）


```
┌─────────────────────────────────────────┐
│  [头像]  @sophia_chen        [设置]     │
│                                         │
│  主权分数：9,850  🔥                     │
│  资产总值：¥ 2,850,000                  │
│  全球排名：Top 0.1%                     │
│                                         │
│  我的馆藏（12 件）                       │
│                                         │
│  ┌───────────┐  ┌───────────┐         │
│  │ Hermès    │  │ Rolex     │         │
│  │ Birkin 30 │  │ Daytona   │         │
│  │ ¥280,000  │  │ ¥350,000  │         │
│  └───────────┘  └───────────┘         │
│                                         │
│  [🔍验真] [💎馆藏] [🌐发现] [👤我的]    │
└─────────────────────────────────────────┘
```

### 2.2 资产详情（沉浸式）

**关键设计点：**

- ✅ 实时估值（基于市场数据）
- ✅ 增值显示（激发投资心理）
- ✅ 所有权历史（建立信任）
- ✅ 品牌认证标识（权威背书）

```
┌─────────────────────────────────────────┐
│  [< 返回]              [分享] [更多]     │
│                                         │
│  [大图展示 + 多图轮播]                   │
│                                         │
│  Hermès Birkin 30                       │
│  黑色 Togo 皮 2024 款                    │
│                                         │
│  当前估值：¥ 280,000                     │
│  购入价格：¥ 250,000                     │
│  增值：+12% 📈                           │
│                                         │
│  资产信息                               │
│  • 资产编号：#RC-2024-001234            │
│  • 激活时间：2024-03-15                 │
│  • 品牌认证：✓ Hermès 官方认证          │
│  • 验证次数：12 次                      │
│                                         │
│  所有权历史                             │
│  • 2024-03-15  品牌激活                 │
│  • 2024-03-20  售出给 @sophia_chen      │
│                                         │
│  [发起过户]  [申请估值]  [查看证书]     │
└─────────────────────────────────────────┘
```

### 2.3 资产过户（流畅体验）

**关键设计点：**

- ✅ 过户费用透明展示（品牌分成 + 平台分成）
- ✅ 安全验证进度可视化
- ✅ 虚拟母卡自动调用（用户无感知）

**过户流程：**

1. 选择接收方（用户名或手机号）
2. 确认过户费用（服务费 2%：品牌 50% + 平台 50%）
3. 生物识别验证
4. 扫描商品标签
5. 虚拟母卡自动验证
6. 确认过户（不可撤销）

---

## 三、身份层（Identity Layer）

### 3.1 主权分数（Sovereignty Score）

> 定位：高净值人群的信用体系

**计算逻辑：**

```typescript
function calculateSovereigntyScore(user: User): number {
  let score = 0

  // 1. 资产总值（权重 40%）—— 每 1 万元 = 1 分，最高 4000 分
  score += Math.min(assetValue / 10000, 4000)

  // 2. 资产数量（权重 20%）—— 每件资产 = 100 分，最高 2000 分
  score += Math.min(assetCount * 100, 2000)

  // 3. 品牌多样性（权重 15%）—— 每个品牌 = 150 分，最高 1500 分
  score += Math.min(brandCount * 150, 1500)

  // 4. 资产增值率（权重 10%）—— 最高 1000 分
  score += Math.min(avgAppreciation * 1000, 1000)

  // 5. 活跃度（权重 10%）—— 每次验真 = 5 分，最高 1000 分
  score += Math.min(verifyCount * 5, 1000)

  // 6. 账号年龄（权重 5%）—— 每年 = 100 分，最高 500 分
  score += Math.min(accountAge * 100, 500)

  return Math.round(score)
}
```

**分数等级：**

| 分数 | 等级 | 全球排名 |
| --- | --- | --- |
| 10,000+ | 💎 钻石级（Diamond） | Top 0.01% |
| 8,000+ | ◆ 黑金级（Platinum） | Top 0.1% |
| 6,000+ | 🥇 黄金级（Gold） | Top 1% |
| 4,000+ | 🥈 白银级（Silver） | Top 10% |
| 2,000+ | 🥉 青铜级（Bronze） | Top 30% |
| 0-2,000 | 普通级（Standard） | 其他 |

### 3.2 身份认证（黑卡级别）

> 定位：主权分数 = 高净值身份证明

**应用场景：**

| 领域 | 场景 |
| --- | --- |
| 金融服务 | 银行高额信用卡、券商融资融券、保险更低保费 |
| 商业服务 | 酒店免押金入住、租车免押金、会所会员资格 |
| 社交场景 | 高端社交 App 认证、相亲平台资产证明、商务社交信任 |

**第三方验证接口：**

```http
POST /api/v1/verify-identity
Authorization: Bearer <third_party_api_key>

{
  "qr_code": "RC_CERT_20240407_ABCD1234",
  "user_consent": true
}
```

```json
// Response
{
  "verified": true,
  "user_id": "user_001",
  "username": "@sophia_chen",
  "sovereignty_score": 9850,
  "rank": "Top 0.1%",
  "asset_value": 2850000,
  "asset_count": 12,
  "account_age_days": 365,
  "verified_at": "2024-04-07T12:00:00Z"
}
```

### 3.3 权益解锁（激励体系）

| 等级 | 权益 |
| --- | --- |
| 💎 钻石级 | 免费估值 10 次/月、免费过户 3 次/月、24/7 专属客服、品牌方直连、金融绿色通道、年度资产报告 |
| ◆ 黑金级 | 免费估值 5 次/月、过户 5 折、专属客服（工作日）、金融优先通道、季度资产报告 |
| 🥇 黄金级 | 免费估值 2 次/月、过户 8 折、在线客服优先、月度资产报告 |
| 🥈 白银级 | 免费估值 1 次/月、过户 9 折 |
| 🥉 青铜级 | 基础权益 |
| 普通级 | 基础功能 |

---

## 四、社交层（Social Layer）

### 4.1 资产展示（可选公开）

> 定位：高净值人群的"朋友圈"

**关键设计点：**

- ✅ 用户可选择是否公开资产
- ✅ 显示用户等级（建立信任）
- ✅ 类似 Instagram 的社交体验
- ✅ 评论、点赞、分享

### 4.2 高净值社交（未来扩展）

- **同好圈子：** 劳力士爱好者圈、Hermès 收藏家圈、茅台投资者圈
- **线下活动：** 品牌方 VIP 活动、高净值人群聚会、资产鉴赏会
- **资产交易：** 用户间资产交易、平台担保交易、过户费分成

---

## 完整 App 架构

### 底部导航

| Tab | 功能 |
| --- | --- |
| 🔍 验真 | 扫码验真、验真结果、风险提示 |
| 💎 馆藏 | 我的馆藏、资产详情、发起过户、资产估值 |
| 🌐 发现 | 动态流、圈子、活动 |
| 👤 我的 | 个人主页、主权分数、身份认证、我的权益、设置、帮助中心 |

---

## 商业闭环

### 1. 用户获取（Acquisition）

| 阶段 | 用户量 | 策略 |
| --- | --- | --- |
| 品牌方导流 | 0 → 1,000 | 商品包装盒内放置"激活卡"，扫码下载 App 并激活资产 |
| 口碑传播 | 1,000 → 10,000 | 社交场景展示 + 邀请机制，双方获得主权分数加成 |
| 金融机构合作 | 10,000 → 100,000 | 银行/券商/保险导流，凭主权分数申请服务 |
| 品牌深度合作 | 100,000+ | 品牌推出"主权分数专享款"，高分用户优先购买权 |

### 2. 用户激活（Activation）

```
下载 App → 扫码验真 → 发现是自己的资产 → 注册账号 → 资产自动入馆
→ 显示主权分数（初始 + 新手奖励） → 引导添加更多资产
```

### 3. 用户留存（Retention）

| 周期 | 策略 |
| --- | --- |
| 日留存（D1） | Push：资产估值更新、有人验真了您的资产、主权分数排名上升 |
| 周留存（D7） | 每周资产报告、权益提醒、社交动态 |
| 月留存（D30） | 月度资产报告（详细分析）、权益总结、排名变化 |

**留存机制核心：**

1. 资产估值更新（每日）→ 每天打开 App 查看增值
2. 主权分数排名（实时）→ 攀比心理
3. 权益到期提醒（每月）→ 不想浪费权益
4. 社交互动（实时）→ 社交粘性

### 4. 用户变现（Revenue）

| 收入来源 | 模式 | 说明 |
| --- | --- | --- |
| 过户手续费（核心） | 交易额 2%，品牌/平台五五分成 | 10 万用户 × 5 件 × 5 万 × 20% 流转率 = 年交易额 50 亿，平台收入 5000 万 |
| 资产估值服务 | ¥99/次（付费），各等级有免费额度 | — |
| 会员订阅 | 基础免费 / 高级 ¥99/月 / 尊享 ¥999/月 | — |
| 金融服务佣金 | 信用卡 ¥200/单、券商 ¥500/单、保险保费 10% | — |
| 品牌方增值服务 | 数据报告 ¥10 万/年、用户画像 ¥5 万/次 | — |

**收入预测（3 年）：**

| 年份 | 用户数 | 资产数 | 年交易额 | 总收入 |
| --- | --- | --- | --- | --- |
| Year 1（冷启动） | 10,000 | 50,000 | 2.5 亿 | 300 万 |
| Year 2（增长期） | 100,000 | 500,000 | 25 亿 | 3,000 万 |
| Year 3（爆发期） | 500,000 | 2,500,000 | 125 亿 | 1.5 亿 |

### 5. 用户推荐（Referral）

| 行为 | 奖励 |
| --- | --- |
| 邀请好友注册 | 邀请人 +200 分，被邀请人 +100 分 |
| 被邀请人添加首件资产 | 邀请人再 +300 分 |
| 分享资产到朋友圈 | +10 分 |
| 分享主权分数到朋友圈 | +20 分 |
| 创建圈子 | +500 分 |
| 邀请好友加入圈子 | +50 分/人 |

---

## 技术架构

### 前端技术栈

| 项目 | 选型 |
| --- | --- |
| 框架 | uni-app (Vue 3)，跨端（H5 / 小程序 / App） |
| UI 风格 | Apple 风格（简洁、高端），黑金色系 |
| 字体 | SF Pro / PingFang SC |
| 核心组件 | RcAssetCard、RcScoreRing、RcLevelBadge、RcVerifyResult、RcTransferFlow |

### 后端技术栈

| 服务 | 说明 |
| --- | --- |
| Rust `rc-api` | 协议核心 |
| Go `go-bff` | 前端聚合 |
| Go `go-iam` | 身份认证 |
| Go `go-score` | 主权分数计算（新增） |
| Go `go-valuation` | 资产估值（新增） |
| Go `go-social` | 社交功能（新增） |
| Go `go-notification` | 通知推送（新增） |

**数据存储：** PostgreSQL（主数据库）+ Redis（缓存 + 排行榜）+ Elasticsearch（搜索）+ OSS（图片存储）

### 新增数据库表

```sql
-- 用户表（扩展）
CREATE TABLE users (
    user_id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(64) UNIQUE,
    phone VARCHAR(32) UNIQUE,
    email VARCHAR(128),
    avatar TEXT,
    sovereignty_score INTEGER NOT NULL DEFAULT 0,
    score_rank INTEGER,
    score_level VARCHAR(32),
    asset_count INTEGER NOT NULL DEFAULT 0,
    asset_value DECIMAL(15, 2) NOT NULL DEFAULT 0,
    total_verify_count INTEGER NOT NULL DEFAULT 0,
    is_public BOOLEAN NOT NULL DEFAULT false,
    follower_count INTEGER NOT NULL DEFAULT 0,
    following_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 主权分数历史表
CREATE TABLE sovereignty_score_history (
    history_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    score INTEGER NOT NULL,
    score_change INTEGER NOT NULL,
    change_reason VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 资产估值表
CREATE TABLE asset_valuations (
    valuation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id VARCHAR(64) NOT NULL REFERENCES assets(asset_id),
    valuation_value DECIMAL(15, 2) NOT NULL,
    valuation_source VARCHAR(64),
    valuation_confidence DECIMAL(3, 2),
    valuated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 社交动态表
CREATE TABLE social_posts (
    post_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    asset_id VARCHAR(64) REFERENCES assets(asset_id),
    content TEXT,
    images TEXT[],
    like_count INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,
    share_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 用户关注表
CREATE TABLE user_follows (
    follow_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    following_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_follow UNIQUE (follower_id, following_id)
);

-- 身份认证记录表
CREATE TABLE identity_verifications (
    verification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    third_party_id VARCHAR(64) NOT NULL,
    third_party_name VARCHAR(128),
    verification_type VARCHAR(64),
    qr_code VARCHAR(128),
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);
```

---

## 竞争壁垒

| 壁垒类型 | 内容 | 效果 |
| --- | --- | --- |
| 技术壁垒 | NTAG 424 DNA + 动态 CMAC、母子标签纠缠、KDF 密钥派生链、状态机 + 审计体系 | 竞争对手难以复制 |
| 数据壁垒 | 真实资产数据、消费品位数据、流转数据、估值数据 | 数据越多价值越大 |
| 网络效应 | 品牌方越多 → 用户越多 → 金融机构越多 → 用户越有价值 | 正向飞轮 |
| 身份壁垒 | 主权分数积累、等级权益 | 用户迁移成本极高 |

---

## 运营策略

| 阶段 | 时间 | 目标 | 核心策略 |
| --- | --- | --- | --- |
| Phase 1 冷启动 | Month 1-3 | 签约 3-5 个品牌，激活 1000 用户 | 高端品牌试点，包装盒激活卡 |
| Phase 2 增长期 | Month 4-12 | 签约 20 个品牌，激活 10,000 用户 | 口碑传播 + 金融机构合作 + KOL |
| Phase 3 爆发期 | Year 2 | 签约 100 个品牌，激活 100,000 用户 | 品牌深度合作 + 金融深度整合 + 线下活动 |

---

## 下一步行动

1. **设计 UI/UX（2 周）：** 主权分数页面、资产馆页面、验真结果页面
2. **实现核心功能（1 个月）：** 扫码验真、资产馆、资产过户、主权分数计算
3. **签约第一个品牌（1 个月）：** 找高端品牌试点、完成技术对接、激活前 100 个用户
4. **验证商业模式（3 个月）：** 完成 100 次过户、验证分成模式、收集用户反馈

> **目标：6 个月内完成 MVP，激活 1000 个用户，验证商业闭环。**

---

## 总结

> **Vault 不是一个 App，而是高净值人群的数字身份证。拥有 Vault 账号并持有真实资产，本身就是一种身份象征。**

**核心价值：**

| 角色 | 价值 |
| --- | --- |
| 对用户 | 证明高净值身份 |
| 对品牌 | 掌握二级市场，持续分成 |
| 对金融机构 | 获取高净值客户 |
| 对平台 | 交易抽成 + 金融佣金 |

**商业模式：** 过户手续费（核心）+ 金融服务佣金（增值）+ 会员订阅（稳定）
