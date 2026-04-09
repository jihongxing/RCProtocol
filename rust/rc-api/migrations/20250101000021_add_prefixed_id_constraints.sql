-- Add forward-looking prefixed ID constraints without forcing immediate legacy cleanup.
-- Existing legacy/demo data remains readable because constraints are created as NOT VALID.
-- New rows and updated rows must satisfy the prefix policy.

ALTER TABLE brands
  ADD CONSTRAINT chk_brands_brand_id_prefix
  CHECK (brand_id LIKE 'brand_%') NOT VALID;

ALTER TABLE products
  ADD CONSTRAINT chk_products_product_id_prefix
  CHECK (product_id LIKE 'product_%') NOT VALID;

ALTER TABLE assets
  ADD CONSTRAINT chk_assets_asset_id_prefix
  CHECK (asset_id LIKE 'asset_%') NOT VALID;

ALTER TABLE batches
  ADD CONSTRAINT chk_batches_batch_id_prefix
  CHECK (batch_id LIKE 'batch_%') NOT VALID;

ALTER TABLE factory_sessions
  ADD CONSTRAINT chk_factory_sessions_session_id_prefix
  CHECK (session_id LIKE 'session_%') NOT VALID;
