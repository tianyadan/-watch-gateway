package bus

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// RouteConfig
// 描述一个固定的公交查询场景。
// Apple Watch / 公共调用方不需要知道 lineId、stationId 等第三方参数。
type RouteConfig struct {
	Hash        string `yaml:"hash"`
	City        string `yaml:"city"`
	CityID      string `yaml:"city_id"`
	Line        string `yaml:"line"`
	LineID      string `yaml:"line_id"`
	Scene       string `yaml:"scene"`
	From        string `yaml:"from"`
	To          string `yaml:"to"`
	TargetOrder int    `yaml:"target_order"`
	StationID   string `yaml:"station_id"`
	StationLat  string `yaml:"station_lat"`
	StationLng  string `yaml:"station_lng"`
	Legacy      string `yaml:"legacy"`
	Bark        bool   `yaml:"bark"`
}

// Name 兼容旧逻辑：优先 legacy，否则用 hash。
func (r RouteConfig) Name() string {
	if r.Legacy != "" {
		return r.Legacy
	}
	return r.Hash
}

type apiResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    RouteData `json:"data"`
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RegisterRoutes 按配置注册 legacy 与公共 hash 接口。
func RegisterRoutes(
	mux *http.ServeMux,
	cfg *AppConfig,
	service *Service,
	barkClient *BarkClient,
	limiter *RateLimiter,
) {
	authToken := cfg.Server.AuthToken

	for _, route := range cfg.Routes {
		route := route

		if route.Legacy != "" {
			pattern := "GET /watch/bus/" + route.Legacy
			mux.HandleFunc(
				pattern,
				requireAuthorization(
					authToken,
					legacyRouteHandler(service, route, barkClient),
				),
			)
		}
	}

	mux.HandleFunc(
		"GET /watch/bus/q/{hash}",
		publicRouteHandler(cfg, service, limiter),
	)
}

// legacyRouteHandler 旧接口：可 Bark，不限流。
func legacyRouteHandler(
	service *Service,
	cfg RouteConfig,
	barkClient *BarkClient,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := service.QueryRoute(r.Context(), cfg)
		if err != nil {
			log.Printf("query bus route %s failed: %v", cfg.Name(), err)
			writeJSON(w, http.StatusBadGateway, errorResponse{
				Code:    502,
				Message: "公交实时数据查询失败",
			})
			return
		}

		if cfg.Bark && barkClient != nil {
			if err := barkClient.PushBusResult(r.Context(), cfg, data); err != nil {
				log.Printf("push bark for route %s failed: %v", cfg.Name(), err)
			} else {
				log.Printf("bark push success: route=%s", cfg.Name())
			}
		}

		writeJSON(w, http.StatusOK, apiResponse{
			Code:    0,
			Message: "success",
			Data:    data,
		})
	}
}

// publicRouteHandler 公共接口：限流 + 只返回 JSON，不推 Bark。
func publicRouteHandler(
	cfg *AppConfig,
	service *Service,
	limiter *RateLimiter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := r.PathValue("hash")
		route, ok := cfg.RouteByHash(hash)
		if !ok {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Code:    404,
				Message: "route not found",
			})
			return
		}

		ip := clientIP(r)
		key := ip + "|" + hash
		if limiter != nil && !limiter.Allow(key, time.Now()) {
			writeJSON(w, http.StatusTooManyRequests, errorResponse{
				Code:    429,
				Message: "rate limit exceeded",
			})
			return
		}

		data, err := service.QueryRoute(r.Context(), route)
		if err != nil {
			log.Printf("query bus route %s failed: %v", route.Name(), err)
			writeJSON(w, http.StatusBadGateway, errorResponse{
				Code:    502,
				Message: "公交实时数据查询失败",
			})
			return
		}

		writeJSON(w, http.StatusOK, apiResponse{
			Code:    0,
			Message: "success",
			Data:    data,
		})
	}
}

// clientIP 优先取 X-Forwarded-For 首段，否则 RemoteAddr。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json response failed: %v", err)
	}
}
