# Spec：Verification V2

> 文档类型：Spec  
> 状态：Draft  
> 权威级别：Implementation Proposal  
> 最后更新：2026-04-09

---

## 1. 目标

本 Spec 定义 `Verification V2`，作为 RCProtocol 从“当前 MVP 验真实现”升级到“终局协议验真定义”的第一版协议接口与判定模型。

它的核心目标是把验真从：

- 芯片证明 + 数据库上下文 + 平台 KMS

升级为：

- 芯片证明 + `AssetCommitment` + `Brand Attestation` + `Platform Attestation`

---

## 2. 当前 V1 与目标 V2 的区别

### 2.1 当前 V1

当前验真链路：

1. 读取 `uid / ctr / cmac`
2. 查询 `assets`
3. 获取 `brand_id / epoch / current_state`
4. 平台 KMS 派生 `K_chip`
5. 验证 `CMAC`
6. 检查 `CTR`
7. 返回验真结果

### 2.2 目标 V2

目标验真链路：

1. 读取 `uid / ctr / cmac`
2. 完成标签动态证明校验
3. 定位 `AssetCommitment`
4. 校验 `Brand Attestation`
5. 校验 `Platform Attestation`
6. 检查协议状态与风险规则
7. 输出终局协议意义上的验真结论

---

## 3. 验真结论分层

### 3.1 V2 必须区分三类结论

#### A. 标签真实性结论

回答：

- 这是不是一个通过了芯片动态认证的标签

#### B. 协议承诺完整性结论

回答：

- 这个标签对应的协议对象是否同时具备品牌承诺与平台承诺

#### C. 协议状态结论

回答：

- 该资产当前是否处于允许正常展示的协议状态

### 3.2 不再允许单一结论偷换全部含义

V2 不应再把：

- “CMAC 通过”
- “数据库里有记录”
- “前端展示正常”

其中任一项，偷换成完整“真品成立”。

---

## 4. 输入定义

### 4.1 请求参数

第一版建议保留与 V1 接近的输入：

- `uid`
- `ctr`
- `cmac`

后续可扩展：

- `enc`
- `sdm_meta`
- `client_context`

### 4.2 规范化要求

- `uid` 使用大写十六进制解析
- `ctr` 使用固定长度十六进制
- `cmac` 使用固定长度十六进制

---

## 5. 处理流程

### 5.1 Step 1：标签动态证明校验

执行：

- `uid / ctr / cmac` 解析
- 芯片密钥定位或恢复
- `CMAC` 校验
- `CTR` replay 检查

若失败，直接返回：

- `tag_authentication_failed`
- `replay_suspected`
- `unknown_tag`

### 5.2 Step 2：定位 `AssetCommitment`

在标签动态证明通过后，定位对应 `AssetCommitment`。

建议路径：

- 先通过 `uid + epoch` 或桥接映射定位
- 取得 `asset_commitment_id`

### 5.3 Step 3：校验 `Brand Attestation`

检查：

- 是否存在品牌承诺
- 品牌承诺签名是否有效
- 品牌承诺是否指向当前 `asset_commitment_id`

### 5.4 Step 4：校验 `Platform Attestation`

检查：

- 是否存在平台承诺
- 平台承诺签名是否有效
- 平台承诺是否指向当前 `asset_commitment_id`

### 5.5 Step 5：状态与风险检查

检查：

- `current_state`
- 风险标记
- 冻结 / 终态限制

### 5.6 Step 6：输出验真结果

输出必须包含：

- 标签真实性状态
- 承诺完整性状态
- 协议状态
- 最终验真结论

---

## 6. 响应结构建议

### 6.1 响应示例

```json
{
  "verification_version": "v2",
  "tag_authentication": "passed",
  "attestation_status": {
    "asset_commitment_id": "...",
    "brand_attestation": "valid",
    "platform_attestation": "valid"
  },
  "protocol_state": {
    "current_state": "Activated",
    "risk_flags": []
  },
  "verification_status": "authentic"
}
```

### 6.2 字段说明

- `verification_version`：固定为 `v2`
- `tag_authentication`：`passed / failed / replay_suspected / unknown_tag`
- `attestation_status.brand_attestation`：`valid / missing / invalid`
- `attestation_status.platform_attestation`：`valid / missing / invalid`
- `protocol_state.current_state`：当前协议状态
- `verification_status`：最终结论

---

## 7. 最终结论规则

### 7.1 `authentic`

必须同时满足：

1. 标签动态证明通过
2. `Brand Attestation` 有效
3. `Platform Attestation` 有效
4. 协议状态允许正常展示

### 7.2 `restricted`

以下任一满足：

- 标签证明通过
- 双承诺有效
- 但状态处于 `Disputed / Tampered / Compromised / Destructed` 等限制态

### 7.3 `incomplete_attestation`

以下任一满足：

- 标签证明通过
- 但品牌承诺缺失 / 无效
- 或平台承诺缺失 / 无效

该状态非常关键，它表达：

> 当前标签真实性成立，但终局协议真品定义尚未成立。

### 7.4 `authentication_failed`

标签动态证明失败。

### 7.5 `unknown_tag`

找不到标签映射或承诺对象。

---

## 8. 与当前 V1 的兼容策略

### 8.1 双接口并存

建议保留：

- `GET /verify` 作为 V1
- `GET /verify/v2` 作为 V2

### 8.2 渐进迁移

第一阶段：

- V1 继续服务现有前端
- V2 用于协议升级验证与灰度联调

第二阶段：

- 前端与品牌 API 可逐步切换到 V2

### 8.3 不建议直接替换

不建议立刻把现有 `/verify` 直接改造成 V2，以免打断当前 MVP 链路。

---

## 9. 审计建议

V2 验真事件建议记录：

- `asset_commitment_id`
- `brand_attestation_status`
- `platform_attestation_status`
- `verification_version`
- `final_verification_status`

---

## 10. 错误码建议

- `ASSET_COMMITMENT_NOT_FOUND`
- `BRAND_ATTESTATION_MISSING`
- `BRAND_ATTESTATION_INVALID`
- `PLATFORM_ATTESTATION_MISSING`
- `PLATFORM_ATTESTATION_INVALID`
- `REPLAY_SUSPECTED`
- `TAG_AUTHENTICATION_FAILED`

---

## 11. 验收标准

本 Spec 最小验收标准：

1. V2 能独立返回结构化验真结论
2. 能区分“标签真实但承诺不完整”与“终局协议真品成立”
3. 能同时展示品牌承诺与平台承诺状态
4. 不破坏现有 V1 验真链路

---

## 12. 非目标

本 Spec 当前不解决：

- 去中心化网络传播
- 跨平台互认
- MPC / 联合派生的最终接入方式
- 前端最终 UI 细节

---

## 13. 关联 Specs

- `spec-asset-commitment.md`
- `spec-brand-attestation.md`
- `spec-platform-attestation.md`
