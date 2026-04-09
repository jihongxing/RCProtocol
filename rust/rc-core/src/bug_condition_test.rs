/// Bug Condition 探索性测试 — 状态机与协议缺陷
/// 
/// 这些测试编码了**期望行为**：在未修复代码上应当 FAIL，证明缺陷存在。
/// 修复后测试通过即确认修复成功。
///
/// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.11**

#[cfg(test)]
mod bug_condition_state_machine {
    use rc_common::types::{ActorRole, AssetAction, AssetState, AssetRecord};
    use rc_common::audit::AuditContext;
    use uuid::Uuid;
    use crate::state_machine::{next_state_for_action, next_state_for_action_with_previous};
    use crate::protocol::apply_action;

    fn make_context(role: ActorRole) -> AuditContext {
        AuditContext {
            trace_id: Uuid::new_v4(),
            actor_id: "test-actor".into(),
            actor_role: role,
            actor_org: None,
            idempotency_key: "idem-1".into(),
            approval_id: None,
            policy_version: None,
            buyer_id: None,
        }
    }

    // ── 1.1: Recover 从终态恢复 ──
    // Bug: apply_action 中 Recover 直接使用 record.previous_state，
    //      客户端可构造 previous_state = Destructed（终态）实现恢复到终态。
    // 期望行为: Recover 应拒绝恢复到终态。
    #[test]
    fn bug_1_1_recover_should_reject_terminal_previous_state() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::Disputed,
            previous_state: Some(AssetState::Destructed),
        };

        let ctx = make_context(ActorRole::Moderator);
        let result = apply_action(&record, AssetAction::Recover, ctx);

        // 期望: Recover 到终态 Destructed 应该被拒绝
        assert!(
            result.is_err(),
            "BUG 1.1: Recover 允许恢复到终态 Destructed，应拒绝"
        );
    }

    // ── 1.2: can_transition(Disputed, *) 对 Recover 应返回可达 ──
    // Bug: next_state_for_action 无 Recover 分支，Recover 从 Disputed 永远返回 None。
    // 期望行为: next_state_for_action_with_previous(Disputed, Recover, Some(合法状态)) 应返回 Some。
    #[test]
    fn bug_1_2_state_machine_should_have_recover_branch() {
        // Recover 在带 previous_state 的版本中应有处理分支
        // 使用 Activated 作为合法的 previous_state（非终态、非 Disputed）
        let result = next_state_for_action_with_previous(
            AssetState::Disputed,
            AssetAction::Recover,
            Some(AssetState::Activated),
        );

        assert!(
            result.is_some(),
            "BUG 1.2: next_state_for_action_with_previous(Disputed, Recover, Some(Activated)) 返回 None — 状态机无 Recover 分支"
        );
        assert_eq!(result, Some(AssetState::Activated));
    }

    // ── 1.3: Destructed 无进入路径 ──
    // Bug: 无任何 Action 可转入 Destructed（无 MarkDestructed）。
    // 期望行为: 应存在 MarkDestructed action 使非终态可进入 Destructed。
    #[test]
    fn bug_1_3_no_action_leads_to_destructed() {
        let all_non_terminal_states = [
            AssetState::PreMinted, AssetState::FactoryLogged, AssetState::Unassigned,
            AssetState::RotatingKeys, AssetState::EntangledPending, AssetState::Activated,
            AssetState::LegallySold, AssetState::Transferred, AssetState::Disputed,
        ];

        let all_actions = [
            AssetAction::BlindLog, AssetAction::StockIn,
            AssetAction::ActivateRotateKeys, AssetAction::ActivateEntangle,
            AssetAction::ActivateConfirm, AssetAction::LegalSell,
            AssetAction::Transfer, AssetAction::Consume, AssetAction::Legacy,
            AssetAction::Freeze, AssetAction::Recover,
            AssetAction::MarkTampered, AssetAction::MarkCompromised,
            AssetAction::MarkDestructed,
        ];

        let mut any_reaches_destructed = false;
        for &state in &all_non_terminal_states {
            for &action in &all_actions {
                if let Some(target) = next_state_for_action(state, action) {
                    if target == AssetState::Destructed {
                        any_reaches_destructed = true;
                    }
                }
            }
        }

        assert!(
            any_reaches_destructed,
            "BUG 1.3: 状态机中无任何 action 可进入 Destructed 终态"
        );
    }

    // ── 1.4: LegalSell owner_id 应为 buyer_id 而非 actor_id ──
    // Bug: persist_action 中 LegalSell 将 owner_id 设为 actor_id（Brand），
    //      而非 buyer_id（Consumer）。
    // 期望行为: apply_action 应要求 buyer_id 上下文，AuditContext 应包含 buyer_id 字段。
    #[test]
    fn bug_1_4_audit_context_should_have_buyer_id_field() {
        // AuditContext 缺少 buyer_id 字段，LegalSell 无法传递买家信息
        // 通过反射式检查: 尝试构造带 buyer_id 的 AuditContext
        // 如果编译通过说明字段已存在（修复后），编译失败说明缺陷存在。
        // 
        // 因为 Rust 无法在运行时检查字段是否存在，我们验证 AuditContext 的 JSON 序列化
        // 是否包含 buyer_id 字段。
        let ctx = make_context(ActorRole::Brand);
        let json = serde_json::to_value(&ctx).unwrap();

        assert!(
            json.get("buyer_id").is_some(),
            "BUG 1.4: AuditContext 缺少 buyer_id 字段，LegalSell 无法传递买家信息"
        );
    }

    // ── 1.11: Platform 执行 BlindLog 无需 approval_id ──
    // Bug: Platform 角色 can_role_initiate 无条件返回 true，
    //      apply_action 不强制要求 approval_id。
    // 期望行为: Platform 执行非治理类动作（如 BlindLog）应要求 approval_id。
    #[test]
    fn bug_1_11_platform_business_action_should_require_approval() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::PreMinted,
            previous_state: None,
        };

        // Platform 执行 BlindLog（业务动作），无 approval_id
        let ctx = AuditContext {
            trace_id: Uuid::new_v4(),
            actor_id: "platform-admin".into(),
            actor_role: ActorRole::Platform,
            actor_org: None,
            idempotency_key: "idem-1".into(),
            approval_id: None, // 无审批
            policy_version: None,
            buyer_id: None,
        };

        let result = apply_action(&record, AssetAction::BlindLog, ctx);

        // 期望: Platform 执行非治理类动作（BlindLog）无 approval_id 应被拒绝
        assert!(
            result.is_err(),
            "BUG 1.11: Platform 执行 BlindLog（非治理动作）无需 approval_id，应强制要求"
        );
    }

    // ── 1.5: RC_AUTH_DISABLED=true 时 Brand 角色 brand_id 为 None ──
    // Bug: build_fallback_claims 不从 X-Brand-Id 读取 brand_id，Brand 角色的 brand_id 永远为 None。
    // 注意: 此测试在 rc-api auth 模块中，这里仅验证 Claims 结构是否支持 brand_id。
    // 实际的 middleware 测试在 rc-api 模块中覆盖。
}
