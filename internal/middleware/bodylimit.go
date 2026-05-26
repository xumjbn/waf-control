package middleware

import (
	"net/http"
)

// MaxBodyBytes 包装请求体大小上限，超出会让后续 Decoder 立即返回 io.ErrUnexpectedEOF
// 或 http.MaxBytesError，handler 自然给 400。防止恶意大 body 耗内存。
// 默认 1MiB —— 业务负载（创建规则 / 配置 / 试运行）都远小于这个量级。
// 个别端点（如 swagger 文档导入）需要大 body 的，自己再包一层提高上限。
const DefaultMaxBodyBytes int64 = 1 << 20 // 1 MiB

func MaxBody(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength != 0 {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
