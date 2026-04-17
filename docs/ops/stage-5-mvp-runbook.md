# Stage 5 MVP 本地运行与联调手册

> 文档类型：Ops  
> 适用阶段：Stage 5 MVP 可交付闭环  
> 最后更新：2026-04-10

---

## 1. 目的

本手册用于固定 RCProtocol 在 Stage 5 阶段的最小运行环境、启动顺序、联调入口、自检方式与常见故障处理。

目标是让团队成员在不依赖口头补充的情况下完成：

1. 启动 PostgreSQL / Redis
2. 启动 `rc-api`
3. 跑通激活两阶段语义：`/activate` 生成承诺与声明，`/activate-entangle` 生成虚拟母卡与母子绑定
4. 执行品牌注册、验真、转移、主链路相关测试
5. 确认系统达到“可演示 / 可回归”状态

---

## 2. 当前本地环境约定

### 2.1 本地基础设施

当前本地默认约定为：

- PostgreSQL：运行在 **Podman** 容器中
- Redis：运行在 **Podman** 容器中
- `rc-api`：本机直接运行
- Go 服务：按需本机直接运行
- 前端：按需本机直接运行

### 2.2 默认端口

若未额外覆盖环境变量，Stage 5 默认端口为：

- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`
- `rc-api`：`localhost:8081`
- `go-gateway`：`localhost:8080`
- `go-bff`：`localhost:8082`
- `go-iam`：`localhost:8083`

---

## 3. 最小运行环境

### 3.1 Stage 5 必需服务

对 Stage 5 来说，以下服务是必需的：

1. PostgreSQL
2. Redis
3. `rust/rc-api`

### 3.2 Stage 5 可选服务

以下服务在不同联调场景下按需启动：

- `go-gateway`
- `go-bff`
- `go-iam`
- `go-workorder`
- `go-webhook`
- `frontend/apps/b-console`
- `frontend/apps/c-app`

### 3.3 双通路最小要求

#### 平台托管运营通路
建议至少启动：

- PostgreSQL
- Redis
- `rc-api`
- `go-bff`（若页面依赖）
- B/C 端前端（若要页面演示）

#### 品牌开放接入通路
建议至少启动：

- PostgreSQL
- Redis
- `rc-api`
- 可选：`go-gateway`

如果当前品牌 API 直接打 `rc-api`，则可以不强依赖 `go-gateway`。

---

## 4. 环境变量基线

推荐至少设置以下变量：

```bash
RC_ROOT_KEY_HEX=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
RC_SYSTEM_ID=rcprotocol-dev
RC_JWT_SECRET=my-super-secret-jwt-key-for-testing-only
RC_API_KEY_SECRET=rc-dev-api-key-secret-do-not-use-in-prod
DATABASE_URL=postgresql://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol
TEST_DATABASE_URL=postgresql://rcprotocol:rcprotocol_dev@localhost:5432/postgres
REDIS_URL=redis://localhost:6379
RC_API_FALLBACK_STRATEGY=Auto
RC_SEED_DATA=true
```

说明：

- 若 Redis 当前不希望参与联调，可把 `RC_API_FALLBACK_STRATEGY=DirectPg`
- 若 PostgreSQL / Redis 在 Podman 中映射了不同端口，请按实际端口覆盖
- 本地密钥仅用于开发联调，不可用于生产

---

## 5. Podman 启动建议

### 5.1 PostgreSQL

如果本地 PostgreSQL 容器尚未运行，可参考：

```bash
podman run -d --name rcprotocol-postgres \
  -e POSTGRES_DB=rcprotocol \
  -e POSTGRES_USER=rcprotocol \
  -e POSTGRES_PASSWORD=rcprotocol_dev \
  -p 5432:5432 \
  postgres:16
```

### 5.2 Redis

如果本地 Redis 容器尚未运行，可参考：

```bash
podman run -d --name rcprotocol-redis \
  -p 6379:6379 \
  redis:7-alpine redis-server --appendonly yes
