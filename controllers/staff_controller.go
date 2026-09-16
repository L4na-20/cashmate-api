package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type staffInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func staffView(user models.User) gin.H {
	return gin.H{
		"id":            user.ID,
		"business_id":   user.BusinessID,
		"name":          user.Name,
		"email":         user.Email,
		"role":          models.NormalizeRole(user.Role),
		"profile_photo": user.ProfilePhoto,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"deleted_at":    user.DeletedAt,
	}
}

func StaffIndex(c *gin.Context) {
	query := config.DB.Model(&models.User{}).Where("business_id = ?", currentBusinessID(c))
	status := parseResourceStatus(c)
	if status != "active" {
		query = query.Unscoped()
	}
	if status == "disabled" {
		query = query.Where("deleted_at IS NOT NULL")
	} else if status == "active" {
		query = query.Where("deleted_at IS NULL")
	}
	query = query.Where("LOWER(role) <> ?", strings.ToLower(models.RoleOwner))
	var staff []models.User
	if err := query.Order("created_at ASC, id ASC").Find(&staff).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal mengambil Staff", nil)
		return
	}
	views := make([]gin.H, 0, len(staff))
	for _, user := range staff {
		views = append(views, staffView(user))
	}
	helpers.Success(c, http.StatusOK, "Staff berhasil diambil", views)
}

func StaffStore(c *gin.Context) {
	var input staffInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "payload Staff tidak valid", nil)
		return
	}
	input.Name = helpers.Sanitize(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(helpers.Sanitize(input.Email)))
	if len(input.Name) < 3 || input.Email == "" || len(input.Password) < 8 {
		helpers.Error(c, http.StatusBadRequest, "name, email, dan password (min 8) wajib valid", nil)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memproses password Staff", nil)
		return
	}
	var user models.User
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Unscoped().Model(&models.User{}).Where("email = ?", input.Email).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrDuplicatedKey
		}
		user = models.User{
			BusinessID:  currentBusinessID(c),
			Name:        input.Name,
			Email:       input.Email,
			Password:    string(hash),
			Role:        models.RoleStaff,
			AuthVersion: 0,
		}
		return tx.Create(&user).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		helpers.Error(c, http.StatusConflict, "email sudah terdaftar", nil)
		return
	}
	if err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal membuat Staff", nil)
		return
	}
	helpers.Success(c, http.StatusCreated, "Staff berhasil dibuat", staffView(user))
}

func StaffDestroy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id Staff tidak valid", nil)
		return
	}
	if uint(id) == currentUserID(c) {
		helpers.Error(c, http.StatusBadRequest, "Owner tidak dapat menonaktifkan akunnya sendiri", nil)
		return
	}
	var user models.User
	if err := config.DB.Where("id = ? AND business_id = ?", id, currentBusinessID(c)).First(&user).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "Staff tidak ditemukan", nil)
		return
	}
	if models.NormalizeRole(user.Role) != models.RoleStaff {
		helpers.Error(c, http.StatusForbidden, "akun tersebut bukan Staff", nil)
		return
	}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).UpdateColumn("auth_version", gorm.Expr("auth_version + 1")).Error; err != nil {
			return err
		}
		return tx.Delete(&user).Error
	}); err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal menonaktifkan Staff", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "Staff berhasil dinonaktifkan", nil)
}

// AuthUsers/AuthDeleteUser remain compatibility aliases for clients using the
// old Owner user routes; both are still tenant-scoped and Staff-only.
func AuthUsers(c *gin.Context) { StaffIndex(c) }

func AuthDeleteUser(c *gin.Context) { StaffDestroy(c) }
