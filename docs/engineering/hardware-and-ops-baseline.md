# 硬件与运维基线

> 文档类型：Engineering  
> 状态：Active  
> 权威级别：Reference

---

## 1. 目标

本文件汇总当前与硬件实验、测试运行、运维执行直接相关的工程基线。

它不是协议真源，但用于指导工程落地与实验执行。

---

## 2. 当前硬件实验基线

当前明确的硬件实验与集成基线包括：

- 读卡器：`ACR122U`
- 标签：`NTAG 424 DNA`
- 重点验证能力：
  - 读卡器连接
  - 标签识别
  - 健康检查
  - 传输与 APDU 路径
  - 复位到 transport baseline
  - provision + readback
  - blind scan 与 CMAC 校验路径

详细操作步骤以 `../archive/` 中保留的 runbook 和当前代码实现为参考，不再扩写为新的并行规范。

---

## 3. 运行与恢复基线

当前生产与测试环境需要具备以下基本能力：

- Redis 故障恢复
- PostgreSQL 故障恢复
- 网络分区后的 CTR 校准
- 钱包快照重建
- 分片扩容
- 地理围栏参数维护

对应手册位于：

- `../ops/disaster-recovery.md`
- `../ops/ctr-calibration.md`
- `../ops/geo-fence-config.md`
- `../ops/shard-expansion.md`
- `../ops/wallet-snapshot-rebuild.md`

---

## 4. 基线原则

### 4.1 运维手册从属于真源

运维手册描述的是执行方式，不是协议真相。若与 Foundation 文档冲突，以 Foundation 为准。

### 4.2 工程手册从属于当前实现

硬件实验、脚本、测试命令应以当前仓库代码与脚本为准；文档只保留稳定入口，不复制大量命令细节。

### 4.3 失败可恢复

所有实验性路径与硬件写入操作，都应优先保证：

- 可复位
- 可恢复
- 可回归验证

---

## 5. 清理结论

本次整理后：

- 硬件 runbook 不再作为顶层主文档存在
- 运维文档继续保留在 `ops/`
- 工程目录只保留当前基线说明与跳转关系
