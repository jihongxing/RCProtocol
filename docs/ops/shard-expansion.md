# 分片扩容运维手册

## 概述

RC-Protocol 使用 UID 一致性哈希进行数据分片。当单分片数据量超过阈值或性能下降时，需要进行分片扩容。

## 当前分片架构

```rust
pub const SHARD_COUNT: u8 = 16;

pub fn get_shard_id(uid: &str) -> u8 {
    let hash = xxh64(uid.as_bytes(), 0);
    (hash % u64::from(SHARD_COUNT)) as u8
}
```

## 扩容触发条件

| 指标 | 阈值 | 说明 |
|------|------|------|
| 单分片记录数 | > 10M | 查询性能下降 |
| 分片数据倾斜 | > 15% | 负载不均衡 |
| 查询延迟 P99 | > 100ms | 用户体验下降 |

## 扩容方案

### 方案 A：倍增扩容（推荐）

将分片数从 16 扩展到 32：

```
原分片 0 → 新分片 0, 16
原分片 1 → 新分片 1, 17
...
原分片 15 → 新分片 15, 31
```

优点：数据迁移量最小（每个分片只需迁移约 50% 数据）

### 方案 B：重新分片

完全重新计算所有 UID 的分片归属。

缺点：需要迁移所有数据，停机时间长。

## 扩容步骤（方案 A）

### 1. 准备阶段

```bash
# 1.1 创建新分片数据库
for i in {16..31}; do
  createdb rc_shard_$i
  psql -d rc_shard_$i -f migrations/008_shard_tables.sql
  psql -d rc_shard_$i -f migrations/009_shard_indexes.sql
done

# 1.2 更新配置（不重启）
export RC_API_SHARD_COUNT=32
export RC_API_SHARD_URLS="postgres://...shard_0,...,postgres://...shard_31"
```

### 2. 数据迁移

```bash
# 2.1 生成迁移脚本
cargo run -p rc-api --features postgres -- shard-migrate generate \
  --old-count 16 \
  --new-count 32 \
  --output migrate_16_to_32.sql

# 2.2 执行迁移（每个原分片）
for i in {0..15}; do
  new_shard=$((i + 16))
  psql -d rc_shard_$i -c "
    INSERT INTO rc_shard_$new_shard.asset_records_hot
    SELECT * FROM asset_records_hot
    WHERE get_shard_id(uid, 32) = $new_shard;
  "
done
```

### 3. 切换阶段

```bash
# 3.1 启用双写模式
export RC_API_SHARD_DUAL_WRITE=true
kubectl rollout restart deployment/rc-api

# 3.2 验证双写正常
cargo run -p rc-api --features postgres -- shard-verify --count 32

# 3.3 切换到新分片数
export RC_API_SHARD_COUNT=32
export RC_API_SHARD_DUAL_WRITE=false
kubectl rollout restart deployment/rc-api
```

### 4. 清理阶段

```bash
# 4.1 删除已迁移数据
for i in {0..15}; do
  new_shard=$((i + 16))
  psql -d rc_shard_$i -c "
    DELETE FROM asset_records_hot
    WHERE get_shard_id(uid, 32) = $new_shard;
  "
done

# 4.2 回收空间
for i in {0..15}; do
  psql -d rc_shard_$i -c "VACUUM FULL asset_records_hot;"
done
```

## 回滚方案

如果扩容过程中出现问题：

```bash
# 1. 停止双写
export RC_API_SHARD_DUAL_WRITE=false

# 2. 恢复原分片数
export RC_API_SHARD_COUNT=16

# 3. 重启服务
kubectl rollout restart deployment/rc-api

# 4. 删除新分片数据库
for i in {16..31}; do
  dropdb rc_shard_$i
done
```

## 监控指标

```yaml
# 扩容期间重点监控
- rcapi_shard_record_count{shard_id="*"}
- rcapi_db_query_latency_seconds{operation="select"}
- rcapi_shard_migration_progress
```

## 注意事项

1. **选择低峰期**：建议在凌晨 2-5 点进行
2. **备份数据**：迁移前完整备份所有分片
3. **灰度发布**：先在测试环境验证
4. **监控告警**：扩容期间加强监控
5. **回滚预案**：确保回滚脚本可用
