package routes

import (
	"cashmate-api/controllers"
	"cashmate-api/middleware"

	"github.com/gin-gonic/gin"
)

// Setup mendaftarkan seluruh route REST API dengan prefix /api.
func Setup(r *gin.Engine) {
	r.Use(middleware.SecurityHeaders())

	api := r.Group("/api")
	api.Use(middleware.CORS())
	api.Use(middleware.RateLimitGeneral())

	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "cashmate-api"})
	})

	// ---------- Auth (public, dengan rate limiter ketat) ----------
	auth := api.Group("/auth")
	auth.Use(middleware.RateLimitAuth())
	{
		auth.POST("/register", controllers.AuthRegister)
		auth.POST("/login", controllers.AuthLogin)
		auth.POST("/refresh", controllers.AuthRefresh)
		auth.POST("/logout", middleware.AuthJWT(), controllers.AuthLogout)
	}

	// ---------- Protected routes (autentikasi wajib) ----------
	protected := api.Group("")
	protected.Use(middleware.AuthJWT())
	{
		// ==== Bisa diakses SEMUA role (staff & owner) ====

		// Wallets: lihat
		protected.GET("/wallets", controllers.WalletsIndex)
		protected.GET("/wallets/:id", controllers.WalletsShow)

		// Categories: lihat
		protected.GET("/categories", controllers.CategoriesIndex)

		// Transactions: kelola pencatatan
		protected.GET("/transactions", controllers.TransactionsIndex)
		protected.POST("/transactions", controllers.TransactionsStore)
		protected.PUT("/transactions/:id", controllers.TransactionsUpdate)
		protected.DELETE("/transactions/:id", controllers.TransactionsDestroy)
		protected.POST("/transactions/:id/restore", controllers.TransactionsRestore)

		// Dashboard
		protected.GET("/dashboard/summary", controllers.DashboardSummary)
	}

	// ---------- Khusus owner ----------
	owner := protected.Group("")
	owner.Use(middleware.RoleOwner())
	{
		// Wallets: kelola (tambah/ubah/hapus/pulihkan)
		owner.POST("/wallets", controllers.WalletsStore)
		owner.PUT("/wallets/:id", controllers.WalletsUpdate)
		owner.DELETE("/wallets/:id", controllers.WalletsDestroy)
		owner.POST("/wallets/:id/restore", controllers.WalletsRestore)

		// Categories: kelola
		owner.POST("/categories", controllers.CategoriesStore)
		owner.PUT("/categories/:id", controllers.CategoriesUpdate)
		owner.DELETE("/categories/:id", controllers.CategoriesDestroy)
		owner.POST("/categories/:id/restore", controllers.CategoriesRestore)

		// Laporan & manajemen user
		owner.GET("/reports/monthly", controllers.MonthlyReport)
		owner.GET("/auth/users", controllers.AuthUsers)
		owner.DELETE("/auth/users/:id", controllers.AuthDeleteUser)
	}
}
