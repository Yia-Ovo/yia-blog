package repo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArticleImagePublishDoesNotReplaceExistingOnWindows(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "published")
	temporary := filepath.Join(directory, "temporary")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishArticleImageFile(temporary, target); !os.IsExist(err) {
		t.Fatalf("existing target must reject replacement: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "original" {
		t.Fatalf("published target changed: %q %v", data, err)
	}
	if _, err := os.Stat(temporary); err != nil {
		t.Fatalf("failed publication must leave the temporary file for cleanup: %v", err)
	}
	newTarget := filepath.Join(directory, "new")
	if err := publishArticleImageFile(temporary, newTarget); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(newTarget); err != nil || string(data) != "replacement" {
		t.Fatalf("new target is incomplete: %q %v", data, err)
	}
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatalf("successful publication did not move temporary file: %v", err)
	}
}

func TestArticleImagePublishSupportsLongPathsOnWindows(t *testing.T) {
	directory := filepath.Join(t.TempDir(), strings.Repeat("long-", 24), strings.Repeat("long-", 24))
	r := NewArticleImageFileRepo(directory)
	data := avatarFilePNG(t)
	name, err := r.Save(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if image, err := r.Read(context.Background(), name); err != nil || !bytes.Equal(image.Data, data) {
		t.Fatalf("long-path image round trip failed: %v", err)
	}
}
