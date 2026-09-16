package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cashmate-api/config"
)

// UploadURLPrefix adalah path publik tempat file upload disajikan.
const UploadURLPrefix = "/uploads/"

var (
	ErrFileEmpty       = errors.New("file kosong")
	ErrFileTooLarge    = errors.New("ukuran file melebihi batas")
	ErrUnsupportedType = errors.New("format gambar tidak didukung")
)

// allowedImages memetakan MIME type yang dideteksi server ke ekstensi file.
// Ekstensi ditentukan server, bukan nama file dari client.
var allowedImages = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// SaveImage memvalidasi lalu menyimpan file gambar ke
// <UPLOAD_DIR>/<subDir>/<random>.<ext> dan mengembalikan URL publiknya.
// subDir dibangun oleh pemanggil dari nilai internal (mis. "avatars/1").
func SaveImage(header *multipart.FileHeader, subDir string) (string, error) {
	if header == nil || header.Size <= 0 {
		return "", ErrFileEmpty
	}
	if header.Size > config.UploadMaxBytes() {
		return "", ErrFileTooLarge
	}

	ext, err := imageExtension(header)
	if err != nil {
		return "", err
	}

	name, err := randomName(ext)
	if err != nil {
		return "", err
	}

	relPath := filepath.Join(filepath.FromSlash(path.Clean("/"+subDir)), name)
	absDir := filepath.Join(config.UploadDir(), filepath.Dir(relPath))
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return "", err
	}

	src, err := header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(filepath.Join(config.UploadDir(), relPath))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return AssetURL(relPath), nil
}

// RemoveImage menghapus file lokal dari URL yang pernah dikembalikan
// SaveImage. URL eksternal (mis. CDN) diabaikan tanpa error.
func RemoveImage(url string) {
	relPath, ok := localPath(url)
	if !ok {
		return
	}
	_ = os.Remove(filepath.Join(config.UploadDir(), relPath))
}

// AssetURL mengubah path relatif (mis. "avatars/1/x.jpg") menjadi URL publik.
func AssetURL(relPath string) string {
	clean := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(relPath)), "/")
	url := UploadURLPrefix + clean
	if base := config.AssetBaseURL(); base != "" {
		return base + url
	}
	return url
}

func imageExtension(header *multipart.FileHeader) (string, error) {
	f, err := header.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	buffer := make([]byte, 512)
	n, err := io.ReadFull(f, buffer)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	if n == 0 {
		return "", ErrFileEmpty
	}

	ext, ok := allowedImages[http.DetectContentType(buffer[:n])]
	if !ok {
		return "", ErrUnsupportedType
	}
	return ext, nil
}

func localPath(url string) (string, bool) {
	cleaned := strings.TrimSpace(url)
	if base := config.AssetBaseURL(); base != "" && strings.HasPrefix(cleaned, base) {
		cleaned = strings.TrimPrefix(cleaned, base)
	}
	if !strings.HasPrefix(cleaned, UploadURLPrefix) {
		return "", false
	}
	rel := strings.TrimPrefix(cleaned, UploadURLPrefix)
	rel = strings.TrimPrefix(path.Clean("/"+rel), "/")
	if rel == "" || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return filepath.FromSlash(rel), true
}

func randomName(ext string) (string, error) {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer) + ext, nil
}
