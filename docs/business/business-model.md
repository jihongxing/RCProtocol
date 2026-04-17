# 商业模型

> 文档类型：Business  
> 状态：Active  
> 权威级别：Authoritative

---

## 1. 商业定位

RCProtocol 的商业本质不是卖标签，而是围绕“资产确权、流转治理、品牌控制与可信验证”建立可持续收费体系。

它通过把实物资产接入统一协议，帮助品牌方、平台方和持有者在交易、治理与后续服务中建立长期价值关系。

---

## 2. 价值主张

### 2.1 对品牌方

- 获得统一资产身份体系
- 获得售后与二级流转中的持续控制力
- 获得生产、激活、销售、异常的治理视图
- 获得过去缺失的二级市场数据与服务入口

### 2.2 对平台方

- 成为协议与治理中枢
- 获取确权、流转与服务型收入
- 沉淀安全、审计与平台能力壁垒

### 2.3 对消费者 / 持有者

- 降低验真成本
- 提升流转信任
- 获得持续性的资产档案体验
- 获得更可控的资产处置路径

---

## 3. 收费结构

当前商业模型收敛为以下几类：

### 3.1 品牌接入与服务费

面向品牌方收取接入、治理、托管与持续服务相关费用。

### 3.2 生产 / 激活相关费用

围绕盲扫登记、身份激活、资产认领等环节形成单件或批次收费。

### 3.3 流转确权费

围绕合法流转、过户与处置过程收取服务费用，可采用固定底费 + 价值比例的方式。

### 3.4 增值服务费

包括但不限于：

- 风险恢复类服务
- 托管 / 释放类服务
- 展示与会员类服务
- 延展性商业能力

### 3.5 第一阶段推荐结算结构

第一阶段建议把结算结构收敛到品牌最容易理解、也最容易签约的三段：

1. **品牌接入费**：覆盖试点上线、配置、培训、联调与首批交付
2. **品牌年费**：覆盖品牌持续托管、运维、审计、后台与接口可用性
3. **流转确权费分成**：当资产发生合法流转、过户或后续处置时，按约定比例分账

对于流转确权费分成，当前建议可以采用：

- 平台 / 品牌 `50 / 50` 作为首批试点的默认商务谈判基线

但必须明确：

- 该比例是**商务策略基线**，不是协议常量
- 最终比例可按品类、品牌议价能力、平台承担义务、售后责任边界做调整
- 任何分账比例都不能反向改变协议状态、权限或真相定义

---

## 4. 动态定价原则

流转或确权类服务可以采用如下原则：

```text
Service Fee = min(Max Cap, Base + Market Value * Rate)
```

其中：

- `Base`：基础服务费
- `Rate`：价值费率
- `Max Cap`：封顶上限

具体费率不是协议常量，应由商业策略决定。

### 4.1 品牌端 ROI 计算口径

品牌端的 ROI 不应只看“付了多少系统费”，而应按以下口径统一：

```text
Brand Annual ROI
= (Dispute Savings
 + Customer Service Savings
 + Secondary Channel Recovery Value
 + Brand Share of Transfer Fees
 + Retained User Relationship Value)
 - (Integration Fee
 + Annual Fee
 + Internal Operating Cost)
```

推荐先跟品牌一起确认的变量包括：

- `Integration Fee`：品牌接入费
- `Annual Fee`：品牌年费
- `Internal Operating Cost`：品牌内部配合人力、培训与流程改造成本
- `Dispute Savings`：因真假争议、售后扯皮、客服处理减少的成本
- `Customer Service Savings`：门店、客服、售后人工判断成本下降
- `Secondary Channel Recovery Value`：品牌拿回二手流转入口后新增的可分账收入或渠道控制价值
- `Brand Share of Transfer Fees`：品牌在流转确权费中的分成收入
- `Retained User Relationship Value`：品牌重新拿回售出后用户与资产的关系入口所带来的长期价值

### 4.2 平台端年度收入口径

平台端不应只看一次性项目收入，而应按年度口径统一：

```text
Platform Annual Revenue
= Integration Fee
 + Annual Fee
 + Activation Fees
 + Platform Share of Transfer Fees
 + Managed Service Fees
```

其中：

- `Integration Fee`：品牌接入费
- `Annual Fee`：品牌年费
- `Activation Fees`：按件或按批次计费的激活收入
- `Platform Share of Transfer Fees`：平台在流转确权费中的分成收入
- `Managed Service Fees`：平台陪跑、运营托管、异常处理等服务收入

### 4.3 首批试点品牌的简化报价框架

首批试点建议统一按以下顺序报价：

1. 先报接入费，覆盖试点上线与首批交付
2. 再报年费，覆盖持续运行与托管
3. 最后再报流转确权费分成，作为长期共赢条款而不是首单成交门槛

这样做的原因是：

- 品牌更容易理解和审批
- 更容易把首单目标收敛到“先上线、先跑通、先验证”
- 流转确权费分成可以在试点后作为续约与扩量谈判筹码

### 4.4 品牌试点测算表（可报价模板）

为避免销售阶段只讲概念，首批品牌试点建议统一使用下表测算。

