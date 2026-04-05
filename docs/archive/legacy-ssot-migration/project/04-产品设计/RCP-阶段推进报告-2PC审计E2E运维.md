# RCP 阶段推进报告（2PC / 审计联查 / E2E / 运维）

## 本轮实现范围

1. 全量跨平面 E2E 联调闭环脚本化
2. NATS 消费者与 2PC 补偿链路实装
3. 审计联合检索（governance + rc-api + rc-kms）
4. 灰度、回滚、故障演练脚本
5. 生产级监控告警规则与运维门槛文档

## 已实现清单

### 1) NATS + 2PC

- 新增事务表与补偿表：
  - `apps/governance-ops/db/migrations/002_2pc_tables.sql`
- 新增 2PC 状态机与补偿处理：
  - `apps/governance-ops/internal/core/two_phase.go`
  - `apps/governance-ops/internal/core/two_phase_processors.go`
- 新增消费者：
  - `apps/governance-ops/internal/worker/consumers.go`
- API 编排调整为“准备事务 -> 发事件 -> 异步执行”：
  - `apps/governance-ops/internal/core/workflows.go`

### 2) 审计联合检索

- Go 联查聚合：
  - `apps/governance-ops/internal/core/audit_federation.go`
- Rust 核心审计查询端点：
  - `crates/rc-api/src/api/ops_audits.rs`
  - 路由接入：`crates/rc-api/src/api/mod.rs`
- Go Rust 客户端支持核心审计查询：
  - `apps/governance-ops/internal/rustapi/client.go`

### 3) E2E 闭环

- 跨平面执行脚本：
  - `deploy/e2e-cross-plane.ps1`
- 回归矩阵：
  - `docs/project/04-产品设计/RCP-跨平面E2E回归矩阵.md`
- 实测报告输出路径：
  - `docs/project/04-产品设计/RCP-B端-跨平面E2E实测记录.md`

### 4) 灰度/回滚/故障演练

- `deploy/canary-rollout.ps1`
- `deploy/rollback-governance.ps1`
- `deploy/fault-drill.ps1`
- 运维门槛文档：
  - `docs/project/04-产品设计/RCP-灰度回滚故障演练与运维验收门槛.md`

### 5) 生产监控告警

- `/metrics` 暴露与基础指标：
  - `apps/governance-ops/internal/metrics/metrics.go`
  - `apps/governance-ops/internal/httpx/router.go`
- 告警规则：
  - `deploy/monitoring/prometheus-rules-governance.yml`

## 构建验证结果

- `go test ./...`（governance-ops）通过
- `cargo check -p rc-api` 通过

## 待实网执行项（不影响代码交付完整性）

- 执行 `deploy/e2e-cross-plane.ps1` 并归档报告
- 执行 3 类故障演练并归档复盘
- 线上观测系统接入 Prometheus 规则并验证告警链路
