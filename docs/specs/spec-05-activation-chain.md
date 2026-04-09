# Spec-05: 激活链路与虚拟母卡生成

> 文档类型：Technical Specification  
> 状态：Draft  
> 优先级：P0（核心链路）  
> 预计工作量：2-3 天  
> 最后更新：2026-04-08

---

## 1. 背景与目标

### 1.1 业务背景

激活（Activation）是 RCProtocol 的核心链路之一，将工厂盲扫登记的资产（FactoryLogged 状态）转化为品牌认领的已激活资产（Activated 状态）。

激活过程包括：
- 品牌认领资产
- 绑定外部 SKU 映射（external_product_id）
- 生成虚拟母卡（Authority Device）
- 建立母子纠缠关系（Entanglement）
- 状态推进（FactoryLogged → Activated）

### 1.2 当前实现状态

**已完成：**
- `POST /assets/:id/activate` 路由已注册（protocol.rs:241-249）
- 状态机逻辑已实现（rc-core）
- 虚拟母卡生成函数已实现（protocol.rs:510-572）
- 数据库表结构完整（authority_devices, asset_entanglements）

**待补齐：**
- 激活接口的请求/响应格式需要调整（支持 external_product_id）
- 虚拟母卡凭证需要返回给调用方
- 需要完善错误处理和权限校验
- 需要编写集成测试

### 1.3 目标

实现完整的激活链路，支持：
1. 品牌方通过 API 激活资产
2. 自动生成虚拟母卡并建立纠缠关系
3. 返回虚拟母卡凭证（用于后续过户验证）
4. 完整的审计日志和错误处理

---

## 2. 接口设计

### 2.1 激活资产

**接口**: `POST /api/v1/assets/:asset_id/activate`

**认证**: JWT (Platform/Brand) 或 API Key (Brand)

**请求头**:
```
Authorization: Bearer <jwt_token>
或
X-Api-Key: <api_key>

X-Trace-Id: <uuid>
X-Idempotency-Key: <unique_key>
```

**请求体**:
```json
{
  "external_product_id": "SKU-2024-LUXURY-001",
  "external_product_name": "经典款手提包",
  "external_product_url": "https://brand.com/products/SKU-2024-LUXURY-001"
}
```

**字段说明**:
- `external_product_id` (必填): 品牌方 SKU ID
- `external_product_name` (可选): SKU 名称（用于展示）
- `external_product_url` (可选): SKU 详情页 URL

**响应体** (200 OK):
```json
{
  "asset_id": "01KNNSWRVHBJQBJVQXJ3JJ6C2N",
  "action": "ActivateRotateKeys",
  "from_state": "FactoryLogged",
  "to_state": "Activated",
  "audit_event_id": "550e8400-e29b-41d4-a716-446655440000",
  "virtual_mother_card": {
    "authority_uid": "vauth-abc123def456",
    "authority_type": "VIRTUAL_APP",
    "credential_hash": "a1b2c3d4e5f6...",
    "epoch": 0
  }
}
```

**错误响应**:
- `400 Bad Request`: 缺少必填字段或格式错误
- `401 Unauthorized`: 认证失败
- `403 Forbidden`: 品牌边界违规（尝试激活其他品牌的资产）
- `404 Not Found`: 资产不存在
- `409 Conflict`: 资产状态不允许激活（非 FactoryLogged 状态）
- `422 Unprocessable Entity`: 状态机转换失败

---

## 3. 数据模型

### 3.1 Assets 表更新

激活时需要更新以下字段：

```sql
UPDATE assets SET
  current_state = 'Activated',
  previous_state = 'FactoryLogged',
  external_product_id = $1,
  external_product_name = $2,
  external_product_url = $3,
  key_epoch = key_epoch + 1,
  updated_at = NOW()
WHERE asset_id = $4
```

### 3.2 Authority Devices 表

虚拟母卡记录：

```sql
INSERT INTO authority_devices (
  device_id,
  authority_uid,
  authority_type,
  brand_id,
  epoch,
  credential_hash,
  bound_user_id,
  status,
  created_at,
  created_by
) VALUES (
  uuid_generate_v4(),
  'vauth-{nanoid(12)}',
  'VIRTUAL_APP',
  $brand_id,
  $epoch,
  $credential_hash,
  $actor_id,
  'Active',
  NOW(),
  $actor_id
)
```

### 3.3 Asset Entanglements 表

母子纠缠关系：

```sql
INSERT INTO asset_entanglements (
  entanglement_id,
  asset_id,
  authority_device_id,
  status,
  created_at,
  created_by
) VALUES (
  uuid_generate_v4(),
  $asset_id,
  $authority_device_id,
  'Active',
  NOW(),
  $actor_id
)
```

---

## 4. 业务规则

### 4.1 状态转换规则

- **前置状态**: FactoryLogged
- **后置状态**: Activated
- **允许的角色**: Platform, Brand (仅限自己品牌的资产)

### 4.2 虚拟母卡生成规则

1. **Authority UID 格式**: `vauth-{nanoid(12)}`
2. **密钥派生**: `K_chip_mother = KMS.derive_mother_key(brand_id, authority_uid, epoch)`
3. **凭证哈希**: `credential_hash = HMAC-SHA256(K_chip_mother, authority_uid)`
4. **密钥清零**: K_chip_mother 使用后自动 ZeroizeOnDrop

