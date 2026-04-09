/// Preservation（保持性）测试 — 合法业务流程不受影响
///
/// 在未修复代码上运行，确认基线行为正确。修复后重新运行确认无回归。
/// 遵循 observation-first 方法论：观察当前行为并记录。
///
/// **Validates: Requirements 3.1, 3.2, 3.4, 3.6**

#[cfg(test)]
mod preservation_state_machine {
    use crate::state_machine::{can_transition, next_state_for_action};
    use rc_common::types::{AssetAction, AssetState};

    // ── 3.1 + 3.2: 状态机正向路径 ──
    // 完整链路 PreMinted → FactoryLogged → ... → LegallySold 正常转换

    #[test]
    fn preservation_3_1_full_chain_preminted_to_legallysold() {
        // 完整正向链路
        let s1 = next_state_for_action(AssetState::PreMinted, AssetAction::BlindLog);
        assert_eq!(s1, Some(AssetState::FactoryLogged), "PreMinted + BlindLog → FactoryLogged");

        let s2 = next_state_for_action(AssetState::FactoryLogged, AssetAction::StockIn);
        assert_eq!(s2, Some(AssetState::Unassigned), "FactoryLogged + StockIn → Unassigned");

        let s3 = next_state_for_action(AssetState::Unassigned, AssetAction::ActivateRotateKeys);
        assert_eq!(s3, Some(AssetState::RotatingKeys), "Unassigned + ActivateRotateKeys → RotatingKeys");

        let s4 = next_state_for_action(AssetState::RotatingKeys, AssetAction::ActivateEntangle);
        assert_eq!(s4, Some(AssetState::EntangledPending), "RotatingKeys + ActivateEntangle → EntangledPending");

        let s5 = next_state_for_action(AssetState::EntangledPending, AssetAction::ActivateConfirm);
        assert_eq!(s5, Some(AssetState::Activated), "EntangledPending + ActivateConfirm → Activated");

        let s6 = next_state_for_action(AssetState::Activated, AssetAction::LegalSell);
        assert_eq!(s6, Some(AssetState::LegallySold), "Activated + LegalSell → LegallySold");
    }

    // ── 3.2: 已有 13 个合法状态转换路径（不含 Recover/MarkDestructed）──

    #[test]
    fn preservation_3_2_all_13_legitimate_transitions() {
        // 定义预期的 13 个合法转换
        let expected_transitions: Vec<(AssetState, AssetAction, AssetState)> = vec![
            (AssetState::PreMinted, AssetAction::BlindLog, AssetState::FactoryLogged),
            (AssetState::FactoryLogged, AssetAction::StockIn, AssetState::Unassigned),
            (AssetState::Unassigned, AssetAction::ActivateRotateKeys, AssetState::RotatingKeys),
            (AssetState::RotatingKeys, AssetAction::ActivateEntangle, AssetState::EntangledPending),
            (AssetState::EntangledPending, AssetAction::ActivateConfirm, AssetState::Activated),
            (AssetState::Activated, AssetAction::LegalSell, AssetState::LegallySold),
            (AssetState::LegallySold, AssetAction::Transfer, AssetState::Transferred),
            (AssetState::Transferred, AssetAction::Transfer, AssetState::Transferred),
            (AssetState::LegallySold, AssetAction::Consume, AssetState::Consumed),
            (AssetState::Transferred, AssetAction::Consume, AssetState::Consumed),
            (AssetState::LegallySold, AssetAction::Legacy, AssetState::Legacy),
            (AssetState::Transferred, AssetAction::Legacy, AssetState::Legacy),
            // Freeze: 非终态非冻结 → Disputed（用 Activated 作为代表）
            (AssetState::Activated, AssetAction::Freeze, AssetState::Disputed),
        ];

        for (from, action, expected_to) in &expected_transitions {
            let result = next_state_for_action(*from, *action);
            assert_eq!(
                result,
                Some(*expected_to),
                "转换失败: {:?} + {:?} 应为 {:?}，实际为 {:?}",
                from, action, expected_to, result
            );
        }
    }

