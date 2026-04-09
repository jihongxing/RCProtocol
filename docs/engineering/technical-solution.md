# 技术方案

> 文档类型：Engineering  
> 状态：Active  
> 权威级别：Execution Baseline

---

## 1. 目标

本文件用于把 RCProtocol 的工程基线进一步落实为可执行技术方案，明确：

- 为什么采用 Rust + Go + uni-app
- 各层分别负责什么
- 服务如何拆分
- 数据如何分工
- 项目目录如何组织
- 按什么顺序启动开发

本文件服从以下权威文档：

- `../foundation/state-machine.md`
- `../foundation/security-model.md`
- `../foundation/roles-and-permissions.md`
- `../foundation/api-and-service-boundaries.md`
- `system-architecture.md`

如本文件与 Foundation 层冲突，以 Foundation 层为准。

---

## 2. 技术选型结论

RCProtocol 采用三段式技术组合：

- **Rust**：密码学、协议核心、状态机、安全校验、KMS 边界
- **Go**：治理编排、审批流、工单流、BFF、报表、后台快速迭代能力
- **uni-app**：C 端与 B 端多端前端交付

### 2.1 Rust 的职责边界

Rust 只负责“不能错、不能漂移、不能绕过”的部分：

- 状态机推进
- 权限最终裁决
- 动态认证
- KDF / 派生逻辑
- 关键安全校验
- 协议级审计事实

### 2.2 Go 的职责边界

Go 负责“需要快速变化、快速交付、业务编排强”的部分：

- 后台聚合接口
- 组织与岗位治理
- 审批流
- 工单流
- 策略中心
- 报表与运营视图
- 文件 / 素材 / 配置类管理

### 2.3 前端的职责边界

前端负责用户交互和多端交付，按使用场景分为两套技术栈：

- **uni-app（c-app）**：C 端验真、资产馆、预览、过户页。面向手机扫码场景，需要跨端能力（H5 / 小程序 / App），uni-app 是合理选择。
- **Vue 3 + Vite + vue-router（b-console）**：B 端治理后台、工厂任务页、审核页。面向桌面浏览器，需要嵌套路由、侧边栏布局、表格密集型交互，标准 Vue SPA 更合适。

前端只负责展示与触发，不定义协议规则。

> **设计决策记录**：b-console 最初设计为 uni-app，后评估发现 uni-app 的页面栈路由、移动端组件生态和 rpx 单位体系不适合桌面端后台管理场景，改为标准 Vue 3 SPA。

---

## 3. 总体架构

```text
[ uni-app Frontend ]
  - C App / H5 / Mini Program
  - B Console / Factory / Moderator

[ Go Service Layer ]
  - Gateway / BFF
  - IAM
  - Approval
  - Workorder
  - Policy
  - Reporting

[ Rust Protocol Layer ]
  - rc-api
  - rc-core
  - rc-crypto
  - rc-kms

[ Data & Infra Layer ]
  - PostgreSQL
  - Redis
  - Object Storage
  - MQ / Job Worker（可选）
```

架构原则：

1. 所有协议写入动作最终进入 Rust
2. Go 层不直接写协议状态真相
3. uni-app 不维护平行状态机
4. Redis 只做投影与缓存，不做真源

---

## 4. Rust 技术方案

### 4.1 Rust crate / service 规划

建议在 `rust/` 目录下拆分：

```text
rust/
├─ rc-common/
├─ rc-core/
├─ rc-crypto/
├─ rc-kms/
└─ rc-api/
```

### 4.2 `rc-common`

职责：

- 公共类型
- DTO
- 错误码
- 枚举定义
- 审计字段结构

### 4.3 `rc-core`

职责：

- 资产生命周期状态机
- 状态转换合法性校验
- 权限最终裁决
- 风险冻结 / 恢复规则
- 协议级领域模型

典型模块：

- `asset_state`
- `transition_validator`
- `permission_checker`
- `risk_decision`
- `ownership_rules`

### 4.4 `rc-crypto`

职责：

- HMAC / CMAC / KDF 封装
- 动态认证消息校验
- 常量时间比较
- 敏感数据零化

建议优先支持：

- `HMAC-SHA256`
- `AES-128 CMAC`
- KDF 派生

### 4.5 `rc-kms`

职责：

- Root / Brand / Chip 级密钥派生
- 软件 KMS 封装
- 与 HSM / Cloud KMS 的后续适配边界

阶段建议：

- Phase 1：软件 KMS
- Phase 2：云 KMS / HSM 接入

### 4.6 `rc-api`

职责：

- 对外提供协议核心 API
- 执行写动作
- 返回正式状态变化结果
- 输出协议级错误

推荐框架：

- `axum`
- `tokio`
- `tower`

原因：

- 类型清晰
- 生态成熟
- 适合高可靠服务边界

