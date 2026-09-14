package config

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// LoadConfig membaca file .env / environment variable.
func LoadConfig() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("config: .env tidak ditemukan, menggunakan environment variable")
	}

	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("DB_USER", "root")
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("DB_NAME", "cashmate")
	viper.SetDefault("JWT_SECRET", "cashmate_default_insecure_access_secret_change_me")
	viper.SetDefault("JWT_REFRESH_SECRET", "cashmate_default_insecure_refresh_secret_change_me")
	viper.SetDefault("ACCESS_TOKEN_TTL_MINUTES", 30)
	viper.SetDefault("REFRESH_TOKEN_TTL_HOURS", 168) // 7 hari
	viper.SetDefault("APP_TIMEZONE", "Asia/Jakarta")
	viper.SetDefault("AUTO_MIGRATE", false)
}

// JWTSecret mengembalikan secret untuk Access Token.
func JWTSecret() string {
	return viper.GetString("JWT_SECRET")
}

// JWTRefreshSecret mengembalikan secret untuk Refresh Token.
func JWTRefreshSecret() string {
	return viper.GetString("JWT_REFRESH_SECRET")
}

// AccessTokenTTL mengembalikan durasi berlaku Access Token.
func AccessTokenTTL() int {
	return viper.GetInt("ACCESS_TOKEN_TTL_MINUTES")
}

// RefreshTokenTTL mengembalikan durasi berlaku Refresh Token.
func RefreshTokenTTL() int {
	return viper.GetInt("REFRESH_TOKEN_TTL_HOURS")
}

// AppTimezone returns the IANA timezone used for business-day calculations.
func AppTimezone() string {
	return viper.GetString("APP_TIMEZONE")
}

// AutoMigrateEnabled controls the development-only GORM schema bootstrap.
// Production should use the checked-in SQL migrations instead.
func AutoMigrateEnabled() bool {
	return viper.GetBool("AUTO_MIGRATE")
}

// AppPort mengembalikan port server.
func AppPort() string {
	return viper.GetString("APP_PORT")
}

// ConnectDB membuat koneksi MySQL menggunakan GORM.
func ConnectDB() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=%s",
		viper.GetString("DB_USER"),
		viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_HOST"),
		viper.GetString("DB_PORT"),
		viper.GetString("DB_NAME"),
		url.QueryEscape(AppTimezone()),
	)

	gormLogger := logger.Default.LogMode(logger.Warn)
	if viper.GetString("APP_ENV") == "debug" {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         gormLogger,
		TranslateError: true,
	})
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}

	DB = db
	fmt.Fprintf(os.Stdout, "[cashmate-api]  koneksi database berhasil\n")
}
