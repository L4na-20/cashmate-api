package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"cashmate-api/helpers"
	"cashmate-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type categoryInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func validateCategory(input *categoryInput) string {
	input.Name = helpers.Sanitize(input.Name)
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	if input.Name == "" {
		return "nama kategori wajib diisi"
	}
	if input.Type != models.TransactionIncome && input.Type != models.TransactionExpense {
		return "tipe kategori harus income atau expense"
	}
	return ""
}

func categoryView(category models.Category) gin.H {
	return gin.H{
		"id":          category.ID,
		"business_id": category.BusinessID,
		"name":        category.Name,
		"type":        category.Type,
		"created_at":  category.CreatedAt,
		"updated_at":  category.UpdatedAt,
		"deleted_at":  category.DeletedAt,
	}
}

func categoryScope(c *gin.Context, includeDeleted bool) *gorm.DB {
	query := configDB().Where("(business_id = ? OR business_id IS NULL)", currentBusinessID(c))
	if includeDeleted {
		return query.Unscoped()
	}
	return query
}

func categoryExists(c *gin.Context, name, categoryType string, excludeID uint) bool {
	query := categoryScope(c, false).Where("name = ? AND type = ?", name, categoryType)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	query.Count(&count)
	return count > 0
}

func CategoriesIndex(c *gin.Context) {
	owner := isOwner(c)
	status := parseResourceStatus(c)
	if !owner {
		status = "active"
	}
	query := categoryScope(c, owner && status != "active")
	if status == "disabled" {
		query = query.Where("deleted_at IS NOT NULL")
	} else if status == "active" {
		query = query.Where("deleted_at IS NULL")
	}
	if categoryType := strings.ToLower(strings.TrimSpace(c.Query("type"))); categoryType == models.TransactionIncome || categoryType == models.TransactionExpense {
		query = query.Where("type = ?", categoryType)
	}
	var categories []models.Category
	if err := query.Order("type ASC, name ASC, id ASC").Find(&categories).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal mengambil kategori", nil)
		return
	}
	views := make([]gin.H, 0, len(categories))
	for _, category := range categories {
		views = append(views, categoryView(category))
	}
	helpers.Success(c, http.StatusOK, "kategori berhasil diambil", views)
}

func CategoriesStore(c *gin.Context) {
	var input categoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "nama dan tipe kategori wajib diisi", nil)
		return
	}
	if msg := validateCategory(&input); msg != "" {
		helpers.Error(c, http.StatusBadRequest, msg, nil)
		return
	}
	if categoryExists(c, input.Name, input.Type, 0) {
		helpers.Error(c, http.StatusConflict, "kategori sudah tersedia", nil)
		return
	}
	businessID := currentBusinessID(c)
	category := models.Category{BusinessID: &businessID, Name: input.Name, Type: input.Type}
	if err := configDB().Create(&category).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal membuat kategori", nil)
		return
	}
	helpers.Success(c, http.StatusCreated, "kategori berhasil dibuat", categoryView(category))
}

func CategoriesUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id kategori tidak valid", nil)
		return
	}
	var category models.Category
	if err := scopedBusiness(configDB(), c).Where("id = ? AND business_id = ?", id, currentBusinessID(c)).First(&category).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "kategori tidak ditemukan", nil)
		return
	}
	var input categoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.Error(c, http.StatusBadRequest, "nama dan tipe kategori wajib diisi", nil)
		return
	}
	if msg := validateCategory(&input); msg != "" {
		helpers.Error(c, http.StatusBadRequest, msg, nil)
		return
	}
	if categoryExists(c, input.Name, input.Type, uint(id)) {
		helpers.Error(c, http.StatusConflict, "kategori dengan nama dan tipe yang sama sudah tersedia", nil)
		return
	}
	if err := scopedBusiness(configDB(), c).Model(&category).Updates(map[string]any{"name": input.Name, "type": input.Type}).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memperbarui kategori", nil)
		return
	}
	category.Name, category.Type = input.Name, input.Type
	helpers.Success(c, http.StatusOK, "kategori berhasil diperbarui", categoryView(category))
}

func CategoriesDestroy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id kategori tidak valid", nil)
		return
	}
	var category models.Category
	if err := scopedBusiness(configDB(), c).Where("id = ? AND business_id = ?", id, currentBusinessID(c)).First(&category).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "kategori tidak ditemukan", nil)
		return
	}
	if err := configDB().Delete(&category).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal menonaktifkan kategori", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "kategori berhasil dinonaktifkan", nil)
}

func CategoriesRestore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		helpers.Error(c, http.StatusBadRequest, "id kategori tidak valid", nil)
		return
	}
	var category models.Category
	if err := scopedBusiness(configDB().Unscoped(), c).Where("id = ? AND business_id = ?", id, currentBusinessID(c)).First(&category).Error; err != nil {
		helpers.Error(c, http.StatusNotFound, "kategori tidak ditemukan", nil)
		return
	}
	if err := configDB().Unscoped().Model(&category).Update("deleted_at", nil).Error; err != nil {
		helpers.Error(c, http.StatusInternalServerError, "gagal memulihkan kategori", nil)
		return
	}
	helpers.Success(c, http.StatusOK, "kategori berhasil dipulihkan", nil)
}
