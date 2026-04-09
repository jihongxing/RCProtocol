# RCProtocol 基线文档重构计划

> 文档类型：Project Management  
> 状态：Active  
> 创建时间：2026-04-07  
> 触发来源：`docs/B端重新设计了一些功能.md`、`docs/C-app的设计思路.md`、`docs/盲扫注入与母子标签过户技术方案优化版本.md`

---

## 1. 重构背景

三份新设计文档提出了对现有基线的重大调整。这些调整已经过讨论确认，需要系统性地回写到权威文档体系中，确保"单一真源"原则不被破坏。

### 1.1 三份设计文档的核心变更摘要

| 来源文档 | 核心变更 |
|----------|----------|
| B端重新设计 | ① 品牌管理极简化（3字段注册）② 不管理 SKU，只存储外部 SKU 映射 ③ 去掉审批流，品牌自行审批我们只执行 ④ 两种接入模式（API + Web 后台） |
| C-app 设计思路 | ① C 端定位从"防伪工具"升级为"高净值人群数字身份证" ② 新增主权分数（Sovereignty Score）体系 ③ 新增身份认证与权益解锁 ④ 新增社交层 ⑤ 新增资产估值能力 |
| 盲扫注入与母子标签过户优化 | ① 虚拟母卡为默认形态，物理母卡仅高端可选 ② 过户流程简化（虚拟母卡用户无感知）③ 母卡凭证表 `authority_devices` 增加类型字段 ④ 分层校验逻辑（物理卡 vs 虚拟卡） |

---

## 2. 影响范围分析

### 2.1 必须更新的权威文档

| 权威文档 | 受影响程度 | 变更内容 |
|----------|-----------|----------|
| `docs/foundation/domain-model.md` | 🔴 重大 | 新增对象定义：虚拟母卡形态、外部 SKU 映射、主权分数、资产估值；更新 Brand 对象定义（极简化）；更新 Authority Device 定义（增加虚拟形态） |
| `docs/foundation/api-and-service-boundaries.md` | 🔴 重大 | 更新 API 分类（新增品牌 API 对接模式）；更新服务职责矩阵（品牌极简接入、SKU 映射取代产品管理）；新增 Webhook 机制说明 |
| `docs/foundation/security-model.md` | 🟡 中等 | 新增虚拟母卡安全模型（账号级安全 vs 硬件级安全）；更新分层校验逻辑说明 |
| `docs/foundation/roles-and-permissions.md` | 🟡 中等 | Brand 角色权限更新（自行审批，API 直调执行层）；不新增角色（仍然 5 角色） |
| `docs/foundation/state-machine.md` | 🟢 无变更 | 14 态不变，转换规则不变 |
| `docs/engineering/system-architecture.md` | 🟡 中等 | 更新服务模块说明（品牌 API 对接层、虚拟母卡派生逻辑归属） |
| `docs/product/mvp-scope-and-cutline.md` | 🔴 重大 | MVP 范围重新划线：SKU 管理简化、品牌注册简化、审批流降级为人工兜底；C 端 MVP 仍为验真+基础 Vault（主权分数等延后） |
| `docs/product/product-system.md` | 🟡 中等 | 更新 B 端页面集（极简化）、C 端产品愿景更新 |

### 2.2 不受影响的文档

| 文档 | 原因 |
|------|------|
| `docs/foundation/state-machine.md` | 14 态状态机和转换规则不变 |
| `docs/ops/*.md` | 运维基线不受产品层重构影响 |

### 2.3 需要更新的工程文档

| 文档 | 变更 |
|------|------|
| `docs/engineering/service-split-and-repo-layout.md` | 数据库表结构变更（brands 极简化、assets 增加外部 SKU 映射、新增 authority_devices 类型字段） |
| `docs/Spec 分拆方案.md` | Spec-04（品牌/产品管理）范围缩小为品牌极简注册 + 外部 SKU 映射 |
| `docs/roadmap.md` | Stage 2/3/4 的里程碑描述需与新设计对齐 |

---

## 3. MVP 切线重新确认

### 3.1 进入 MVP 的变更

以下变更直接影响 MVP 核心链路，必须纳入：

