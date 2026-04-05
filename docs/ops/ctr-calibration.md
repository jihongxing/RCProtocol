# CTR 校准运维手册

## 概述

CTR（Counter）是 NFC 芯片的单调递增计数器，用于防重放攻击。由于采用三级缓冲架构（L1 进程内缓存 → L2 Redis → L3 PostgreSQL），在故障恢复或数据迁移后可能出现不一致。本手册描述 CTR 校准流程。

## 架构背景

```
L1 (DashMap)  ──►  L2 (Redis)  ──►  L3 (PostgreSQL)
   进程内缓存        分布式缓存         持久化存储
   TTL: 5min        TTL: 24h          永久
```

## 不一致场景

| 场景 | 原因 | 影响 |
|------|------|------|
| Redis 宕机恢复 | AOF 重放不完整 | Redis CTR < PG CTR |
| 进程崩溃 | L1 脏数据未刷盘 | Redis CTR < 实际 CTR |
| 网络分区 | 刷盘失败 | Redis CTR > PG CTR |
| 数据迁移 | 迁移脚本遗漏 | 数据缺失 |

## 校准工具

### 一致性检查

```bash
# 检查所有 UID 的 CTR 一致性
cargo run -p rc-api --features postgres -- ctr-calibrate check

# 输出示例
Checking CTR consistency...
Total UIDs: 125,432
Consistent: 125,420
Mismatches: 12

Mismatch details:
  UID: 04A31B2C3D4E5F  Redis: 1523  PG: 1520  Action: UseRedis
  UID: 04B42C3D4E5F60  Redis: 892   PG: 895   Action: UseMax
  ...
```

### 自动校准

```bash
# 自动校准（使用较大值）
cargo run -p rc-api --features postgres -- ctr-calibrate auto

# 输出示例
Auto-calibrating CTR values...
Corrected: 12 UIDs
  04A31B2C3D4E5F: Redis 1523 → PG updated to 1523
  04B42C3D4E5F60: PG 895 → Redis updated to 895
  ...
```

### 手动校准

```bash
# 手动校准单个 UID
cargo run -p rc-api --features postgres -- ctr-calibrate uid \
  --uid 04A31B2C3D4E5F \
  --action use-redis

# 可选 action:
#   use-redis   - 使用 Redis 值更新 PG
#   use-pg      - 使用 PG 值更新 Redis
#   use-max     - 使用较大值
#   set-value   - 设置指定值（需要 --value 参数）
```

## 校准策略

### 默认策略：UseMax

```
if redis_ctr > pg_ctr:
    UPDATE pg SET ctr = redis_ctr
else:
    SET redis_ctr = pg_ctr
```

理由：CTR 必须单调递增，使用较大值可避免重放攻击误判。

### 特殊情况：ManualReview

以下情况需要人工审核：
- Redis CTR 与 PG CTR 差值 > 1000
- UID 在审计日志中有异常记录
- 资产处于 Disputed 状态

## 定时任务

建议配置定时校准任务：

```yaml
# Kubernetes CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: ctr-calibration
spec:
  schedule: "0 3 * * *"  # 每天凌晨 3 点
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: calibrator
            image: rc-api:latest
            command: ["rc-api", "ctr-calibrate", "auto"]
```

## 监控告警

```yaml
# Prometheus 告警规则
- alert: CtrMismatchDetected
  expr: rcapi_ctr_mismatch_count > 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "CTR 不一致检测"
    description: "发现 {{ $value }} 个 UID 的 CTR 值不一致"
```

## 故障恢复流程

1. **确认故障范围**
   ```bash
   cargo run -p rc-api --features postgres -- ctr-calibrate check
   ```

2. **备份当前数据**
   ```bash
   pg_dump -t ctr_records > ctr_backup_$(date +%Y%m%d).sql
   redis-cli BGSAVE
   ```

3. **执行校准**
   ```bash
   cargo run -p rc-api --features postgres -- ctr-calibrate auto
   ```

4. **验证结果**
   ```bash
   cargo run -p rc-api --features postgres -- ctr-calibrate check
   ```

5. **记录审计日志**
   - 记录校准时间、影响 UID 数量、操作人员
