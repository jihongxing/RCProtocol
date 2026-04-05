# RCP 跨平面 E2E 回归矩阵（主链路全绿）

## 1. 覆盖范围

- GovernanceOps（Go）控制平面
- SovereignCorePlane（Rust: rc-api/rc-kms/rc-core）
- BConsole（React）调用链

## 2. 主链路矩阵

| 用例ID | 链路 | 入口 | 期望结果 |
| --- | --- | --- | --- |
| E2E-01 | delegation 2PC | `/ops/v1/policies/delegation/:id/apply` | 返回 accepted，2PC 最终 COMMITTED 或 COMPENSATION_REQUIRED |
| E2E-02 | workorder 2PC | `/ops/v1/workorders/:id/advance` | 返回 accepted，2PC 状态正确迁移 |
| E2E-03 | factory task 2PC | `/ops/v1/factory/tasks/:id/execute` | 返回 accepted，产生 2PC 事务与事件 |
| E2E-04 | 联合审计检索 | `/ops/v1/audits/search` | 返回 governance + core 合并事件 |
| E2E-05 | Rust 对接探针 | `/v1/ops/context` | 正确回显 trace/idempotency/approval/claims |

## 3. 执行方式

- 一键脚本：`deploy/e2e-cross-plane.ps1`
- 输出报告：`docs/project/04-产品设计/RCP-B端-跨平面E2E实测记录.md`

## 4. 判定标准

- 所有关键用例接口返回符合契约（code/status/error）。
- 2PC 状态机无非法跳转（PREPARED->APPLYING->COMMITTED/COMPENSATION_REQUIRED）。
- TraceId 能在联合审计结果中检索到。
- 失败路径可进入补偿轨道且有审计记录。