    #[test]
    fn preservation_3_2_freeze_works_on_all_non_terminal_non_frozen_states() {
        // Freeze 应对所有非终态非冻结状态有效
        let freezeable_states = [
            AssetState::PreMinted,
            AssetState::FactoryLogged,
            AssetState::Unassigned,
            AssetState::RotatingKeys,
            AssetState::EntangledPending,
            AssetState::Activated,
            AssetState::LegallySold,
            AssetState::Transferred,
        ];

        for &state in &freezeable_states {
            let result = next_state_for_action(state, AssetAction::Freeze);
            assert_eq!(
                result,
                Some(AssetState::Disputed),
                "Freeze 从 {:?} 应转至 Disputed",
                state
            );
        }
    }

    #[test]
    fn preservation_3_2_mark_tampered_works_on_non_terminal() {
        // MarkTampered 对非终态有效
        let non_terminal = [
            AssetState::PreMinted, AssetState::FactoryLogged,
            AssetState::Unassigned, AssetState::RotatingKeys,
            AssetState::EntangledPending, AssetState::Activated,
            AssetState::LegallySold, AssetState::Transferred,
            AssetState::Disputed,
        ];

        for &state in &non_terminal {
            let result = next_state_for_action(state, AssetAction::MarkTampered);
            assert_eq!(
                result,
                Some(AssetState::Tampered),
                "MarkTampered 从 {:?} 应转至 Tampered",
                state
            );
        }
    }

    #[test]
    fn preservation_3_2_mark_compromised_works_on_non_terminal() {
        // MarkCompromised 对非终态有效
        let non_terminal = [
            AssetState::PreMinted, AssetState::FactoryLogged,
            AssetState::Unassigned, AssetState::RotatingKeys,
            AssetState::EntangledPending, AssetState::Activated,
            AssetState::LegallySold, AssetState::Transferred,
            AssetState::Disputed,
        ];

        for &state in &non_terminal {
            let result = next_state_for_action(state, AssetAction::MarkCompromised);
            assert_eq!(
                result,
                Some(AssetState::Compromised),
                "MarkCompromised 从 {:?} 应转至 Compromised",
                state
            );
        }
    }

    #[test]
    fn preservation_3_2_terminal_states_block_transitions() {
        // 终态不应有任何合法转换（除 MarkTampered/MarkCompromised 可能适用的情况外）
        let terminal_states = [
            AssetState::Consumed,
            AssetState::Legacy,
            AssetState::Destructed,
        ];

        let business_actions = [
            AssetAction::BlindLog, AssetAction::StockIn,
            AssetAction::ActivateRotateKeys, AssetAction::ActivateEntangle,
            AssetAction::ActivateConfirm, AssetAction::LegalSell,
            AssetAction::Transfer, AssetAction::Consume,
            AssetAction::Legacy, AssetAction::Freeze,
        ];

        for &state in &terminal_states {
            for &action in &business_actions {
                let result = next_state_for_action(state, action);
                assert_eq!(
                    result, None,
                    "终态 {:?} 不应接受 {:?}，但得到 {:?}",
                    state, action, result
                );
            }
        }
    }

    #[test]
    fn preservation_3_2_can_transition_positive() {
        // can_transition 对合法转换应返回 true
        assert!(can_transition(AssetState::PreMinted, AssetState::FactoryLogged));
        assert!(can_transition(AssetState::FactoryLogged, AssetState::Unassigned));
        assert!(can_transition(AssetState::Activated, AssetState::LegallySold));
        assert!(can_transition(AssetState::LegallySold, AssetState::Transferred));
        assert!(can_transition(AssetState::LegallySold, AssetState::Consumed));
    }
}

/// Preservation 测试 — apply_action 保持性
///
/// **Validates: Requirements 3.1, 3.4, 3.6**
#[cfg(test)]
mod preservation_protocol {
    use crate::protocol::apply_action;
    use rc_common::audit::AuditContext;
    use rc_common::types::{ActorRole, AssetAction, AssetRecord, AssetState};
    use uuid::Uuid;

