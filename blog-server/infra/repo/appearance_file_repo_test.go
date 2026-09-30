package repo

import (
	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"testing/fstest"
)

func TestAppearanceCatalogAndContentAddressing(t *testing.T) {
	data := []byte("prepared image bytes")
	sum := sha256.Sum256(data)
	id := hex.EncodeToString(sum[:])[:20] + ".jpg"
	catalog, _ := json.Marshal([]string{id})
	files := fstest.MapFS{
		consts.AppearanceCatalogPath:      &fstest.MapFile{Data: catalog},
		consts.AppearanceAssetPrefix + id: &fstest.MapFile{Data: data},
		"private.txt":                     &fstest.MapFile{Data: []byte("not public")},
	}
	r := NewAppearanceFileRepo(files)
	ids, err := r.List(context.Background())
	if err != nil || len(ids) != 1 {
		t.Fatalf("List: %v %v", ids, err)
	}
	ids[0] = "changed"
	again, _ := r.List(context.Background())
	if again[0] != id {
		t.Fatal("returned mutable repository state")
	}
	loaded, err := r.Read(context.Background(), id)
	if err != nil || string(loaded) != string(data) {
		t.Fatalf("Read: %q %v", loaded, err)
	}
	for _, candidate := range []string{"../private.txt", id + "/../../private.txt", "private.txt", "00000000000000000000.jpg"} {
		if _, err := r.Read(context.Background(), candidate); !errors.Is(err, domainrepo.ErrAppearanceImageNotFound) {
			t.Fatalf("unlisted path accepted: %s: %v", candidate, err)
		}
	}
	files[consts.AppearanceAssetPrefix+id].Data = []byte("replaced content")
	if _, err := r.Read(context.Background(), id); err == nil {
		t.Fatal("mutable content accepted under immutable URL")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.Read(ctx, id); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored")
	}
}

func TestAppearanceRejectsInvalidCatalog(t *testing.T) {
	for _, catalog := range []string{`["../private.jpg"]`, `["00000000000000000000.jpg","00000000000000000000.jpg"]`, `broken`} {
		r := NewAppearanceFileRepo(fstest.MapFS{consts.AppearanceCatalogPath: &fstest.MapFile{Data: []byte(catalog)}})
		if _, err := r.List(context.Background()); err == nil {
			t.Fatalf("invalid catalog accepted: %s", catalog)
		}
	}
}
