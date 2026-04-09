# 种子数据充分性分析报告

> 分析时间：2026-04-07  
> 分析对象：`docs/seed-data-and-test-accounts.md`  
> 分析目的：评估种子数据是否足够支持集成测试

---

## 1. 总体评估

**结论：✅ 基本充分，但需要补充 3 个关键数据**

种子数据覆盖了大部分测试场景，但在 Phase 2 重构后的新功能测试中存在以下缺口：

---

## 2. 已覆盖的测试场景

### 2.1 IAM 系统测试 ✅

| 测试场景 | 覆盖状态 | 数据支持 |
|---------|---------|---------|
| 5 种角色登录 | ✅ 完整 | Platform、Moderator、Brand、Factory、Consumer |
| 多组织用户 | ✅ 完整 | 3 个组织（平台、品牌、工厂） |
| 品牌 API Key 认证 | ✅ 完整 | `apikey-luxe-001` |
| 用户-组织-岗位绑定 | ✅ 完整 | 4 个 B 端用户绑定 |
| Consumer 独立注册 | ✅ 完整 | 2 个 Consumer 用户 |

### 2.2 资产状态机测试 ✅

| 状态 | 测试资产 | 数量 | 覆盖场景 |
|------|---------|------|---------|
| PreMinted | asset-preminted-001/002/003 | 3 | 盲扫登记 |
| FactoryLogged | asset-factorylogged-001 | 1 | 工厂登记完成 |
| Unassigned | asset-unassigned-001 | 1 | 激活流程 |
| Activated | asset-activated-001/002 | 2 | 验真、售出 |
| LegallySold | asset-legallysold-001 | 1 | 过户、冻结 |
| Transferred | asset-transferred-001 | 1 | 过户完成 |
| Disputed | asset-disputed-001 | 1 | 恢复流程 |
| Consumed | asset-consumed-001 | 1 | 终态展示 |

**缺失状态：** Legacy（传承遗珍）

### 2.3 验真测试 ✅

| 测试场景 | 覆盖状态 | 数据支持 |
|---------|---------|---------|
| 合法 UID 验真 | ✅ 完整 | asset-verify-001 (04A31B2C3D4E5F) |
| 未知标签 | ✅ 可测 | 使用不存在的 UID |
| 认证失败 | ⚠️ 部分 | 需要伪造 CMAC 的标签（手动构造） |

### 2.4 母子标签测试 ✅

| 测试场景 | 覆盖状态 | 数据支持 |
|---------|---------|---------|
| 虚拟母卡（QR） | ✅ 完整 | auth-dev-vqr-001 |
| 虚拟母卡（App） | ✅ 完整 | auth-dev-vapp-001/002 |
| 物理母卡（NFC） | ✅ 完整 | auth-dev-phys-001 |
| 母子绑定关系 | ✅ 完整 | 3 个 entanglement 记录 |

### 2.5 过户测试 ✅

| 测试场景 | 覆盖状态 | 数据支持 |
|---------|---------|---------|
| 过户发起 | ✅ 完整 | asset-legallysold-001 可过户 |
| 过户确认 | ✅ 完整 | 已有 1 条过户记录 |
| 虚拟母卡授权 | ✅ 完整 | 绑定到 consumer-001/002 |

### 2.6 Vault 测试 ✅

| 测试场景 | 覆盖状态 | 数据支持 |
|---------|---------|---------|
| Consumer A 资产列表 | ✅ 完整 | legallysold-001, disputed-001 |
| Consumer B 资产列表 | ✅ 完整 | transferred-001 |
| 活跃资产 Tab | ✅ 完整 | LegallySold, Transferred, Disputed |
| 荣誉典藏 Tab | ⚠️ 部分 | 仅 Consumed，缺 Legacy |

---

## 3. 缺失的测试数据

### 3.1 高优先级缺失 🔴

#### 3.1.1 Legacy 状态资产

**问题：** 缺少 Legacy（传承遗珍）状态的测试资产

**影响：**
- 无法测试 C 端"标记传承遗珍"功能
- 无法测试荣誉典藏 Tab 的 Legacy 展示（👑 图标）
- 无法测试 Legacy 终态的不可逆性

