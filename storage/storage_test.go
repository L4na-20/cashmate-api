package storage

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

var pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

func multipartHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("photo", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	form, err := multipart.NewReader(body, writer.Boundary()).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("read form: %v", err)
	}
	return form.File["photo"][0]
}

func TestSaveAndRemoveImage(t *testing.T) {
	dir := t.TempDir()
	viper.Set("UPLOAD_DIR", dir)
	viper.Set("UPLOAD_MAX_SIZE_MB", 5)
	viper.Set("ASSET_BASE_URL", "")
	defer viper.Reset()

	url, err := SaveImage(multipartHeader(t, "proof.png", pngSignature), "transactions/7")
	if err != nil {
		t.Fatalf("SaveImage: %v", err)
	}
	if !strings.HasPrefix(url, "/uploads/transactions/7/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("unexpected url %q", url)
	}

	path := filepath.Join(dir, "transactions", "7", filepath.Base(url))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file not found: %v", err)
	}

	RemoveImage(url)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, stat err = %v", err)
	}
}

func TestSaveImageRejectsUnsupportedContent(t *testing.T) {
	viper.Set("UPLOAD_DIR", t.TempDir())
	viper.Set("UPLOAD_MAX_SIZE_MB", 5)
	viper.Set("ASSET_BASE_URL", "")
	defer viper.Reset()

	if _, err := SaveImage(multipartHeader(t, "note.txt", []byte("this is not an image")), "avatars/1"); err != ErrUnsupportedType {
		t.Fatalf("err = %v, want ErrUnsupportedType", err)
	}
}

func TestAssetURLUsesConfiguredBase(t *testing.T) {
	viper.Set("ASSET_BASE_URL", "https://example.test/cashmate/")
	defer viper.Reset()

	if got := AssetURL("avatars/1/a.png"); got != "https://example.test/cashmate/uploads/avatars/1/a.png" {
		t.Fatalf("AssetURL = %q", got)
	}
}
