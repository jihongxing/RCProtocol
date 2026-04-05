# RCP BConsole 信息架构、页面流与角色可见性矩阵

## 1. 角色定义（B端）

- `PlatformAdmin`：平台超级管理员（对应 Rust `Admin`）。
- `BrandAdmin`：品牌管理员（对应 Rust `Brand`）。
- `FactoryOperator`：工厂操作员（对应 Rust `Factory`）。
- `RiskOfficer`：风控专员（Rust 侧使用 `Admin` + 风控策略约束）。
- `Auditor`：审计员（Rust 侧使用 `Admin` 只读策略）。
- `PartnerOps`：生态合作方运营（Rust 侧 `Brand` 或受限委托 token）。

## 2. 控制台 IA（一级导航）

1. 总览驾驶舱
2. 组织与权限
3. 品牌与SKU主数据
4. 工厂与发行任务
5. 信任与策略中心
6. 资产主权与流转
7. 风险与争议
8. 审计与合规
9. 系统配置

## 3. 页面到 Rust 能力映射

| 菜单/页面 | 关键操作 | Rust接口映射 |
| --- | --- | --- |
| 总览驾驶舱 | 资产总量、激活率、风险告警 | `vault` `valuation` `credit` `legacy-gallery` |
| 组织列表 | 组织创建、岗位配置、成员分配 | Rust 认证 + `auth/revoke`（其余由Go治理库） |
| 角色策略页 | 角色绑定 API 范围 | `claims/require_role` + delegation scope |
| 品牌注册页 | 新品牌入驻 | `POST /v1/brand/register` |
| 品牌配额页 | 查看配额、资产分页 | `GET /v1/brand/quota` `/v1/brand/assets` |
| SKU中心 | 创建产品、多语翻译、版本发布 | `brand_products` 三类端点 |
| 批次管理 | 创建批次、暂停/恢复、统计 | `factory_batch` 全套端点 |
| 会话管理 | 开始/结束会话、会话追踪 | `factory_session` 全套端点 |
| 盲扫工位 | 扫码写入与结果回执 | `POST /v1/factory/blind-log` |
| 纠缠激活 | 母子绑定激活 | `POST /v1/brand/entangle-active` |
| 委托授权 | 签发/校验/吊销/轮换 delegation | `permission/delegation/*` |
| 使用权许可 | issue/revoke/list usage token | `permission` 三类端点 |
| 地理围栏策略 | 围栏创建、位置上报 | `geo-fence` 两类端点 |
| 再验证中心 | 发起 re-verify | `POST /v1/asset/:aid/re-verify` |
| 主权操作台 | 查询/锁定/释放主权 | `asset/:uid/sovereignty/*` |
| 资产转移台 | 发起/确认/取消转移 | `transfer` 三段端点 |
| 恢复中心 | recovery 五段流程 | `recovery` 五段端点 |
| 再生中心 | rebirth 三段流程+lineage | `rebirth` + `lineage` |
| 风险告警中心 | 黑名单、信用评分、异常工单 | `blacklist` `credit` + Go工单 |
| 审计检索 | 操作追踪、审计导出 | BusinessAudit + KMS Audit 聚合 |
| 系统参数 | 域名、语言、法律状态、属性词表 | `domains` `i18n` `legal` `attributes` |

## 4. 关键页面流（端到端）

### 4.1 品牌入驻到发布

1. `BrandAdmin` 提交品牌信息。
2. `PlatformAdmin` 审批通过。
3. 系统调用 Rust `brand/register`。
4. 自动跳转 SKU 初始化页，调用 `brand_products/create`。
5. 发布成功写入审计并进入“可发行”状态。

### 4.2 发行任务闭环

1. `FactoryOperator` 创建批次与会话。
2. 扫描写入 blind-log。
3. 品牌执行 entangle 激活。
4. 页面展示成功率、失败原因、补扫清单。

### 4.3 主权转移与恢复

1. `BrandAdmin/PartnerOps` 发起转移或恢复工单。
2. `RiskOfficer` 审核风险评分。
3. 系统推进 Rust 阶段接口。
4. `Auditor` 对关键操作进行复核与归档。

## 5. 角色可见性矩阵

| 菜单 | PlatformAdmin | BrandAdmin | FactoryOperator | RiskOfficer | Auditor | PartnerOps |
| --- | --- | --- | --- | --- | --- | --- |
| 总览驾驶舱 | R | R | R(受限) | R | R | R(受限) |
| 组织与权限 | RW | R(本组织) | - | R | R | - |
| 品牌与SKU | RW | RW(本品牌) | R(只读) | R | R | RW(受授权品牌) |
| 工厂与发行任务 | RW | RW | RW | R | R | R |
| 信任与策略中心 | RW | RW(受限) | - | RW | R | RW(受委托) |
| 资产主权与流转 | RW | RW | R | RW | R | RW(受限) |
| 风险与争议 | RW | R | R(告警只读) | RW | R | R |
| 审计与合规 | R | R(本品牌) | R(本人) | R | RW | R(本域) |
| 系统配置 | RW | R(局部) | - | R | R | - |

说明：

- `RW`：可读写；`R`：只读；`-`：不可见。
- 所有“写”动作必须触发审批流和审计日志。

## 6. 前端实现边界（可落地）

- 前端不直接调用 Rust，必须经 Go BFF。
- 前端只持有业务会话，不持有核心密钥或签名材料。
- 所有高风险按钮（rotation/revoke/recovery complete）必须二次确认并展示影响范围。

## 7. 验收要求（UI层）

- 每个页面都可追溯到至少一个 Rust 端点。
- 每个角色访问菜单时，展示内容与权限矩阵一致。
- 每个写操作均在“操作详情”页可查到审批号、trace_id、操作者和时间戳。
