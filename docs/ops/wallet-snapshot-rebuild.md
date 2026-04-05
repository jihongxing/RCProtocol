# 钱包快照重建流程

## 概述

钱包快照是存储在 Redis 中的反规范化数据，用于加速钱包信息和资产列表查询。当 Redis 数据丢失或不一致时，需要从 PostgreSQL 重建快照。

## 数据结构

```
# Redis Hash - 钱包基本信息
wallet:{wid}
  ├── display_name: "老钱的藏馆"
  ├── avatar_url: "https://..."
  ├── asset_count: 42
  ├── total_value_usd: 1500000
  └── updated_at: "2026-03-17T10:00:00Z"

# Redis Sorted Set - 资产列表（score = 估值）
wallet:{wid}:assets
  ├── aid_001: 150000
  ├── aid_002: 80000
  └── aid_003: 50000
```

## 重建触发条件

| 场景 | 触发方式 |
|------|----------|
| Redis 数据丢失 | 自动（查询时发现缺失） |
| 一致性校验失败 | 手动触发 |
| 数据迁移后 | 批量重建 |
| 定期维护 | 定时任务 |

## 单钱包重建

### 通过 API 触发

```bash
# 重建单个钱包快照
curl -X POST http://rc-api/v1/admin/wallet-snapshot/rebuild \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"wid": "wid-123456"}'

# 响应
{
  "wid": "wid-123456",
  "asset_count": 42,
  "total_value_usd": 1500000,
  "rebuild_time_ms": 125
}
```

### 通过 CLI 触发

```bash
cargo run -p rc-api --features postgres -- wallet-snapshot rebuild \
  --wid wid-123456
```

## 批量重建

### 重建所有钱包

```bash
# 批量重建（每批 100 个，间隔 1 秒）
cargo run -p rc-api --features postgres -- wallet-snapshot rebuild-all \
  --batch-size 100 \
  --interval-ms 1000

# 输出
Rebuilding wallet snapshots...
Progress: 1000/5000 (20%)
Progress: 2000/5000 (40%)
...
Completed: 5000 wallets rebuilt in 52.3s
```

### 重建指定钱包列表

```bash
# 从文件读取钱包 ID 列表
cat > wallets.txt << EOF
wid-001
wid-002
wid-003
EOF

cargo run -p rc-api --features postgres -- wallet-snapshot rebuild-list \
  --file wallets.txt
```

## 一致性校验

### 校验单个钱包

```bash
curl http://rc-api/v1/admin/wallet-snapshot/verify/wid-123456 \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 响应
{
  "wid": "wid-123456",
  "is_consistent": true,
  "redis_count": 42,
  "pg_count": 42,
  "redis_total_value": 1500000,
  "pg_total_value": 1500000,
  "mismatched_assets": []
}
```

### 批量校验

```bash
cargo run -p rc-api --features postgres -- wallet-snapshot verify-all

# 输出
Verifying wallet snapshots...
Total wallets: 5000
Consistent: 4995
Inconsistent: 5

Inconsistent wallets:
  wid-001: redis_count=42, pg_count=43
  wid-002: redis_total=1500000, pg_total=1500500
  ...
```

## 自动重建策略

### 查询时自动重建

当查询钱包快照时发现 Redis 中不存在，系统会自动从 PostgreSQL 重建：

```rust
pub async fn get_snapshot(&self, wid: &str) -> Result<WalletSnapshot, SnapshotError> {
    // 1. 尝试从 Redis 获取
    if let Some(snapshot) = self.get_snapshot_from_redis(wid).await? {
        return Ok(snapshot);
    }
    
    // 2. Redis 未命中，从 PG 获取并缓存
    let snapshot = self.get_snapshot_from_pg(wid).await?;
    self.cache_snapshot(&snapshot).await?;
    Ok(snapshot)
}
```

### 定时校验和重建

```yaml
# Kubernetes CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: wallet-snapshot-verify
spec:
  schedule: "0 4 * * *"  # 每天凌晨 4 点
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: verifier
            image: rc-api:latest
            command: 
            - rc-api
            - wallet-snapshot
            - verify-and-rebuild
            - --auto-fix
```

## 监控指标

```
# 快照命中率
rcapi_wallet_snapshot_hits_total
rcapi_wallet_snapshot_misses_total

# 快照重建次数
rcapi_wallet_snapshot_rebuilds_total

# 快照查询延迟
rcapi_wallet_snapshot_latency_seconds
```

## 告警配置

```yaml
# Prometheus 告警规则
- alert: WalletSnapshotHitRateLow
  expr: |
    rate(rcapi_wallet_snapshot_hits_total[5m]) /
    (rate(rcapi_wallet_snapshot_hits_total[5m]) + rate(rcapi_wallet_snapshot_misses_total[5m]))
    < 0.9
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "钱包快照命中率过低"
    description: "当前命中率 {{ $value | humanizePercentage }}"

- alert: WalletSnapshotRebuildSpike
  expr: increase(rcapi_wallet_snapshot_rebuilds_total[5m]) > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "钱包快照重建次数异常"
    description: "过去 5 分钟重建了 {{ $value }} 个快照"
```

## 故障排除

### 问题：重建速度慢

**原因**：PostgreSQL 查询慢或 Redis 写入慢

**解决**：
```bash
# 检查 PG 查询性能
EXPLAIN ANALYZE SELECT * FROM asset_ownership WHERE wid = 'wid-123456';

# 检查 Redis 写入延迟
redis-cli -h redis-master --latency

# 调整批次大小
cargo run -p rc-api --features postgres -- wallet-snapshot rebuild-all \
  --batch-size 50
```

### 问题：一致性校验总是失败

**原因**：并发写入导致数据竞争

**解决**：
```bash
# 暂停写入
kubectl scale deployment/rc-api --replicas=0

# 重建快照
cargo run -p rc-api --features postgres -- wallet-snapshot rebuild-all

# 恢复服务
kubectl scale deployment/rc-api --replicas=3
```