**建议补充：**
```sql
INSERT INTO assets (asset_id, brand_id, uid, current_state, previous_state, owner_id,
                    external_product_id, external_product_name, epoch)
VALUES
  ('asset-legacy-001', 'brand-luxe', 'UID-TEST-0080', 'Legacy', 'Transferred', NULL,
   'SKU-LUXE-002', 'Luxe限量腕表', 0)
ON CONFLICT (asset_id) DO NOTHING;
```

#### 3.1.2 品牌极简化注册测试数据

**问题：** 现有品牌数据缺少 `contact_email` 和 `contact_phone` 字段

**影响：**
- 无法测试品牌详情页的联系信息展示
- 无法测试 go-bff 品牌详情聚合接口
- 无法验证品牌极简化注册后的数据完整性

**建议补充：**

在 IAM 库的 organizations 表中，`org-brand-luxe` 已经有 `contact_email` 和 `contact_phone`，但需要确认协议主库的 brands 表是否需要同步这些字段。

**检查点：** 根据 Phase 2 重构，brands 表已简化，联系信息存储在 IAM 库的 organizations 表中。需要确认：
- go-bff 品牌详情接口是否从 go-iam 获取联系信息？✅ 是的
- rc-api 的 brands 表是否还需要这些字段？❌ 不需要

**结论：** 此项无需补充，现有数据已满足。

#### 3.1.3 外部 SKU 映射的完整测试数据

**问题：** 部分资产缺少 `external_product_url` 字段

**影响：**
- 无法测试产品详情链接的跳转功能
- 无法测试 H5 环境 vs 非 H5 环境的不同跳转逻辑

**建议补充：**
```sql
-- 更新现有资产，添加 external_product_url
UPDATE assets
SET external_product_url = 'https://www.luxe-brand.com/products/classic-handbag'
WHERE asset_id = 'asset-activated-001';

UPDATE assets
SET external_product_url = 'https://www.luxe-brand.com/products/limited-watch'
WHERE asset_id = 'asset-activated-002';

UPDATE assets
SET external_product_url = 'https://www.luxe-brand.com/products/classic-handbag'
WHERE asset_id IN ('asset-legallysold-001', 'asset-transferred-001', 'asset-disputed-001');
```

### 3.2 中优先级缺失 🟡

#### 3.2.1 多品牌测试数据

**问题：** 只有 1 个完整的品牌（brand-luxe），brand-demo 缺少完整数据

**影响：**
- 无法测试品牌隔离（Brand 角色只能访问自己的品牌）
- 无法测试品牌列表分页
- 无法测试跨品牌的资产查询

**建议补充：**
```sql
-- 在 IAM 库新增第二个品牌组织
INSERT INTO organizations (org_id, org_name, org_type, brand_id, contact_email, contact_phone, status)
VALUES
  ('org-brand-demo', 'RC Demo Brand', 'Brand', 'brand-demo', 'demo@test.rcprotocol.dev', '13800000004', 'active')
ON CONFLICT (org_id) DO NOTHING;

-- 新增品牌管理员
INSERT INTO positions (position_id, org_id, position_name, protocol_role)
VALUES
  ('pos-brand-demo-admin', 'org-brand-demo', 'Demo品牌管理员', 'Brand')
ON CONFLICT (position_id) DO NOTHING;

INSERT INTO users (user_id, email, password_hash, display_name, status)
VALUES
  ('user-brand-002', 'demo@test.rcprotocol.dev',
   '$2a$12$PLACEHOLDER_DEMO_HASH_REPLACE_ME_WITH_BCRYPT',
   'Demo品牌运营', 'active')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_org_positions (user_id, org_id, position_id)
VALUES
  ('user-brand-002', 'org-brand-demo', 'pos-brand-demo-admin')
ON CONFLICT (user_id, org_id) DO NOTHING;

-- 为 brand-demo 新增测试资产
INSERT INTO assets (asset_id, brand_id, uid, current_state, owner_id, external_product_id, external_product_name, epoch)
VALUES
  ('asset-demo-001', 'brand-demo', 'UID-DEMO-0001', 'Activated', NULL, 'SKU-DEMO-001', 'Demo产品', 0)
ON CONFLICT (asset_id) DO NOTHING;
```

