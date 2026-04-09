-- M2: 审批表 pending 唯一索引 (Bug 1.44)
-- 防止同资源同类型两个并发 pending 审批
CREATE UNIQUE INDEX IF NOT EXISTS uq_approvals_pending_resource
  ON approvals(resource_type, resource_id, type)
  WHERE status = 'pending';
