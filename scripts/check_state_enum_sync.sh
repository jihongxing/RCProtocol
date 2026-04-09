#!/usr/bin/env bash
# M12: CI 检查脚本 — 确保 BFF AllStates 与 Rust AssetState 枚举一致
# 用法: bash scripts/check_state_enum_sync.sh

set -euo pipefail

RUST_FILE="rust/rc-common/src/types.rs"
BFF_FILE="services/go-bff/internal/viewmodel/mapper.go"

if [ ! -f "$RUST_FILE" ] || [ ! -f "$BFF_FILE" ]; then
    echo "ERROR: source files not found"
    exit 1
fi

# 从 Rust AssetState 枚举提取变体名（排除 derive/impl 等行）
RUST_STATES=$(grep -oP '^\s+(PreMinted|FactoryLogged|Unassigned|RotatingKeys|EntangledPending|Activated|LegallySold|Transferred|Consumed|Legacy|Disputed|Tampered|Compromised|Destructed)\b' "$RUST_FILE" | tr -d ' ,' | sort)

# 从 BFF AllStates 提取枚举值
BFF_STATES=$(grep -oP '"(PreMinted|FactoryLogged|Unassigned|RotatingKeys|EntangledPending|Activated|LegallySold|Transferred|Consumed|Legacy|Disputed|Tampered|Compromised|Destructed)"' "$BFF_FILE" | tr -d '"' | sort -u)

DIFF=$(diff <(echo "$RUST_STATES") <(echo "$BFF_STATES") || true)

if [ -z "$DIFF" ]; then
    echo "OK: BFF AllStates matches Rust AssetState enum (14 states)"
    exit 0
else
    echo "MISMATCH: BFF AllStates differs from Rust AssetState enum"
    echo "$DIFF"
    exit 1
fi
