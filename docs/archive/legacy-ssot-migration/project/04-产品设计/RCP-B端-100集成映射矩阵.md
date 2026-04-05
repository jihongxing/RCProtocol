# RCP B端 100%集成映射矩阵（Rust核心对齐）

## 1. 审计口径

- 目标：证明 B 端能力均可通过现有 `rc-api`/`rc-kms`/`rc-core` 落地，不新增平行核心引擎。
- 判定标准：
  - `直接复用`：现有 endpoint/service 可直接支撑。
  - `在位扩展`：仅在现有 crate 内新增模块或 endpoint，不新建核心协议服务。
- 基线代码：
  - 路由基线：`crates/rc-api/src/api/mod.rs`
  - 应用状态基线：`crates/rc-api/src/state.rs`
  - KMS状态与服务基线：`crates/rc-kms/src/state.rs`
  - 信任策略基线：`crates/rc-core/src/trust/mod.rs`

## 2. 域能力 -> 现有代码映射

| B端能力域 | 现有入口（API/Service） | 现有支撑度 | 在位扩展点（必须在现有crate中） |
|---|---|---|---|
| 品牌注册与配额 | `POST /v1/brand/register` `GET /v1/brand/quota` `GET /v1/brand/assets`；`BrandManagementService` | 直接复用 | 增加品牌状态机（draft/active/suspended）到 `rc-api` 的 brand 模块 |
| SKU/产品主数据 | `POST/GET/PUT /v1/brand/:brand_id/products...`；`BrandProductService` | 直接复用 | 新增版本发布/回滚 endpoint 于 `rc-api/api/brand_products.rs` |
| 工厂批次与会话 | `factory_batch` + `factory_session` 路由；`BatchService` `OperatorSessionService` | 直接复用 | 增加批次审批引用字段（approval_id） |
| 盲扫入库与纠缠激活 | `POST /v1/factory/blind-log` `POST /v1/brand/entangle-active`；`rc-kms` blind_log/entangle | 直接复用 | 增加治理侧幂等键传递（header/claim） |
| 委托授权（Hybrid信任） | `/v1/permission/delegation/{issue,verify,revoke,rotate}`；`DelegationTokenService` | 直接复用 | 增加 scope 模板化策略与审批来源字段 |
| 使用权许可 | `/v1/permission/issue` `/revoke` `/asset/:aid`；`PermissionService` | 直接复用 | 增加企业策略约束（时间窗、地域、次数） |
| 主权锁与资产操作 | `/v1/asset/:uid/sovereignty/*` `/consume`；`SovereigntyService` | 直接复用 | 增加“组织级二次确认”钩子 |
| 转移/恢复/再生 | `transfer` `recovery` `rebirth` 全链路端点 | 直接复用 | 增加争议工单ID贯穿字段 |
| 再验证与风控 | `/v1/asset/:aid/re-verify` `geo-fence` `blacklist` `credit` | 直接复用 | 在 `RiskEngine` 增加策略版本号 |
| 审计与不可抵赖 | `BusinessAuditStore` + `rc-kms::AuditStore` + delegation审计事件 | 直接复用 | 扩展审计事件类型：审批、回滚、策略发布 |
| 协议参数治理 | `rc-kms::config`（epoch/system_id/...） + `rc-core::trust` | 在位扩展 | 在 `rc-api` 增加“治理参数发布”入口，最终写入现有配置通道 |
| 密钥仪式与设备可信 | `rc-kms` device/ceremony API + `KeyCeremonyService` | 直接复用 | 增加 B 端可见性聚合查询，不改底层仪式逻辑 |
| 多语言、法律语义 | `i18n` `legal` `attributes` 公共端点 | 直接复用 | 增加企业私有词条覆盖层（在 `rc-api` 追加） |
| Vault与资产视图 | `/v1/vault/*` `legacy-gallery` `valuation` | 直接复用 | 增加品牌级聚合看板查询 endpoint |

## 3. Go治理后端到Rust核心调用矩阵

| Go治理服务 | 调用Rust入口 | 数据一致性策略 | 回滚策略 |
|---|---|---|---|
| Org/IAM编排 | `rc-api` JWT认证与角色守卫（claims/require_role） | 外层RBAC + 内层角色双重校验 | 先撤销外层token，再撤销内层授权 |
| 审批流引擎 | `brand` `brand_products` `permission` `delegation` 端点 | 审批单号作为幂等键写入审计扩展字段 | 审批撤销触发反向API（revoke/suspend） |
| 策略中心 | `permission` `delegation` `geo-fence` `re-verify` | 策略版本号随请求下发 | 策略回退到上一版本并冻结新写入 |
| 运营工单中心 | `recovery` `rebirth` `transfer` | 工单ID贯穿 trace 字段 | 使用状态机补偿（cancel/revoke） |
| 报表中心 | `vault` `credit` `valuation` + 审计读模型 | T+0读模型，T+1归档 | 报表层可重放，不影响核心账本 |

## 4. 缺口与实现边界（可落地约束）

- 必须新增但不越界的内容：
  - `rc-api`：治理编排端点（审批引用、策略版本、组织上下文）。
  - `rc-kms`：审计事件类型和查询聚合扩展。
  - `rc-core`：仅补充 trust policy 参数校验，不新增业务流程。
- 禁止项（确保100%集成）：
  - 禁止在 Go 平面实现 KDF、签名验签、EV2、密钥派生。
  - 禁止在 Go 平面维护资产主状态真源。
  - 禁止复制一套“新KMS”或“新协议引擎”。

## 5. 集成完成判定

- 每个 B 端菜单都能落到“至少一个现有 Rust endpoint/service”。
- 每个在位扩展点都明确归属 `rc-api`/`rc-kms`/`rc-core` 之一。
- 所有核心链路（blind-log/entangle/verify/transfer/recovery）仅由 Rust 执行。
