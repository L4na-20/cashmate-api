package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/middleware"
	"cashmate-api/models"
	"cashmate-api/storage"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type registerInput struct {
	BusinessName string `json:"business_name"`
	Name         string `json:"name"`
	OwnerName    string `json:"owner_name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
}

type loginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func userView(user *models.User) gin.H {
	return gin.H{
		"id":            user.ID,
		"business_id":   user.BusinessID,
		"name":          user.Name,
		"email":         user.Email,
		"role":          models.NormalizeRole(user.Role),
		"profile_photo": user.ProfilePhoto,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	}
}

func businessView(business *models.Business) gin.H {
	return gin.H{"id": business.ID, "name": business.Name}
}

func issueTokens(user *models.User) (string, string, error) {
	role := models.NormalizeRole(user.Role)
	access, err := helpers.GenerateAccessToken(user.ID, user.BusinessID, user.Email, role, user.AuthVersion)
	if err != nil {
		return "", "", err
	}
	refresh, err := helpers.GenerateRefreshToken(user.ID, user.BusinessID, user.Email, role, user.AuthVersion)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// AuthRegister creates a new tenant, its Owner, and the default Cash wallet
// atomically. Public registration can never create a Staff account.
func AuthRegister(c *gin.Context) {
	var input registerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "data registrasi tidak valid", nil)
		return
	}

	input.BusinessName = helpers.Sanitize(input.BusinessName)
	input.Name = helpers.Sanitize(input.Name)
	if input.Name == "" {
		input.Name = helpers.Sanitize(input.OwnerName)
	}
	input.Email = strings.ToLower(strings.TrimSpace(helpers.Sanitize(input.Email)))
	if len(input.BusinessName) < 2 || len(input.Name) < 3 || input.Email == "" || len(input.Password) < 8 {
		helpers.Error(c, http.StatusBadRequest, "business_name, name, email, dan password (min 8) wajib valid", nil)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memproses password", nil)
		return
	}

	var user models.User
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Unscoped().Model(&models.User{}).Where("email = ?", input.Email).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return gorm.ErrDuplicatedKey
		}

		business := models.Business{Name: input.BusinessName}
		if err := tx.Create(&business).Error; err != nil {
			return err
		}
		user = models.User{
			BusinessID:  business.ID,
			Name:        input.Name,
			Email:       input.Email,
			Password:    string(hash),
			Role:        models.RoleOwner,
			AuthVersion: 0,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return tx.Create(&models.Wallet{
			BusinessID: business.ID,
			Name:       "Cash",
			Balance:    0,
			Currency:   models.CurrencyIDR,
		}).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		helpers.Error(c, http.StatusConflict, "email sudah terdaftar", nil)
		return
	}
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal membuat Business dan Owner", nil)
		return
	}

	var business models.Business
	if err := config.DB.First(&business, user.BusinessID).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "registrasi berhasil tetapi Business tidak dapat dibaca", nil)
		return
	}
	helpers.Success(c, http.StatusCreated, "registrasi berhasil", gin.H{
		"user":     userView(&user),
		"business": businessView(&business),
	})
}

// AuthLogin verifies credentials and returns access/refresh tokens.
func AuthLogin(c *gin.Context) {
	var input loginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "email dan password wajib diisi", nil)
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	var user models.User
	if err := config.DB.Preload("Business").Where("email = ?", email).First(&user).Error; err != nil {
		helpers.Error(c, http.StatusUnauthorized, "email atau password salah", nil)
		return
	}
	if user.Business == nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		helpers.Error(c, http.StatusUnauthorized, "email atau password salah", nil)
		return
	}
	if models.NormalizeRole(user.Role) == "" {
		helpers.Error(c, http.StatusUnauthorized, "akun tidak memiliki role yang valid", nil)
		return
	}

	access, refresh, err := issueTokens(&user)
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal membuat token", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "login berhasil", gin.H{
		"user":          userView(&user),
		"business":      businessView(user.Business),
		"access_token":  access,
		"refresh_token": refresh,
	})
}

// AuthRefresh issues a new access token after rechecking the current database
// user. Refresh tokens are invalidated by auth_version on logout.
func AuthRefresh(c *gin.Context) {
	var input refreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "refresh_token wajib diisi", nil)
		return
	}
	claims, err := helpers.ParseRefreshToken(strings.TrimSpace(input.RefreshToken))
	if err != nil {
		helpers.Error(c, http.StatusUnauthorized, "refresh token tidak valid", nil)
		return
	}

	var user models.User
	if err := config.DB.Preload("Business").First(&user, claims.UserID).Error; err != nil || user.Business == nil || user.AuthVersion != claims.AuthVersion {
		helpers.Error(c, http.StatusUnauthorized, "refresh token tidak valid", nil)
		return
	}
	access, _, err := issueTokens(&user)
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal membuat token", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "token berhasil diperbarui", gin.H{"access_token": access})
}

// AuthLogout increments auth_version, invalidating access and refresh tokens
// for the user immediately. This intentionally logs out all active sessions.
func AuthLogout(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	if userID == 0 {
		helpers.Error(c, http.StatusUnauthorized, "sesi tidak valid", nil)
		return
	}
	if err := config.DB.Model(&models.User{}).Where("id = ? AND business_id = ?", userID, middleware.CurrentBusinessID(c)).UpdateColumn("auth_version", gorm.Expr("auth_version + 1")).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "logout gagal", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "logout berhasil", nil)
}

// AuthMe returns the active authenticated user and Business context.
func AuthMe(c *gin.Context) {
	user, ok := c.Get("current_user")
	if !ok {
		helpers.Error(c, http.StatusUnauthorized, "sesi tidak valid", nil)
		return
	}
	current, ok := user.(*models.User)
	if !ok || current.Business == nil {
		helpers.Error(c, http.StatusUnauthorized, "sesi tidak valid", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "sesi aktif", gin.H{
		"user":     userView(current),
		"business": businessView(current.Business),
	})
}

// AuthUpdateProfilePhoto memperbarui foto profil user yang sedang login dari
// multipart field "photo". File lama dihapus setelah update berhasil.
func AuthUpdateProfilePhoto(c *gin.Context) {
	value, ok := c.Get("current_user")
	current, valid := value.(*models.User)
	if !ok || !valid {
		helpers.Error(c, http.StatusUnauthorized, "sesi tidak valid", nil)
		return
	}

	header, err := c.FormFile("photo")
	if err != nil {
		helpers.Error(c, http.StatusBadRequest, "field file 'photo' wajib diisi", nil)
		return
	}

	url, err := storage.SaveImage(header, fmt.Sprintf("avatars/%d", current.ID))
	if err != nil {
		helpers.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := config.DB.Model(&models.User{}).
		Where("id = ? AND business_id = ?", current.ID, current.BusinessID).
		Update("profile_photo", url).Error; err != nil {
		storage.RemoveImage(url)
		helpers.Error(c, http.StatusInternalServerError, "gagal memperbarui foto profil", nil)
		return
	}

	if previous := current.ProfilePhoto; previous != "" && previous != url {
		storage.RemoveImage(previous)
	}
	current.ProfilePhoto = url

	helpers.Success(c, http.StatusOK, "foto profil berhasil diperbarui", userView(current))
}
