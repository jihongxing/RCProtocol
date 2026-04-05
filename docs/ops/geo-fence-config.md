# 地理围栏配置指南

## 概述

地理围栏速度检测用于识别可能的欺诈行为。当资产在短时间内出现在相距很远的位置时，系统会触发告警。

## 工作原理

```
验证请求 (uid, lat, lng)
        │
        ▼
┌───────────────────┐
│ 获取上次位置和时间 │ ← Redis GEO
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Haversine 距离计算 │
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ 速度 = 距离 / 时间 │
└───────────────────┘
        │
        ▼
    速度 > 阈值?
    /        \
   是         否
   │          │
   ▼          ▼
 告警       正常
```

## 配置参数

### 环境变量

```bash
# 是否启用地理围栏检测（默认 true）
RC_API_GEO_FENCE_ENABLED=true

# 默认速度阈值（km/h，默认 1000）
RC_API_GEO_FENCE_DEFAULT_THRESHOLD=1000

# 位置数据 TTL（秒，默认 86400 = 24h）
RC_API_GEO_FENCE_LOCATION_TTL=86400

# 最小时间间隔（秒，默认 60）
# 两次验证间隔小于此值时跳过速度检测
RC_API_GEO_FENCE_MIN_INTERVAL=60
```

### 配置文件

```toml
# config.toml
[geo_fence]
enabled = true
default_speed_threshold_kmh = 1000.0
location_ttl_secs = 86400
min_interval_secs = 60
```

## 品牌自定义阈值

不同品牌可以设置不同的速度阈值：

```bash
# 通过 API 设置品牌阈值
curl -X POST http://rc-api/v1/admin/geo-fence/threshold \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "brand_id": "brand-luxury-watches",
    "threshold_kmh": 500
  }'

# 查询品牌阈值
curl http://rc-api/v1/admin/geo-fence/threshold/brand-luxury-watches \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 删除品牌阈值（恢复使用默认值）
curl -X DELETE http://rc-api/v1/admin/geo-fence/threshold/brand-luxury-watches \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## 阈值设置建议

| 资产类型 | 建议阈值 | 说明 |
|----------|----------|------|
| 奢侈品手表 | 500 km/h | 高价值，需要严格检测 |
| 艺术品 | 300 km/h | 通常不会快速移动 |
| 电子产品 | 1000 km/h | 可能通过航空运输 |
| 汽车配件 | 200 km/h | 通常随车移动 |

## 告警处理

### 速度违规告警

当检测到速度异常时，系统会：

1. 记录审计日志（SECURITY 级别）
2. 发送 Prometheus 指标
3. 可选：触发 Webhook 通知

```yaml
# Prometheus 告警规则
- alert: GeoFenceSpeedViolation
  expr: increase(rcapi_geo_fence_violations_total[5m]) > 0
  for: 1m
  labels:
    severity: warning
  annotations:
    summary: "检测到地理围栏速度违规"
    description: "过去 5 分钟内检测到 {{ $value }} 次速度违规"
```

### 人工审核流程

1. 查看违规详情
   ```bash
   curl http://rc-api/v1/admin/geo-fence/violations?since=2026-03-17 \
     -H "Authorization: Bearer $ADMIN_TOKEN"
   ```

2. 分析是否为误报
   - 检查资产是否通过航空运输
   - 检查是否为 GPS 漂移
   - 检查是否为时钟同步问题

3. 处理决定
   - 误报：标记为已审核
   - 真实违规：冻结资产，通知品牌方

## 监控指标

```
# 地理围栏检测总数（按结果分类）
rcapi_geo_fence_checks_total{result="normal"}
rcapi_geo_fence_checks_total{result="violation"}
rcapi_geo_fence_checks_total{result="first_verification"}

# 地理围栏检测延迟
rcapi_geo_fence_latency_seconds

# 速度违规总数
rcapi_geo_fence_violations_total
```

## 故障排除

### 问题：所有请求都返回 FirstVerification

**原因**：Redis 位置数据丢失

**解决**：
```bash
# 检查 Redis 连接
redis-cli -h redis-master ping

# 检查位置数据
redis-cli -h redis-master GEOPOS asset_locations "04A31B2C3D4E5F"
```

### 问题：误报率过高

**原因**：阈值设置过低

**解决**：
```bash
# 分析历史速度分布
curl http://rc-api/v1/admin/geo-fence/stats \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 调整阈值
export RC_API_GEO_FENCE_DEFAULT_THRESHOLD=1200
kubectl rollout restart deployment/rc-api
```

### 问题：检测延迟过高

**原因**：Redis GEO 操作慢

**解决**：
```bash
# 检查 Redis 延迟
redis-cli -h redis-master --latency

# 检查 Redis 内存使用
redis-cli -h redis-master INFO memory
```