```

### 5.3 检查容器状态

```bash
podman ps
```

至少应看到 PostgreSQL 与 Redis 容器处于运行状态。

---

## 6. 启动顺序

推荐按以下顺序启动：

### Step 1：确认 PostgreSQL 与 Redis 可连通

- PostgreSQL 端口可访问
- Redis 端口可访问

### Step 2：启动 `rc-api`

Windows PowerShell：

```powershell
pwsh ./scripts/start-api.ps1
```

Linux/macOS / Git Bash：

```bash
bash ./scripts/start-api.sh
```

### Step 3：执行基础健康检查

```bash
curl http://localhost:8081/healthz
```

预期返回：

- `status = ok`
- `redis = connected` 或 `not_configured`

### Step 4：运行 Stage 5 回归脚本

- 品牌 API 闭环：`scripts/test-brand-registration.sh` / `.ps1`
- 验真 V1 / V2：`scripts/test-verify-v2.sh` / `.ps1`
- 主链路回归：`scripts/stage5-main-chain.sh` / `.ps1`
- 异常流矩阵：`scripts/stage5-error-matrix.sh` / `.ps1`
- 性能基线：`scripts/stage5-perf-baseline.sh` / `.ps1`

---

## 7. 数据库与 sqlx 基线

### 7.1 启动后自动 migration

`rc-api` 启动时会自动执行 `sqlx::migrate!()`，因此本地联调通常不需要手工执行 migration。

### 7.2 `sqlx` 编译问题处理

若本地编译遇到 `sqlx` 检查问题：

1. 先确认 `DATABASE_URL` 可连接
2. 再运行：

```bash
cargo test -p rc-api --test brand_registration_integration -- --nocapture
```

如果仍需离线缓存，可在数据库连通时执行：

```bash
cargo sqlx prepare
```

### 7.3 测试数据库

默认测试数据库建议使用：

```text
postgresql://rcprotocol:rcprotocol_dev@localhost:5432/postgres
```

---

## 8. Redis 角色说明

Stage 5 中 Redis 的定位是：

1. CTR 缓存 / 校准辅助
2. 钱包快照与查询加速的运行时缓存
3. 非权威加速层

Redis **不是** 当前 Stage 5 的权威真相源。

### 8.1 Redis 故障时的降级方式

如只想完成 MVP 联调，可设置：

```bash
RC_API_FALLBACK_STRATEGY=DirectPg
```

此时：

- `rc-api` 启动时不强连 Redis
- `/healthz` 中 Redis 状态会显示 `not_configured`
- 主链路仍可验证，但会失去 Redis 缓存加速路径

### 8.2 Redis 重建参考

Redis 快照与恢复流程可参考：

- `docs/ops/wallet-snapshot-rebuild.md`
- `docs/ops/disaster-recovery.md`

---

## 9. Stage 5 标准联调入口

### 9.1 品牌 API 接入闭环

```bash
bash ./scripts/test-brand-registration.sh
```

或：

```powershell
pwsh ./scripts/test-brand-registration.ps1
```

覆盖：

- 品牌注册
- 邮箱唯一性
- API Key 轮换
- 旧 Key 失效 / 新 Key 生效
- API Key 列表
- Brand 权限限制

### 9.2 验真 V1 / V2 并行验证

```bash
bash ./scripts/test-verify-v2.sh
```

或：

```powershell
pwsh ./scripts/test-verify-v2.ps1
```

覆盖：

- `/verify`
- `/verify/v2`
- 结构化响应对照

### 9.3 主链路回归

```bash
bash ./scripts/stage5-main-chain.sh
```

或：

```powershell
pwsh ./scripts/stage5-main-chain.ps1
```

覆盖：

- Rust 核心写路径测试
- 激活 / 承诺 / 详情联调
- 转移联调
- 验真集成测试
- 前端关键页面测试（可选）

### 9.4 异常流回归

```bash
bash ./scripts/stage5-error-matrix.sh
```

或：

```powershell
pwsh ./scripts/stage5-error-matrix.ps1
```

覆盖：

- replay suspected
- authentication failed
- incomplete attestation
- API Key 失效
- 冻结 / 恢复限制行为
- 转移 reject / conflict

### 9.5 性能基线采集

```bash
bash ./scripts/stage5-perf-baseline.sh
```

或：

```powershell
pwsh ./scripts/stage5-perf-baseline.ps1
```

输出：

- `/healthz` 可用性检查
- `verify` / `verify v2` 简单延迟样本
- 写路径测试耗时摘要

---

## 10. 如何判断系统已达到“可演示状态”

满足以下条件即可视为进入 Stage 5 可演示状态：

1. `rc-api` 成功启动
2. `/healthz` 返回正常
3. 品牌注册脚本通过
4. 验真 V1 / V2 对照脚本通过
5. 主链路回归脚本通过
6. 异常流矩阵脚本通过

---

## 11. 常见问题

### 11.1 `RC_JWT_SECRET` 长度不足

现象：`rc-api` 启动时报 JWT secret 长度错误。

处理：

- 将 `RC_JWT_SECRET` 设置为至少 32 字节

### 11.2 PostgreSQL 端口不通

现象：`connect postgres` 失败。

处理：

- 用 `podman ps` 确认 PostgreSQL 容器正在运行
- 检查 `DATABASE_URL` 里的端口是否正确
- 确认容器端口映射已暴露到宿主机

### 11.3 Redis 不通

现象：健康检查里 Redis 显示 `disconnected`。

处理：

- 若当前仅做主链路联调，可切换 `RC_API_FALLBACK_STRATEGY=DirectPg`
- 若需要 Redis 能力，检查 `REDIS_URL` 与 Podman 端口映射

### 11.4 品牌注册脚本提示 `PLATFORM_TOKEN` 缺失

处理：

- 先运行 `scripts/generate-platform-token.sh`
- 或手工设置 `PLATFORM_TOKEN`

### 11.5 测试数据库冲突

处理：

- 确认 `TEST_DATABASE_URL` 指向可创建临时 schema / 数据的 PostgreSQL 实例
- 如本地残留脏数据，可清理测试库或重启容器后重跑

---

## 12. 关联文档

- `docs/specs/spec-stage-5-mvp-delivery.md`
- `docs/tasks/task-stage-5-mvp-delivery.md`
- `docs/ops/stage-5-error-matrix.md`
- `docs/ops/stage-5-mvp-acceptance.md`
- `docs/ops/stage-5-performance-baseline.md`
- `docs/ops/wallet-snapshot-rebuild.md`
- `docs/ops/disaster-recovery.md`
