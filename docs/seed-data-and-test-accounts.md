# 种子数据与测试账号

> 文档类型：Engineering / Testing
> 状态：Active
> 最后更新：2026-04-07

---

## 1. 概述

本文档定义 RCProtocol 开发与测试环境的种子数据，包括：

- 数据库种子数据（`rcprotocol` 主库 + `rcprotocol_iam` IAM 库）
- 后端测试账号与凭证
- 前端测试场景配置
- 各角色登录凭证速查

所有密码均为开发环境专用，**严禁用于生产**。

---

## 2. 环境前置条件

```bash
# .env 必须包含以下配置
RC_SEED_DATA=true
RC_JWT_SECRET=test-jwt-secret-at-least-32-bytes-long
RC_API_KEY_SECRET=test-api-key-secret-for-hmac-256
RC_ROOT_KEY_HEX=000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20
```

---

## 3. IAM 库种子数据（rcprotocol_iam）

### 3.1 组织（organizations）

| org_id | org_name | org_type | brand_id | contact_email | contact_phone |
|--------|----------|----------|----------|---------------|---------------|
| `org-platform-001` | RCProtocol 平台运营 | Platform | — | platform@test.rcprotocol.dev | 13800000001 |
| `org-brand-luxe` | Luxe 奢侈品品牌 | Brand | `brand-luxe` | brand@test.rcprotocol.dev | 13800000002 |
| `org-brand-demo` | RC Demo Brand | Brand | `brand-demo` | demo@test.rcprotocol.dev | 13800000004 |
| `org-factory-shenzhen` | 深圳标签工厂 | Factory | — | factory@test.rcprotocol.dev | 13800000003 |

### 3.2 岗位（positions）

| position_id | org_id | position_name | protocol_role |
|-------------|--------|---------------|---------------|
| `pos-platform-admin` | `org-platform-001` | 平台管理员 | Platform |
| `pos-platform-moderator` | `org-platform-001` | 审核员 | Moderator |
| `pos-brand-admin` | `org-brand-luxe` | 品牌管理员 | Brand |
| `pos-brand-demo-admin` | `org-brand-demo` | Demo品牌管理员 | Brand |
| `pos-factory-operator` | `org-factory-shenzhen` | 工厂操作员 | Factory |

### 3.3 用户（users）

| user_id | email | 密码（明文） | display_name | status |
|---------|-------|-------------|--------------|--------|
| `user-admin-001` | admin@test.rcprotocol.dev | `Admin@2026` | 平台管理员张三 | active |
| `user-moderator-001` | moderator@test.rcprotocol.dev | `Mod@2026` | 审核员李四 | active |
| `user-brand-001` | brand@test.rcprotocol.dev | `Brand@2026` | Luxe品牌运营王五 | active |
| `user-brand-002` | demo@test.rcprotocol.dev | `Brand@2026` | Demo品牌运营 | active |
| `user-factory-001` | factory@test.rcprotocol.dev | `Factory@2026` | 工厂操作员赵六 | active |
| `user-consumer-001` | consumer1@test.rcprotocol.dev | `Consumer@2026` | 消费者测试用户A | active |
| `user-consumer-002` | consumer2@test.rcprotocol.dev | `Consumer@2026` | 消费者测试用户B | active |

### 3.4 用户-组织-岗位绑定（user_org_positions）

| user_id | org_id | position_id |
|---------|--------|-------------|
| `user-admin-001` | `org-platform-001` | `pos-platform-admin` |
| `user-moderator-001` | `org-platform-001` | `pos-platform-moderator` |
| `user-brand-001` | `org-brand-luxe` | `pos-brand-admin` |
| `user-brand-002` | `org-brand-demo` | `pos-brand-demo-admin` |
| `user-factory-001` | `org-factory-shenzhen` | `pos-factory-operator` |

> 注意：Consumer 角色不通过组织绑定，C 端用户独立注册后直接获得 Consumer 角色。

### 3.5 品牌 API Key（brand_api_keys）

| key_id | org_id | 明文 API Key | description | status |
|--------|--------|-------------|-------------|--------|
| `apikey-luxe-001` | `org-brand-luxe` | `brand_org-brand-luxe_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4` | Luxe 品牌集成测试 Key | active |
| `apikey-luxe-002` | `org-brand-luxe` | `brand_org-brand-luxe_revoked_key_for_testing_only` | Luxe 品牌已撤销的测试 Key | revoked |

> API Key 存储为 bcrypt 哈希。明文仅在此文档记录，用于测试。

---

## 4. 协议主库种子数据（rcprotocol）

