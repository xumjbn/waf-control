package httputil

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SanitizeDBError 把 pgx/pg 错误转成对用户安全的提示。原 err 仅供调用方 slog。
// 用例：handler 里 `slog.Error(...); httputil.SanitizeDBError(err)`，回前端的字符串。
// 设计目的：避免把表名/列名/SQL 片段泄露给前端用户。
func SanitizeDBError(err error) (status int, safeMsg string) {
	if err == nil {
		return http.StatusOK, ""
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusNotFound, "资源不存在"
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return http.StatusConflict, "记录已存在（唯一约束冲突）"
		case "23503": // foreign_key_violation
			return http.StatusBadRequest, "关联资源不存在或仍被引用"
		case "23502": // not_null_violation
			return http.StatusBadRequest, "缺少必填字段"
		case "23514": // check_violation
			return http.StatusBadRequest, "字段值不合法"
		case "42P01": // undefined_table
			return http.StatusInternalServerError, "服务暂不可用（schema 缺失）"
		case "42703": // undefined_column
			return http.StatusInternalServerError, "服务暂不可用（schema 不匹配）"
		case "53300", "57P03": // too_many_connections / cannot_connect_now
			return http.StatusServiceUnavailable, "数据库繁忙，请稍后重试"
		}
	}
	// 网络/超时类
	msg := err.Error()
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "context deadline") {
		return http.StatusGatewayTimeout, "请求超时"
	}
	return http.StatusInternalServerError, "服务器内部错误"
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   msg,
	})
}

func JSONList(w http.ResponseWriter, data interface{}, total, page, limit int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
		Meta:    &Meta{Total: total, Page: page, Limit: limit},
	})
}

func Decode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
