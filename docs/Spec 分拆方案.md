# Spec 分拆方案

> 最后更新：2026-04-07（对齐品牌极简化、外部 SKU 映射、虚拟母卡、品牌 API 对接等基线变更）

基于项目文档定义的四阶段路线（Phase 1→2→3→4）和 MVP 的 7 个核心模块，整个项目拆成 16 个 spec，按依赖关系排序。每个 spec 都是可独立交付、可独立验收的最小单元。

---

## Phase 1：Rust 协议主链路（Spec 1 ~ 6）

这是整个项目的地基，必须先完成。

| Spec | 名称 | 核心交付 | 依赖 | 当前状态 |
|------|------|---------|------|---------|
| Spec-01 | rc-crypto 密码学基础库 | HMAC-SHA256 派生、AES-128 CMAC 校验、常量时间比较、Zeroize 安全约束 | 无 | 空壳，需从零实现 |
| Spec-02 | rc-kms 密钥管理服务 | Root→Brand→Chip 三级 KDF 派生链、K_honey 派生、**K_chip_mother 虚拟母卡密钥派生**、软件 KMS 封装 | Spec-01 | 空壳，需从零实现 |
| Spec-03 | rc-api 鉴权与租户上下文 | JWT 解析、**API Key 认证**、actor/role/org/brand 上下文提取、替换当前 header 字符串透传 | 无 | 未实现，当前 Authorization 只是透传 |
| Spec-04 | rc-api 品牌极简注册与外部 SKU 映射 | **POST /brands（3 字段极简注册）**、**API Key 签发**、**资产外部 SKU 映射写入**、品牌隔离校验。~~不再包含 POST /brands/{brandId}/products~~（已废弃产品管理） | Spec-03 | 未实现，当前无品牌写接口 |
| Spec-05 | rc-api 动态认证、验真与分层授权校验 | verify 接口接入真实 CMAC 校验、CTR 防重放检测、UID/CTR/CMAC 验证闭环、**虚拟母卡凭证生成与校验**、**分层授权校验（物理卡 CMAC vs 虚拟卡 Token）**、**过户接口含虚拟母卡自动调用** | Spec-01, Spec-02 | verify 接口存在但只做了 DB 查询，无真实认证 |
| Spec-06 | 数据库 migration 管理与测试夹具 | 引入 sqlx-migrate 或 refinery、改造 init SQL 为版本化 migration、**brands 表极简化**、**assets 表增加外部 SKU 映射字段**、**authority_devices 表（含虚拟/物理类型）**、**asset_entanglements 表**、开发/测试用 fixture | 无 | 当前用 init SQL 直接执行 |

### Phase 1 变更说明

- **Spec-02**：新增虚拟母卡密钥 `K_chip_mother` 的派生逻辑
- **Spec-03**：新增 API Key 认证方式（品牌方 API 对接）
- **Spec-04**：范围大幅缩小——从"品牌与产品管理"改为"品牌极简注册 + 外部 SKU 映射 + API Key 签发"。不再有 Product CRUD
- **Spec-05**：范围扩大——新增虚拟母卡凭证生成、分层授权校验、过户接口
- **Spec-06**：数据库 schema 对齐新设计（brands 极简化、assets 增加外部映射、新增 authority_devices/asset_entanglements）

---

## Phase 2：Go 治理编排层（Spec 7 ~ 11）

在 Rust 协议主链路稳定后，补齐治理编排。

| Spec | 名称 | 核心交付 | 依赖 | 当前状态 |
|------|------|---------|------|---------|
| Spec-07 | go-gateway 统一接入网关 | JWT 校验、**API Key 认证与品牌上下文映射**、trace-id 注入、路由转发到 rc-api 和 Go 服务、限流、统一错误码映射 | Spec-03 | 空壳 healthz |
| Spec-08 | go-iam 身份与组织治理 | 用户/组织/岗位/角色 CRUD、品牌组织结构、治理侧 RBAC、登录与 token 签发、**API Key 管理（签发/轮换/吊销）** | Spec-07 | 空壳 healthz |
| Spec-09 | go-bff 前端聚合接口 | C 端资产列表/详情/验真结果聚合、B 端 dashboard/**品牌资产列表（含外部 SKU 映射）**/工厂任务列表聚合、ViewModel 输出、**Webhook 基础推送** | Spec-07, Spec-04 | 空壳 healthz |
| Spec-10 | go-approval 审批流 | ~~品牌发布审批、策略审批~~、恢复审批、高风险动作审批、审批状态流转 | Spec-08 | **MVP 可选/延后**——品牌方自行审批，系统不承担品牌业务审批。仅保留平台级高风险操作审批（冻结恢复等），可先人工兜底 |
| Spec-11 | go-workorder 工单服务 | 风险工单、争议工单、恢复工单、人工审核结论流转、工单与 rc-api freeze/recover 联动 | Spec-08 | 空壳 healthz |

### Phase 2 变更说明

- **Spec-07**：新增 API Key 认证能力，品牌 API 调用在 Gateway 完成认证后映射为内部上下文
- **Spec-08**：新增 API Key 管理功能
- **Spec-09**：聚合接口对齐新设计（外部 SKU 映射展示、Webhook 推送）
- **Spec-10**：**降级为 MVP 可选**。品牌方自行审批，go-approval 在 MVP 中不是必需服务。仅保留平台级高风险操作审批，可先人工兜底
- **Spec-11**：不变

---

