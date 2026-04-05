# 服务与 API 边界

> 文档类型：Foundation  
> 状态：Active  
> 权威级别：Authoritative

---

## 1. 权威说明

本文件定义 RCProtocol 的服务边界、职责分工、调用约束、接口分类与执行规则。

凡涉及以下问题，均以本文件为准：

- 某项能力由哪个服务负责
- 哪些动作属于协议核心真源
- 治理层如何调用协议层
- 写操作必须满足哪些前置条件
- API 如何按资源与流程分类

---

## 2. 总体边界原则

### 2.1 协议核心是真源

以下能力只能由协议核心负责：

- 状态机推进
- 权限校验
- 密钥派生
- 动态认证
- 冻结 / 恢复 / 终态推进
- 协议级审计事实记录

### 2.2 治理层不承载主权真相

治理层可以承担：

- 草稿
- 审批
- 组织岗位
- 策略编排
- 工单推进
- 聚合查询
- 报表视图

但治理层不得：

- 直接写协议核心表替代状态流转
- 绕过协议核心推进资产状态
- 复制一套独立的状态真相

### 2.3 读写分离原则

- **查询**：允许聚合、缓存、投影
- **写入**：必须收敛到协议核心
- **审计**：关键写入必须带审计上下文

### 2.4 资源中心原则

接口设计以资源为中心，而不是以前端页面为中心。

核心资源包括：

- Brand
- Product
- Asset
- Batch
- Session
- Delegation
- Workorder
- Wallet Snapshot

---

## 3. 服务划分

## 3.1 Rust 协议核心

Rust 协议核心承担：

- 状态机校验与推进
- 权限矩阵校验
- 认证消息校验
- 安全判定
- 资产与品牌核心写操作

它是唯一的协议执行真源。

## 3.2 Rust KMS

Rust KMS 承担：

- Root / Brand / Chip 级密钥派生
- 安全敏感认证支持
- 安全边界内密钥运算
- 风险相关密钥辅助能力

KMS 不负责页面逻辑、审批逻辑和报表逻辑。

## 3.3 Go Gateway / BFF

Go Gateway / BFF 承担：

- 前端接入聚合
- 会话透传
- 跨服务响应拼装
- 错误标准化映射

它不负责协议规则判定。

## 3.4 Go IAM

Go IAM 承担：

- 组织结构
- 岗位模板
- RBAC / ABAC 治理策略
- 治理身份映射

它负责“治理侧谁能申请什么”，不负责“协议侧最终是否能执行”。

## 3.5 Go Approval / Policy / Workorder

这些服务承担：

- 审批流
- 策略版本管理
- 工单编排
- 人工决策上下文

它们只能组织动作，不直接产生协议事实。

## 3.6 Reporting / Audit Projection

这些服务承担：

- 审计检索
- 报表聚合
- 投影查询
- 看板汇总

它们可以读真源、建投影，但不能反向覆盖真源。

---

## 4. 服务职责矩阵

| 能力 | Rust Core | Rust KMS | Go Governance | 备注 |
|------|-----------|----------|---------------|------|
| 状态转换合法性 | 是 | 否 | 否 | 唯一真源 |
| 角色权限最终裁决 | 是 | 否 | 否 | Go 可预检查 |
| 品牌 / 产品发布生效 | 是 | 否 | 编排 | Go 先审批 |
| 盲扫登记 | 是 | 可辅助 | 编排 | Factory 流程 |
| 激活推进 | 是 | 是 | 编排 | 包含密钥相关操作 |
| 委托授权治理 | 核心校验 | 可辅助 | 编排 | 双层权限 |
| 风险冻结 / 恢复 | 是 | 可辅助 | 工单编排 | 审核流进入 Core |
| 报表 / 看板 | 否 | 否 | 是 | 来源于真源 |

---

## 5. API 分类

### 5.1 Public Verification API

用途：

- 验真
- 基础公开信息查询

特点：

- 尽量只读
- 限流优先
- 无需暴露治理信息

### 5.2 Core Business API

用途：

- 品牌注册
- 产品管理
- 盲扫登记
- 激活
- 销售合法化
- 过户
- 冻结 / 恢复

特点：

- 必须经过协议核心
- 有严格状态校验
- 需要角色与边界判定

### 5.3 Governance API

用途：

- 审批流
- 工单推进
- 组织与角色治理
- 策略版本维护

特点：

