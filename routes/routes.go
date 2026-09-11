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
		// Wallets
		protected.GET("/wallets", controllers.WalletsIndex)
		protected.GET("/wallets/:id", controllers.WalletsShow)
		protected.POST("/wallets", controllers.WalletsStore)
		protected.PUT("/wallets/:id", controllers.WalletsUpdate)
		protected.DELETE("/wallets/:id", controllers.WalletsDestroy)

		// Categories
		protected.GET("/categories", controllers.CategoriesIndex)
		protected.POST("/categories", controllers.CategoriesStore)
		protected.PUT("/categories/:id", controllers.CategoriesUpdate)
		protected.DELETE("/categories/:id", controllers.CategoriesDestroy)

		// Transactions
		protected.GET("/transactions", controllers.TransactionsIndex)
		protected.POST("/transactions", controllers.TransactionsStore)
		protected.PUT("/transactions/:id", controllers.TransactionsUpdate)
		protected.DELETE("/transactions/:id", controllers.TransactionsDestroy)

		// Dashboard & Reports
		protected.GET("/dashboard/summary", controllers.DashboardSummary)
		protected.GET("/reports/monthly", controllers.MonthlyReport)
	}
}