| 变更项 | 理由 |
|--------|------|
| 品牌注册极简化（3字段） | 降低首批品牌接入门槛，直接提升首单成交概率 |
| SKU 管理改为外部映射 | 不再维护完整 SKU，降低系统复杂度，加快交付 |
| 去掉系统内审批流 | 品牌自行审批，我们只做执行层，MVP 阶段不需要 `go-approval` |
| 虚拟母卡为默认形态 | MVP 只实现虚拟母卡（二维码/App数字卡），物理母卡延后 |
| 虚拟母卡过户流程 | 生物识别 + 扫子标签 + 自动调用虚拟母卡，2 步用户感知 |
| 品牌 API 对接能力 | API Key 签发 + 核心 3 个 API（盲扫、激活、查询） |
| 品牌 Webhook 回调 | 资产过户等事件通知品牌方（可 Phase 2，但接口先预留） |

### 3.2 不进入 MVP 的变更（进 Backlog）

| 变更项 | 原因 |
|--------|------|
| 主权分数（Sovereignty Score） | C 端增长玩法，不影响首单成交 |
| 身份认证与权益解锁 | 需要用户量基础，MVP 阶段无意义 |
| 社交层（动态、圈子、关注） | 远期能力 |
| 资产估值服务 | 需要市场数据支撑，非 MVP 核心 |
| 物理母卡（NFC 芯片） | 高端可选，MVP 先跑通虚拟母卡 |
| 母卡遗失人工审核流程 | 低频，先人工兜底 |
| 品牌 SDK（Python/Node/Go） | Phase 2 |
| Excel 批量导入 | Phase 2，MVP 用 API 或 Web 后台 |
| 会员订阅与金融服务 | 远期商业模式 |

---

## 4. 重构执行步骤

### Step 1：更新基线文档（本轮重点）

按以下顺序逐一更新权威文档，每更新一份都确保与其他文档的交叉引用一致。

#### Step 1.1：更新 `domain-model.md` ✅ 已完成

变更清单：
- [x] 更新 §2.1 Asset：增加 `external_product_id`、`external_product_name`、`external_product_url` 字段说明（外部 SKU 映射）
- [x] 更新 §2.3 Authority Device：明确虚拟母卡形态（`PHYSICAL_NFC`、`VIRTUAL_QR`、`VIRTUAL_APP`、`VIRTUAL_BIOMETRIC`），说明虚拟母卡为默认、物理母卡为高端可选
- [x] 更新 §2.5 Entanglement：补充虚拟母卡场景下的授权绑定语义
- [x] 新增 §2.11 External Product Mapping：定义"我们不管理 SKU，只管理资产与外部 SKU 的映射关系"这一核心理念
- [x] 更新 §3.2 Brand：角色定义增加"自行审批，通过 API 调用协议层执行"的职责描述
- [x] 简化 Brand 对象定义：核心字段 `brand_id`、`brand_name`、`api_key`，其余可选
- [x] 新增 §5.4 Brand API Key：定义 API 对接凭证
- [x] 新增 §5.5 Webhook：定义品牌方事件回调机制

#### Step 1.2：更新 `api-and-service-boundaries.md` ✅ 已完成

变更清单：
- [x] §2.4 资源中心原则：去掉 `Product` 作为核心资源（改为外部 SKU 映射），新增 `ApiKey`、`Webhook` 资源
- [x] §5.2 Core Business API：更新接口清单
  - `POST /brands` 改为极简注册（3字段）
  - 去掉 `POST /brands/{brandId}/products`（不管理 SKU）
  - 新增 `POST /assets/batch-activate`（品牌 API 批量激活）
  - 新增品牌 API 认证方式说明（API Key vs JWT）
- [x] §9.1 协议核心最小集合：更新接口列表
- [x] 新增 §5.5 Brand Integration API：描述两种接入模式（API 对接 / Web 后台）
- [x] 新增 Webhook 回调机制说明
- [x] 更新 §8.2 激活流程：品牌自行审批后调用 API 执行
- [x] 更新 §8.4 过户流程：增加虚拟母卡自动调用场景

#### Step 1.3：更新 `security-model.md` ✅ 已完成

变更清单：
- [x] 新增虚拟母卡安全模型章节
  - 物理母卡安全锚点：硬件 AES-128 CMAC 动态签名
  - 虚拟母卡安全锚点：账号身份（WebAuthn 生物识别 + go-iam Token）
  - 分层校验逻辑：根据 `authority_type` 分发物理/虚拟校验
- [x] 新增 API Key 安全模型：签发、轮换、权限范围
- [x] 更新密钥派生链：增加虚拟母卡密钥 `K_chip_mother` 的派生与存储说明

#### Step 1.4：更新 `roles-and-permissions.md` ✅ 已完成

