package viewmodel

import (
	"testing"

	"pgregory.net/rapid"
)

// --- Unit Tests ---

func TestMapState_AllKnownStates(t *testing.T) {
	// 14 种已知状态全部映射正确
	expected := map[string]string{
		"PreMinted":        "预铸造",
		"FactoryLogged":    "已登记",
		"Unassigned":       "待分配",
		"RotatingKeys":     "密钥轮换中",
		"EntangledPending": "绑定确认中",
		"Activated":        "已激活",
		"LegallySold":      "已售出",
		"Transferred":      "已过户",
		"Consumed":         "已消费",
		"Legacy":           "传承遗珍",
		"Disputed":         "争议中",
		"Tampered":         "已篡改",
		"Compromised":      "已失陷",
		"Destructed":       "已销毁",
	}
	for state, wantLabel := range expected {
		got := MapState(state)
		if got != wantLabel {
			t.Errorf("MapState(%q) = %q, want %q", state, got, wantLabel)
		}
	}
}

func TestMapState_UnknownState(t *testing.T) {
	unknowns := []string{"", "unknown", "Active", "ACTIVATED", "pending", "foo_bar"}
	for _, s := range unknowns {
		got := MapState(s)
		if got != "未知状态" {
			t.Errorf("MapState(%q) = %q, want %q", s, got, "未知状态")
		}
	}
}

func TestMapBadges_KnownStates(t *testing.T) {
	cases := []struct {
		state string
		want  []string
	}{
		{"Activated", []string{"verified"}},
		{"LegallySold", []string{"verified"}},
		{"Transferred", []string{"verified"}},
		{"Disputed", []string{"frozen"}},
		{"PreMinted", []string{}},
		{"FactoryLogged", []string{}},
		{"Unassigned", []string{}},
		{"RotatingKeys", []string{}},
		{"EntangledPending", []string{}},
		{"Consumed", []string{}},
		{"Legacy", []string{}},
		{"Tampered", []string{}},
		{"Compromised", []string{}},
		{"Destructed", []string{}},
	}
	for _, tc := range cases {
		got := MapBadges(tc.state)
		if !slicesEqual(got, tc.want) {
			t.Errorf("MapBadges(%q) = %v, want %v", tc.state, got, tc.want)
		}
	}
}

func TestMapBadges_UnknownState(t *testing.T) {
	got := MapBadges("SomeRandomState")
	if len(got) != 0 {
		t.Errorf("MapBadges(unknown) = %v, want empty slice", got)
	}
}

// --- Property Tests ---

// TestKnownStateMappingCompleteness — Property 3: 已知状态映射完整性
// **Validates: Requirements FR-10 (10.1~10.4)**
// 从 AllStates 随机选择，验证 MapState 返回非"未知状态"，MapBadges 与映射表一致
func TestKnownStateMappingCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		idx := rapid.IntRange(0, len(AllStates)-1).Draw(t, "stateIndex")
		state := AllStates[idx]

		label := MapState(state)
		if label == "未知状态" {
			t.Fatalf("MapState(%q) returned fallback for known state", state)
		}

		// MapBadges must match the StateBadges table exactly
		gotBadges := MapBadges(state)
		expectedBadges, hasBadges := StateBadges[state]
		if hasBadges {
			if !slicesEqual(gotBadges, expectedBadges) {
				t.Fatalf("MapBadges(%q) = %v, want %v", state, gotBadges, expectedBadges)
			}
		} else {
			if len(gotBadges) != 0 {
				t.Fatalf("MapBadges(%q) = %v, want empty slice", state, gotBadges)
			}
		}
	})
}

// TestUnknownStateFallback — Property 4: 未知状态 fallback
// **Validates: Requirements FR-10 (10.3)**
// 生成不属于 AllStates 的随机字符串，验证 MapState 返回"未知状态"，MapBadges 返回空数组
func TestUnknownStateFallback(t *testing.T) {
	knownSet := make(map[string]struct{}, len(AllStates))
	for _, s := range AllStates {
		knownSet[s] = struct{}{}
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate random string and filter out known states
		s := rapid.String().Draw(t, "randomState")
		if _, known := knownSet[s]; known {
			t.Skip("generated known state, skipping")
		}

		label := MapState(s)
		if label != "未知状态" {
			t.Fatalf("MapState(%q) = %q, want %q", s, label, "未知状态")
		}

		badges := MapBadges(s)
		if len(badges) != 0 {
			t.Fatalf("MapBadges(%q) = %v, want empty slice", s, badges)
		}
	})
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
