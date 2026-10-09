package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// 旧部署配置可能保留已移除的键，升级时仍须读取其他有效配置。
func TestLegacyRouterPrefixConfigCompatibility(t *testing.T) {
	for _, prefix := range []string{"", "  router_prefix: /legacy/custom/v9\n"} {
		content := "gin:\n  mode: debug\n" + prefix
		t.Run(prefix, func(t *testing.T) {
			t.Run("local", func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
				previous := localConfigPath
				localConfigPath = path
				t.Cleanup(func() { localConfigPath = previous })
				conf, err := getFromLocal()
				if err != nil {
					t.Fatal(err)
				}
				if conf.Gin.Mode != "debug" {
					t.Fatalf("gin.mode = %q, want debug", conf.Gin.Mode)
				}
			})
			t.Run("remote_decode", func(t *testing.T) {
				v := viper.New()
				setDefaultConfig(v)
				if err := v.ReadConfig(strings.NewReader(content)); err != nil {
					t.Fatal(err)
				}
				var conf GlobalConfig
				if err := v.Unmarshal(&conf); err != nil {
					t.Fatal(err)
				}
				if conf.Gin.Mode != "debug" {
					t.Fatalf("gin.mode = %q, want debug", conf.Gin.Mode)
				}
			})
		})
	}
}
