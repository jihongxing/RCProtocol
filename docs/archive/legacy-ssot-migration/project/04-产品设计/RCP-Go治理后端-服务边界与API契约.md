# RCP Go治理后端服务边界与 Rust API契约（落地版）

## 1. 目标与硬约束

- Go 后端承担运营治理编排，不承载协议主权逻辑。
- Rust 核心（`rc-api`/`rc-kms`/`rc-core`）是唯一执行与真源。
- 所有治理动作必须可审计、可幂等、可回滚。

## 2. Go治理后端服务切分

| 服务 | 职责 | 只读/写入 | 依赖Rust接口 |
|---|---|---|---|
| `go-bff-gateway` | 统一鉴权、路由聚合、前端BFF | 读写 | 全部业务端点（经策略网关） |
| `go-org-iam` | 组织、岗位、角色模板、策略映射 | 读写（治理库） | JWT签发校验、`/v1/auth/revoke` |
| `go-approval-flow` | 草稿-评审-冻结-发布工作流 | 读写（治理库） | `brand/products/permission/delegation` |
| `go-policy-center` | scope模板、风控策略版本管理 | 读写（治理库） | `delegation` `geo-fence` `re-verify` |
| `go-ops-audit` | 治理动作审计、检索、归档 | 只读聚合+写审计 | `BusinessAuditStore` 查询扩展 |
| `go-workorder` | 争议、恢复、转移工单编排 | 读写（治理库） | `transfer` `recovery` `rebirth` |
| `go-reporting` | 报表与看板聚合 | 只读 | `vault` `valuation` `credit` 等 |

## 3. Rust契约（统一调用规则）

### 3.1 请求头与追踪

- 必传：
  - `Authorization: Bearer <jwt>`
  - `X-Trace-Id: <uuid>`
  - `X-Idempotency-Key: <uuid>`（写操作）
  - `X-Approval-Id: <approval_id>`（需审批的动作）
  - `X-Policy-Version: <version>`（策略化动作）
- 语义：
  - Rust 若收到同一幂等键且请求体 hash 一致，返回同结果。
  - 请求体 hash 不一致时返回 `409 CONFLICT`。

### 3.2 错误码映射

| Rust侧典型响应 | Go侧标准化码 | Go侧处理 |
|---|---|---|
| `400 BAD_REQUEST` | `RCP-400-INVALID_INPUT` | 直接回前端，附字段级错误 |
| `401/403` | `RCP-401-AUTH` / `RCP-403-AUTHZ` | 触发会话检查/权限提示 |
| `404` | `RCP-404-NOT_FOUND` | 标记工单不可重试 |
| `409` | `RCP-409-CONFLICT` | 幂等冲突或状态竞争，进入补偿分支 |
| `422` | `RCP-422-UNPROCESSABLE` | 语义错误，要求修正数据 |
| `5xx` | `RCP-500-UPSTREAM` | 熔断+重试（指数退避） |

### 3.3 鉴权模型（双层）

- 外层（Go）：组织RBAC/ABAC（岗位、组织、审批状态）。
- 内层（Rust）：`Claims.role` + 业务域校验（`brand_id`/`wid`/`scope`）。
- 原则：外层允许不代表最终允许，Rust 保留最终拒绝权。

## 4. 核心流程编排（Go -> Rust）

### 4.1 品牌与SKU发布

1. Go 创建变更草稿（品牌或SKU）。
2. 审批通过后，Go 调用 Rust：
   - `POST /v1/brand/register`
   - `POST /v1/brand/:brand_id/products`
   - `PUT /v1/brand/:brand_id/products/:product_id/translations/:locale`
3. Go 记录审计与版本快照。

补偿路径：
- 发布失败时，SKU停留 `pending_publish`，不回写“已生效”状态。

### 4.2 发行/纠缠任务

1. Go 创建批次与会话：
   - `POST /v1/factory/batches`
   - `POST /v1/factory/sessions`
2. 扫描流程：
   - `POST /v1/factory/blind-log`
   - `POST /v1/brand/entangle-active`
3. Go 汇总成功率、失败原因并生成工厂看板。

补偿路径：
- 会话异常则 `DELETE /v1/factory/sessions/:session_id` 收敛资源。

### 4.3 委托授权与信任切换

1. Go 按审批结果签发委托：
   - `POST /v1/permission/delegation/issue`
2. 运行时校验与吊销：
   - `POST /v1/permission/delegation/verify`
   - `POST /v1/permission/delegation/revoke`
3. 安全轮换：
   - `POST /v1/permission/delegation/rotate`

补偿路径：
- 发现高风险后先 revoke，再冻结相关策略版本。

### 4.4 转移/恢复/争议工单

1. 工单驱动调用：
   - `transfer` 三段式
   - `recovery` 五段式
   - `rebirth` 三段式
2. 每段都绑定 `X-Trace-Id` 和工单ID。

补偿路径：
- 任一阶段失败进入工单“待人工决议”，不自动跨阶段推进。

## 5. Go接口草案（对BConsole）

> 这些接口只做治理编排，最终会调用 Rust。

- `POST /ops/v1/brands/{brandId}/publish`
  - 入参：`approval_id`, `idempotency_key`
  - 动作：调用 Rust 品牌/SKU 端点并写审计
- `POST /ops/v1/factory/tasks/{taskId}/execute`
  - 动作：批量编排 blind-log + entangle
- `POST /ops/v1/policies/delegation/{policyId}/apply`
  - 动作：签发/吊销 delegation token
- `POST /ops/v1/workorders/{woId}/advance`
  - 动作：推进 transfer/recovery/rebirth 的某一阶段

## 6. 非功能门槛

- 可用性：Go治理层单点故障不影响 Rust 核心验证链路。
- 性能：治理编排 P95 < 300ms（不含上游业务处理时长）。
- 安全：所有写操作必须带审批与幂等键；审计写入失败则主流程失败。
- 可运维：每个 trace_id 可串联到 Rust 审计事件。

## 7. 实施清单（可直接分工）

- 后端：
  - 落地 `go-bff-gateway` + `go-approval-flow` 最小骨架
  - 建立 Rust client SDK（重试、幂等、错误映射）
- 平台：
  - 打通统一追踪（trace/span）和审计事件汇聚
  - 配置熔断与重试策略
- 安全：
  - 签发策略、token撤销、scope模板进入审批管控
