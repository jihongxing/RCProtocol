# 故障恢复运维手册

## 概述

本手册描述 RC-Protocol 生产环境的故障恢复流程，涵盖 Redis、PostgreSQL、网络分区等常见故障场景。

## 故障分级

| 级别 | 描述 | 响应时间 | 示例 |
|------|------|----------|------|
| P0 | 服务完全不可用 | 15 分钟 | 主库宕机、Redis 集群故障 |
| P1 | 核心功能受损 | 30 分钟 | 单分片不可用、从库同步延迟 |
| P2 | 性能下降 | 2 小时 | 缓存命中率下降、查询变慢 |
| P3 | 非关键功能异常 | 24 小时 | 监控指标缺失、日志丢失 |

## Redis 故障恢复

### 场景 1：Redis 单节点宕机

**症状**：
- `rcapi_ctr_l2_redis_misses_total` 急剧上升
- 服务日志出现 Redis 连接错误

**恢复步骤**：

```bash
# 1. 确认故障节点
redis-cli -h redis-master ping

# 2. 检查 Sentinel 状态（如果使用 Sentinel）
redis-cli -h sentinel-1 -p 26379 sentinel master mymaster

# 3. 手动触发故障转移（如果 Sentinel 未自动切换）
redis-cli -h sentinel-1 -p 26379 sentinel failover mymaster

# 4. 验证新主节点
redis-cli -h redis-new-master ping

# 5. 更新应用配置（如果需要）
kubectl set env deployment/rc-api REDIS_URL=redis://redis-new-master:6379
```

### 场景 2：Redis 数据丢失

**症状**：
- AOF 文件损坏
- 重启后数据为空

**恢复步骤**：

```bash
# 1. 停止 Redis
systemctl stop redis

# 2. 检查 AOF 文件
redis-check-aof --fix appendonly.aof

# 3. 如果 AOF 无法修复，使用 RDB 快照
cp /backup/dump.rdb /var/lib/redis/dump.rdb

# 4. 重启 Redis
systemctl start redis

# 5. 从 PostgreSQL 重建 CTR 缓存
cargo run -p rc-api --features postgres -- ctr-rebuild
```

## PostgreSQL 故障恢复

### 场景 1：主库宕机

**症状**：
- 写操作全部失败
- `rcapi_db_query_errors_total` 急剧上升

**恢复步骤**：

```bash
# 1. 确认主库状态
pg_isready -h pg-primary

# 2. 检查从库同步状态
psql -h pg-replica -c "SELECT pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn();"

# 3. 提升从库为主库
pg_ctl promote -D /var/lib/postgresql/data

# 4. 更新 PgBouncer 配置
sed -i 's/pg-primary/pg-replica/g' /etc/pgbouncer/pgbouncer.ini
pgbouncer -R

# 5. 更新应用配置
kubectl set env deployment/rc-api DATABASE_URL=postgres://pg-replica:5432/rcprotocol

# 6. 验证服务恢复
curl -s http://rc-api/health | jq .
```

### 场景 2：单分片不可用

**症状**：
- 部分 UID 查询失败
- 错误日志显示特定分片连接失败

**恢复步骤**：

```bash
# 1. 确认故障分片
for i in {0..15}; do
  pg_isready -h pg-shard-$i && echo "Shard $i: OK" || echo "Shard $i: FAILED"
done

# 2. 启用降级模式（跳过故障分片）
kubectl set env deployment/rc-api RC_API_SKIP_SHARDS=5

# 3. 修复故障分片
# ... 根据具体原因处理

# 4. 恢复正常模式
kubectl set env deployment/rc-api RC_API_SKIP_SHARDS=""
```

### 场景 3：数据损坏

**症状**：
- 查询返回异常数据
- 校验和错误

**恢复步骤**：

```bash
# 1. 停止写入
kubectl scale deployment/rc-api --replicas=0

# 2. 从备份恢复
pg_restore -h pg-primary -d rcprotocol /backup/rcprotocol_$(date +%Y%m%d).dump

# 3. 验证数据完整性
cargo run -p rc-api --features postgres -- db-verify

# 4. 恢复服务
kubectl scale deployment/rc-api --replicas=3
```

## 网络分区恢复

### 场景：Redis 与 PostgreSQL 网络分区

**症状**：
- CTR 数据不一致
- 部分写入成功但未持久化

**恢复步骤**：

```bash
# 1. 确认网络恢复
ping redis-master
ping pg-primary

# 2. 检查 CTR 一致性
cargo run -p rc-api --features postgres -- ctr-calibrate check

# 3. 执行校准
cargo run -p rc-api --features postgres -- ctr-calibrate auto

# 4. 验证审计日志完整性
cargo run -p rc-api --features postgres -- audit-verify
```

## 降级策略

### Redis 不可用时的降级

```bash
# 启用直接 PG 模式
kubectl set env deployment/rc-api RC_API_FALLBACK_STRATEGY=DirectPg

# 监控性能影响
watch -n 5 'curl -s http://rc-api/metrics | grep rcapi_ctr_update_latency'
```

### 从库不可用时的降级

```bash
# 所有读请求路由到主库
kubectl set env deployment/rc-api RC_API_READ_FROM_PRIMARY=true
```

## 恢复后检查清单

- [ ] 服务健康检查通过
- [ ] CTR 一致性校验通过
- [ ] 审计日志完整性校验通过
- [ ] 监控指标恢复正常
- [ ] 告警已清除
- [ ] 事故报告已提交
