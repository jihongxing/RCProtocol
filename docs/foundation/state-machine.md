# 状态机

> 文档类型：Foundation  
> 状态：Active  
> 权威级别：Authoritative

---

## 1. 权威说明

本文件是 RCProtocol 资产状态机的唯一文档定义源。

当前基线采用 **14 态模型**。任何旧文档中的 10 态、三态心智模型、扩展概念态，均不得替代本文件。

---

## 2. 完整状态表

| 状态码 | 枚举名 | 含义 | 分类 |
|--------|--------|------|------|
| 0 | `PreMinted` | 预铸造，UID 已录入但尚未进入供应链 | Virgin |
| 1 | `FactoryLogged` | 工厂已登记，盲扫入库完成 | Virgin |
| 10 | `Unassigned` | 已入库待品牌认领 | Virgin |
| 11 | `RotatingKeys` | 密钥轮换中，瞬态 | Enlightened |
| 12 | `EntangledPending` | 绑定待确认，瞬态 | Enlightened |
| 2 | `Activated` | 已激活，品牌绑定与密钥注入完成 | Enlightened |
| 3 | `LegallySold` | 合法售出，资产完成销售确权 | Tangled |
| 4 | `Transferred` | 已过户，可继续流转 | Tangled |
| 5 | `Consumed` | 已消耗，终态 | Terminal |
| 6 | `Legacy` | 遗珍，终态 | Terminal |
| 7 | `Tampered` | 已篡改，终态 | Terminal |
| 8 | `Compromised` | 已失陷，终态 | Terminal |
| 9 | `Destructed` | 已销毁，终态 | Terminal |
| 13 | `Disputed` | 争议冻结，可逆冻结态 | Frozen |

---

## 3. 状态分类

### 3.1 Virgin

- `PreMinted`
- `FactoryLogged`
- `Unassigned`

表示资产仍处于初始 / 待认领阶段。

### 3.2 Enlightened

- `RotatingKeys`
- `EntangledPending`
- `Activated`

表示资产正在或已经完成密钥注入与绑定建立。

### 3.3 Tangled

- `LegallySold`
- `Transferred`

表示资产已经完成销售或流转，进入持有关系有效期。

### 3.4 Terminal

- `Consumed`
- `Legacy`
- `Tampered`
- `Compromised`
- `Destructed`

终态不可逆。

### 3.5 Frozen

- `Disputed`

冻结态可恢复，但恢复必须经过治理流程。

---

## 4. 允许的核心转换

### 4.1 工厂盲扫路径

`PreMinted -> FactoryLogged -> Unassigned`

### 4.2 品牌激活路径

`Unassigned -> RotatingKeys -> EntangledPending -> Activated`

### 4.3 销售与流转路径

`Activated -> LegallySold -> Transferred`

`Transferred -> Transferred`

### 4.4 用户终态路径

`LegallySold -> Consumed | Legacy | Transferred`

`Transferred -> Consumed | Legacy | Transferred`

### 4.5 安全与治理路径

任何非终态可进入：

- `Tampered`
- `Compromised`
- `Disputed`

`Disputed` 可恢复至冻结前状态，或在审计结论下推进至 `Compromised`。

---

## 5. 状态机约束

### 5.1 终态不可逆

以下状态一旦进入，不允许再流转：

- `Consumed`
- `Legacy`
- `Tampered`
- `Compromised`
- `Destructed`

### 5.2 瞬态不可长期停留

以下状态是流程中间态：

- `RotatingKeys`
- `EntangledPending`

它们不应成为长期稳定业务状态。

### 5.3 冻结优先于业务动作

当资产处于 `Disputed` 时，禁止一切正常流转动作，直到恢复完成。

---

## 6. 与产品叙事的关系

以下表述只作为业务层叙事，不替代正式状态：

- 主权锁定
- 荣誉态
- 赋灵
- 纠缠

如果产品需要展示这些概念，应映射到正式状态或正式规则，而不是新增平行状态。

---

## 7. 清理结论

本次整理确认：

- 10 态版本废止
- 三态心智模型仅保留为分类辅助，不作为状态定义
- 任何新状态必须先修改本文件，再允许进入产品与工程文档