---

## 5. Go 技术方案

### 5.1 服务拆分建议

建议在 `services/` 目录下拆分：

```text
services/
├─ go-gateway/
├─ go-bff/
├─ go-iam/
├─ go-approval/
├─ go-workorder/
├─ go-policy/
└─ go-reporting/
```

### 5.2 `go-gateway`

职责：

- 统一接入
- JWT 校验
- trace id 注入
- 路由到 Rust / Go 服务
- 错误码归一化

### 5.3 `go-bff`

职责：

- 面向 uni-app 的聚合接口
- 页面 ViewModel 输出
- 屏蔽多服务细节
- 降低前端请求次数

建议后期可按端拆分：

- `go-bff-c`
- `go-bff-b`

### 5.4 `go-iam`

职责：

- 用户 / 组织 / 岗位 / 角色
- 品牌组织结构
- 后台菜单权限
- 治理侧 RBAC / ABAC

### 5.5 `go-approval`

职责：

- 发布审批
- 策略审批
- 恢复审批
- 高风险动作审批

### 5.6 `go-workorder`

职责：

- 风险工单
- 争议工单
- 恢复工单
- 人工审核结论流转

### 5.7 `go-policy`

职责：

- 地理围栏策略
- 风险阈值策略
- 品牌策略配置
- 策略版本发布

### 5.8 `go-reporting`

职责：

- 看板
- 报表
- 运营统计
- 品牌 / 工厂维度数据分析

---

## 6. 前端方案

### 6.1 前端应用划分

建议在 `frontend/apps/` 下拆两套应用，采用不同技术栈：

```text
frontend/
├─ apps/
│  ├─ c-app/          # uni-app（Vue 3 + Vite），面向 C 端验真
│  └─ b-console/      # Vue 3 + Vite + vue-router，面向 B 端治理后台
└─ packages/
   ├─ api/            # HTTP 请求封装（c-app 用 uni.request，b-console 用 fetch）
   ├─ auth/
   ├─ state/          # 全局状态管理（Vue 3 Composition API）
   ├─ ui/             # 通用 UI 组件（按端分组件实现）
   └─ utils/          # 工具函数与类型定义（跨端共享）
```

### 6.2 技术栈决策

| 应用 | 框架 | 路由 | 构建 | 目标平台 | 原因 |
|------|------|------|------|----------|------|
| c-app | uni-app (Vue 3) | uni-app 页面栈 | @dcloudio/vite-plugin-uni | H5 / 小程序 / App | 验真场景为手机扫码，需跨端能力 |
| b-console | Vue 3 | vue-router | Vite | 桌面浏览器 H5 | 后台管理需要嵌套路由、侧边栏、表格密集交互 |

### 6.3 `c-app`

框架：uni-app (Vue 3 + Vite)

面向：

- 验真用户
- 持有者
- 交易参与方

页面：

- 验真页
- 资产列表
- 资产详情
- 预览页
- 过户确认页
- 荣誉态页面

### 6.4 `b-console`

框架：Vue 3 + Vite + vue-router

面向：

- 平台运营
- 品牌管理员
- 工厂
- 审核员

页面：

- 品牌 / 产品治理
- 批次 / 会话 / 盲扫任务
- 激活页
- 工单审核页
- 风控页
- 报表页

### 6.5 前端设计约束

前端必须遵守：

- 不自定义状态机
- 不用页面逻辑替代权限规则
- 所有关键操作走 BFF / Gateway
- 页面状态来源于后端正式字段

推荐后端统一返回：

- `state`
- `state_label`
- `allowed_actions`
- `risk_flags`
- `display_badges`

---

## 7. 数据与存储方案

### 7.1 PostgreSQL

作为唯一权威存储，建议承载：

- `brands`
- `products`
- `assets`
- `asset_state_history`
- `asset_ownership`
- `asset_transfers`
- `factory_batches`
- `factory_sessions`
- `blind_scan_logs`
- `users`
- `orgs`
- `roles`
- `approvals`
- `workorders`
- `policies`
- `audit_events`
- `idempotency_records`

### 7.2 Redis

作为缓存 / 投影 / 快照层，建议承载：

- 钱包快照
- 热点验真缓存
- 列表查询投影
- 限流计数
- 幂等短期缓存

### 7.3 Object Storage

用于：

- 品牌素材
- 产品图片
- 附件
- 导入导出文件

### 7.4 MQ / Worker

前期可选，不强制。

适合异步化的能力：

- 快照刷新
- 报表统计
- 异步通知
- 投影更新

如前期规模较小，可先用：

- PostgreSQL job table
- Go / Rust worker

---

## 8. 核心链路落地

### 8.1 验真链路

```text
uni-app -> go-gateway / go-bff -> rc-api -> rc-core + rc-crypto -> PostgreSQL / Redis
```

