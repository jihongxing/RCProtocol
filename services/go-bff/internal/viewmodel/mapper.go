package viewmodel

// StateLabel 14 种协议状态 → 中文展示文案
// 来源：docs/foundation/state-machine.md
var StateLabel = map[string]string{
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

// StateBadges 状态 → 前端展示徽章
var StateBadges = map[string][]string{
	"Activated":   {"verified"},
	"LegallySold": {"verified"},
	"Transferred": {"verified"},
	"Disputed":    {"frozen"},
}

// AllStates 14 种已知协议状态枚举
var AllStates = []string{
	"PreMinted", "FactoryLogged", "Unassigned", "RotatingKeys",
	"EntangledPending", "Activated", "LegallySold", "Transferred",
	"Consumed", "Legacy", "Disputed", "Tampered", "Compromised", "Destructed",
}

// MapState 已知状态返回中文文案，未知返回 "未知状态"
func MapState(state string) string {
	if label, ok := StateLabel[state]; ok {
		return label
	}
	return "未知状态"
}

// MapBadges 已知状态返回对应徽章数组，无徽章或未知状态返回空数组
func MapBadges(state string) []string {
	if badges, ok := StateBadges[state]; ok {
		return badges
	}
	return []string{}
}
