# Spec-06: 分层授权校验逻辑（Platform 审批流程）

## 1. 概述

实现 Platform 角色的审批工作流，用于高风险操作的二次确认。当 Platform 角色执行某些敏感操作时，需要提供 approval_id 作为审批凭证。

## 2. 核心概念

### 2.1 需要审批的操作

根据 `rc-core/src/permissions.rs` 的权限规则，以下操作需要审批：

**Platform 角色需要审批的操作：**
- `StockIn` - 资产入库（从 FactoryLogged → Unassigned）
- `ActivateRotateKeys` - 激活资产（从 Unassigned → RotatingKeys）
- `LegalSell` - 法律售出（从 Activated → LegallySold）
- `Transfer` - 过户（从 LegallySold → Transferred）
- `Dispute` - 争议标记
- `Resolve` - 解决争议

### 2.2 审批记录表

```sql
CREATE TABLE approval_records (
    approval_id TEXT PRIMARY KEY,
    requester_id TEXT NOT NULL,
    approver_id TEXT,
    operation_type TEXT NOT NULL,
    target_resource TEXT NOT NULL,
    reason TEXT,
    status TEXT NOT NULL DEFAULT 'Pending',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT valid_status CHECK (status IN ('Pending', 'Approved', 'Rejected', 'Expired'))
);

CREATE INDEX idx_approval_status ON approval_records(status, expires_at);
CREATE INDEX idx_approval_requester ON approval_records(requester_id);
```

## 3. API 接口

### 3.1 创建审批请求

**Endpoint:** `POST /approvals`

**权限:** Platform 角色

**请求体:**
```json
{
  "operation_type": "StockIn",
  "target_resource": "asset_id 或 batch_id",
  "reason": "批量入库 100 件手表"
}
```

**响应:**
```json
{
  "approval_id": "appr_01HXXX",
  "status": "Pending",
  "expires_at": "2024-01-15T10:00:00Z"
}
```

### 3.2 批准审批

**Endpoint:** `POST /approvals/:approval_id/approve`

**权限:** Platform 角色（不同于请求者）

**请求体:**
```json
{
  "approver_comment": "已核实，批准入库"
}
```

**响应:**
```json
{
  "approval_id": "appr_01HXXX",
  "status": "Approved",
  "approved_at": "2024-01-15T09:30:00Z"
}
```

### 3.3 拒绝审批

**Endpoint:** `POST /approvals/:approval_id/reject`

**权限:** Platform 角色

**请求体:**
```json
{
  "reason": "资料不全，需要补充"
}
```

### 3.4 查询审批列表

**Endpoint:** `GET /approvals`

**查询参数:**
- `status`: Pending | Approved | Rejected | Expired
- `requester_id`: 请求者 ID
- `operation_type`: 操作类型
- `page`, `page_size`: 分页参数