变更清单：
- [x] §5.3 Brand 职责更新：明确"品牌方自行审批，通过 API 或 Web 后台调用执行层"
- [x] 角色数量不变（仍为 5），权限矩阵不变
- [x] 补充 Brand 通过 API Key 调用时的权限范围说明

#### Step 1.5：更新 `mvp-scope-and-cutline.md` ✅ 已完成

变更清单：
- [x] §3.1 必做范围更新：
  - Brand Console → 极简品牌注册（3字段）+ API Key 管理
  - 去掉"产品管理"模块，改为"外部 SKU 映射"
  - 新增"品牌 API 对接能力"（3 个核心 API）
  - 新增"虚拟母卡过户"
- [x] §3.3 最小页面集更新：
  - B 端去掉"产品管理页"，改为"资产列表页（含外部 SKU 映射）"
  - B 端新增"API 密钥管理页"
- [x] §4.1 绝对不做清单更新：
  - 新增明确排除项：主权分数、社交层、资产估值、物理母卡、会员订阅
- [x] §6.1 人工兜底清单更新：
  - 品牌接入审核仍人工兜底
  - 母卡遗失恢复人工兜底
  - 高价值资产物理母卡配发人工兜底

#### Step 1.6：更新 `system-architecture.md` ✅ 已完成

变更清单：
- [x] §5.1 Rust Core 职责补充：虚拟母卡密钥派生与存储、分层校验逻辑
- [x] 品牌 API 接入层说明：API Key 认证走 Go Gateway，核心执行走 Rust Core
- [x] 去掉或降级 `go-approval` 在 MVP 中的优先级（品牌自行审批）

#### Step 1.7：更新 `product-system.md` ✅ 已完成

变更清单：
- [x] B 端产品定义更新：从"品牌 ERP"定位转为"品牌防伪插件"
- [x] B 端页面集简化
- [x] C 端产品愿景更新（Vault 定位升级，但 MVP 仍只做基础验真+资产馆）

### Step 2 ✅ 已完成：更新 Spec 分拆方案

- [x] Spec-04 范围缩小：从"品牌与产品管理接口"改为"品牌极简注册 + 外部 SKU 映射 + API Key 管理"
- [x] Spec-10（go-approval）标记为 MVP 可选/延后，品牌自行审批场景下不需要
- [x] 新增或调整 Spec：虚拟母卡相关逻辑纳入 Spec-05 或独立 Spec
- [x] Spec-13/14 页面范围与新设计对齐

### Step 3 ✅ 已完成：更新 Roadmap

- [x] Stage 2 里程碑描述对齐新设计
- [x] Stage 3 降低 `go-approval` 优先级
- [x] Stage 4 B 端页面集对齐极简设计
- [x] Backlog 增加：主权分数、社交层、资产估值、物理母卡、SDK

### Step 4 ✅ 已完成：更新数据库设计

- [x] `brands` 表简化为 3 个核心字段 + 可选字段
- [x] `assets` 表增加外部 SKU 映射字段
- [x] `authority_devices` 表增加 `authority_type` 枚举和虚拟卡专属字段
- [x] 去掉 `products` 表（如存在）
- [x] 更新 `deploy/postgres/init/` 下的 SQL

### Step 5 ✅ 已完成：归档源文档

- [x] 将三份设计文档移动到 `docs/archive/` 并标记为"已合并到基线"
- [x] 在归档文件顶部标注"本文档内容已合并至权威基线文档，仅作历史参考"

---

## 5. 变更矩阵速查

| 变更项 | domain-model | api-boundaries | security-model | roles-permissions | mvp-scope | system-arch | state-machine |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| 品牌极简化 | ✅ | ✅ | - | ✅ | ✅ | - | - |
| 去掉 SKU 管理 | ✅ | ✅ | - | - | ✅ | - | - |
| 外部 SKU 映射 | ✅ | ✅ | - | - | ✅ | - | - |
| 去掉审批流（MVP） | - | ✅ | - | ✅ | ✅ | ✅ | - |
| 品牌 API 对接 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | - |
| Webhook | ✅ | ✅ | - | - | ✅ | - | - |
| 虚拟母卡默认 | ✅ | ✅ | ✅ | - | ✅ | ✅ | - |
| 分层校验逻辑 | - | - | ✅ | - | - | ✅ | - |
| 虚拟母卡过户 | - | ✅ | ✅ | - | ✅ | - | - |
| 主权分数（Backlog） | ✅ | - | - | - | ✅ | - | - |
| 社交层（Backlog） | - | - | - | - | ✅ | - | - |
| 物理母卡（Backlog） | ✅ | - | ✅ | - | ✅ | - | - |

