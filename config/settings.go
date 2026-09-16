package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AllowedOrigins mengembalikan daftar origin CORS dari env CORS_ORIGINS
// (dipisahkan koma). Jika kosong, memakai daftar default untuk development.
func AllowedOrigins() []string {
	raw := viper.GetString("CORS_ORIGINS")
	if raw == "" {
		return []string{
			"http://localhost:8000",
			"http://127.0.0.1:8000",
			"http://localhost:3000",
		}
	}

	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// BusinessLocation returns the configured business timezone and falls back to
// Asia/Jakarta when an invalid value is supplied.
func BusinessLocation() *time.Location {
	location, err := time.LoadLocation(AppTimezone())
	if err != nil {
		return time.FixedZone("Asia/Jakarta", 7*60*60)
	}
	return location
}

// UploadDir adalah folder penyimpanan file upload (foto transaksi & avatar).
func UploadDir() string {
	return viper.GetString("UPLOAD_DIR")
}

// UploadMaxBytes adalah batas ukuran satu file upload dalam byte.
func UploadMaxBytes() int64 {
	sizeMB := viper.GetInt64("UPLOAD_MAX_SIZE_MB")
	if sizeMB <= 0 {
		sizeMB = 5
	}
	return sizeMB * 1024 * 1024
}

// AssetBaseURL adalah prefix absolut opsional untuk URL file upload. Jika
// kosong, URL dikembalikan sebagai path relatif ("/uploads/...").
func AssetBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(viper.GetString("ASSET_BASE_URL")), "/")
}

// TrustedProxies mengembalikan daftar proxy yang dipercaya (misal IP nginx)
// untuk menghitung ClientIP yang benar pada rate limiter.
func TrustedProxies() []string {
	raw := viper.GetString("TRUSTED_PROXIES")
	if raw == "" {
		return nil
	}

	var proxies []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies = append(proxies, p)
		}
	}
	return proxies
}