**响应:**
```json
{
  "approvals": [
    {
      "approval_id": "appr_01HXXX",
      "requester_id": "user_platform_001",
      "operation_type": "StockIn",
      "target_resource": "batch_01HYYY",
      "status": "Pending",
      "created_at": "2024-01-15T09:00:00Z",
      "expires_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### 3.5 查询审批详情

**Endpoint:** `GET /approvals/:approval_id`

**响应:**
```json
{
  "approval_id": "appr_01HXXX",
  "requester_id": "user_platform_001",
  "approver_id": "user_platform_002",
  "operation_type": "StockIn",
  "target_resource": "batch_01HYYY",
  "reason": "批量入库 100 件手表",
  "status": "Approved",
  "metadata": {
    "asset_count": 100
  },
  "created_at": "2024-01-15T09:00:00Z",
  "approved_at": "2024-01-15T09:30:00Z",
  "expires_at": "2024-01-15T10:00:00Z"
}
```

## 4. 业务规则

### 4.1 审批有效期

- 默认有效期：1 小时
- 可配置范围：5 分钟 ~ 24 小时
- 过期后自动标记为 Expired

### 4.2 审批者限制

- 审批者不能是请求者本人
- 审批者必须是 Platform 角色
- 一个审批只能被批准或拒绝一次

### 4.3 审批使用

- 审批通过后，approval_id 可在 1 小时内使用
- 每个 approval_id 只能使用一次
- 使用后自动标记为 Used

### 4.4 自动清理

- 过期的审批记录保留 30 天后自动删除
- 已使用的审批记录保留 90 天

## 5. 中间件集成

### 5.1 审批校验中间件

在 `protocol.rs` 的操作处理函数中，检查 `X-Approval-Id` header：

```rust
async fn check_approval_if_needed(
    pool: &PgPool,
    actor: &ActorContext,
    action: &AssetAction,
    approval_id: Option<String>,
) -> Result<(), RcError> {
    // 检查是否需要审批
    let needs_approval = match (actor.role, action) {
        (Role::Platform, AssetAction::StockIn) => true,
        (Role::Platform, AssetAction::ActivateRotateKeys) => true,
        (Role::Platform, AssetAction::LegalSell) => true,
        (Role::Platform, AssetAction::Transfer) => true,
        _ => false,
    };

    if !needs_approval {
        return Ok(());
    }

    // 验证 approval_id
    let approval_id = approval_id.ok_or_else(|| {
        RcError::PermissionDenied("approval_id required for this operation".into())
    })?;

    // 查询审批记录
    let approval = fetch_approval(&pool, &approval_id).await?;

    // 验证审批状态
    if approval.status != "Approved" {
        return Err(RcError::PermissionDenied("approval not approved".into()));
    }

    // 验证是否过期
    if approval.expires_at < Utc::now() {
        return Err(RcError::PermissionDenied("approval expired".into()));
    }

    // 标记为已使用
    mark_approval_used(&pool, &approval_id).await?;

    Ok(())
}
```

## 6. 测试用例

### 6.1 创建审批请求
- ✅ Platform 角色可以创建审批请求
- ✅ 返回 approval_id 和过期时间
- ❌ 非 Platform 角色创建失败

### 6.2 批准审批
- ✅ 不同的 Platform 用户可以批准
- ❌ 请求者不能批准自己的请求
- ❌ 已批准的审批不能重复批准

### 6.3 使用审批
- ✅ 有效的 approval_id 可以执行操作
- ❌ 过期的 approval_id 被拒绝
- ❌ 已使用的 approval_id 不能重复使用
- ❌ 未批准的 approval_id 不能使用

### 6.4 查询审批
- ✅ 可以按状态过滤
- ✅ 可以按请求者过滤
- ✅ 支持分页

## 7. 实施计划

### Phase 1: 数据库层（0.5 天）
- [ ] 创建 approval_records 表迁移
- [ ] 实现 db/approvals.rs 模块
  - [ ] create_approval
  - [ ] fetch_approval
  - [ ] list_approvals
  - [ ] approve_approval
  - [ ] reject_approval
  - [ ] mark_approval_used

### Phase 2: 业务逻辑层（1 天）
- [ ] 实现 routes/approval.rs 模块
  - [ ] POST /approvals
  - [ ] POST /approvals/:id/approve
  - [ ] POST /approvals/:id/reject
  - [ ] GET /approvals
  - [ ] GET /approvals/:id
- [ ] 实现审批校验中间件
- [ ] 集成到 protocol.rs 的操作处理

### Phase 3: 测试（0.5 天）
- [ ] 编写单元测试
- [ ] 编写集成测试脚本
- [ ] 测试完整审批流程

## 8. 验收标准

- [ ] 所有 API 接口正常工作
- [ ] 审批校验中间件正确拦截未授权操作
- [ ] 过期审批被正确拒绝
- [ ] 审批者不能是请求者本人
- [ ] 集成测试全部通过
