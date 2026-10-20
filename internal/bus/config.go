package bus

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppConfig 启动时从 YAML 加载的全局配置。
type AppConfig struct {
	Server ServerConfig  `yaml:"server"`
	Routes []RouteConfig `yaml:"routes"`

	byHash   map[string]RouteConfig
	byLegacy map[string]RouteConfig
}

// ServerConfig HTTP 与鉴权相关配置。
type ServerConfig struct {
	Addr      string `yaml:"addr"`
	AuthToken string `yaml:"auth_token"`
}

// LoadConfig 读取并校验配置文件。
func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	if err := cfg.validateAndIndex(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// RouteByHash 按 hash 查找线路配置。
func (c *AppConfig) RouteByHash(hash string) (RouteConfig, bool) {
	r, ok := c.byHash[hash]
	return r, ok
}

// RouteByLegacy 按 legacy 名称查找线路配置。
func (c *AppConfig) RouteByLegacy(name string) (RouteConfig, bool) {
	r, ok := c.byLegacy[name]
	return r, ok
}

// validateAndIndex 校验必填项与唯一性，并建立索引。
func (c *AppConfig) validateAndIndex() error {
	if strings.TrimSpace(c.Server.AuthToken) == "" {
		return fmt.Errorf("server.auth_token is required")
	}
	if strings.TrimSpace(c.Server.Addr) == "" {
		c.Server.Addr = ":28085"
	}
	if len(c.Routes) == 0 {
		return fmt.Errorf("routes must not be empty")
	}

	c.byHash = make(map[string]RouteConfig, len(c.Routes))
	c.byLegacy = make(map[string]RouteConfig)

	for i, r := range c.Routes {
		if err := validateRoute(r, i); err != nil {
			return err
		}
		if _, exists := c.byHash[r.Hash]; exists {
			return fmt.Errorf("duplicate route hash: %s", r.Hash)
		}
		c.byHash[r.Hash] = r

		if r.Legacy == "" {
			continue
		}
		if _, exists := c.byLegacy[r.Legacy]; exists {
			return fmt.Errorf("duplicate legacy name: %s", r.Legacy)
		}
		c.byLegacy[r.Legacy] = r
	}

	return nil
}

// validateRoute 校验单条线路必填字段。
func validateRoute(r RouteConfig, index int) error {
	prefix := fmt.Sprintf("routes[%d]", index)
	required := []struct {
		name  string
		value string
	}{
		{"hash", r.Hash},
		{"city", r.City},
		{"city_id", r.CityID},
		{"line", r.Line},
		{"line_id", r.LineID},
		{"scene", r.Scene},
		{"from", r.From},
		{"to", r.To},
		{"station_id", r.StationID},
		{"station_lat", r.StationLat},
		{"station_lng", r.StationLng},
	}
	for _, f := range required {
		if strings.TrimSpace(f.value) == "" {
			return fmt.Errorf("%s.%s is required", prefix, f.name)
		}
	}
	if r.TargetOrder <= 0 {
		return fmt.Errorf("%s.target_order must be > 0", prefix)
	}
	return nil
}