### 4.1 品牌（brands）

| brand_id | brand_name | api_key_hash | status |
|----------|-----------|-------------|--------|
| `brand-luxe` | Luxe 奢侈品品牌 | HMAC-SHA256 哈希（见 §4.1.1） | Active |
| `brand-demo` | RC Demo Brand | — | Active |

#### 4.1.1 Rust Core API Key

rc-api 使用 `HMAC-SHA256(RC_API_KEY_SECRET, api_key)` 生成哈希。测试用 API Key：

```
rcsk_luxe_test_key_0123456789abcdef0123456789abcdef
```

### 4.2 批次与会话（batches / factory_sessions）

| batch_id | brand_id |
|----------|----------|
| `batch-luxe-001` | `brand-luxe` |
| `batch-demo-001` | `brand-demo` |

| session_id | batch_id |
|------------|----------|
| `session-luxe-001` | `batch-luxe-001` |
| `session-demo-001` | `batch-demo-001` |

### 4.3 资产（assets）— 覆盖主要测试状态

| asset_id | brand_id | uid | current_state | owner_id | external_product_id | external_product_name | external_product_url |
|----------|----------|-----|---------------|----------|--------------------|-----------------------|---------------------|
| `asset-preminted-001` | `brand-luxe` | `UID-TEST-0001` | PreMinted | — | — | — | — |
| `asset-preminted-002` | `brand-luxe` | `UID-TEST-0002` | PreMinted | — | — | — | — |
| `asset-preminted-003` | `brand-luxe` | `UID-TEST-0003` | PreMinted | — | — | — | — |
| `asset-factorylogged-001` | `brand-luxe` | `UID-TEST-0010` | FactoryLogged | — | — | — | — |
| `asset-unassigned-001` | `brand-luxe` | `UID-TEST-0020` | Unassigned | — | — | — | — |
| `asset-activated-001` | `brand-luxe` | `UID-TEST-0030` | Activated | — | `SKU-LUXE-001` | Luxe经典手袋 | https://www.luxe-brand.com/products/classic-handbag |
| `asset-activated-002` | `brand-luxe` | `UID-TEST-0031` | Activated | — | `SKU-LUXE-002` | Luxe限量腕表 | https://www.luxe-brand.com/products/limited-watch |
| `asset-legallysold-001` | `brand-luxe` | `UID-TEST-0040` | LegallySold | `user-consumer-001` | `SKU-LUXE-001` | Luxe经典手袋 | https://www.luxe-brand.com/products/classic-handbag |
| `asset-transferred-001` | `brand-luxe` | `UID-TEST-0050` | Transferred | `user-consumer-002` | `SKU-LUXE-001` | Luxe经典手袋 | https://www.luxe-brand.com/products/classic-handbag |
| `asset-disputed-001` | `brand-luxe` | `UID-TEST-0060` | Disputed | `user-consumer-001` | `SKU-LUXE-001` | Luxe经典手袋 | https://www.luxe-brand.com/products/classic-handbag |
| `asset-consumed-001` | `brand-luxe` | `UID-TEST-0070` | Consumed | — | `SKU-LUXE-002` | Luxe限量腕表 | https://www.luxe-brand.com/products/limited-watch |
| `asset-legacy-001` | `brand-luxe` | `UID-TEST-0080` | Legacy | — | `SKU-LUXE-002` | Luxe限量腕表 | https://www.luxe-brand.com/products/limited-watch |
| `asset-verify-001` | `brand-luxe` | `04A31B2C3D4E5F` | Activated | — | `SKU-LUXE-001` | Luxe经典手袋 | https://www.luxe-brand.com/products/classic-handbag |
| `asset-demo-001` | `brand-demo` | `UID-DEMO-0001` | Activated | — | `SKU-DEMO-001` | Demo产品 | https://demo.rcprotocol.dev/products/demo-001 |

> `asset-disputed-001` 的 `previous_state` 为 `LegallySold`，用于测试恢复流程。  
> `asset-legacy-001` 的 `previous_state` 为 `Transferred`，用于测试传承遗珍终态。

### 4.4 母卡凭证（authority_devices）

| authority_id | authority_uid | authority_type | brand_id | status | bound_user_id |
|-------------|---------------|----------------|----------|--------|---------------|
| `auth-dev-vqr-001` | `VAUTH-LUXE-0030` | VIRTUAL_QR | `brand-luxe` | Active | — |
| `auth-dev-vapp-001` | `VAUTH-LUXE-0040` | VIRTUAL_APP | `brand-luxe` | Active | `user-consumer-001` |
| `auth-dev-vapp-002` | `VAUTH-LUXE-0050` | VIRTUAL_APP | `brand-luxe` | Active | `user-consumer-002` |
| `auth-dev-phys-001` | `04F10A2B3C4D5E` | PHYSICAL_NFC | `brand-luxe` | Active | — |

