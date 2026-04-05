# RCProtocol Monorepo Scaffold

这是基于当前 `docs/engineering/` 基线生成的最小可演进脚手架。

## 当前目标

- 保持 `Rust Core` 作为协议真源
- 保持 `Go` 作为治理编排与聚合层
- 保持 `frontend` 作为 uni-app 多端前端落点
- 保持 `deploy` / `tools` / `scripts` 为基础支撑层
- 补齐协议主链路的最小状态推进、权限裁决、审计结构与 PostgreSQL 初始化骨架

## 目录

```text
RCProtocol/
├─ docs/
├─ rust/
├─ services/
├─ frontend/
├─ deploy/
├─ scripts/
└─ tools/
```

## 当前脚手架原则

1. 先搭模块边界，不先堆复杂业务实现
2. 先保证目录和依赖方向正确，再逐步填充代码
3. 所有协议写路径最终都应收敛到 `rust/rc-api`
4. Go 服务只做治理与聚合，不做协议真相写入
5. 前端只做展示和触发，不定义平行状态机
6. PostgreSQL 作为权威真源，初始化脚本只承载协议主链路必需表

## 当前已补齐的关键骨架

- `rc-common`：状态、角色、动作、资产记录、审计上下文与审计事件
- `rc-core`：动作到状态的映射、角色动作裁决、冻结约束、恢复约束、审计事件生成
- `rc-api`：完整资产主链路动作接口、严格 header 校验、幂等冲突校验、真实 `verify` 只读查询
- `deploy/postgres/init/001_init.sql`：品牌、产品、批次、会话、资产、状态事件、幂等记录表
- `deploy/postgres/init/002_seed.sql`：本地联调种子数据
- `scripts/local-main-chain.ps1`：Windows PowerShell 主链路联调脚本
- `scripts/local-reset-and-assert.ps1`：Windows PowerShell 重置说明与结果断言脚本

## 本地联调建议

1. 用 compose 启动 PostgreSQL、Redis、rc-api
2. 让 `001_init.sql` 与 `002_seed.sql` 自动初始化数据库
3. 运行 `scripts/local-main-chain.ps1`
4. 运行 `scripts/local-reset-and-assert.ps1`
5. 观察 `asset_state_events`、`assets`、`idempotency_records` 三张表变化

## 下一步建议

1. 补鉴权解析，把 `Authorization` 拆成真实 actor / tenant 上下文
2. 引入 migration 管理与测试夹具
3. 为 `verify` 增加更完整的公开信息投影
4. 将 Go Gateway / BFF 接到真实 `rc-api` 契约
