#!/bin/bash
# 生成 Platform 角色的测试 JWT token

cd "$(dirname "$0")/../rust/rc-api"

# 默认值与本地 Podman 环境保持一致；允许外部 export 覆盖
export RC_JWT_SECRET="${RC_JWT_SECRET:-my-super-secret-jwt-key-for-testing-only}"
export DATABASE_URL="${DATABASE_URL:-postgresql://rcprotocol:rcprotocol_dev@localhost:5432/rcprotocol}"

cargo run --release --example generate_jwt