## Phase 3：uni-app 前端双端（Spec 12 ~ 15）

在接口与状态语义稳定后，实现前端。

| Spec | 名称 | 核心交付 | 依赖 | 当前状态 |
|------|------|---------|------|---------|
| Spec-12 | frontend 基础设施与共享包 | uni-app 工程初始化、packages/api（请求封装/token 注入/错误处理）、packages/state（登录态/品牌上下文）、packages/ui（状态徽章/风险卡片/通用组件） | Spec-07, Spec-09 | 空壳 workspace |
| Spec-13 | b-console B 端治理后台 | 登录页、**品牌极简注册页（3 字段）**、**API Key 管理页**、**资产列表页（含外部 SKU 映射）**、批次/会话管理页、盲扫任务页、激活页（**含外部 SKU 映射填写**）、售出确认页、基础审计页。~~不再有产品管理页~~ | Spec-12, Spec-09 | 空壳 |
| Spec-14 | c-app C 端验真与资产馆 | 验真页（扫码/NFC → 验真结果、**显示资产归属者信息**）、基础资产详情页（**含外部 SKU 跳转**）、基础 Vault 持有列表页 | Spec-12, Spec-09 | 空壳 |
| Spec-15 | c-app 过户与终态流程 | 过户发起页（**虚拟母卡 2 步操作：生物识别 + 扫子标签**）、接收方确认页、Consumed/Legacy 荣誉态展示 | Spec-14 | 不存在 |

### Phase 3 变更说明

- **Spec-13**：页面集大幅简化——去掉产品管理页，新增品牌极简注册页和 API Key 管理页，资产列表含外部 SKU 映射
- **Spec-14**：验真结果增加归属者展示，资产详情增加外部 SKU 跳转
- **Spec-15**：过户对齐虚拟母卡 2 步操作流程

---

## Phase 4：投影、运维与增强（Spec 16）

| Spec | 名称 | 核心交付 | 依赖 | 当前状态 |
|------|------|---------|------|---------|
| Spec-16 | Redis 投影与运维增强 | 钱包快照写入/重建、验真热缓存、列表投影、定时校验任务、ops 运维脚本落地（CTR 校准/快照重建/灾难恢复） | Spec-05, Spec-09 | Redis 已在 compose 中但未使用 |

---

## 依赖关系图

```
Spec-01 (rc-crypto)
  └─→ Spec-02 (rc-kms，含虚拟母卡密钥)
  └─→ Spec-05 (动态认证/验真/分层授权校验)

Spec-03 (鉴权上下文，含 API Key 认证)
  └─→ Spec-04 (品牌极简注册/外部 SKU 映射/API Key 签发)
  └─→ Spec-07 (go-gateway，含 API Key 路由)
        └─→ Spec-08 (go-iam，含 API Key 管理)
        │     └─→ Spec-10 (go-approval) ← MVP 可选/延后
        │     └─→ Spec-11 (go-workorder)
        └─→ Spec-09 (go-bff，含 Webhook)
              └─→ Spec-12 (frontend 基础设施)
                    └─→ Spec-13 (b-console 极简版)
                    └─→ Spec-14 (c-app 验真)
                          └─→ Spec-15 (c-app 过户，虚拟母卡)

Spec-06 (migration，含新 schema) ── 独立，可并行

Spec-16 (Redis/运维) ── 最后
```

---

## 推荐执行顺序

可以有两条并行线：

- **主线**：Spec-01 → 02 → 05（密码学 → KMS（含虚拟母卡密钥）→ 验真 + 分层授权校验，硬核安全链路）
- **副线**：Spec-03 → 04 → 06（鉴权（含 API Key）→ 品牌极简注册 + 外部 SKU 映射 → migration，业务基础设施）

两条线汇合后进入 Spec-07 → 08 → 09 → 11（Go 层，**跳过 Spec-10**），再进入 Spec-12 → 13 → 14 → 15（前端），最后 Spec-16 收尾。

**关键变更**：Spec-10（go-approval）在 MVP 阶段可跳过或延后，品牌方自行审批。Spec-11（go-workorder）不依赖 Spec-10 即可独立实现基础工单能力。

---

## Backlog Spec（不进入当前冲刺）

| 方向 | 说明 | 前置条件 |
|------|------|---------|
| 物理母卡支持 | PHYSICAL_NFC 形态的完整支持 | Spec-05 完成 |
| 品牌 SDK | Python / Node.js / Go SDK | Spec-04, Spec-07 完成 |
| 主权分数体系 | Sovereignty Score 计算与展示 | C 端 MVP 验证后 |
| 社交层 | 动态、圈子、关注 | C 端用户量达标后 |
| 资产估值 | 基于市场数据的估值能力 | 需外部数据源 |
| 高级审批流 | go-approval 完整能力 | 多品牌场景需求验证后 |
| Excel 批量导入 | Web 后台批量操作增强 | Spec-13 完成 |
| Webhook 高级配置 | 事件过滤、重试策略 | Spec-09 完成 |


Spec 分拆方案的核心变更：

Spec-04 从"品牌与产品管理"缩小为"品牌极简注册 + 外部 SKU 映射 + API Key 签发"
Spec-05 扩大范围，纳入虚拟母卡凭证生成和分层授权校验
Spec-10（go-approval）降级为 MVP 可选/延后
Spec-02/03/06/07/08/09/13/14/15 均增加对应的新能力描述
新增 Backlog Spec 清单（物理母卡、SDK、主权分数、社交层等）