package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPasswordPreservesSpacesAndRejectsMultipleLines(t *testing.T) {
	for _, suffix := range []string{"", "\n", "\r\n"} {
		password, err := readPassword(strings.NewReader("  Owner-pass73 " + suffix))
		if err != nil || password != "  Owner-pass73 " {
			t.Fatal("password whitespace was changed")
		}
	}
	for _, input := range []string{"Owner-pass73\nsecond", strings.Repeat("x", 1000), "Owner\x00pass73"} {
		if _, err := readPassword(strings.NewReader(input)); err == nil {
			t.Fatal("unsafe or oversized input accepted")
		}
	}
}

func TestReadConfigIsLoopbackOnlyAndDoesNotEchoBrokenSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	for _, host := range []string{"127.0.0.1", "::1", "192.0.2.1", "localhost"} {
		data := "mysql:\n  host: '" + host + "'\n  port: 13306\n  database: yia_blog\n  user: yia_blog\n  password: private-test-value\n"
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := readConfig(path)
		wantOK := host == "127.0.0.1" || host == "::1"
		if (err == nil) != wantOK {
			t.Fatalf("loopback restriction failed for %s", host)
		}
	}
	if err := os.WriteFile(path, []byte("mysql:\n  password: [private-test-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readConfig(path)
	if err == nil || strings.Contains(err.Error(), "private-test-value") {
		t.Fatal("parser error disclosed the private configuration")
	}
}
