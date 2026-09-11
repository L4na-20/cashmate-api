package controllers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---------- Token Blacklist (in-memory) ----------
// Untuk menyederhanakan, invalidasi token dilakukan lewat blacklist
// in-memory. Pada produksi besar, ganti dengan Redis/database.
var (
	blacklistMu sync.Mutex
	blacklist   = map[string]time.Time{} // token -> expiry
)

func blacklistToken(token string, expiry time.Time) {
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	blacklist[token] = expiry
}

func isBlacklisted(token string) bool {
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	exp, ok := blacklist[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(blacklist, token)
		return false
	}
	return true
}

// refreshStore menyimpan refresh token yang sedang aktif per user
// agar bisa di-revoke pada saat logout.
var (
	refreshMu sync.Mutex
	refreshDB = map[uint]string{} // user_id -> refresh token
)

func storeRefreshToken(userID uint, token string) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	refreshDB[userID] = token
}

func getRefreshToken(userID uint) (string, bool) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	t, ok := refreshDB[userID]
	return t, ok
}

func deleteRefreshToken(userID uint) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	delete(refreshDB, userID)
}

// ---------- Input structs ----------

type registerInput struct {
	Name     string `json:"name" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register membuat akun baru dengan password ter-hash (Bcrypt cost 10).
func AuthRegister(c *gin.Context) {
	var input registerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid: name (min 3), email, password (min 8)"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(helpers.Sanitize(input.Email)))
	input.Name = helpers.Sanitize(input.Name)

	var count int64
	config.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email sudah terdaftar"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost) // cost 10
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memproses password"})
		return
	}

	user := models.User{
		Name:     input.Name,
		Email:    email,
		Password: string(hash),
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Buat wallet default "Cash" untuk user baru.
	if err := config.DB.Create(&models.Wallet{
		UserID:   user.ID,
		Name:     "Cash",
		Balance:  0,
		Currency: "IDR",
	}).Error; err != nil {
		// Jangan gagalkan registrasi hanya karena wallet; tapi log saja.
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "registrasi berhasil",
		"data": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}

// Login memverifikasi kredensial dan mengembalikan Access + Refresh Token.
func AuthLogin(c *gin.Context) {
	var input loginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email dan password wajib diisi"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))

	var user models.User
	err := config.DB.Where("email = ?", email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
		return
	}

	accessToken, err := helpers.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token"})
		return
	}

	refreshToken, err := helpers.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token"})
		return
	}

	storeRefreshToken(user.ID, refreshToken)

	c.JSON(http.StatusOK, gin.H{
		"message": "login berhasil",
		"data": gin.H{
			"user": gin.H{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

// Refresh menghasilkan Access Token baru dari Refresh Token yang valid.
func AuthRefresh(c *gin.Context) {
	var input refreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token wajib diisi"})
		return
	}

	if isBlacklisted(input.RefreshToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token tidak valid"})
		return
	}

	claims, err := helpers.ParseRefreshToken(input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Pastikan refresh token yang dipakai adalah milik user tsb (anti token reuse).
	if stored, ok := getRefreshToken(claims.UserID); !ok || stored != input.RefreshToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token tidak valid"})
		return
	}

	newAccess, err := helpers.GenerateAccessToken(claims.UserID, claims.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "token berhasil diperbarui",
		"access_token": newAccess,
	})
}

// Logout menblacklist token & menghapus refresh token user.
func AuthLogout(c *gin.Context) {
	accessHeader := c.GetHeader("Authorization")
	var refreshToken string

	// Ambil refresh token dari body bila dikirim.
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&body)
	refreshToken = body.RefreshToken

	if accessHeader != "" {
		parts := strings.SplitN(accessHeader, " ", 2)
		if len(parts) == 2 {
			tok := strings.TrimSpace(parts[1])
			blacklistToken(tok, time.Now().Add(time.Minute*30))
		}
	}

	// Ambil user_id dari context (jika route dilindungi AuthJWT).
	var userID uint
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(uint); ok2 {
			userID = id
		}
	}

	// Jika refresh token disertakan, blacklist & invalidasi session.
	if refreshToken != "" {
		blacklistToken(refreshToken, time.Now().Add(time.Hour*24*7))
		if userID == 0 {
			if claims, err := helpers.ParseRefreshToken(refreshToken); err == nil {
				userID = claims.UserID
			}
		}
	}

	if userID != 0 {
		deleteRefreshToken(userID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "logout berhasil, token telah diinvalidasi"})
}
