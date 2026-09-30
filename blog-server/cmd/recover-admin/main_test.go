package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoveryPasswordInput(t *testing.T) {
	for _, test := range []struct {
		input, want string
		bad         bool
	}{
		{"SampleOnly123\r\n", "SampleOnly123", false},
		{" Sample123 ", " Sample123 ", false},
		{"First123\nSecond456", "", true},
		{strings.Repeat("a", 21), "", true},
		{strings.Repeat("a", 100), "", true},
		{"bad\x00value1", "", true},
		{"", "", true},
	} {
		got, err := readPassword(strings.NewReader(test.input))
		if (err != nil) != test.bad || !test.bad && got != test.want {
			t.Fatalf("unexpected result for input length %d", len(test.input))
		}
	}
}

func TestRecoveryRejectsRemoteAndProductionConfig(t *testing.T) {
	valid := "server:\n  env: local\nmysql:\n  host: 127.0.0.1\n  port: 3307\n  user: local\n  database: yia_blog\nredis:\n  addr: 127.0.0.1:6379\n  db_index: 0\n"
	for _, test := range []struct {
		text  string
		valid bool
	}{
		{valid, true},
		{strings.Replace(valid, "env: local", "env: prod", 1), false},
		{strings.Replace(valid, "host: 127.0.0.1", "host: 192.0.2.1", 1), false},
		{strings.Replace(valid, "addr: 127.0.0.1", "addr: 192.0.2.1", 1), false},
		{strings.Replace(valid, "6379", "70000", 1), false},
		{"invalid: [private-value", false},
	} {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(test.text), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := readConfig(path)
		if (err == nil) != test.valid {
			t.Fatal("configuration boundary failed")
		}
		if err != nil && strings.Contains(err.Error(), "private-value") {
			t.Fatal("configuration contents leaked")
		}
	}
}

func TestRecoveryRejectsAmbiguousModes(t *testing.T) {
	for _, modes := range [][]string{
		{}, {"-check", "-verify-stdin"}, {"-check", "-password-stdin"},
		{"-verify-stdin", "-password-stdin"}, {"-check", "-verify-stdin", "-password-stdin"},
	} {
		args := append([]string{"-name", "super_admin", "-config", filepath.Join(t.TempDir(), "missing.yaml")}, modes...)
		_, _, status, message := run(args, strings.NewReader("SampleOnly123"))
		if status != "failed" || !strings.HasPrefix(message, "参数错误") {
			t.Fatal("ambiguous mode was not rejected before configuration or database access")
		}
	}
	args := []string{"-name", "super_admin", "-config", filepath.Join(t.TempDir(), "missing.yaml"), "-verify-stdin"}
	_, _, status, message := run(args, strings.NewReader("SampleOnly123"))
	if status != "failed" || message != "无法读取本地配置文件。" {
		t.Fatal("read-only mode was not accepted independently")
	}
}
