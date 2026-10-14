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
		handlePermissionRoute(ledgerRoute, "GET", "", authz.LedgerRead, controller.GetLedgerEntries)
		handlePermissionRoute(ledgerRoute, "GET", "/summary", authz.LedgerRead, controller.GetLedgerSummary)
		handlePermissionRoute(ledgerRoute, "GET", "/settings", authz.LedgerRead, controller.GetLedgerSettings)
		handlePermissionRoute(ledgerRoute, "GET", "/export", authz.LedgerRead, controller.ExportLedgerEntries)
		handlePermissionRoute(ledgerRoute, "POST", "", authz.LedgerWrite, controller.CreateLedgerEntry)
		handlePermissionRoute(ledgerRoute, "PUT", "/settings", authz.LedgerWrite, controller.UpdateLedgerSettings)
		handlePermissionRoute(ledgerRoute, "PUT", "/:id", authz.LedgerWrite, controller.UpdateLedgerEntry)
		handlePermissionRoute(ledgerRoute, "POST", "/batch-delete", authz.LedgerDelete, controller.DeleteLedgerEntries)
		handlePermissionRoute(ledgerRoute, "DELETE", "/:id", authz.LedgerDelete, controller.DeleteLedgerEntry)
	}
}
