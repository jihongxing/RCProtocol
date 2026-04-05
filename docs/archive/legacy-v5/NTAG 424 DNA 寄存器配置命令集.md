这份文档是整个项目的**“生产协议”**。它的作用是告诉读写器，如何把一枚通用的空白芯片，正式转化为属于 V5.0 协议的**“资产主权密钥”**。

这套命令集的核心在于启用 **SDM (Secure Dynamic Messaging)** 模式，让 URL 的 `cmac` 和 `ctr` 字段在感应时自动生成。

---

# 《V5.0 协议：NTAG 424 DNA 寄存器配置命令集 (绝密版)》

**适用对象：** 开发人员、工厂“点睛”设备、自研点睛 App。

**安全级别：** 涉及 Master Key 配置，严禁泄露。

---

## 一、 预备阶段：密钥分配（Key Derivation）

在发送命令前，系统需为每枚 UID 生成独立的密钥。

- **$K_0$ (Master Key)**: 用于后续修改配置。
    
- **$K_1$ (SDM Meta Key)**: 用于加密 UID（可选）。
    
- **$K_2$ (SDM File Read Key)**: 用于生成动态 CMAC（核心）。
    

---

## 二、 核心指令集（APDU 命令流）

### 1. 认证与选卡 (Select & Auth)

首先必须通过 AES 认证，获取对芯片配置文件的修改权限。

- **Command**: `90 AF 00 00 09 00 00 00 00 00 00 00 00 00 00`
    
- **逻辑**: 使用初始密钥（默认全 0）进行认证。
    

### 2. 配置 SDM 动态参数镜像 (Set File Settings)

这是最关键的一步，定义 URL 里哪些部分是动态变化的。

- **Command**: `90 5F 00 02 [Length] [Payload]`
    
- **Payload 配置参数**：
    
    - **`FileOption`**: 设置为 `0x40`（启用 SDM）。
        
    - **`AccessRights`**: 设置为 `0x1211`（定义 $K_2$ 为读取和认证密钥）。
        
    - **`SDMOptions`**: `0x81`（启用 ASCII 模式，确保 URL 可读）。
        
    - **`SDMCtrOffset`**: 定义计数器 `ctr` 在 URL 中的起始字节位置。
        
    - **`SDMMACRefOffset`**: 定义 `cmac` 在 URL 中的起始字节位置。
        
    - **`SDMMACOffset`**: 定义 `cmac` 动态密文的填充位置。
        

### 3. 写入 V5.0 基础 URL (Write NDEF)

将带有占位符的 H5 链接写入文件。

- **示例数据**: `https://v5.auth/v?u=00000000000000&c=000000&m=0000000000000000`
    
- **注意**: `u` (UID)、`c` (CTR)、`m` (MAC) 的位置必须与第 2 步定义的 Offset 精确对应。
    

### 4. 锁定计数器与保护 (Enable Privacy)

- **指令**: 开启 `Mirror CTR`，确保每感应一次，芯片内部计数器自动累加，并触发 `cmac` 更新。
    

### 5. 最终锁定 (Permanently Lock)

- **注意**: 一旦确认配置无误，修改 $K_0$ 并锁定配置区。
    
- **风险提示**: **锁定后不可逆**。如果配置错误，芯片将变成废卡。
    

---

## 三、 校验逻辑演示（后端伪代码）

当你的 H5 接收到请求后，后端应执行以下逻辑：



```Python
def verify_v5_tag(uid, ctr, cmac_from_url):
    # 1. 从安全数据库获取该 UID 对应的 K2
    k2 = get_key_by_uid(uid)
    
    # 2. 检查计数器是否异常
    if int(ctr) <= last_recorded_ctr(uid):
        return "REPLAY_ATTACK_DETECTED" # 重放攻击，链接已失效

    # 3. 构造计算素材 (根据 SDM 配置的模式)
    # 素材通常包含：UID + CTR + 其他可选字段
    sv = construct_sdm_vector(uid, ctr)
    
    # 4. 使用 K2 进行 AES-128 CMAC 计算
    expected_cmac = aes_cmac(k2, sv)
    
    # 5. 比对
    if cmac_from_url == expected_cmac:
        update_last_ctr(uid, ctr) # 验证通过，更新计数器
        return "SUCCESS_GENUINE_ASSET"
    else:
        return "COUNTERFEIT_DETECTED" # 伪造芯片
```

---

## 四、 给你的落地建议

1. **先跑通“明文”模式**：刚开始实验时，先别急着加加密，先确保手机靠近能跳出网页。
    
2. **善用工具**：如果你觉得手动写 APDU 指令太痛苦，可以使用 **NXP TagXplorer** 软件，它有图形界面可以直接勾选 SDM 参数，勾选完它会自动生成这些指令。
    
3. **批量生产**：当你跟品牌方谈成后，这套指令会集成到你的“生产端 App”里，工人只需要把手机放在包包的芯片位置，App 一秒钟内就会自动发完这些指令。
    

**这套“底层密令”是 V5.0 协议的命根子。建议你收到测试芯片后，找一个安静的下午，用 ACR122U 尝试发送第一条配置指令。**