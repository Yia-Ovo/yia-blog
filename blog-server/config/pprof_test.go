package config

import "testing"

func TestLocalPprofConfiguration(t *testing.T) {
	for _, tc := range []struct {
		env, host string
		allowed   bool
	}{
		{"dev", "127.0.0.1", true}, {"local", "127.0.0.1", true}, {"test", "::1", true},
		{"prod", "127.0.0.1", false}, {"", "127.0.0.1", false},
		{"dev", "", false}, {"dev", "0.0.0.0", false}, {"dev", "::", false},
		{"dev", "localhost", false}, {"dev", "192.0.2.1", false},
	} {
		t.Run(tc.env+"/"+tc.host, func(t *testing.T) {
			conf := &GlobalConfig{
				Server: ServerConfig{EnablePprof: true, Env: tc.env, HttpHost: tc.host, HttpPort: 8081},
				Gin:    GinConfig{Mode: "debug"}, Mysql: &MysqlConfig{Host: "localhost", User: "test", Database: "test"},
				Redis: RedisConfig{Addr: "localhost:6379"},
			}
			if err := validateConfig(conf); (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v err=%v", tc.allowed, err)
			}
			conf.Server.EnablePprof = false
			if err := validateConfig(conf); err != nil {
				t.Fatalf("disabled pprof must retain old compatibility: %v", err)
			}
		})
	}
}