    fn make_context(role: ActorRole) -> AuditContext {
        AuditContext {
            trace_id: Uuid::new_v4(),
            actor_id: "test-actor".into(),
            actor_role: role,
            actor_org: None,
            idempotency_key: format!("idem-{}", Uuid::new_v4()),
            approval_id: None,
            policy_version: None,
            buyer_id: None,
        }
    }

    // ── 3.1: 非 Disputed 状态的合法非 Recover 动作正常执行 ──

    #[test]
    fn preservation_3_1_blindlog_preminted() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::PreMinted,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Factory);
        let result = apply_action(&record, AssetAction::BlindLog, ctx);
        assert!(result.is_ok(), "BlindLog 从 PreMinted 应成功");
        let (next, event) = result.unwrap();
        assert_eq!(next.current_state, AssetState::FactoryLogged);
        assert_eq!(event.to_state, AssetState::FactoryLogged);
    }

    #[test]
    fn preservation_3_1_legalsell_activated() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::Activated,
            previous_state: None,
        };
        let mut ctx = make_context(ActorRole::Brand);
        ctx.buyer_id = Some("consumer-1".into());
        let result = apply_action(&record, AssetAction::LegalSell, ctx);
        assert!(result.is_ok(), "LegalSell 从 Activated 应成功");
        let (next, _) = result.unwrap();
        assert_eq!(next.current_state, AssetState::LegallySold);
    }

    // ── 3.4: Platform 执行治理类动作（Freeze）继续允许 ──

    #[test]
    fn preservation_3_4_platform_freeze_allowed() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::Activated,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Platform);
        let result = apply_action(&record, AssetAction::Freeze, ctx);
        assert!(result.is_ok(), "Platform Freeze 应成功");
        let (next, _) = result.unwrap();
        assert_eq!(next.current_state, AssetState::Disputed);
    }

    #[test]
    fn preservation_3_4_platform_mark_tampered_allowed() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::Activated,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Platform);
        let result = apply_action(&record, AssetAction::MarkTampered, ctx);
        assert!(result.is_ok(), "Platform MarkTampered 应成功");
        let (next, _) = result.unwrap();
        assert_eq!(next.current_state, AssetState::Tampered);
    }

    #[test]
    fn preservation_3_4_platform_mark_compromised_allowed() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::Activated,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Platform);
        let result = apply_action(&record, AssetAction::MarkCompromised, ctx);
        assert!(result.is_ok(), "Platform MarkCompromised 应成功");
        let (next, _) = result.unwrap();
        assert_eq!(next.current_state, AssetState::Compromised);
    }

    #[test]
    fn preservation_3_4_moderator_freeze_allowed() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::LegallySold,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Moderator);
        let result = apply_action(&record, AssetAction::Freeze, ctx);
        assert!(result.is_ok(), "Moderator Freeze 应成功");
    }

    // ── 3.6: 非 LegalSell 动作 owner_id 语义不变 ──
    // apply_action 不直接写 owner_id，但确认 AuditEvent 中无 buyer_id 干扰

    #[test]
    fn preservation_3_6_non_legalsell_no_owner_change_in_event() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::PreMinted,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Factory);
        let (_, event) = apply_action(&record, AssetAction::BlindLog, ctx).unwrap();
        // BlindLog 事件不应涉及 owner_id 变更
        assert_eq!(event.action, AssetAction::BlindLog);
        assert_eq!(event.from_state, Some(AssetState::PreMinted));
        assert_eq!(event.to_state, AssetState::FactoryLogged);
    }

    #[test]
    fn preservation_3_6_transfer_preserves_semantics() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::LegallySold,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Consumer);
        let (next, event) = apply_action(&record, AssetAction::Transfer, ctx).unwrap();
        assert_eq!(next.current_state, AssetState::Transferred);
        assert_eq!(event.action, AssetAction::Transfer);
    }

    // ── 3.1: freeze 记录 previous_state ──

    #[test]
    fn preservation_3_1_freeze_records_previous_state() {
        let record = AssetRecord {
            asset_id: "asset-1".into(),
            brand_id: "brand-1".into(),
            current_state: AssetState::Activated,
            previous_state: None,
        };
        let ctx = make_context(ActorRole::Platform);
        let (next, _) = apply_action(&record, AssetAction::Freeze, ctx).unwrap();
        assert_eq!(next.current_state, AssetState::Disputed);
        // Freeze 应记录冻结前状态以便将来 Recover
        assert_eq!(next.previous_state, Some(AssetState::Activated));
    }
}

