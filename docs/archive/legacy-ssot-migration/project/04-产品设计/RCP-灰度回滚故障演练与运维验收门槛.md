# RCP 灰度回滚、故障演练与运维验收门槛

## 1. 灰度策略

- 5% -> 30% -> 100% 三阶段切流。
- 阶段门槛：
  - 5xx 比例 < 1%
  - `COMPENSATION_REQUIRED` 每 10 分钟 < 20
  - 审计写入连续性正常（无 15 分钟断流）

## 2. 回滚策略

- 回滚触发条件（任一满足）：
  - 连续 10 分钟 5xx 比例 > 1%
  - 高风险写接口出现系统性拒绝或超时
  - 审计链路不可用且无法快速恢复
- 回滚动作：
  - 执行 `deploy/rollback-governance.ps1`
  - 暂停高风险写入口（delegation rotate/revoke、workorder complete）
  - 保留审计与只读接口

## 3. 故障演练

- 演练脚本：`deploy/fault-drill.ps1`
- 场景：
  - `nats_down`
  - `redis_down`
  - `rust_unreachable`
- 演练输出：
  - 错误率曲线
  - 2PC 状态分布
  - 审计事件完整性
  - 恢复时长（RTO）

## 4. 运维验收门槛（GA）

- [ ] 灰度三阶段全部通过
- [ ] 至少完成 3 类故障演练并有复盘记录
- [ ] 回滚演练成功，RTO <= 30 分钟
- [ ] Prometheus 告警规则生效并完成告警链路验证
- [ ] 审计联合检索可在 5 分钟内定位任一 TraceId 全链路事件