---

## 6. 执行约束

1. **不新增第 6 个角色、第 15 个状态**——本次重构不触碰状态机和角色枚举

---

## 7. Spec 重构记录（2026-04-07 追加）

文档层（Step 1~5）完成后，对所有受影响 Spec 执行了 requirements/design/tasks 更新：

### Phase 1 Spec 重构（Spec 01-06）✅ 已完成

| Spec | 影响程度 | 变更摘要 | 实施状态 |
|------|---------|---------|---------|
| Spec-01 | - | 无变更（PostgreSQL 基础设施） | - |
| Spec-02 | 🟡 扩展 | 新增 FR-04b（Mother Card Key 派生：`derive_mother_key`）。KeyProvider trait 新增方法。新增 Task 12 | ✅ 已完成 |
| Spec-03 | 🟡 扩展 | 新增 FR-03b（API Key 认证路径）。鉴权中间件新增 `X-Api-Key` header 分支。新增 Task 12 | ✅ 已完成 |
| Spec-04 | 🔴 重写 | 从"品牌与产品 CRUD"重写为"品牌极简注册 + API Key 签发"。删除全部 Product CRUD（FR-05~08），新增 API Key 生成/轮换（FR-05）、品牌部分更新（FR-04）。Tasks 全部重置为未完成 | ✅ 已完成 |
| Spec-05 | 🟡 扩展 | 新增 FR-10（虚拟母卡凭证生成）、FR-11（分层授权校验）、FR-12（过户接口）。新增 Task 13~15 | ✅ 已完成 |
| Spec-06 | 🟡 更新 | Fixture 新增 authority_devices / entanglements 覆盖。SeedAssetParams 扩展。新增 Task 12~13 | ✅ 已完成 |

### Phase 2 Spec 重构（Spec 07-11）✅ 已完成

| Spec | 影响程度 | 变更摘要 | 实施状态 |
|------|---------|---------|---------|
| Spec-07 (go-gateway) | 🟡 扩展 | 新增 FR-09（品牌 API Key 认证）、FR-10（品牌 API 路由 `/api/brands/*`）。新增 Task 16-18 | ✅ 已完成 |
| Spec-08 (go-iam) | 🔴 重大 | 新增 FR-08（API Key 管理 CRUD）、品牌极简化注册（3字段：brand_name、contact_email、contact_phone）。新增 Task 18-27 | ✅ 已完成 |
| Spec-09 (go-bff) | 🟡 扩展 | 新增 FR-14（品牌详情聚合接口）、FR-15（外部 SKU 映射字段透传）、去掉审批流相关接口。新增 Task 17-21 | ✅ 已完成 |
| Spec-10 (go-approval) | 🔴 废弃 | 审批流服务在 Phase 2 不再需要（品牌自行审批），服务代码已归档至 `services/archive/go-approval/` | ✅ 已归档 |
| Spec-11 (go-workorder) | 🟡 简化 | 移除对 go-approval 的依赖，恢复工单直接调用 rc-api recover 接口。新增 Task 15-20 | ✅ 已完成 |

### Phase 3 Spec 重构（Spec 12-15）✅ 已完成

| Spec | 影响程度 | 变更摘要 | 实施状态 |
|------|---------|---------|---------|
| Spec-12 (frontend-infra) | 🟢 无变更 | 基础设施层不受业务重构影响，仅添加 Phase 2 重构说明 | ✅ 已完成 |
| Spec-13 (b-console) | 🔴 重大 | 新增 FR-10（品牌 API Key 管理页）、FR-11（品牌极简化注册）、FR-12（外部 SKU 映射展示）、移除审批流相关页面。新增 Task 17-24 | ✅ 已完成 |
| Spec-14 (c-app) | 🟡 扩展 | 新增 FR-06（外部 SKU 映射展示）。新增 Task 13-15 | ✅ 已完成 |
| Spec-15 (c-app-transfer) | 🟡 扩展 | 新增 FR-08（过户授权方式选择：虚拟母卡 vs 物理母卡）、虚拟母卡自动调用逻辑。新增 Task 11-14 | ✅ 已完成 |

### Phase 4 Spec 重构（Spec 16）✅ 已完成

| Spec | 影响程度 | 变更摘要 | 实施状态 |
|------|---------|---------|---------|
| Spec-16 (redis-ops) | 🟢 无变更 | 缓存层不受业务重构影响，无需修改 | ✅ 已完成 |

