package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"cashmate-api/config"
	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
)

// categoryInput menampung payload create & update kategori.
type categoryInput struct {
	Name string `json:"name" binding:"required,min=1"`
	Type string `json:"type" binding:"required,oneof=income expense"`
}

// CategoriesIndex menampilkan kategori global + kategori milik user yang login.
func CategoriesIndex(c *gin.Context) {
	userID := mustUserID(c)
	q := config.DB.Model(&models.Category{}).
		Where("user_id IS NULL OR user_id = ?", userID)

	if t := c.Query("type"); t != "" {
		t = strings.ToLower(t)
		if t == "income" || t == "expense" {
			q = q.Where("type = ?", t)
		}
	}

	var categories []models.Category
	if err := q.Order("type ASC, name ASC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// validateCategory memvalidasi & menormalkan input kategori.
func validateCategory(input *categoryInput) string {
	input.Name = helpers.Sanitize(input.Name)
	if input.Name == "" {
		return "nama kategori wajib diisi"
	}
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	if input.Type != "income" && input.Type != "expense" {
		return "tipe kategori harus 'income' atau 'expense'"
	}
	return ""
}

// categoryExists cek kategori dgn nama+tipe yang sama (global atau milik user).
func categoryExists(userID uint, name, categoryType string, excludeID uint) bool {
	var count int64
	q := config.DB.Model(&models.Category{}).
		Where("name = ? AND type = ? AND (user_id IS NULL OR user_id = ?)", name, categoryType, userID)
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

// CategoriesStore membuat kategori baru milik user yang login.
func CategoriesStore(c *gin.Context) {
	var input categoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama dan tipe kategori wajib diisi"})
		return
	}

	if msg := validateCategory(&input); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userID := mustUserID(c)
	if categoryExists(userID, input.Name, input.Type, 0) {
		c.JSON(http.StatusConflict, gin.H{"error": "kategori sudah tersedia"})
		return
	}

	category := models.Category{
		UserID: &userID,
		Name:   input.Name,
		Type:   input.Type,
	}

	if err := config.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "kategori berhasil dibuat",
		"data":    category,
	})
}

// CategoriesUpdate mengubah kategori milik user (anti-IDOR).
func CategoriesUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id kategori tidak valid"})
		return
	}

	userID := mustUserID(c)
	var category models.Category
	// User hanya boleh memodifikasi kategori miliknya (bukan global).
	if err := config.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "kategori tidak ditemukan"})
		return
	}

	var input categoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama dan tipe kategori wajib diisi"})
		return
	}

	if msg := validateCategory(&input); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	if categoryExists(userID, input.Name, input.Type, uint(id)) {
		c.JSON(http.StatusConflict, gin.H{"error": "kategori dengan nama dan tipe yang sama sudah tersedia"})
		return
	}

	category.Name = input.Name
	category.Type = input.Type

	if err := config.DB.Save(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "kategori berhasil diperbarui",
		"data":    category,
	})
}

// CategoriesDestroy menghapus kategori milik user (anti-IDOR).
func CategoriesDestroy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id kategori tidak valid"})
		return
	}

	userID := mustUserID(c)
	var category models.Category
	if err := config.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "kategori tidak ditemukan"})
		return
	}

	var used int64
	config.DB.Model(&models.Transaction{}).
		Where("category_id = ? AND type = ?", id, category.Type).
		Count(&used)
	if used > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "kategori masih digunakan oleh transaksi. Hapus/pindahkan transaksinya dulu."})
		return
	}

	if err := config.DB.Delete(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "kategori berhasil dihapus"})
}

// CategoriesRestore memulihkan kategori milik user yang sudah di-soft delete.
func CategoriesRestore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id kategori tidak valid"})
		return
	}

	userID := mustUserID(c)
	var category models.Category
	if err := config.DB.Unscoped().Where("id = ? AND user_id = ?", id, userID).
		First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "kategori tidak ditemukan"})
		return
	}

	if err := config.DB.Unscoped().Model(&category).Update("deleted_at", nil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "kategori berhasil dipulihkan"})
}
