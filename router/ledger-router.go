package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func registerLedgerRoutes(apiRouter *gin.RouterGroup) {
	ledgerRoute := apiRouter.Group("/ledger")
	ledgerRoute.Use(middleware.DisableCache(), middleware.AdminAuth())
	{
		ledgerRoute.GET("", middleware.RequirePermission(authz.LedgerRead), controller.GetLedgerEntries)
		ledgerRoute.GET("/summary", middleware.RequirePermission(authz.LedgerRead), controller.GetLedgerSummary)
		ledgerRoute.GET("/settings", middleware.RequirePermission(authz.LedgerRead), controller.GetLedgerSettings)
		ledgerRoute.GET("/export", middleware.RequirePermission(authz.LedgerRead), controller.ExportLedgerEntries)
		ledgerRoute.POST("", middleware.RequirePermission(authz.LedgerWrite), controller.CreateLedgerEntry)
		ledgerRoute.PUT("/settings", middleware.RequirePermission(authz.LedgerWrite), controller.UpdateLedgerSettings)
		ledgerRoute.PUT("/:id", middleware.RequirePermission(authz.LedgerWrite), controller.UpdateLedgerEntry)
		ledgerRoute.POST("/batch-delete", middleware.RequirePermission(authz.LedgerDelete), controller.DeleteLedgerEntries)
		ledgerRoute.DELETE("/:id", middleware.RequirePermission(authz.LedgerDelete), controller.DeleteLedgerEntry)
	}
}