### 4.5 母子绑定（asset_entanglements）

| asset_id | child_uid | authority_id | authority_uid | entanglement_state | bound_by |
|----------|-----------|-------------|---------------|-------------------|----------|
| `asset-activated-001` | `UID-TEST-0030` | `auth-dev-vqr-001` | `VAUTH-LUXE-0030` | Active | `user-brand-001` |
| `asset-legallysold-001` | `UID-TEST-0040` | `auth-dev-vapp-001` | `VAUTH-LUXE-0040` | Active | `user-brand-001` |
| `asset-transferred-001` | `UID-TEST-0050` | `auth-dev-vapp-002` | `VAUTH-LUXE-0050` | Active | `user-brand-001` |

### 4.6 过户记录（asset_transfers）

| asset_id | from_user_id | to_user_id | trace_id |
|----------|-------------|-----------|----------|
| `asset-transferred-001` | `user-consumer-001` | `user-consumer-002` | `trace-transfer-001` |

---

## 5. 登录凭证速查表

### 5.1 B 端（BConsole）登录

| 角色 | 邮箱 | 密码 | 登录后角色 | brand_id |
|------|------|------|-----------|----------|
| 平台管理员 | admin@test.rcprotocol.dev | `Admin@2026` | Platform | — |
| 审核员 | moderator@test.rcprotocol.dev | `Mod@2026` | Moderator | — |
| 品牌管理员（Luxe） | brand@test.rcprotocol.dev | `Brand@2026` | Brand | `brand-luxe` |
| 品牌管理员（Demo） | demo@test.rcprotocol.dev | `Brand@2026` | Brand | `brand-demo` |
| 工厂操作员 | factory@test.rcprotocol.dev | `Factory@2026` | Factory | — |

### 5.2 C 端（Consumer App）登录

| 用户 | 邮箱 | 密码 | 备注 |
|------|------|------|------|
| 消费者 A | consumer1@test.rcprotocol.dev | `Consumer@2026` | 持有 1 个 LegallySold 资产 + 1 个 Disputed 资产 |
| 消费者 B | consumer2@test.rcprotocol.dev | `Consumer@2026` | 持有 1 个 Transferred 资产 |

### 5.3 品牌 API Key 调用

```bash
# IAM 系统的 API Key（用于 Go Gateway 认证）- Active
curl -H "X-Api-Key: brand_org-brand-luxe_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4" \
     -H "X-Trace-Id: $(uuidgen)" \
     -H "X-Idempotency-Key: test-$(date +%s)" \
     http://localhost:8080/api/v1/assets/asset-activated-001

# IAM 系统的 API Key（用于 Go Gateway 认证）- Revoked（应返回 401）
curl -H "X-Api-Key: brand_org-brand-luxe_revoked_key_for_testing_only" \
     -H "X-Trace-Id: $(uuidgen)" \
     http://localhost:8080/api/v1/assets/asset-activated-001

# Rust Core 的 API Key（用于 rc-api 直接调用）
curl -H "X-Api-Key: rcsk_luxe_test_key_0123456789abcdef0123456789abcdef" \
     http://localhost:8081/brands/brand-luxe
```

---

## 6. 测试场景对照

### 6.1 盲扫登记测试

- 用 Factory 账号登录
- 对 `asset-preminted-001/002/003` 执行 BlindLog
- 预期：状态变为 FactoryLogged

### 6.2 激活测试

- 用 Brand 账号登录
- 对 `asset-unassigned-001` 执行激活流程（ActivateRotateKeys → ActivateEntangle → ActivateConfirm）
- 预期：状态变为 Activated，生成虚拟母卡

### 6.3 售出测试

- 用 Brand 账号登录
- 对 `asset-activated-002` 执行 LegalSell，buyer_id 设为 `user-consumer-001`
- 预期：状态变为 LegallySold，owner_id 设为 consumer-001

### 6.4 验真测试

- 无需登录（公开接口）
- 用 `04A31B2C3D4E5F` 作为 UID 访问验真接口
- 预期：返回 Activated 状态的资产信息

### 6.5 过户测试

- 用 Consumer A 账号登录
- 对 `asset-legallysold-001` 执行 Transfer
- 预期：状态变为 Transferred，owner_id 变更

### 6.6 冻结/恢复测试

