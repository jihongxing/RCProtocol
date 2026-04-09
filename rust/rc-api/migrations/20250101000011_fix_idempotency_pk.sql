-- M2: 幂等键复合主键 (Bug 1.43)
-- 当前主键仅 idempotency_key，不含 resource_type 作用域
-- 不同 actor/接口使用相同 key 会产生跨资源意外幂等拦截
ALTER TABLE idempotency_records DROP CONSTRAINT idempotency_records_pkey;
ALTER TABLE idempotency_records ADD PRIMARY KEY (idempotency_key, resource_type);
