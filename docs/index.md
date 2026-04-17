# RCProtocol 文档中心

> 本目录是 **RCProtocol 的唯一文档源（Single Source of Truth, SSOT）**。
> 除 `archive/` 历史归档外，项目中的产品、协议、工程、运维与商业定义均以本目录中的当前结构为准。

---

## 1. 项目是什么

RCProtocol 是一个面向高价值实物资产的数字主权协议与平台。

它以 `NTAG 424 DNA` 等物理标签为锚点，通过：

- 资产唯一身份
- 母子标签纠缠 / 授权绑定
- KDF 密钥派生与动态认证
- 全生命周期状态机
- 平台 / 品牌 / 工厂 / 用户 / 审核员多角色治理
- B 端治理后台与 C 端验证 / 藏馆 / 流转能力

实现对实物资产的：

- 防伪验真
- 激活确权
- 权属流转
- 风险冻结
- 运维审计
- 商业清算

---

## 2. 文档权威规则

### 2.1 唯一文档源原则

本目录下除 `archive/` 外的文档，构成当前唯一有效文档源。

### 2.2 优先级

1. `foundation/`：协议、状态、权限、安全、边界的最终定义
2. `engineering/`：系统实现、服务拆分、运行与运维基线
3. `product/`：产品能力、用户流程、交互与业务旅程
4. `business/`：商业模式、合作策略、战略边界
5. `ops/`：生产运维手册与故障处理流程
6. `archive/`：历史方案、讨论稿、旧版本，不可作为当前实现依据

### 2.3 当前唯一文档集合

以下文件组成当前唯一有效文档体系：

- `index.md`
- `foundation/project-overview.md`
- `foundation/domain-model.md`
- `foundation/state-machine.md`
- `foundation/roles-and-permissions.md`
- `foundation/protocol-definition.md`
- `foundation/security-model.md`
- `foundation/protocol-gap-analysis.md`
- `foundation/target-protocol-architecture.md`
- `foundation/api-and-service-boundaries.md`
- `engineering/system-architecture.md`
- `engineering/technical-solution.md`
- `engineering/service-split-and-repo-layout.md`
- `engineering/spec-implementation-workflow.md`
- `engineering/hardware-and-ops-baseline.md`
- `product/product-system.md`
- `product/mvp-scope-and-cutline.md`
- `product/commercial-scenario-and-buyer.md`
- `business/business-model.md`
- `ops/` 下运维手册
- `ops/stage-5-mvp-runbook.md`
- `ops/stage-5-error-matrix.md`
- `ops/stage-5-performance-baseline.md`
- `ops/stage-5-mvp-acceptance.md`

以下文件属于 **当前协议演进实现提案（Draft Specs）**，可作为后续 Stage 7~10 的实现设计输入，但尚未替代现有落地基线：

- `specs/spec-asset-commitment.md`
- `specs/spec-brand-attestation.md`
- `specs/spec-platform-attestation.md`
- `specs/spec-verification-v2.md`
- `specs/spec-stage-5-mvp-delivery.md`（其中 Stage 5 主链路已明确为两阶段激活：`/activate` 负责承诺/声明，`/activate-entangle` 负责虚拟母卡/母子绑定）

以下文件属于 **当前协议演进任务拆解（Implementation Tasks）**，用于把 Draft Specs 进一步转成可排期、可编码、可验收的执行任务：

- `tasks/task-asset-commitment-migration.md`
- `tasks/task-attestation-flows.md`
- `tasks/task-verification-v2-implementation.md`
- `tasks/task-stage-5-mvp-delivery.md`

以下文件属于 **代码级实施计划（Code Plans）**，用于把任务文档进一步映射到当前仓库的模块、函数、migration 与测试文件：

- `tasks/code-plan-asset-commitment.md`
- `tasks/code-plan-attestations-and-verify-v2.md`

### 2.4 开发落地顺序

为避免文档再次发散，后续开发默认按以下顺序落地：

1. `foundation/` 先定义规则
2. `product/` 再定义页面与业务流程映射
3. `engineering/` 再定义系统实现与运行方式
4. `ops/` 最后定义运维执行与恢复流程

任何新需求若涉及状态、权限、安全或接口语义，必须先更新 `foundation/`，再进入产品与工程文档。

### 2.5 单一定义原则

以下主题只允许在指定文档中定义一次，其余文档只能引用：

- 项目定位与边界：`foundation/project-overview.md`
- 术语与核心对象：`foundation/domain-model.md`
- 状态机：`foundation/state-machine.md`
- 角色与权限：`foundation/roles-and-permissions.md`
- 协议术语定义：`foundation/protocol-definition.md`
- 安全与密钥体系：`foundation/security-model.md`
- 当前实现与目标协议差距：`foundation/protocol-gap-analysis.md`
- 目标协议终局架构：`foundation/target-protocol-architecture.md`
- 服务与 API 边界：`foundation/api-and-service-boundaries.md`
- 工程基线：`engineering/system-architecture.md`
- 技术选型与分层实现方案：`engineering/technical-solution.md`
- 服务拆分与仓库布局：`engineering/service-split-and-repo-layout.md`
- spec-driven 实施工作流：`engineering/spec-implementation-workflow.md`
- 产品主流程：`product/product-system.md`
- MVP 范围与切线：`product/mvp-scope-and-cutline.md`
- 商业场景与买家：`product/commercial-scenario-and-buyer.md`
- 商业模型：`business/business-model.md`