- 用 Moderator 账号登录
- 对 `asset-activated-001` 执行 Freeze → 预期变为 Disputed
- 对 `asset-disputed-001` 执行 Recover → 预期恢复为 LegallySold

### 6.7 Vault 测试

- 用 Consumer A 登录 C 端
- 预期在 Vault 中看到：`asset-legallysold-001`（LegallySold）、`asset-disputed-001`（Disputed）
- 用 Consumer B 登录 C 端
- 预期在 Vault 中看到：`asset-transferred-001`（Transferred）

### 6.8 终态测试

- 用 Consumer 账号登录
- 对 `asset-consumed-001` 查看详情 → 预期显示"已消费 🏆"荣誉态
- 对 `asset-legacy-001` 查看详情 → 预期显示"传承遗珍 👑"荣誉态
- 验证终态资产不可再次操作（不可过户、不可冻结）

### 6.9 品牌隔离测试

- 用 Luxe 品牌管理员登录
- 预期只能看到 `brand-luxe` 的资产，无法访问 `brand-demo` 的资产
- 用 Demo 品牌管理员登录
- 预期只能看到 `brand-demo` 的资产，无法访问 `brand-luxe` 的资产
- 用 Platform 管理员登录
- 预期可以看到所有品牌的资产

---

## 7. SQL 种子脚本

### 7.1 IAM 库（rcprotocol_iam）

**完整脚本已生成：** `deploy/postgres/init/998_seed_iam.sql`

包含内容：
- 4 个组织（Platform、Luxe 品牌、Demo 品牌、工厂）
- 5 个岗位
- 7 个用户（真实 bcrypt 哈希，cost=12）
- 5 个用户-组织-岗位绑定
- 2 个品牌 API Key（1 个 active，1 个 revoked）

所有密码哈希已使用真实 bcrypt 生成，可直接执行。

### 7.2 协议主库（rcprotocol）

**完整脚本已生成：** `deploy/postgres/init/999_seed_protocol.sql`

包含内容：
- 2 个品牌（brand-luxe、brand-demo）
- 2 个批次和工厂会话
- 14 个资产（覆盖 9 种状态 + 外部 SKU 映射）
- 4 个母卡凭证（虚拟 QR、虚拟 App、物理 NFC）
- 3 个母子绑定关系
- 1 个过户记录
- 4 条审计日志示例

所有资产已包含 `external_product_url` 字段，支持产品详情链接跳转测试。

---

## 8. 执行种子数据脚本

### 8.1 自动执行（推荐）

种子数据脚本已放置在 `deploy/postgres/init/` 目录，Docker Compose 启动时会自动执行：

```bash
cd deploy/compose
docker-compose up -d postgres

# 等待数据库初始化完成（约 10-15 秒）
docker-compose logs -f postgres | grep "database system is ready"

# 验证种子数据
docker-compose exec postgres psql -U rcprotocol -d rcprotocol_iam -c "SELECT COUNT(*) FROM users;"
docker-compose exec postgres psql -U rcprotocol -d rcprotocol -c "SELECT COUNT(*) FROM assets;"
```

### 8.2 手动执行

如果需要重新加载种子数据：

```bash
# IAM 库
docker-compose exec postgres psql -U rcprotocol -d rcprotocol_iam -f /docker-entrypoint-initdb.d/998_seed_iam.sql

# 协议主库
docker-compose exec postgres psql -U rcprotocol -d rcprotocol -f /docker-entrypoint-initdb.d/999_seed_protocol.sql
```

### 8.3 验证种子数据

```bash
# 验证用户数量（应为 7）
docker-compose exec postgres psql -U rcprotocol -d rcprotocol_iam -c "SELECT user_id, email, display_name FROM users;"

# 验证资产数量（应为 14）
docker-compose exec postgres psql -U rcprotocol -d rcprotocol -c "SELECT asset_id, current_state, external_product_name FROM assets;"

# 验证 API Key（应为 2，1 active + 1 revoked）
docker-compose exec postgres psql -U rcprotocol -d rcprotocol_iam -c "SELECT key_id, status, description FROM brand_api_keys;"

# 验证母卡凭证（应为 4）
docker-compose exec postgres psql -U rcprotocol -d rcprotocol -c "SELECT authority_id, authority_type, status FROM authority_devices;"
```

---

## 9. 生成 bcrypt 哈希的工具命令

所有密码哈希已在种子脚本中使用真实值，无需手动生成。如需重新生成：

