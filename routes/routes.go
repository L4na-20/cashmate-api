package routes

import (
	"net/http"

	"cashmate-api/config"
	"cashmate-api/controllers"
	"cashmate-api/middleware"
	"github.com/gin-gonic/gin"
)

// Setup registers all REST endpoints under /api.
func Setup(r *gin.Engine) {
	r.Use(middleware.SecurityHeaders())

	// File upload (foto transaksi & foto profil) disajikan sebagai aset statis.
	r.Static("/uploads", config.UploadDir())

	api := r.Group("/api")
	api.Use(middleware.CORS())

	api.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	api.Use(middleware.RateLimitGeneral())

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "cashmate-api"})
	})

	auth := api.Group("/auth")
	auth.Use(middleware.RateLimitAuth())
	auth.POST("/register", controllers.AuthRegister)
	auth.POST("/login", controllers.AuthLogin)
	auth.POST("/refresh", controllers.AuthRefresh)

	protected := api.Group("")
	protected.Use(middleware.AuthJWT())
	protected.POST("/auth/logout", controllers.AuthLogout)
	protected.GET("/auth/me", controllers.AuthMe)
	protected.PUT("/auth/me/photo", controllers.AuthUpdateProfilePhoto)

	// Owner and Staff may read active Business wallets/categories and create
	// transactions. Controllers still apply role-safe response filtering.
	protected.GET("/wallets", controllers.WalletsIndex)
	protected.GET("/wallets/:id", controllers.WalletsShow)
	protected.GET("/categories", controllers.CategoriesIndex)
	protected.GET("/transactions", controllers.TransactionsIndex)
	protected.POST("/transactions", controllers.TransactionsStore)

	owner := protected.Group("")
	owner.Use(middleware.RoleOwner())
	owner.GET("/dashboard/summary", controllers.DashboardSummary)
	owner.GET("/reports/monthly", controllers.MonthlyReport)

	owner.POST("/staff", controllers.StaffStore)
	owner.GET("/staff", controllers.StaffIndex)
	owner.DELETE("/staff/:id", controllers.StaffDestroy)

	owner.POST("/wallets", controllers.WalletsStore)
	owner.PUT("/wallets/:id", controllers.WalletsUpdate)
	owner.DELETE("/wallets/:id", controllers.WalletsDestroy)
	owner.POST("/wallets/:id/restore", controllers.WalletsRestore)

	owner.POST("/categories", controllers.CategoriesStore)
	owner.PUT("/categories/:id", controllers.CategoriesUpdate)
	owner.DELETE("/categories/:id", controllers.CategoriesDestroy)
	owner.POST("/categories/:id/restore", controllers.CategoriesRestore)

	owner.PUT("/transactions/:id", controllers.TransactionsUpdate)
	owner.DELETE("/transactions/:id", controllers.TransactionsDestroy)
	owner.POST("/transactions/:id/restore", controllers.TransactionsRestore)
	owner.DELETE("/transactions/:id/photos/:photo_id", controllers.TransactionPhotoDestroy)

	// Temporary compatibility aliases for clients using the old names. They
	// remain Owner-only and tenant-scoped.
	owner.GET("/auth/users", controllers.AuthUsers)
	owner.DELETE("/auth/users/:id", controllers.AuthDeleteUser)

}