| 项目 | 变量 | 示例值 | 说明 |
|---|---|---:|---|
| 品牌接入费 | `Integration Fee` | 80,000 | 覆盖试点上线、联调、培训与首批交付 |
| 品牌年费 | `Annual Fee` | 120,000 | 覆盖后台、接口、运维、审计与持续托管 |
| 首批激活量 | `Activated Units` | 2,000 | 首批试点导入并完成激活的资产数量 |
| 单件激活费 | `Activation Fee Per Unit` | 8 | 按件计费，可换成按批次计费 |
| 年度有效流转量 | `Transfer Count` | 300 | 首批品牌一年内发生的合法流转次数 |
| 单笔流转均价 | `Avg Transfer GMV` | 3,000 | 用于估算价值比例收费 |
| 流转服务底费 | `Transfer Base Fee` | 39 | 每笔流转基础收费 |
| 流转价值费率 | `Transfer Rate` | 1.0% | 按成交价值抽取的服务费率 |
| 品牌分成比例 | `Brand Share` | 50% | 当前首批试点默认谈判基线 |
| 平台分成比例 | `Platform Share` | 50% | 当前首批试点默认谈判基线 |
| 品牌内部配合成本 | `Internal Operating Cost` | 60,000 | 品牌内部对接、培训、流程配合的人力成本 |
| 假货 / 争议处理年损失 | `Dispute Savings` | 120,000 | 若项目生效后预计可减少的损失 |
| 客服 / 门店判断年成本节省 | `Customer Service Savings` | 80,000 | 由标准化验真与登记带来的节省 |
| 二手渠道回收价值 | `Secondary Channel Recovery Value` | 180,000 | 品牌拿回流转入口后的年度保守价值 |
| 售后关系沉淀价值 | `Retained User Relationship Value` | 100,000 | 品牌重建售出后用户关系的保守估值 |

基于上表，可先用如下方式估算：

```text
Activation Revenue
= Activated Units * Activation Fee Per Unit

Transfer Fee Per Order
= Transfer Base Fee + Avg Transfer GMV * Transfer Rate

Annual Transfer Fee Pool
= Transfer Count * Transfer Fee Per Order

Brand Annual Cash Out
= Integration Fee + Annual Fee + Internal Operating Cost

Brand Annual Cash In
= Dispute Savings
 + Customer Service Savings
 + Secondary Channel Recovery Value
 + Retained User Relationship Value
 + Annual Transfer Fee Pool * Brand Share

Platform Annual Revenue
= Integration Fee
 + Annual Fee
 + Activation Revenue
 + Annual Transfer Fee Pool * Platform Share
```

按上面的示例值计算：

```text
Activation Revenue = 2,000 * 8 = 16,000
Transfer Fee Per Order = 39 + 3,000 * 1.0% = 69
Annual Transfer Fee Pool = 300 * 69 = 20,700
Brand Annual Cash Out = 80,000 + 120,000 + 60,000 = 260,000
Brand Annual Cash In = 120,000 + 80,000 + 180,000 + 100,000 + 20,700 * 50%
                     = 490,350
Brand Annual Net Value = 490,350 - 260,000 = 230,350
Platform Annual Revenue = 80,000 + 120,000 + 16,000 + 20,700 * 50%
                        = 226,350
```

这个示例的意义不是给所有品牌统一报价，而是：

- 提供一张统一的商务测算骨架
- 让品牌方理解“我们付出的是什么，换回来的是什么”
- 让平台内部可以快速判断单品牌试点是否值得推进

### 4.5 报价使用原则

使用试点测算表时，必须遵守以下原则：

1. 示例值只能作为首轮商务沟通模板，正式报价必须按目标品类和品牌体量重算。
2. `Secondary Channel Recovery Value` 与 `Retained User Relationship Value` 必须采用保守口径，避免把想象力收入直接写进报价。
3. 流转服务费率和分成比例必须在商务条款中单独确认，不能在销售沟通中被表述成协议固定规则。
4. 若品牌当前尚无真实二手流转数据，应优先用低、中、高三档情景测算，而不是只给单点乐观值。

---

## 5. 商业边界

### 5.1 当前确认纳入的商业能力

- 品牌接入收入
- 激活 / 确权收入
- 流转收入
- 运维与托管相关服务收入

### 5.2 当前不作为真源承诺的能力

以下内容可保留为未来战略方向，但当前不作为项目真源承诺：

- 泛金融化叙事
- 大规模画像售卖
- 未落地的全球清算体系
- 尚未实现的通用资产金融产品

---

## 6. 与产品和协议的关系

商业模式不能反向篡改协议真源。

也就是说：

- 收费模式不能发明新状态
- 商业包装不能替代权限边界
- 愿景叙事不能视为已实现能力

产品能力是否收费，建立在已有协议能力和工程能力之上。

---

## 7. 本次整理结论

本次整理后：

- 商业白皮书、专题商业模型、融资导向表述被收敛为单一商业文档
- 对外叙事仍可继续使用，但不再进入核心协议与工程真源层
- 商业模型只保留对当前开发与系统规划真正有约束力的内容

---

## 8. 关联文档

- 项目总览：`../foundation/project-overview.md`
- 产品体系：`../product/product-system.md`
- 服务边界：`../foundation/api-and-service-boundaries.md`
