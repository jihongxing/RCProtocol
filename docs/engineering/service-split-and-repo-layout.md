# 服务拆分与仓库布局

> 文档类型：Engineering  
> 状态：Active  
> 权威级别：Execution Guide

---

## 1. 目标

本文件在 `technical-solution.md` 基础上，进一步明确 RCProtocol 的工程落地组织方式，重点回答：

- 服务应该怎么拆
- Rust / Go / uni-app 代码放在哪里
- 各服务之间如何协作
- 哪些模块必须优先建设
- 仓库目录如何保证后续可扩展

如本文件与 Foundation 层定义冲突，以 Foundation 层为准；如与 `system-architecture.md` 冲突，以后者为准。

---

## 2. 总体落位

推荐采用单仓库（monorepo）组织方式：

```text
RCProtocol/
├─ docs/
├─ frontend/
├─ services/
├─ rust/
├─ deploy/
├─ scripts/
└─ tools/
```

原因：

- 协议、业务、前端和文档可以统一版本管理
- 状态机 / API / 页面流程的联动修改更容易收敛
- 适合当前仍在快速演进中的协议型项目

---

## 3. Rust 工程布局

建议目录：

```text
rust/
├─ Cargo.toml
├─ rc-common/
├─ rc-core/
├─ rc-crypto/
├─ rc-kms/
└─ rc-api/
```

### 3.1 `rc-common`

用于沉淀：

- 公共错误码
- 公共类型
- DTO
- 审计字段
- 常量与枚举

### 3.2 `rc-core`

用于沉淀：

- 状态机
- 状态迁移校验
- 角色权限最终裁决
- 风险冻结 / 恢复规则
- 生命周期核心领域对象

### 3.3 `rc-crypto`

用于沉淀：

- KDF
- HMAC / CMAC
- 动态认证消息校验
- 常量时间比较
- 安全工具函数

### 3.4 `rc-kms`

用于沉淀：

- Root / Brand / Chip 级密钥派生
- 软件 KMS 实现
- 云 KMS / HSM 适配边界

### 3.5 `rc-api`

作为可部署服务，负责：

- 对外提供协议 API
- 调用 `rc-core`
- 调用 `rc-kms`
- 输出正式状态变更与错误语义

---

## 4. Go 工程布局

建议目录：

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

### 4.1 `go-gateway`

职责：

- 统一入口
- JWT 校验
- 请求透传
- trace id 注入
- 统一错误码映射
- 限流与基础防护

### 4.2 `go-bff`

职责：

- 面向 uni-app 的聚合接口
- 输出页面友好的 ViewModel
- 聚合 Rust Core 与多个 Go 服务数据

### 4.3 `go-iam`

职责：

- 用户、组织、岗位、角色管理
- 治理侧权限模型
- 品牌 / 工厂 / 平台主体关系管理

### 4.4 `go-approval`

职责：

- 发布审批
- 策略审批
- 恢复审批
- 高风险动作审批

### 4.5 `go-workorder`

职责：

- 风险工单
- 争议工单
- 恢复工单
- 人工审核流转

### 4.6 `go-policy`

职责：

- 地理围栏策略
- 风险阈值策略
- 频率限制策略
- 品牌侧策略版本维护

### 4.7 `go-reporting`

职责：

- 看板
- 报表
- 运营统计
- 数据聚合导出

---

## 5. uni-app 工程布局

建议目录：

```text
frontend/
├─ apps/
│  ├─ c-app/
│  └─ b-console/
└─ packages/
   ├─ api/
   ├─ auth/
   ├─ state/
   ├─ ui/
   └─ utils/
```

### 5.1 `c-app`

面向：

- 验真用户
- 合法持有者
- 交易参与方

核心页面：

- 验真页
- 资产列表
- 资产详情
- 预览页
- 过户确认页
- 荣誉态页面

### 5.2 `b-console`

面向：

- 平台运营
- 品牌管理员
- 工厂人员
- 审核员

核心页面：

- 品牌 / 产品治理
- 批次与会话
- 盲扫任务
- 激活页
- 工单审核页
- 风控页
- 报表页

### 5.3 `packages/api`

统一封装：

