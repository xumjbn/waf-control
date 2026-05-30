package monitor

// protection.go —— NW · 01 dashboard「站点防护评分」雷达的真数据驱动。
//
// 之前雷达是硬编码 [88,76,92,81,70,96]。这里把 6 个维度全部用真实配置 / 运行数据
// 计算出 0-100 分。各维度的数据来源（注释在每个子查询）：
//   1 注入防护  policies 中 sqli/rce/lfi-rfi/xss 类规则的启用比例
//   2 Bot 抗扰  bot_challenges 启用比例（无则看 bot 类规则启用比例）
//   3 CC 抗压   policies 中 rate-limit 类规则启用比例
//   4 认证安全  api_endpoints 中 auth_type != 'None' 的比例
//   5 数据脱敏  代理指标：sites 的 SSL 启用比例（传输层加密）
//   6 业务可用  nodes 在线比例
//
// 前端轴标签固定来自设计稿（RADAR_AXES），这里只按该顺序返回 6 个分数。

import (
	"context"
	"net/http"
	"time"
)

type ProtectionScore struct {
	// 顺序与前端 RADAR_AXES 对齐：注入/Bot/CC/认证/脱敏/可用。
	Scores      []int     `json:"scores"`
	GeneratedAt time.Time `json:"generated_at"`
}

// pct 计算 num/den*100，den<=0 返回 0。clamp 到 [0,100]。
func pct(num, den int64) int {
	if den <= 0 {
		return 0
	}
	v := int(float64(num) / float64(den) * 100)
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// scalarInt 跑一个返回单个 bigint 的查询；出错返回 0。
func (r *Repository) scalarInt(ctx context.Context, q string, args ...any) int64 {
	var v int64
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&v); err != nil {
		return 0
	}
	return v
}

func (r *Repository) ProtectionScoreSnapshot(ctx context.Context) ProtectionScore {
	// 1 注入防护
	injTotal := r.scalarInt(ctx, `SELECT COUNT(*) FROM policies WHERE category IN ('sqli','rce','lfi-rfi','xss')`)
	injOn := r.scalarInt(ctx, `SELECT COUNT(*) FROM policies WHERE category IN ('sqli','rce','lfi-rfi','xss') AND COALESCE(is_enabled,false) = true`)

	// 2 Bot 抗扰：优先 bot_challenges；无则 bot 类规则启用率
	botTotal := r.scalarInt(ctx, `SELECT COUNT(*) FROM bot_challenges`)
	botOn := r.scalarInt(ctx, `SELECT COUNT(*) FROM bot_challenges WHERE COALESCE(enabled,false) = true`)
	botScore := pct(botOn, botTotal)
	if botTotal == 0 {
		bt := r.scalarInt(ctx, `SELECT COUNT(*) FROM policies WHERE category = 'bot'`)
		bo := r.scalarInt(ctx, `SELECT COUNT(*) FROM policies WHERE category = 'bot' AND COALESCE(is_enabled,false) = true`)
		botScore = pct(bo, bt)
	}

	// 3 CC 抗压
	ccTotal := r.scalarInt(ctx, `SELECT COUNT(*) FROM policies WHERE category = 'rate-limit'`)
	ccOn := r.scalarInt(ctx, `SELECT COUNT(*) FROM policies WHERE category = 'rate-limit' AND COALESCE(is_enabled,false) = true`)

	// 4 认证安全
	apiTotal := r.scalarInt(ctx, `SELECT COUNT(*) FROM api_endpoints`)
	apiAuth := r.scalarInt(ctx, `SELECT COUNT(*) FROM api_endpoints WHERE auth_type IS NOT NULL AND auth_type <> '' AND auth_type <> 'None'`)

	// 5 数据脱敏（代理：传输层 SSL 启用率）
	siteTotal := r.scalarInt(ctx, `SELECT COUNT(*) FROM sites`)
	siteSSL := r.scalarInt(ctx, `SELECT COUNT(*) FROM sites WHERE COALESCE(ssl_enabled,false) = true`)

	// 6 业务可用（在线节点比例）
	nodeTotal := r.scalarInt(ctx, `SELECT COUNT(*) FROM nodes`)
	nodeOnline := r.scalarInt(ctx, `SELECT COUNT(*) FROM nodes WHERE status = 'online'`)

	return ProtectionScore{
		Scores: []int{
			pct(injOn, injTotal),
			botScore,
			pct(ccOn, ccTotal),
			pct(apiAuth, apiTotal),
			pct(siteSSL, siteTotal),
			pct(nodeOnline, nodeTotal),
		},
		GeneratedAt: time.Now(),
	}
}

// ProtectionScoreHandler GET /monitor/protection-score
func (h *Handler) ProtectionScoreHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.ProtectionScoreSnapshot(r.Context()))
}