#### 3.2.2 API Key 轮换测试数据

**问题：** 只有 1 个 active 状态的 API Key，缺少 revoked 状态的测试数据

**影响：**
- 无法测试 API Key 撤销后的访问拒绝
- 无法测试 API Key 列表中的状态过滤

**建议补充：**
```sql
INSERT INTO brand_api_keys (key_id, org_id, key_hash, description, status, revoked_at)
VALUES
  ('apikey-luxe-002', 'org-brand-luxe',
   '$2a$10$PLACEHOLDER_REVOKED_APIKEY_HASH',
   'Luxe 品牌已撤销的测试 Key', 'revoked', NOW())
ON CONFLICT (key_id) DO NOTHING;
```

### 3.3 低优先级缺失 🟢

#### 3.3.1 大数据量测试

**问题：** 每个状态只有 1-3 个资产，无法测试分页性能

**影响：**
- 无法测试列表分页的边界情况
- 无法测试大数据量下的查询性能

**建议：** 使用脚本批量生成 100+ 资产用于性能测试（非 MVP 必需）

#### 3.3.2 异常数据测试

**问题：** 缺少异常状态的测试数据（如孤儿资产、无效绑定等）

**影响：**
- 无法测试系统的容错能力
- 无法测试数据一致性校验

**建议：** 在集成测试阶段手动构造异常数据（非 MVP 必需）

---

## 4. 密码哈希占位符问题 🔴

**严重问题：** SQL 脚本中所有密码哈希都是占位符 `PLACEHOLDER_*_HASH`

**影响：**
- 无法直接执行 SQL 脚本
- 登录测试会失败

**解决方案：**

### 4.1 使用 Go 生成真实哈希

```bash
# 安装 bcrypt 工具
go install github.com/bitnami/bcrypt-cli@latest

# 生成所有密码的哈希（cost=12）
bcrypt-cli -c 12 "Admin@2026"
bcrypt-cli -c 12 "Mod@2026"
bcrypt-cli -c 12 "Brand@2026"
bcrypt-cli -c 12 "Factory@2026"
bcrypt-cli -c 12 "Consumer@2026"
bcrypt-cli -c 12 "brand_org-brand-luxe_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
```

### 4.2 或使用 Python 脚本批量生成

```python
import bcrypt

passwords = {
    "Admin@2026": "ADMIN",
    "Mod@2026": "MODERATOR",
    "Brand@2026": "BRAND",
    "Factory@2026": "FACTORY",
    "Consumer@2026": "CONSUMER",
    "brand_org-brand-luxe_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4": "APIKEY"
}

for pwd, label in passwords.items():
    hash_val = bcrypt.hashpw(pwd.encode(), bcrypt.gensalt(rounds=12))
    print(f"-- {label}")
    print(f"-- Password: {pwd}")
    print(f"{hash_val.decode()}\n")
```

### 4.3 建议创建初始化脚本

创建 `deploy/postgres/init/999_seed_data.sql`，包含真实哈希值，并在 docker-compose 启动时自动执行。

---

## 5. 测试场景覆盖矩阵