- Gateway / BFF 请求
- Token 注入
- 错误码处理
- 文件上传下载

### 5.4 `packages/state`

统一管理：

- 登录态
- 当前品牌 / 当前组织
- 当前资产 / 当前工单
- 页面可用动作

### 5.5 `packages/ui`

统一封装：

- 资产状态徽章
- 验真结果卡片
- 风险提示卡
- 工单状态组件
- 通用表单与列表部件

---

## 6. 服务依赖关系

建议依赖方向固定如下：

```text
uni-app -> go-gateway / go-bff
                |
                +-> go-iam / go-approval / go-workorder / go-policy / go-reporting
                |
                +-> rc-api -> rc-core / rc-kms / rc-crypto
```

约束：

- uni-app 不直接访问 Rust 内部模块
- Go 服务不直接写协议状态表
- 只有 `rc-api` 能落协议核心写动作

---

## 7. 共享规范目录建议

建议新增共享目录以控制跨服务一致性：

```text
tools/
├─ schemas/
├─ api-contracts/
└─ codegen/
```

用途：

- OpenAPI / JSON Schema
- 前后端共享契约
- DTO 或 SDK 生成脚本

如果前期不想引入代码生成，也应至少统一维护：

- 错误码表
- 资源字段表
- 状态枚举表

---

## 8. 首批必须创建的模块

为了尽快进入可开发状态，首批建议只创建以下模块：

### 8.1 Rust

- `rc-common`
- `rc-core`
- `rc-crypto`
- `rc-kms`
- `rc-api`

### 8.2 Go

- `go-gateway`
- `go-bff`
- `go-iam`
- `go-approval`
- `go-workorder`

### 8.3 Frontend

- `frontend/apps/c-app`
- `frontend/apps/b-console`
- `frontend/packages/api`
- `frontend/packages/state`

### 8.4 Infra

- PostgreSQL
- Redis
- 基础对象存储

`go-policy`、`go-reporting`、MQ、代码生成等能力可在第二阶段补齐。

---

## 9. 首批接口归属建议

### 9.1 Rust `rc-api`

建议先承载：

- `POST /brands`
- `POST /brands/{brandId}/products`
- `POST /factory/batches`
- `POST /factory/sessions`
- `POST /factory/blind-log`
- `POST /assets/{assetId}/activate`
- `POST /assets/{assetId}/legal-sell`
- `POST /assets/{assetId}/transfer`
- `POST /assets/{assetId}/freeze`
- `POST /assets/{assetId}/recover`
- `GET /verify`

### 9.2 Go `go-bff`

建议先承载：

- `GET /app/assets`
- `GET /app/assets/{assetId}`
- `GET /app/workorders/{id}`
- `GET /console/dashboard`
- `GET /console/brands/{brandId}/products`
- `GET /console/factory/tasks`

### 9.3 Go `go-approval` / `go-workorder`

建议先承载：

- `POST /ops/brands/{brandId}/publish`
- `POST /ops/workorders/{workorderId}/advance`
- `POST /ops/recovery/{assetId}/submit`

---

## 10. 分阶段落地顺序

### Phase 1：协议主链路

必须先完成：

- Rust 核心模块
- PostgreSQL
- 基础验真链路
- 盲扫登记
- 激活
- 冻结 / 恢复

### Phase 2：治理编排

补齐：

- Gateway
- IAM
- Approval
- Workorder
- BFF

### Phase 3：uni-app 页面上线

补齐：

- C 端验真和资产馆
- B 端批次 / 工单 / 激活页
- 页面动作与正式状态对齐

### Phase 4：投影与报表

补齐：

- Redis 快照
- Reporting
- Policy
- 异步投影更新

---

## 11. 当前结论

当前最稳妥的工程落地方式是：

- 用 `rust/` 固化协议和安全内核
- 用 `services/` 承载 Go 业务编排与快速迭代
- 用 `frontend/` 承载 uni-app 多端应用
- 用 `docs/engineering/` 持续维护工程与实现说明

这样既能保证协议层强约束，又能保留业务层足够快的迭代速度。

---

## 12. 关联文档

- `system-architecture.md`
- `technical-solution.md`
- `../foundation/api-and-service-boundaries.md`
- `../product/product-system.md`
