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
- `foundation/security-model.md`
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
- 安全与密钥体系：`foundation/security-model.md`
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
5. `foundation/security-model.md`
6. `product/mvp-scope-and-cutline.md`
7. `product/commercial-scenario-and-buyer.md`
8. `engineering/system-architecture.md`
9. `engineering/technical-solution.md`
10. `engineering/service-split-and-repo-layout.md`
11. `engineering/spec-implementation-workflow.md`
12. `product/product-system.md`

### 面向研发

1. `foundation/state-machine.md`
2. `foundation/security-model.md`
3. `foundation/api-and-service-boundaries.md`
4. `product/mvp-scope-and-cutline.md`
5. `product/commercial-scenario-and-buyer.md`
6. `engineering/system-architecture.md`
7. `engineering/technical-solution.md`
8. `engineering/service-split-and-repo-layout.md`
9. `engineering/spec-implementation-workflow.md`
10. `ops/`

### 面向产品 / 设计

1. `foundation/project-overview.md`
2. `product/mvp-scope-and-cutline.md`
3. `product/commercial-scenario-and-buyer.md`
4. `foundation/domain-model.md`
5. `product/product-system.md`
6. `business/business-model.md`
7. `engineering/spec-implementation-workflow.md`

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
│  ├─ security-model.md
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

后续新增文档时，必须先判断其归属层级；若内容已在现有权威文档中定义，则应修改原文档，而不是新增重复文档。