/// Preservation PBT — 随机合法 (AssetState, AssetAction) 组合保持性
///
/// **Validates: Requirements 3.1, 3.2**
#[cfg(test)]
mod preservation_pbt_state_machine {
    use crate::state_machine::next_state_for_action;
    use proptest::prelude::*;
    use rc_common::types::{AssetAction, AssetState};

    // 所有合法状态转换对（文档定义的 13 条，不含 Recover/MarkDestructed）
    fn legal_transition_pairs() -> Vec<(AssetState, AssetAction, AssetState)> {
        vec![
            (AssetState::PreMinted, AssetAction::BlindLog, AssetState::FactoryLogged),
            (AssetState::FactoryLogged, AssetAction::StockIn, AssetState::Unassigned),
            (AssetState::Unassigned, AssetAction::ActivateRotateKeys, AssetState::RotatingKeys),
            (AssetState::RotatingKeys, AssetAction::ActivateEntangle, AssetState::EntangledPending),
            (AssetState::EntangledPending, AssetAction::ActivateConfirm, AssetState::Activated),
            (AssetState::Activated, AssetAction::LegalSell, AssetState::LegallySold),
            (AssetState::LegallySold, AssetAction::Transfer, AssetState::Transferred),
            (AssetState::Transferred, AssetAction::Transfer, AssetState::Transferred),
            (AssetState::LegallySold, AssetAction::Consume, AssetState::Consumed),
            (AssetState::Transferred, AssetAction::Consume, AssetState::Consumed),
            (AssetState::LegallySold, AssetAction::Legacy, AssetState::Legacy),
            (AssetState::Transferred, AssetAction::Legacy, AssetState::Legacy),
        ]
    }

    proptest! {
        // 从合法转换表中随机选取，验证 next_state_for_action 与文档一致
        #[test]
        fn pbt_legal_transitions_match_documentation(
            idx in 0..12usize
        ) {
            let pairs = legal_transition_pairs();
            let (from, action, expected) = pairs[idx];
            let result = next_state_for_action(from, action);
            prop_assert_eq!(
                result,
                Some(expected),
                "转换 {:?} + {:?} 应为 {:?}",
                from,
                action,
                expected
            );
        }

        // 随机选取非终态，验证 Freeze 均转至 Disputed
        #[test]
        fn pbt_freeze_any_non_terminal_non_frozen(idx in 0..8usize) {
            let non_terminal = [
                AssetState::PreMinted, AssetState::FactoryLogged,
                AssetState::Unassigned, AssetState::RotatingKeys,
                AssetState::EntangledPending, AssetState::Activated,
                AssetState::LegallySold, AssetState::Transferred,
            ];
            let state = non_terminal[idx];
            let result = next_state_for_action(state, AssetAction::Freeze);
            prop_assert_eq!(result, Some(AssetState::Disputed));
        }

        // 终态 + 业务动作 → None
        #[test]
        fn pbt_terminal_states_reject_business_actions(
            state_idx in 0..3usize,
            action_idx in 0..6usize
        ) {
            let terminals = [AssetState::Consumed, AssetState::Legacy, AssetState::Destructed];
            let business = [
                AssetAction::BlindLog, AssetAction::StockIn,
                AssetAction::LegalSell, AssetAction::Transfer,
                AssetAction::Consume, AssetAction::Legacy,
            ];
            let result = next_state_for_action(terminals[state_idx], business[action_idx]);
            prop_assert_eq!(result, None);
        }
    }
}
