package bus

import (
	"os"
	"path/filepath"
	"testing"
)

// 写入临时 YAML 并调用 LoadConfig，便于断言校验逻辑。
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadConfig_Valid(t *testing.T) {
	path := writeTempConfig(t, `
server:
  addr: ":28085"
  auth_token: "secret"
routes:
  - hash: "abcd1234"
    city: "示例市"
    city_id: "000"
    line: "1"
    line_id: "0000-1-1"
    scene: "回家"
    from: "A"
    to: "B"
    target_order: 11
    station_id: "0000-1"
    station_lat: "0.0"
    station_lng: "0.0"
    legacy: "home"
    bark: true
`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Server.Addr != ":28085" {
		t.Fatalf("addr = %q", cfg.Server.Addr)
	}
	if cfg.Server.AuthToken != "secret" {
		t.Fatalf("auth_token = %q", cfg.Server.AuthToken)
	}
	if len(cfg.Routes) != 1 {
		t.Fatalf("routes len = %d", len(cfg.Routes))
	}
	r := cfg.Routes[0]
	if r.Hash != "abcd1234" || r.Legacy != "home" || !r.Bark {
		t.Fatalf("route mismatch: %+v", r)
	}
	byHash, ok := cfg.RouteByHash("abcd1234")
	if !ok || byHash.LineID != "0000-1-1" {
		t.Fatalf("RouteByHash failed: ok=%v route=%+v", ok, byHash)
	}
	byLegacy, ok := cfg.RouteByLegacy("home")
	if !ok || byLegacy.Hash != "abcd1234" {
		t.Fatalf("RouteByLegacy failed: ok=%v route=%+v", ok, byLegacy)
	}
}

func TestLoadConfig_DuplicateHash(t *testing.T) {
	path := writeTempConfig(t, `
server:
  auth_token: "secret"
routes:
  - hash: "samehash"
    city: "示例市"
    city_id: "000"
    line: "1"
    line_id: "0000-1-1"
    scene: "a"
    from: "A"
    to: "B"
    target_order: 1
    station_id: "1"
    station_lat: "1"
    station_lng: "1"
  - hash: "samehash"
    city: "示例市"
    city_id: "000"
    line: "1"
    line_id: "0000-1-0"
    scene: "b"
    from: "C"
    to: "D"
    target_order: 2
    station_id: "2"
    station_lat: "2"
    station_lng: "2"
`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected duplicate hash error")
	}
}

func TestLoadConfig_ExampleFile(t *testing.T) {
	cfg, err := LoadConfig("../../config.example.yaml")
	if err != nil {
		t.Fatalf("LoadConfig config.example.yaml: %v", err)
	}
	if _, ok := cfg.RouteByHash("demo-home"); !ok {
		t.Fatal("missing demo-home hash")
	}
	if _, ok := cfg.RouteByLegacy("work"); !ok {
		t.Fatal("missing legacy work")
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	path := writeTempConfig(t, `
server:
  auth_token: "secret"
routes:
  - hash: "abcd1234"
    city: "示例市"
    line: "1"
    scene: "回家"
    from: "A"
    to: "B"
`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected missing field error")
	}
}
