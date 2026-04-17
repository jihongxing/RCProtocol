# Stage 5 性能基线（第一版）

> 文档类型：Ops  
> 适用阶段：Stage 5 MVP 可交付闭环  
> 状态：Baseline v1  
> 最后更新：2026-04-10

---

## 1. 目标

本基线用于记录 Stage 5 当前最小可复测性能对象，作为后续优化与验收的对照。

当前优先对象：

1. `/healthz`
2. `/verify`
3. `/verify/v2`
4. 核心 Rust 集成测试耗时

---

## 2. 测试环境

当前默认测试环境：

- 宿主机：Windows 本地开发环境
- 数据库：Podman 中 PostgreSQL
- 缓存：Podman 中 Redis
- API：本机运行 `rc-api`
- 可选降级：`RC_API_FALLBACK_STRATEGY=DirectPg`

---

## 3. 测试方法

### 3.1 健康检查

- 连续请求 `/healthz`
- 记录总耗时与均值

### 3.2 验真接口

- 连续请求 `/verify`
- 连续请求 `/verify/v2`
- 使用固定 UID / CTR / CMAC 样本
- 记录总耗时、平均耗时、近似 P95

### 3.3 写路径替代指标

Stage 5 当前用核心 Rust 集成测试作为写路径耗时近似观测：

- `activation_integration`
- `transfer_integration`

说明：

- 这不是最终生产压测
- 但可以作为当前本地联调可复测的最小基线

---

## 4. 指标记录模板

每次执行 `scripts/stage5-perf-baseline.*` 后，建议补录如下内容：

```text
日期：
执行人：
环境：Podman(PostgreSQL/Redis) + local rc-api
Fallback：Auto / DirectPg

/healthz:
- count:
- avg_ms:
- p95_ms:

/verify:
- count:
- avg_ms:
- p95_ms:

/verify/v2:
- count:
- avg_ms:
- p95_ms:

activation_integration:
- elapsed:

transfer_integration:
- elapsed:

观察到的瓶颈：
- 
```

---

## 5. 当前建议目标

Stage 5 推荐参考目标：

- `/verify` P95 ≤ 800ms
- 激活写入成功率 ≥ 99%
- 联调脚本连续运行无随机失败

说明：

本阶段若暂未达到目标，也必须先形成可复测结果，而不是没有基线。

---

## 6. 当前已知瓶颈方向

优先关注：

1. PostgreSQL 本地连接与 migration 启动耗时
2. Redis 不可用时的 DirectPg 降级成本
3. `/verify/v2` 相比 `/verify` 的额外承诺查询开销
4. 集成测试初始化数据库成本

---

## 7. 自动化入口

- `scripts/stage5-perf-baseline.sh`
- `scripts/stage5-perf-baseline.ps1`

---

## 8. 备注

本基线是 Stage 5 的第一版最小性能记录，不替代后续专门压测、并发测试和生产容量评估。
