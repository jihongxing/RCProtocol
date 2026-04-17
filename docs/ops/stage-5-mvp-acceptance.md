# Stage 5 MVP 验收清单与结论

> 文档类型：Ops  
> 适用阶段：Stage 5 MVP 可交付闭环  
> 最后更新：2026-04-10

---

## 1. 目的

本文件用于把 Stage 5 的验收对象从“口头判断”转成固定清单，并给出当前阶段结论。

---

## 2. 技术验收清单

- [x] PostgreSQL / Redis / `rc-api` 最小运行环境已固定
- [x] 已补 Stage 5 本地运行与联调手册
- [x] 品牌 API 闭环有脚本与集成测试
- [x] `/verify` 与 `/verify/v2` 有联调脚本与集成测试
- [x] 激活 / 承诺 / 资产详情链路有集成测试
- [x] 激活绑定 / 虚拟母卡链路有集成测试
- [x] 转移链路有集成测试
- [x] 异常流矩阵已固定
- [x] 性能基线模板已固定

---

## 3. 产品演示清单

- [x] 可演示品牌注册与 API Key 轮换
- [x] 可演示 `/verify` 与 `/verify/v2` 的差异
- [x] 可演示两阶段激活：承诺/声明生成与虚拟母卡绑定分离
- [x] 可演示资产详情与承诺状态读取
- [x] 可演示转移主流程与 reject / confirm 异常场景
- [x] 可解释 `authentication_failed` / `replay_suspected` / `incomplete_attestation`

---

## 4. 品牌试点预演清单

- [x] 平台可创建品牌
- [x] 平台可签发 API Key
- [x] 品牌可使用 API Key 读取品牌资源
- [x] 品牌可进行 API Key 轮换
- [x] 品牌接入脚本可复用

---

## 5. 异常流验收清单

- [x] 已覆盖认证失败
- [x] 已覆盖 replay suspected
- [x] 已覆盖 incomplete attestation
- [x] 已覆盖 API Key 失效
- [x] 已覆盖 transfer reject / conflict
- [x] 已覆盖冻结 / 恢复限制行为

---

## 6. 当前结论

### 6.1 已完成内容

Stage 5 当前已完成以下收口：

1. 已补齐 Stage 5 运行手册
2. 已固定 Podman + PostgreSQL + Redis 的本地环境说明
3. 已收口激活两阶段语义：`/activate` 负责承诺/声明，`/activate-entangle` 负责虚拟母卡/母子绑定
4. 已补品牌 API 联调脚本与集成测试基线引用
5. 已补 V1 / V2 验真并行联调入口
6. 已补异常流矩阵
7. 已补性能基线文档与采集脚本入口
8. 已补 MVP 验收清单

### 6.2 当前判断

在当前文档、脚本、测试与运行手册范围内，Stage 5 可判断为：

> **已形成“可交付 MVP 的第一版验收闭环”。**

它表示：

- MVP 的最小环境可说明
- 主链路与关键链路有自动化入口
- 品牌接入与验真升级能力可并行演示
- 异常流、运行手册、性能基线不再缺位

但它 **不表示**：

- 终局协议已完成
- 高可用生产交付已完成
- 完整品牌试点运营流程已全部产品化

### 6.3 后续建议

Stage 5 完成后，建议优先进入：

1. 持续补 CI 回归
2. 品牌试点预演与真实数据联调
3. Stage 6 试点交付验证
4. 在真实反馈下再推进 Stage 7~10

---

## 7. 关联文件

- `docs/specs/spec-stage-5-mvp-delivery.md`
- `docs/tasks/task-stage-5-mvp-delivery.md`
- `docs/ops/stage-5-mvp-runbook.md`
- `docs/ops/stage-5-error-matrix.md`
- `docs/ops/stage-5-performance-baseline.md`