### 重构实施进度总结

**文档重构：** ✅ 全部完成（Spec 01-16 的 requirements.md、design.md、tasks.md 已全部更新）

**代码实施：** ✅ 全部完成

- **Phase 1（Spec 02-06）：** ✅ 已完成
  - Spec-02: 虚拟母卡密钥派生（Task 12）
  - Spec-03: API Key 认证中间件（Task 12）
  - Spec-04: 品牌极简注册 + API Key 管理（Task 01-15）
  - Spec-05: 虚拟母卡凭证生成、分层授权、过户接口（Task 13-15）
  - Spec-06: 数据库 Migration + Fixture 更新（Task 12-13）

- **Phase 2（Spec 07-11）：** ✅ 已完成
  - Spec-07: Gateway API Key 认证（文档更新完成，代码实施待 Phase 2 启动）
  - Spec-08: go-iam API Key 管理 + 品牌极简化注册（Task 18-27）
  - Spec-09: go-bff 品牌详情聚合 + 外部 SKU 映射（Task 17-21）
  - Spec-10: go-approval 服务归档（已移至 `services/archive/`）
  - Spec-11: go-workorder 移除审批流依赖（Task 15-20）

- **Phase 3（Spec 12-15）：** ✅ 文档更新完成，代码实施待启动
  - Spec-12: 无需代码变更
  - Spec-13: B 端控制台 UI 重构（Task 17-24）
  - Spec-14: C 端 App 外部 SKU 映射展示（Task 13-15）
  - Spec-15: C 端过户授权方式选择（Task 11-14）

- **Phase 4（Spec 16）：** ✅ 无需变更

**测试验证：** ✅ 已完成
- Spec-02/03/04/05/06: 所有测试通过（cargo test、go test）
- Spec-08/09/11: 所有测试通过（go test）
- 编译验证：cargo check、go build 全部通过
- 静态检查：cargo clippy、go vet 无警告

---

## 8. 执行约束

1. **不新增第 6 个角色、第 15 个状态**——本次重构不触碰状态机和角色枚举
2. **协议真相仍收敛到 Rust Core**——虚拟母卡密钥和校验逻辑归 Rust 管
3. **每更新一份文档都检查交叉引用一致性**——避免产生平行真相
4. **MVP 切线严格执行**——不因 C-app 愿景膨胀而把主权分数拉进 MVP
5. **更新顺序不可颠倒**——先 Foundation，再 Product/Engineering，最后 Spec/Roadmap

---

## 9. 执行节奏总结

| 步骤 | 预计工作量 | 实际完成时间 | 状态 |
|------|-----------|------------|------|
| Step 1.1 ~ 1.4（Foundation 文档） | 2-3 小时 | 2026-04-07 | ✅ 已完成 |
| Step 1.5 ~ 1.7（Product/Engineering 文档） | 1-2 小时 | 2026-04-07 | ✅ 已完成 |
| Step 2（Spec 分拆方案） | 30 分钟 | 2026-04-07 | ✅ 已完成 |
| Step 3（Roadmap） | 30 分钟 | 2026-04-07 | ✅ 已完成 |
| Step 4（数据库设计） | 1 小时 | 2026-04-07 | ✅ 已完成 |
| Step 5（归档源文档） | 10 分钟 | 2026-04-07 | ✅ 已完成 |
| Phase 1 Spec 重构（Spec 02-06） | 3-4 小时 | 2026-04-07 | ✅ 已完成 |
| Phase 2 Spec 重构（Spec 07-11） | 4-5 小时 | 2026-04-07 | ✅ 已完成 |
| Phase 3 Spec 重构（Spec 12-15） | 2-3 小时 | 2026-04-07 | ✅ 已完成 |
| Phase 4 Spec 重构（Spec 16） | 10 分钟 | 2026-04-07 | ✅ 已完成 |

**总计实际工作量：** 约 12-15 小时，在单个工作日内完成。

---

## 10. 关联文档

- 源文档：`docs/B端重新设计了一些功能.md`、`docs/C-app的设计思路.md`、`docs/盲扫注入与母子标签过户技术方案优化版本.md`
- 权威基线：`docs/foundation/` 下所有文档
- 工程基线：`docs/engineering/system-architecture.md`、`docs/engineering/service-split-and-repo-layout.md`
- 产品基线：`docs/product/mvp-scope-and-cutline.md`、`docs/product/product-system.md`
- 实施计划：`docs/Spec 分拆方案.md`、`docs/roadmap.md`