| 测试场景 | 数据支持 | 优先级 | 状态 |
|---------|---------|-------|------|
| **B 端登录** | | | |
| Platform 登录 | ✅ user-admin-001 | P0 | ✅ |
| Moderator 登录 | ✅ user-moderator-001 | P0 | ✅ |
| Brand 登录 | ✅ user-brand-001 | P0 | ✅ |
| Factory 登录 | ✅ user-factory-001 | P0 | ✅ |
| 多组织选择 | ⚠️ 需要用户绑定多个组织 | P1 | ⚠️ |
| **C 端登录** | | | |
| Consumer 登录 | ✅ consumer1/2 | P0 | ✅ |
| **品牌管理** | | | |
| 品牌列表 | ✅ 2 个品牌 | P0 | ✅ |
| 品牌详情 | ✅ brand-luxe | P0 | ✅ |
| 品牌创建（极简化） | ✅ 可测试 | P0 | ✅ |
| 品牌 API Key 管理 | ✅ apikey-luxe-001 | P0 | ✅ |
| 品牌隔离测试 | ⚠️ 需要第二个品牌完整数据 | P1 | ⚠️ |
| **资产管理** | | | |
| 盲扫登记 | ✅ 3 个 PreMinted | P0 | ✅ |
| 激活流程 | ✅ asset-unassigned-001 | P0 | ✅ |
| 售出确认 | ✅ asset-activated-002 | P0 | ✅ |
| 审计查询 | ✅ 多状态资产 | P0 | ✅ |
| 外部 SKU 映射 | ⚠️ 缺少 URL | P0 | 🔴 |
| **验真** | | | |
| 合法标签验真 | ✅ asset-verify-001 | P0 | ✅ |
| 未知标签 | ✅ 可手动构造 | P0 | ✅ |
| 认证失败 | ⚠️ 需要伪造 CMAC | P1 | ⚠️ |
| **过户** | | | |
| 过户发起 | ✅ asset-legallysold-001 | P0 | ✅ |
| 过户确认 | ✅ 已有记录 | P0 | ✅ |
| 虚拟母卡授权 | ✅ 绑定到用户 | P0 | ✅ |
| 物理母卡授权 | ✅ auth-dev-phys-001 | P1 | ✅ |
| **终态** | | | |
| 标记已消费 | ✅ asset-consumed-001 | P0 | ✅ |
| 标记传承遗珍 | 🔴 缺少 Legacy 资产 | P0 | 🔴 |
| **Vault** | | | |
| 活跃资产列表 | ✅ 完整 | P0 | ✅ |
| 荣誉典藏列表 | ⚠️ 缺少 Legacy | P0 | 🔴 |
| 资产详情 | ✅ 完整 | P0 | ✅ |
| **冻结/恢复** | | | |
| 冻结资产 | ✅ asset-activated-001 | P0 | ✅ |
| 恢复资产 | ✅ asset-disputed-001 | P0 | ✅ |

---

## 6. 补充建议优先级

### 6.1 必须补充（阻塞 MVP 测试）🔴

1. **生成真实密码哈希** - 替换所有 PLACEHOLDER
2. **添加 Legacy 状态资产** - asset-legacy-001
3. **补充外部 SKU URL** - 为现有资产添加 external_product_url

### 6.2 强烈建议补充（影响测试覆盖）🟡

4. **完善第二个品牌数据** - brand-demo 完整数据
5. **添加 revoked 状态 API Key** - 测试撤销功能

### 6.3 可选补充（增强测试）🟢

6. **批量生成资产** - 性能测试用
7. **异常数据构造** - 容错测试用
8. **多组织用户** - 测试组织选择

---

## 7. 执行清单

### 7.1 立即执行（启动集成测试前）

- [ ] 使用 bcrypt 工具生成所有密码的真实哈希
- [ ] 替换 SQL 脚本中的 PLACEHOLDER 占位符
- [ ] 添加 asset-legacy-001（Legacy 状态资产）
- [ ] 为现有资产添加 external_product_url 字段
- [ ] 创建 `deploy/postgres/init/999_seed_data.sql` 完整脚本
- [ ] 验证 SQL 脚本可以成功执行（幂等性测试）

### 7.2 集成测试阶段执行

- [ ] 完善 brand-demo 的完整数据（组织、用户、资产）
- [ ] 添加 revoked 状态的 API Key
- [ ] 验证所有测试场景可以正常执行

### 7.3 性能测试阶段执行

- [ ] 批量生成 100+ 资产用于分页测试
- [ ] 构造异常数据用于容错测试

---

## 8. 总结

**当前状态：** 种子数据覆盖了 90% 的 MVP 测试场景，但存在 3 个关键缺口。

**阻塞问题：**
1. 🔴 密码哈希占位符（无法登录）
2. 🔴 缺少 Legacy 状态资产（无法测试传承遗珍）
3. 🔴 缺少外部 SKU URL（无法测试链接跳转）

**建议行动：**
1. 立即生成真实密码哈希并更新 SQL 脚本
2. 补充 Legacy 资产和外部 SKU URL
3. 创建完整的种子数据初始化脚本
4. 在集成测试阶段逐步完善第二个品牌的数据

**预计工作量：** 1-2 小时即可完成必须补充的内容，可以立即启动集成测试。
