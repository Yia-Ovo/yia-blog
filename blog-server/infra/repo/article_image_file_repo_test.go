package repo

import (
	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestArticleImageAtomicHashStorageAndPathSafety(t *testing.T) {
	for _, format := range []string{"png", "jpg"} {
		t.Run(format, func(t *testing.T) {
			directory := t.TempDir()
			first, second := NewArticleImageFileRepo(directory), NewArticleImageFileRepo(directory)
			var encoded bytes.Buffer
			source := image.NewNRGBA(image.Rect(0, 0, 40, 20))
			if format == "png" {
				if err := png.Encode(&encoded, source); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := jpeg.Encode(&encoded, source, nil); err != nil {
					t.Fatal(err)
				}
			}
			data := encoded.Bytes()
			sum := sha256.Sum256(data)
			expected := hex.EncodeToString(sum[:]) + "." + format
			var wg sync.WaitGroup
			for i := 0; i < 12; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					r := first
					if i%2 == 1 {
						r = second
					}
					name, err := r.Save(context.Background(), data)
					if err != nil || name != expected {
						t.Errorf("save %q: %v", name, err)
					}
				}(i)
			}
			wg.Wait()
			got, err := first.Read(context.Background(), expected)
			if err != nil || !bytes.Equal(got.Data, data) || got.Width != 40 || got.Height != 20 {
				t.Fatalf("read: %v", err)
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 {
				t.Fatalf("atomic write leftovers: %v %v", entries, err)
			}
			for _, bad := range []string{"../outside.png", "..\\outside.png", "file.png", strings.ToUpper(expected), strings.Repeat("a", 64) + ".svg", strings.Repeat("b", 64) + "." + format} {
				if _, err := first.Read(context.Background(), bad); !errors.Is(err, domainrepo.ErrArticleImageNotFound) {
					t.Fatalf("unsafe or missing name %q: %v", bad, err)
				}
			}
			wrongExtension := strings.TrimSuffix(expected, "."+format) + ".jpg"
			if format == "jpg" {
				wrongExtension = strings.TrimSuffix(expected, ".jpg") + ".png"
			}
			if err := os.WriteFile(filepath.Join(directory, wrongExtension), data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := first.Read(context.Background(), wrongExtension); !errors.Is(err, domainrepo.ErrArticleImageNotFound) {
				t.Fatal("content-type mismatch served")
			}
			if err := os.WriteFile(filepath.Join(directory, expected), []byte("tampered"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := first.Read(context.Background(), expected); err == nil {
				t.Fatal("changed immutable content served")
			}
		})
	}
}

func TestArticleImageStorageRejectsInvalidOrCanceled(t *testing.T) {
	r := NewArticleImageFileRepo(t.TempDir())
	var oversized bytes.Buffer
	if err := png.Encode(&oversized, image.NewNRGBA(image.Rect(0, 0, 1921, 1))); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{nil, []byte("not an image"), make([]byte, consts.ArticleImageMaxStoredBytes+1), oversized.Bytes()} {
		if _, err := r.Save(context.Background(), data); err == nil {
			t.Fatal("invalid normalized image stored")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.Save(ctx, avatarFilePNG(t)); err == nil {
		t.Fatal("canceled write stored")
	}
	if _, err := r.Read(ctx, strings.Repeat("a", 64)+".png"); err == nil {
		t.Fatal("canceled read accepted")
	}
	if NewArticleImageFileRepo("").directory != consts.ArticleImageDefaultDirectory {
		t.Fatal("default directory changed")
	}
}
