# RCProtocol 设计文档模板体系

> 文档类型：Product  
> 状态：Active  
> 权威级别：Execution Template

---

## 1. 目标

本目录用于承载 RCProtocol 的页面 / 模块级设计文档模板。

它不重写 `foundation/` 中已经确定的状态、权限、安全和接口真相，而是把这些真相翻译为：

- 模块目标
- 用户路径
- 页面结构
- 状态展示
- 接口映射
- 审计要求
- 异常流处理
- 验收标准

---

## 2. 使用原则

1. 先确认需求是否已被 `docs/foundation/` 与 `docs/product/` 现有权威文档覆盖
2. 若只是某个 MVP 模块的页面 / 交互细化，可使用本目录模板落地
3. 模板中的状态、角色、动作名必须引用正式定义，不得新发明平行枚举
4. 若新增内容影响状态、权限、安全或接口语义，必须先更新上层 SSOT，再回填模板实例

---

## 3. 模板体系结构

- `00-module-design-template.md`：通用模块设计模板
- `01-brand-console.md`：品牌 / 产品管理模块模板实例
- `02-factory-flow.md`：工厂盲扫与批次 / 会话模块模板实例
- `03-activation.md`：资产认领与激活模块模板实例
- `04-verification.md`：C 端验真模块模板实例
- `05-legal-sell-vault.md`：合法售出 / 基础 Vault 模块模板实例
- `06-audit.md`：基础审计查询模块模板实例

---

## 4. 适用范围

本模板体系只覆盖当前 MVP 的 6 个核心模块：

1. Brand Console
2. Factory Flow
3. Activation
4. Verification
5. Legal Sell / Vault
6. Audit

不覆盖：

- 完整二级交易系统
- 荣誉态体系扩展
- 高级 Preview 分享模型
- 高级报表中心
- 多品牌复杂策略中心

---

## 5. 文档填写纪律

### 5.1 必填项

每个模块文档至少必须填写：

- 一句话商业价值
- 模块目标与边界
- 目标角色
- 最小用户路径
- 页面结构
- 状态映射
- 接口动作映射
- 异常流
- 审计要求
- 验收标准
- 不做什么

### 5.2 禁止事项

禁止在模板实例中：

- 重写状态机定义
- 重写角色权限定义
- 发明新的安全前提
- 直接复制历史归档叙事当现行规则
- 把视觉文案写成协议事实

---

## 6. 推荐使用流程

1. 先在 `foundation/` 确认状态、权限、安全与 API 边界
2. 再在 `product/` 确认 MVP 范围与主流程
3. 使用本目录模板填写模块设计
4. 模块文档确认后，再进入页面实现与联调

---

## 7. 关联文档

- `../mvp-scope-and-cutline.md`
- `../product-system.md`
- `../../foundation/state-machine.md`
- `../../foundation/roles-and-permissions.md`
- `../../foundation/api-and-service-boundaries.md`
- `../../engineering/spec-implementation-workflow.md`
