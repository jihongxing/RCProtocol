# RCP B端 Big Bang 联调与 GA 验收报告（阶段性）

## 1. 交付范围确认

- 已按 5 份设计文档落地实现核心骨架：
  - 契约真相源：`contracts/`
  - Go 治理后端：`apps/governance-ops/`
  - 前端控制台：`apps/bconsole/`
  - Rust 在位扩展：`crates/rc-api/src/api/ops_context.rs`
- 未修改协议计划文件本体，仅新增实现与验收材料。

## 2. 关键产物清单

- 契约：
  - `contracts/openapi/governance-ops.v1.yaml`
  - `contracts/proto/governance_events.proto`
  - `contracts/spec/governance-semantics.md`
- Go：
  - Gin 服务入口、路由、业务编排、Rust API 客户端
  - sqlc 配置与 Postgres 迁移脚本（含 RLS 基础策略）
  - Redis 幂等、NATS 事件发布挂点
- 前端：
  - React + Vite 控制台骨架
  - 角色可见导航与关键页面流
  - 对 `/ops/v1` 契约调用示例
- Rust：
  - `/v1/ops/context` 元数据对齐端点（trace/idempotency/approval/claims）

## 3. 运行验证记录

- Go 后端编译/测试：
  - 命令：`go mod tidy; go test ./...`
  - 结果：通过
- 前端构建：
  - 命令：`npm run build`
  - 结果：通过
- Rust 在位扩展编译检查：
  - 命令：`cargo check -p rc-api`
  - 结果：通过（存在既有 warning，不阻断）

## 4. GA 清单对照（可打勾）

### 4.1 集成覆盖率

- [x] B 端治理能力有对应 Rust 映射入口（通过 Go 编排调用）
- [x] 无独立核心协议引擎（Go 不承载 KDF/签名/EV2）
- [x] 新增能力以在位扩展或外围治理实现

### 4.2 质量门槛

- [x] Go 治理服务可编译测试通过
- [x] 前端可构建
- [x] Rust 在位扩展可编译检查通过
- [ ] 全量跨平面集成自动化 E2E（需环境化联调执行）

### 4.3 安全门槛

- [x] 幂等语义、审批门禁、错误码标准已冻结为契约
- [x] 审计事件与 trace 贯穿字段已定义
- [ ] 委托轮换/吊销压力与故障演练（待联调）

### 4.4 运维门槛

- [x] 提供启动脚本（Go/BConsole）
- [ ] 灰度发布与回滚演练实测记录（待执行）
- [ ] 监控告警指标接入（待接入现网观测）

## 5. 待完成项（GA 前）

1. 接入真实 `sqlc generate` 产物并替换临时存取逻辑。
2. 打通 `ops_audit_events` 与 Rust 侧审计联合检索。
3. 完成 NATS 消费者与 2PC 工序补偿任务。
4. 增加 E2E 测试：`blind-log -> entangle -> verify -> transfer/recovery/rebirth`。
5. 输出灰度与回滚演练记录，补齐 GA 剩余勾选项。
