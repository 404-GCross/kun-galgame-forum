// Package trust is kungal's BFF integration with the infra Trust & Safety
// service: report intake (Phase 1), enforcement callbacks (Phase 2), and the
// moderator inbox proxy (Phase 3).
package trust

// ReportReason mirrors an infra-seeded global report reason
// (trust_report_reason). The trust service exposes reasons only via its
// JWT-gated admin API — there is no S2S "list reasons for my site" endpoint —
// so kungal keeps the stable global set here to feed the report dropdown. Keep
// in sync with infra migrate/migrate.go SeedReasons; if infra later ships an
// S2S reasons feed, serve that instead of this constant.
type ReportReason struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Severity int    `json:"severity"`
}

// GlobalReasons are the six seeded global reasons (severity 1=low … 3=high).
var GlobalReasons = []ReportReason{
	{Key: "abuse", Label: "辱骂骚扰", Severity: 2},
	{Key: "spam", Label: "垃圾信息", Severity: 1},
	{Key: "illegal", Label: "违法内容", Severity: 3},
	{Key: "rating_mislabel", Label: "分级标注错误", Severity: 1},
	{Key: "copyright", Label: "版权侵权", Severity: 2},
	{Key: "other", Label: "其他", Severity: 1},
}
