package bus

import (
	"encoding/json"
	"net/http"
)

// requireAuthorization 校验请求头 Authorization 是否等于配置中的 token。
func requireAuthorization(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != token {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(errorResponse{
				Code:    403,
				Message: "Forbidden",
			})
			return
		}
		next(w, r)
	}
}