- 不直接写协议真相
- 必要时调用 Core Business API 落地

### 5.4 Admin / Ops API

用途：

- 快照重建
- 校验修复
- 只读诊断
- 运维恢复

特点：

- 权限要求更高
- 通常只对内部开放

---

## 6. 写操作统一约束

所有关键写操作必须满足以下约束。

## 6.1 请求头

必传头：

- `Authorization`
- `X-Trace-Id`
- `X-Idempotency-Key`

按需传递：

- `X-Approval-Id`
- `X-Policy-Version`
- `X-Actor-Org`

## 6.2 幂等规则

- 相同幂等键 + 相同请求体：返回同一结果
- 相同幂等键 + 不同请求体：返回冲突
- 幂等记录至少覆盖关键业务写操作

## 6.3 审批规则

以下动作原则上需要审批上下文：

- 品牌正式发布
- 策略应用
- 委托签发 / 撤销
- 风险恢复
- 平台级覆盖操作

## 6.4 审计规则

关键写操作必须记录：

- actor
- role
- resource id
- pre-state
- post-state
- trace id
- idempotency key
- approval id（如有）
- result

---

## 7. 错误语义

治理层允许做用户友好的错误映射，但不得抹平协议核心语义。

建议标准映射：

| 协议核心结果 | 治理层标准化 | 含义 |
|--------------|--------------|------|
| `400` | `INVALID_INPUT` | 输入非法 |
| `401` | `AUTH_REQUIRED` | 未认证 |
| `403` | `FORBIDDEN` | 无权执行 |
| `404` | `NOT_FOUND` | 资源不存在 |
| `409` | `CONFLICT` | 幂等冲突或状态竞争 |
| `422` | `UNPROCESSABLE` | 语义不满足 |
| `5xx` | `UPSTREAM_FAILURE` | 上游失败 |

---

## 8. 典型业务流程边界

## 8.1 品牌与产品发布

治理层负责：

1. 创建草稿
2. 发起审批
3. 记录版本
4. 调用协议核心发布

协议核心负责：

1. 品牌注册 / 校验
2. 产品写入 / 更新
3. 生效写入
4. 返回正式结果

## 8.2 工厂盲扫与激活

治理层负责：

1. 创建批次与会话
2. 分发任务
3. 汇总执行结果

协议核心负责：

1. 盲扫登记
2. 状态推进为 `FactoryLogged` / `Unassigned`
3. 激活阶段推进到 `RotatingKeys` / `EntangledPending` / `Activated`
4. 执行认证与密钥约束

## 8.3 销售合法化

治理层负责：

- 销售事件接入
- 业务审批或对账

协议核心负责：

- 校验当前是否可从 `Activated` 到 `LegallySold`
- 更新所有权相关事实
- 生成审计记录

## 8.4 过户

治理层负责：

- 发起工单或交易流程
- 组织双方确认
- 承载支付外部上下文

协议核心负责：

- 校验角色
- 校验状态
- 校验绑定与安全前提
- 将状态推进到 `Transferred`

## 8.5 冻结 / 恢复 / 安全终态

治理层负责：

- 风险工单
- 审核分派
- 审批与人工结论

协议核心负责：

- 推进到 `Disputed`
- 恢复到前状态
- 或推进到 `Tampered` / `Compromised`

---

## 9. 最小开发接口集合

为了便于落地，当前建议优先保障以下最小接口集合：

### 9.1 协议核心最小集合

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

### 9.2 治理层最小集合

- `POST /ops/brands/{brandId}/publish`
- `POST /ops/factory/tasks/{taskId}/execute`
- `POST /ops/policies/{policyId}/apply`
- `POST /ops/workorders/{workorderId}/advance`

这些路径是职责示意，不强制限定最终 URI 命名，但职责边界必须一致。

---

## 10. 当前明确废止的做法

以下做法视为无效：

- 在多个文档中并行维护不同版本的接口规范
- 治理层直接写库推进协议状态
- 页面流程直接替代后端状态流转定义
- 历史草案接口继续作为现行开发依据

---

## 11. 关联文档

- 项目边界：`project-overview.md`
- 状态机：`state-machine.md`
- 权限规则：`roles-and-permissions.md`
- 安全模型：`security-model.md`
- 工程拓扑：`../engineering/system-architecture.md`
- 产品流程：`../product/product-system.md`
- 运维手册：`../ops/`
