package repo

import (
	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func avatarFilePNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestAvatarFilesAtomicContentAddressedAndRestricted(t *testing.T) {
	directory := t.TempDir()
	r := NewAvatarFileRepo(directory)
	data := avatarFilePNG(t)
	var wg sync.WaitGroup
	names := make(chan string, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name, err := r.Save(context.Background(), data)
			if err != nil {
				t.Error(err)
				return
			}
			names <- name
		}()
	}
	wg.Wait()
	close(names)
	var name string
	for result := range names {
		if name != "" && result != name {
			t.Fatal("same bytes have different names")
		}
		name = result
	}
	if !storedAvatarName.MatchString(name) {
		t.Fatal("non-hash filename")
	}
	got, err := r.Read(context.Background(), name)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("read: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("atomic writes left extra files: %v %v", entries, err)
	}
	for _, bad := range []string{"../outside.png", "..\\outside.png", "file.png", strings.Repeat("a", 64) + ".jpg", strings.ToUpper(name), strings.Repeat("b", 64) + ".png"} {
		if _, err := r.Read(context.Background(), bad); !errors.Is(err, domainrepo.ErrAvatarNotFound) {
			t.Fatalf("unsafe or missing name %q: %v", bad, err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, name), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(context.Background(), name); err == nil {
		t.Fatal("modified content served under immutable URL")
	}
	if _, err := r.Save(context.Background(), []byte("not png")); err == nil {
		t.Fatal("non PNG stored")
	}
	if _, err := r.Save(context.Background(), make([]byte, consts.AvatarMaxStoredBytes+1)); err == nil {
		t.Fatal("oversized image stored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.Save(ctx, data); err == nil {
		t.Fatal("canceled upload written")
	}
}