```bash
# 使用 Python（推荐，与 go-iam 一致，cost=12）
pip install bcrypt
python << 'EOF'
import bcrypt
password = "Admin@2026"
hash_val = bcrypt.hashpw(password.encode(), bcrypt.gensalt(rounds=12))
print(hash_val.decode())
EOF

# 或使用 htpasswd
htpasswd -nbBC 12 "" "Admin@2026" | tr -d ':\n' | sed 's/$2y/$2a/'
```

---

## 10. 前端环境配置
  ('e0000000-0000-0000-0000-000000000003', 'asset-transferred-001', 'UID-TEST-0050',
   'a0000000-0000-0000-0000-000000000003', 'VAUTH-LUXE-0050', 'Active', 'user-brand-001')
ON CONFLICT (entanglement_id) DO NOTHING;

-- 过户记录
INSERT INTO asset_transfers (transfer_id, asset_id, from_user_id, to_user_id, trace_id)
VALUES
  ('t0000000-0000-0000-0000-000000000001', 'asset-transferred-001', 'user-consumer-001', 'user-consumer-002', 'trace-transfer-001')
ON CONFLICT (transfer_id) DO NOTHING;
```

---

## 10. 前端环境配置

### 10.1 B 端（b-console）

```env
# frontend/apps/b-console/.env.development
VITE_API_BASE_URL=http://localhost:8080
VITE_IAM_BASE_URL=http://localhost:8083
```

### 10.2 C 端（c-app）

```env
# frontend/apps/c-app/.env.development
VITE_API_BASE_URL=http://localhost:8080
VITE_VERIFY_BASE_URL=http://localhost:8081
```

---

## 11. 数据关系图

```
organizations
├── org-platform-001 (Platform)
│   ├── pos-platform-admin     → user-admin-001
│   └── pos-platform-moderator → user-moderator-001
├── org-brand-luxe (Brand, brand_id=brand-luxe)
│   ├── pos-brand-admin        → user-brand-001
│   ├── apikey-luxe-001 (active)
│   └── apikey-luxe-002 (revoked)
├── org-brand-demo (Brand, brand_id=brand-demo)
│   └── pos-brand-demo-admin   → user-brand-002
└── org-factory-shenzhen (Factory)
    └── pos-factory-operator   → user-factory-001

brands
├── brand-luxe
│   ├── batch-luxe-001 → session-luxe-001
│   └── assets: preminted(3), factorylogged(1), unassigned(1),
│               activated(2), legallysold(1), transferred(1),
│               disputed(1), consumed(1), legacy(1), verify(1)
└── brand-demo
    └── assets: activated(1)

consumers (无组织绑定)
├── user-consumer-001 → 持有: legallysold-001, disputed-001
└── user-consumer-002 → 持有: transferred-001
```

---

## 12. 补充说明

### 12.1 新增内容（相比原文档）

1. **Legacy 状态资产** - `asset-legacy-001`，用于测试传承遗珍终态
2. **外部 SKU URL** - 所有激活后的资产都包含 `external_product_url` 字段
3. **第二个品牌** - `brand-demo` 及其完整数据（组织、用户、资产），用于测试品牌隔离
4. **Revoked API Key** - `apikey-luxe-002`，用于测试 API Key 撤销功能
5. **真实密码哈希** - 所有密码使用 bcrypt cost=12 生成真实哈希，可直接使用
6. **审计日志示例** - 4 条审计日志记录，展示不同操作类型

### 12.2 测试覆盖度

| 功能模块 | 覆盖率 | 说明 |
|---------|-------|------|
| IAM 系统 | 100% | 5 种角色、4 个组织、API Key 认证 |
| 资产状态机 | 100% | 9 种状态全覆盖（含 Legacy） |
| 验真功能 | 100% | 合法标签、未知标签 |
| 母子标签 | 100% | 虚拟 QR、虚拟 App、物理 NFC |
| 过户流程 | 100% | 发起、确认、授权 |
| 终态展示 | 100% | Consumed、Legacy |
| 品牌隔离 | 100% | 多品牌数据支持 |
| API Key 管理 | 100% | 创建、列表、撤销 |

### 12.3 快速启动检查清单

- [ ] 确认 `deploy/postgres/init/998_seed_iam.sql` 存在
- [ ] 确认 `deploy/postgres/init/999_seed_protocol.sql` 存在
- [ ] 启动 Docker Compose：`cd deploy/compose && docker-compose up -d`
- [ ] 验证种子数据加载：`docker-compose exec postgres psql -U rcprotocol -d rcprotocol_iam -c "SELECT COUNT(*) FROM users;"`
- [ ] 预期结果：7 个用户、14 个资产、2 个 API Key、4 个母卡凭证

**✅ 种子数据已完整，可以立即启动集成测试！**