### 2.6 归档原则

`archive/` 中保留历史材料、早期构想、被替代文档、提案与讨论记录。

规则：

- 归档文档不再作为开发依据
- 如与当前文档冲突，以本目录当前体系为准
- 后续若保留旧稿，统一放入 `archive/`

---

## 3. 阅读路径

### 面向新成员

建议按顺序阅读：

1. `foundation/project-overview.md`
2. `foundation/domain-model.md`
3. `foundation/state-machine.md`
4. `foundation/roles-and-permissions.md`
5. `foundation/protocol-definition.md`
6. `foundation/security-model.md`
7. `foundation/protocol-gap-analysis.md`
8. `foundation/target-protocol-architecture.md`
9. `foundation/api-and-service-boundaries.md`
10. `product/mvp-scope-and-cutline.md`
11. `product/commercial-scenario-and-buyer.md`
12. `engineering/system-architecture.md`
13. `engineering/technical-solution.md`
14. `engineering/service-split-and-repo-layout.md`
15. `engineering/spec-implementation-workflow.md`
16. `product/product-system.md`

### 面向研发

1. `foundation/protocol-definition.md`
2. `foundation/state-machine.md`
3. `foundation/security-model.md`
4. `foundation/api-and-service-boundaries.md`
5. `foundation/protocol-gap-analysis.md`
6. `foundation/target-protocol-architecture.md`
7. `product/mvp-scope-and-cutline.md`
8. `product/commercial-scenario-and-buyer.md`
9. `engineering/system-architecture.md`
10. `engineering/technical-solution.md`
11. `engineering/service-split-and-repo-layout.md`
12. `engineering/spec-implementation-workflow.md`
13. `specs/spec-asset-commitment.md`
14. `specs/spec-brand-attestation.md`
15. `specs/spec-platform-attestation.md`
16. `specs/spec-verification-v2.md`
17. `tasks/task-asset-commitment-migration.md`
18. `tasks/task-attestation-flows.md`
19. `tasks/task-verification-v2-implementation.md`
20. `tasks/code-plan-asset-commitment.md`
21. `tasks/code-plan-attestations-and-verify-v2.md`
22. `ops/`

### 面向产品 / 设计

1. `foundation/project-overview.md`
2. `foundation/protocol-definition.md`
3. `foundation/protocol-gap-analysis.md`
4. `foundation/target-protocol-architecture.md`
5. `product/mvp-scope-and-cutline.md`
6. `product/commercial-scenario-and-buyer.md`
7. `foundation/domain-model.md`
8. `product/product-system.md`
9. `business/business-model.md`
10. `engineering/spec-implementation-workflow.md`

### 面向合作 / 商业 / 对外方案

1. `foundation/project-overview.md`
2. `product/mvp-scope-and-cutline.md`
3. `product/commercial-scenario-and-buyer.md`
4. `product/product-system.md`
5. `business/business-model.md`

---

## 4. 目录结构

```text
docs/
├─ index.md
├─ foundation/
│  ├─ project-overview.md
│  ├─ domain-model.md
│  ├─ state-machine.md
│  ├─ roles-and-permissions.md
│  ├─ protocol-definition.md
│  ├─ security-model.md
│  ├─ protocol-gap-analysis.md
│  ├─ target-protocol-architecture.md
│  └─ api-and-service-boundaries.md
├─ engineering/
│  ├─ system-architecture.md
│  ├─ technical-solution.md
│  ├─ service-split-and-repo-layout.md
│  ├─ spec-implementation-workflow.md
│  └─ hardware-and-ops-baseline.md
├─ product/
│  ├─ product-system.md
│  ├─ mvp-scope-and-cutline.md
│  └─ commercial-scenario-and-buyer.md
├─ specs/
│  ├─ spec-04-brand-registration.md
│  ├─ spec-05-activation-chain.md
│  ├─ spec-06-approval-workflow.md
│  ├─ spec-06-id-unification.md
│  ├─ spec-asset-commitment.md
│  ├─ spec-brand-attestation.md
│  ├─ spec-platform-attestation.md
│  └─ spec-verification-v2.md
├─ tasks/
│  ├─ task-17-implementation-status.md
│  ├─ task-spec-04-brand-registration.md
│  ├─ task-spec-06-id-unification.md
│  ├─ task-asset-commitment-migration.md
│  ├─ task-attestation-flows.md
│  ├─ task-verification-v2-implementation.md
│  ├─ code-plan-asset-commitment.md
│  └─ code-plan-attestations-and-verify-v2.md
├─ business/
│  └─ business-model.md
├─ ops/
│  └─ ...
└─ archive/
   └─ ...
```

---

## 5. 当前整理结论

本次整理后：

- 旧的重复总纲、整理版、专题汇编已退出当前文档体系
- 顶层只保留新的统一结构
- `ops/` 保留运维手册
- `archive/` 保留历史材料

### 当前实现进展（摘录）

- `Spec-04: 品牌极简注册与 API Key 管理`：✅ 已落地实现
  - 已完成品牌注册、API Key 轮换、品牌详情、API Key 列表
  - 已补齐 migration、集成测试与联调脚本
  - 标准联调入口：`scripts/test-brand-registration.sh` / `scripts/test-brand-registration.ps1`

后续新增文档时，必须先判断其归属层级；若内容已在现有权威文档中定义，则应修改原文档，而不是新增重复文档。