### 4.3 权限校验

- **Platform 角色**: 可激活任意品牌的资产
- **Brand 角色**: 只能激活自己品牌的资产（brand_id 匹配）
- **其他角色**: 无权限激活

### 4.4 幂等性保证

- 使用 X-Idempotency-Key 防止重复激活
- 相同 idempotency_key 返回缓存的响应
- 不同请求内容但相同 idempotency_key 返回 409 Conflict

---

## 5. 实现任务

### Phase 1: 调整激活接口（0.5 天）

**Task 5.1**: 修改 ActivateRequest 结构体
- 文件: `rust/rc-api/src/routes/protocol.rs`
- 添加 external_product_id, external_product_name, external_product_url 字段
- 验证 external_product_id 非空

**Task 5.2**: 修改 ActivateResponse 结构体
- 添加 virtual_mother_card 字段
- 包含 authority_uid, authority_type, credential_hash, epoch

### Phase 2: 完善虚拟母卡生成（1 天）

**Task 5.3**: 重构 generate_virtual_mother_card 函数
- 返回虚拟母卡信息（而不是仅记录日志）
- 返回类型: `Result<VirtualMotherCard, RcError>`

**Task 5.4**: 在 execute_asset_action 中集成虚拟母卡生成
- 激活成功后调用 generate_virtual_mother_card
- 将虚拟母卡信息添加到响应中
- 失败时记录警告但不阻塞激活响应

**Task 5.5**: 更新 persist_action 函数
- 在更新 assets 表时同时更新 external_product_id 等字段
- 从 AuditContext 或请求 payload 中获取这些字段

### Phase 3: 数据库层实现（0.5 天）

**Task 5.6**: 创建 db/activation.rs 模块
- 实现 update_asset_with_product_mapping 函数
- 在事务中更新 assets 表的 external_product_* 字段

**Task 5.7**: 修改 persist_action 函数
- 在 ActivateRotateKeys 动作时调用 update_asset_with_product_mapping

### Phase 4: 测试与验证（1 天）

**Task 5.8**: 编写单元测试
- 测试 ActivateRequest 验证逻辑
- 测试虚拟母卡生成逻辑
- 测试权限校验

**Task 5.9**: 编写集成测试脚本
- 创建 scripts/test-activation.sh
- 测试完整激活流程（盲扫 → 激活）
- 验证虚拟母卡凭证生成
- 测试错误场景（状态不匹配、权限不足等）

**Task 5.10**: 更新文档
- 更新 docs/api/brand-api-guide.md
- 添加激活接口示例

---

## 6. 测试用例

### 6.1 正常流程

1. **盲扫资产**
   ```bash
   POST /api/v1/assets/blind-scan
   {
     "uid": "04A1B2C3D4E5F6",
     "brand_id": "brand_01KNNSWRVHBJQBJVQXJ3JJ6C2N",
     "batch_id": "550e8400-e29b-41d4-a716-446655440000"
   }
   ```

2. **激活资产**
   ```bash
   POST /api/v1/assets/{asset_id}/activate
   {
     "external_product_id": "SKU-2024-001",
     "external_product_name": "经典款手提包"
   }
   ```

3. **验证结果**
   - 资产状态变为 Activated
   - 返回虚拟母卡凭证
   - authority_devices 表有新记录
   - asset_entanglements 表有新记录
   - asset_state_events 表有审计日志

### 6.2 错误场景

1. **资产不存在**: 404 Not Found
2. **资产状态不匹配**: 409 Conflict（已激活的资产不能再次激活）
3. **品牌边界违规**: 403 Forbidden（Brand A 尝试激活 Brand B 的资产）
4. **缺少必填字段**: 400 Bad Request
5. **幂等性冲突**: 409 Conflict（相同 idempotency_key 但不同请求内容）

---

## 7. 验收标准

- [ ] 激活接口支持 external_product_id 等字段
- [ ] 激活成功后自动生成虚拟母卡
- [ ] 响应中包含虚拟母卡凭证信息
- [ ] 权限校验正确（Brand 角色只能激活自己品牌的资产）
- [ ] 幂等性保证正确
- [ ] 数据库事务完整（assets, authority_devices, asset_entanglements, asset_state_events 全部更新）
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试脚本通过
- [ ] 文档更新完整

---

## 8. 依赖与风险

### 8.1 依赖

- rc-kms: derive_mother_key 函数
- rc-crypto: HMAC-SHA256 计算
- 数据库表: assets, authority_devices, asset_entanglements

### 8.2 风险

- **密钥泄露风险**: K_chip_mother 必须使用 ZeroizeOnDrop，确保用后清零
- **并发冲突**: 同一资产被并发激活时可能导致数据不一致（通过幂等性和数据库约束缓解）
- **虚拟母卡生成失败**: 不应阻塞激活响应，记录警告日志即可

---

## 9. 后续优化

- 支持批量激活接口（POST /api/v1/batches/:batch_id/activate）
- 支持物理母卡激活（PHYSICAL_NFC 类型）
- 虚拟母卡凭证加密存储（当前为明文哈希）
- 激活审批流程（需要 Platform 审批后才能激活）
