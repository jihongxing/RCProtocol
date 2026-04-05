我们将抛弃 `is_valid` 这种简陋的布尔逻辑，转而采用一套描述资产“生命状态”的高维结构。

## Endpoint: `GET /v1/resolve/{sun_msg}`

#### 请求参数

- `sun_msg`: 芯片生成的包含加密签名（CMAC）和计数器（CTR）的原始消息。
    

#### 响应结构 (Response Schema)



```JSON
{
  "protocol": "Regalis Clavis v1.1",
  "status": "SOVEREIGN_AUTHENTIC", 
  "timestamp": "2026-03-18T02:08:28Z",
  
  "asset_identity": {
    "rc_id": "RC-OBJ-04AB94D2151990",
    "display_name": "Valhalla Limited Series No. 42",
    "brand_id": "VALHALLA_LTD",
    "origin_factory": "FAC-CN-09"
  },

  "sovereignty_proof": {
    "is_entangled": true,            // 密钥是否已从出厂态翻转为主权态
    "ownership_lock": "LOCKED",      // 资产锁定状态：UNLOCKED, LOCKED, TRANSFERRING
    "vitality_index": 18,            // 扫描计数器 (CTR)，代表资产被交互的频次
    "entanglement_date": "2026-02-15T09:00:00Z"
  },

  "trust_score": {
    "authenticity_level": "MAXIMUM", // 正统等级
    "security_checks": {
      "cryptographic_verification": "PASSED", // 加密签名验证
      "replay_attack_check": "PASSED",        // 重放检测
      "geo_proximity_check": "WARN",          // 地理位置异常预警（若有）
      "tamper_status": "INTACT"               // 物理防篡改线状态
    }
  },

  "claims": {
    "insurance_eligible": true,
    "warranty_expiry": "2028-02-15",
    "transfer_restricted": false
  }
}
```


---

## 🚥 标准主权状态码 (Sovereign Status Codes)

|**状态码**|**含义 (Sovereign Meaning)**|**业务逻辑指导**|
|---|---|---|
|**`SOVEREIGN_AUTHENTIC`**|**主权正统**|资产状态完美，所有权清晰，建议展示“数字证书”。|
|**`FACTORY_NEW`**|**纯净出厂**|仅通过盲扫，尚未激活纠缠。建议引导进行“开箱激活”。|
|**`DISPUTED_SIGNAL`**|**权属争议**|计数器异常或在多地同时被扫描。建议标记为“受疑资产”。|
|**`SOVEREIGN_REVOKED`**|**主权吊销**|品牌方标记为失窃、报废或违约。资产在数字世界已“死亡”。|
|**`LEGACY_DISCOVERY`**|**历史识别**|识别到芯片 UID 但无协议密钥记录。视为“未定义物体”。|