流程：

1. 前端扫码获得动态参数
2. Go 接入层校验基本上下文
3. Rust 执行认证与状态校验
4. Rust 返回正式结果
5. Go 拼装前端展示字段

### 8.2 工厂盲扫链路

```text
factory frontend -> go-bff -> rc-api -> PostgreSQL
```

流程：

1. Go 创建 batch / session
2. 工厂扫描 UID
3. Rust 写入状态推进
4. Go 汇总任务结果

### 8.3 激活链路

```text
brand console -> go-approval / go-policy -> rc-api -> rc-kms -> PostgreSQL
```

流程：

1. 品牌发起激活
2. Go 校验审批与策略
3. Rust 调用 KMS 执行密钥逻辑
4. Rust 推进到 `Activated`
5. 写入审计事实

### 8.4 过户链路

```text
c-app -> go-bff / go-workorder -> rc-api -> PostgreSQL -> Redis refresh
```

流程：

1. 持有者发起转移
2. 接收方确认
3. Rust 校验状态、权限与风险前提
4. 状态写入
5. 快照异步更新

### 8.5 冻结与恢复链路

```text
moderator / risk system -> go-workorder -> rc-api -> PostgreSQL
```

流程：

1. 风险触发冻结
2. Rust 推进 `Disputed`
3. 审核员形成结论
4. 恢复或进入安全终态

---

## 9. 通信与协议建议

### 9.1 外部 API

建议优先使用：

- HTTP REST
- JSON

原因：

- uni-app 集成简单
- 调试成本低
- 前中期足够稳定

### 9.2 内部服务通信

前期建议：

- Go -> Rust 使用 HTTP/JSON

后期在性能或接口规模足够大时，再考虑：

- gRPC

### 9.3 统一请求约束

所有关键写接口统一要求：

- `Authorization`
- `X-Trace-Id`
- `X-Idempotency-Key`
- `X-Approval-Id`（按需）
- `X-Policy-Version`（按需）

---

## 10. 安全与认证方案

### 10.1 身份认证

建议：

- Gateway 层统一接 JWT
- Rust 层执行最终动作判定

JWT 最少字段：

- `sub`
- `role`
- `org_id`
- `brand_id`
- `scopes`

### 10.2 幂等

以下动作必须具备幂等能力：

- 激活
- 售出
- 过户
- 冻结
- 恢复
- 策略应用

### 10.3 审计

关键接口统一记录：

- actor
- role
- trace id
- resource id
- pre-state
- post-state
- result
- approval id

### 10.4 Rust 安全编码要求

- 敏感密钥对象禁止随意复制
- 敏感数据支持零化
- 常量时间比较
- 日志脱敏
- 明确区分业务错误与安全错误

---

## 11. 仓库组织建议

建议采用 monorepo：

```text
RCProtocol/
├─ docs/
├─ frontend/
│  ├─ apps/
│  └─ packages/
├─ services/
│  ├─ go-gateway/
│  ├─ go-bff/
│  ├─ go-iam/
│  ├─ go-approval/
│  ├─ go-workorder/
│  ├─ go-policy/
│  └─ go-reporting/
├─ rust/
│  ├─ rc-common/
│  ├─ rc-core/
│  ├─ rc-crypto/
│  ├─ rc-kms/
│  └─ rc-api/
├─ deploy/
└─ scripts/
```

这样可以保证：

- 文档、前后端、协议内核统一版本管理
- 规则变更可以联动到实现
- 更适合协议型项目长期演进

---

## 12. 开发阶段建议

### Phase 1：先做 Rust 协议主链路

优先完成：

- 状态机
- 权限模型
- 密钥派生
- 验真接口
- 激活接口
- 过户接口
- 冻结 / 恢复接口

### Phase 2：补齐 Go 治理层

完成：

- Gateway
- BFF
- IAM
- Approval
- Workorder
- Policy

### Phase 3：上线 uni-app 双端

完成：

- C 端验真与资产馆
- B 端基础治理后台
- 页面与接口联调

### Phase 4：补齐投影、报表与运维增强

完成：

- Redis 快照
- 报表聚合
- 定时修复任务
- 恢复自动化

---

## 13. 当前结论

RCProtocol 最适合采用：

- **Rust 固化协议与安全真相**
- **Go 承担业务快速迭代与治理编排**
- **uni-app 完成 C 端跨端交付，Vue 3 SPA 完成 B 端桌面治理后台**

这是兼顾安全性、一致性、开发效率和跨端交付效率的最优组合。

---

## 14. 关联文档

- `system-architecture.md`
- `../foundation/api-and-service-boundaries.md`
- `../foundation/security-model.md`
- `../foundation/state-machine.md`
- `../product/product-system.md`
- `../ops/`
