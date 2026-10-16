package middleware

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
)

type accessTokenRuleKind int

const (
	// accessTokenRuleScope requires the token to carry scope.
	accessTokenRuleScope accessTokenRuleKind = iota + 1
	// accessTokenRuleAny accepts every valid scoped token; the handler applies
	// its own checks.
	accessTokenRuleAny
	// accessTokenRuleSession rejects every access token, legacy ones included.
	accessTokenRuleSession
	// Settings rules accept a section grant; the handler applies key-level authorization.
	accessTokenRuleSettings
)

type accessTokenRouteRule struct {
	kind  accessTokenRuleKind
	scope string
}

var (
	accessTokenAnyRule      = accessTokenRouteRule{kind: accessTokenRuleAny}
	accessTokenSessionRule  = accessTokenRouteRule{kind: accessTokenRuleSession}
	accessTokenSettingsRule = accessTokenRouteRule{kind: accessTokenRuleSettings}
)

func accessTokenScopeRule(scope string) accessTokenRouteRule {
	return accessTokenRouteRule{kind: accessTokenRuleScope, scope: scope}
}

// accessTokenRouteRules declares the token rule of every dashboard route that
// is not guarded by RequirePermission, keyed by method and gin full path.
// Casbin-guarded routes declare themselves through
// DeclareAccessTokenPermissionRoute when they are registered.
var accessTokenRouteRules = map[string]accessTokenRouteRule{
	// router/api-router.go: top level
	"GET /api/models":                     accessTokenScopeRule("profile:read"),
	"GET /api/status/test":                accessTokenScopeRule("log:read"),
	"GET /api/pricing":                    accessTokenAnyRule,
	"GET /api/model-documentation":        accessTokenAnyRule,
	"GET /api/perf-metrics/summary":       accessTokenAnyRule,
	"GET /api/perf-metrics":               accessTokenAnyRule,
	"GET /api/rankings":                   accessTokenAnyRule,
	"POST /api/oauth/state":               accessTokenSessionRule,
	"POST /api/oauth/email/bind/start":    accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/email/bind/resend":   accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/email/bind":          accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/wechat/bind":         accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/telegram/bind/start": accessTokenSessionRule,
	"GET /api/oauth/:provider":            accessTokenSessionRule,
	"GET /api/verify/methods":             accessTokenAnyRule,
	"POST /api/verify":                    accessTokenAnyRule,

	// router/api-router.go: /api/user (self)
	"GET /api/user/sessions":                       accessTokenSessionRule,
	"DELETE /api/user/sessions/:sid":               accessTokenSessionRule,
	"POST /api/user/sessions/revoke-others":        accessTokenSessionRule,
	"GET /api/user/self/groups":                    accessTokenScopeRule("profile:read"),
	"GET /api/user/self/activity":                  accessTokenScopeRule("profile:read"),
	"GET /api/user/self":                           accessTokenScopeRule("profile:read"),
	"GET /api/user/models":                         accessTokenScopeRule("profile:read"),
	"PUT /api/user/self":                           accessTokenScopeRule("profile:write"),
	"DELETE /api/user/self":                        accessTokenScopeRule("account_security:write"),
	"GET /api/user/access_tokens":                  accessTokenSessionRule,
	"GET /api/user/access_tokens/catalog":          accessTokenSessionRule,
	"GET /api/user/access_tokens/scopes":           accessTokenSessionRule,
	"POST /api/user/access_tokens":                 accessTokenSessionRule,
	"PATCH /api/user/access_tokens/:id":            accessTokenSessionRule,
	"DELETE /api/user/access_tokens/:id":           accessTokenSessionRule,
	"DELETE /api/user/access_tokens/legacy":        accessTokenSessionRule,
	"GET /api/user/passkey":                        accessTokenScopeRule("account_security:read"),
	"POST /api/user/passkey/register/begin":        accessTokenScopeRule("account_security:write"),
	"POST /api/user/passkey/register/finish":       accessTokenScopeRule("account_security:write"),
	"POST /api/user/passkey/verify/begin":          accessTokenAnyRule,
	"POST /api/user/passkey/verify/finish":         accessTokenAnyRule,
	"DELETE /api/user/passkey":                     accessTokenScopeRule("account_security:write"),
	"GET /api/user/aff":                            accessTokenScopeRule("wallet:read"),
	"GET /api/user/topup/info":                     accessTokenScopeRule("wallet:read"),
	"GET /api/user/topup/self":                     accessTokenScopeRule("wallet:read"),
	"POST /api/user/topup":                         accessTokenScopeRule("wallet:write"),
	"POST /api/user/pay":                           accessTokenScopeRule("wallet:write"),
	"POST /api/user/amount":                        accessTokenScopeRule("wallet:read"),
	"POST /api/user/stripe/pay":                    accessTokenScopeRule("wallet:write"),
	"POST /api/user/stripe/amount":                 accessTokenScopeRule("wallet:read"),
	"POST /api/user/creem/pay":                     accessTokenScopeRule("wallet:write"),
	"POST /api/user/waffo/amount":                  accessTokenScopeRule("wallet:read"),
	"POST /api/user/waffo/pay":                     accessTokenScopeRule("wallet:write"),
	"POST /api/user/waffo-pancake/amount":          accessTokenScopeRule("wallet:read"),
	"POST /api/user/waffo-pancake/pay":             accessTokenScopeRule("wallet:write"),
	"POST /api/user/aff_transfer":                  accessTokenScopeRule("wallet:write"),
	"PUT /api/user/setting":                        accessTokenScopeRule("profile:write"),
	"GET /api/user/2fa/status":                     accessTokenScopeRule("account_security:read"),
	"POST /api/user/2fa/setup":                     accessTokenScopeRule("account_security:write"),
	"POST /api/user/2fa/enable":                    accessTokenScopeRule("account_security:write"),
	"POST /api/user/2fa/disable":                   accessTokenScopeRule("account_security:write"),
	"POST /api/user/2fa/backup_codes":              accessTokenScopeRule("account_security:write"),
	"GET /api/user/checkin":                        accessTokenScopeRule("wallet:read"),
	"POST /api/user/checkin":                       accessTokenScopeRule("wallet:write"),
	"GET /api/user/oauth/bindings":                 accessTokenScopeRule("account_security:read"),
	"DELETE /api/user/oauth/bindings/:provider_id": accessTokenScopeRule("account_security:write"),

	// router/api-router.go: /api/user (admin)
	"GET /api/user/":                                    accessTokenScopeRule("user:read"),
	"GET /api/user/topup":                               accessTokenScopeRule("billing:read"),
	"POST /api/user/topup/complete":                     accessTokenScopeRule("billing:write"),
	"GET /api/user/search":                              accessTokenScopeRule("user:read"),
	"GET /api/user/:id/oauth/bindings":                  accessTokenScopeRule("user:read"),
	"DELETE /api/user/:id/oauth/bindings/:provider_id":  accessTokenScopeRule("user:write"),
	"DELETE /api/user/:id/bindings/:binding_type":       accessTokenScopeRule("user:write"),
	"GET /api/user/:id":                                 accessTokenScopeRule("user:read"),
	"POST /api/user/":                                   accessTokenScopeRule("user:write"),
	"POST /api/user/manage":                             accessTokenScopeRule("user:write"),
	"PUT /api/user/":                                    accessTokenScopeRule("user:write"),
	"DELETE /api/user/:id":                              accessTokenScopeRule("user:write"),
	"DELETE /api/user/:id/reset_passkey":                accessTokenScopeRule("user:write"),
	"GET /api/user/2fa/stats":                           accessTokenScopeRule("user:read"),
	"DELETE /api/user/:id/2fa":                          accessTokenScopeRule("user:write"),
	"GET /api/user/tags":                                accessTokenScopeRule("user:read"),
	"POST /api/user/tags":                               accessTokenScopeRule("user:write"),
	"PUT /api/user/tags/:tag_id":                        accessTokenScopeRule("user:write"),
	"DELETE /api/user/tags/:tag_id":                     accessTokenScopeRule("user:write"),
	"GET /api/user/options":                             accessTokenScopeRule("user:read"),
	"POST /api/user/options/resolve":                    accessTokenScopeRule("user:read"),
	"POST /api/user/batch-quota":                        accessTokenScopeRule("user:write"),
	"GET /api/user/quota-summary":                       accessTokenScopeRule("user:read"),
	"GET /api/user/quota-summary/settings":              accessTokenScopeRule("user:read"),
	"PUT /api/user/quota-summary/settings":              accessTokenScopeRule("user:write"),
	"GET /api/user/quota-summary/users":                 accessTokenScopeRule("user:read"),
	"POST /api/user/quota-summary/users/resolve":        accessTokenScopeRule("user:read"),
	"GET /api/user/agent-detail/:id":                    accessTokenScopeRule("user:read"),
	"GET /api/user/:id/usage-summary":                   accessTokenScopeRule("user:read"),
	"GET /api/user/:id/risk":                            accessTokenScopeRule("user:read"),
	"PUT /api/user/:id/risk":                            accessTokenScopeRule("user:write"),
	"PUT /api/user/:id/tag":                             accessTokenScopeRule("user:write"),
	"GET /api/user/:id/quota-increases":                 accessTokenScopeRule("user:read"),
	"GET /api/user/:id/login-ips":                       accessTokenScopeRule("user:read"),
	"PUT /api/user/:id/login-ips":                       accessTokenScopeRule("user:write"),
	"GET /api/user/:id/login-devices":                   accessTokenScopeRule("user:read"),
	"PUT /api/user/:id/login-devices":                   accessTokenScopeRule("user:write"),
	"GET /api/user/:id/request-content":                 accessTokenScopeRule("user:read"),
	"GET /api/user/:id/request-content/:log_id":         accessTokenScopeRule("user:read"),
	"PUT /api/user/:id/request-content":                 accessTokenScopeRule("user:write"),
	"DELETE /api/user/:id/request-content":              accessTokenScopeRule("user:write"),
	"GET /api/user/agent/users":                         accessTokenScopeRule("wallet:read"),
	"PUT /api/user/agent/topup-link":                    accessTokenScopeRule("wallet:write"),
	"GET /api/user/invoice":                             accessTokenScopeRule("wallet:read"),
	"GET /api/user/invoice/applications":                accessTokenScopeRule("billing:read"),
	"GET /api/user/invoice/email-templates":             accessTokenScopeRule("billing:read"),
	"GET /api/user/invoice/applications/:id/attachment": accessTokenScopeRule("wallet:read"),
	"POST /api/user/invoice/apply":                      accessTokenScopeRule("wallet:write"),
	"POST /api/user/invoice/applications/:id/cancel":    accessTokenScopeRule("wallet:write"),
	"POST /api/user/invoice/applications/:id/complete":  accessTokenScopeRule("billing:write"),
	"POST /api/user/invoice/applications/:id/withdraw":  accessTokenScopeRule("billing:write"),
	"PUT /api/user/invoice/applications/:id":            accessTokenScopeRule("billing:write"),
	"GET /api/announcements":                            accessTokenScopeRule("profile:read"),

	// router/api-router.go: /api/subscription
	"GET /api/subscription/plans":                                    accessTokenScopeRule("wallet:read"),
	"GET /api/subscription/self":                                     accessTokenScopeRule("wallet:read"),
	"PUT /api/subscription/self/preference":                          accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/balance/pay":                             accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/epay/pay":                                accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/stripe/pay":                              accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/creem/pay":                               accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/waffo-pancake/pay":                       accessTokenScopeRule("wallet:write"),
	"GET /api/subscription/admin/plans":                              accessTokenScopeRule("billing:read"),
	"POST /api/subscription/admin/plans":                             accessTokenScopeRule("billing:write"),
	"PUT /api/subscription/admin/plans/:id":                          accessTokenScopeRule("billing:write"),
	"PATCH /api/subscription/admin/plans/:id":                        accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/bind":                              accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/plans/:id/subscriptions/reset":     accessTokenScopeRule("billing:write"),
	"GET /api/subscription/admin/users/:id/subscriptions":            accessTokenScopeRule("billing:read"),
	"POST /api/subscription/admin/users/:id/subscriptions":           accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/users/:id/subscriptions/reset":     accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/user_subscriptions/:id/invalidate": accessTokenScopeRule("billing:write"),
	"DELETE /api/subscription/admin/user_subscriptions/:id":          accessTokenScopeRule("billing:write"),

	// router/api-router.go: /api/option
	"GET /api/option/":                                           accessTokenSettingsRule,
	"GET /api/option/site_blocked_ips":                           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("security.ssrf"))),
	"PUT /api/option/site_blocked_ips":                           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("security.ssrf"))),
	"GET /api/option/request_policy":                             accessTokenSettingsRule,
	"PATCH /api/option/request_policy":                           accessTokenSettingsRule,
	"PUT /api/option/":                                           accessTokenSettingsRule,
	"PUT /api/option/passkey/domains":                            accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("auth.passkey"))),
	"GET /api/option/model_pricing":                              accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.model-pricing"))),
	"PATCH /api/option/model_pricing":                            accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.model-pricing"))),
	"POST /api/option/model_pricing/convert":                     accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.model-pricing"))),
	"POST /api/option/model_pricing/preview":                     accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.model-pricing"))),
	"POST /api/option/payment_compliance":                        accessTokenSessionRule,
	"GET /api/option/channel_affinity_cache":                     accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("models.channel-affinity"))),
	"DELETE /api/option/channel_affinity_cache":                  accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("models.channel-affinity"))),
	"POST /api/option/rest_model_ratio":                          accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.model-pricing"))),
	"GET /api/option/waffo-pancake/catalog":                      accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.payment"))),
	"POST /api/option/waffo-pancake/pair":                        accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.payment"))),
	"POST /api/option/waffo-pancake/save":                        accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.payment"))),
	"POST /api/option/waffo-pancake/subscription-product":        accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.payment"))),
	"GET /api/option/waffo-pancake/subscription-product-options": accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.payment"))),

	// router/api-router.go: /api/custom-oauth-provider, /api/ratio_sync
	"POST /api/custom-oauth-provider/discovery": accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("auth.custom-oauth"))),
	"GET /api/custom-oauth-provider/":           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("auth.custom-oauth"))),
	"GET /api/custom-oauth-provider/:id":        accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("auth.custom-oauth"))),
	"POST /api/custom-oauth-provider/":          accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("auth.custom-oauth"))),
	"PUT /api/custom-oauth-provider/:id":        accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("auth.custom-oauth"))),
	"DELETE /api/custom-oauth-provider/:id":     accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("auth.custom-oauth"))),
	"GET /api/ratio_sync/channels":              accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.model-pricing"))),
	"POST /api/ratio_sync/fetch":                accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("billing.model-pricing"))),

	// router/api-router.go: /api/performance, /api/system-task, /api/system-info
	"GET /api/performance/stats":                   accessTokenScopeRule("ops:read"),
	"DELETE /api/performance/disk_cache":           accessTokenScopeRule("ops:write"),
	"POST /api/performance/reset_stats":            accessTokenScopeRule("ops:write"),
	"POST /api/performance/gc":                     accessTokenScopeRule("ops:write"),
	"GET /api/performance/logs":                    accessTokenScopeRule("ops:read"),
	"DELETE /api/performance/logs":                 accessTokenScopeRule("ops:write"),
	"GET /api/performance/image-storage":           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.logs"))),
	"PUT /api/performance/image-storage":           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.logs"))),
	"POST /api/performance/image-storage/test":     accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.logs"))),
	"GET /api/performance/image-storage/stats":     accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.logs"))),
	"POST /api/performance/image-storage/cleanup":  accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.logs"))),
	"POST /api/performance/image-storage/purge":    accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.logs"))),
	"POST /api/system-task/log-cleanup":            accessTokenScopeRule("ops:write"),
	"GET /api/system-task/list":                    accessTokenScopeRule("ops:read"),
	"DELETE /api/system-task/history":              accessTokenScopeRule("ops:write"),
	"GET /api/system-task/current":                 accessTokenScopeRule("ops:read"),
	"GET /api/system-task/:task_id":                accessTokenScopeRule("ops:read"),
	"GET /api/system-info/instances":               accessTokenScopeRule("ops:read"),
	"DELETE /api/system-info/stale-instances":      accessTokenScopeRule("ops:write"),
	"DELETE /api/system-info/instances/:node_name": accessTokenScopeRule("ops:write"),

	// router/api-router.go: /api/plugin/task
	"GET /api/plugin/task":                           accessTokenScopeRule("plugin:read"),
	"POST /api/plugin/task":                          accessTokenScopeRule("plugin:write"),
	"PUT /api/plugin/task":                           accessTokenScopeRule("plugin:write"),
	"GET /api/plugin/task/runtime/status":            accessTokenScopeRule("plugin:read"),
	"GET /api/plugin/task/marketplace/sources":       accessTokenScopeRule("plugin:read"),
	"PUT /api/plugin/task/marketplace/sources":       accessTokenScopeRule("plugin:write"),
	"GET /api/plugin/task/:key":                      accessTokenScopeRule("plugin:read"),
	"GET /api/plugin/task/:key/icon":                 accessTokenScopeRule("plugin:read"),
	"GET /api/plugin/task/:key/versions":             accessTokenScopeRule("plugin:read"),
	"POST /api/plugin/task/:key/activate":            accessTokenScopeRule("plugin:write"),
	"POST /api/plugin/task/:key/status":              accessTokenScopeRule("plugin:write"),
	"POST /api/plugin/task/:key/dryrun":              accessTokenScopeRule("plugin:write"),
	"DELETE /api/plugin/task/:key/versions/:version": accessTokenScopeRule("plugin:write"),

	// router/channel-router.go: the key route is RootAuth, not Casbin-guarded.
	"POST /api/channel/:id/key": accessTokenScopeRule(service.AccessTokenScopeOf(authz.ChannelSecretView)),

	// router/authz-router.go
	"GET /api/authz/catalog": accessTokenScopeRule("user:read"),

	// router/api-router.go: /api/token
	"GET /api/token/":            accessTokenScopeRule("api_key:read"),
	"GET /api/token/search":      accessTokenScopeRule("api_key:read"),
	"GET /api/token/auto-groups": accessTokenScopeRule("api_key:read"),
	"GET /api/token/:id":         accessTokenScopeRule("api_key:read"),
	"POST /api/token/:id/key":    accessTokenScopeRule("api_key:reveal"),
	"POST /api/token/":           accessTokenScopeRule("api_key:write"),
	"PUT /api/token/":            accessTokenScopeRule("api_key:write"),
	"DELETE /api/token/:id":      accessTokenScopeRule("api_key:write"),
	"POST /api/token/batch":      accessTokenScopeRule("api_key:write"),
	"POST /api/token/batch/keys": accessTokenScopeRule("api_key:reveal"),

	// router/api-router.go: /api/redemption
	"GET /api/redemption/":           accessTokenScopeRule("redemption:read"),
	"GET /api/redemption/search":     accessTokenScopeRule("redemption:read"),
	"GET /api/redemption/:id":        accessTokenScopeRule("redemption:read"),
	"POST /api/redemption/":          accessTokenScopeRule("redemption:write"),
	"POST /api/redemption/batch":     accessTokenScopeRule("redemption:write"),
	"PUT /api/redemption/":           accessTokenScopeRule("redemption:write"),
	"DELETE /api/redemption/invalid": accessTokenScopeRule("redemption:write"),
	"DELETE /api/redemption/:id":     accessTokenScopeRule("redemption:write"),

	// router/api-router.go: /api/audit, /api/log, /api/data
	"GET /api/audit/self":                               accessTokenScopeRule("usage:read"),
	"GET /api/log/":                                     accessTokenScopeRule("log:read"),
	"GET /api/log/stat":                                 accessTokenScopeRule("log:read"),
	"GET /api/log/self/stat":                            accessTokenScopeRule("usage:read"),
	"GET /api/log/channel_affinity_usage_cache":         accessTokenScopeRule("log:read"),
	"GET /api/log/search":                               accessTokenScopeRule("log:read"),
	"GET /api/log/self":                                 accessTokenScopeRule("usage:read"),
	"GET /api/log/self/search":                          accessTokenScopeRule("usage:read"),
	"GET /api/log/export":                               accessTokenScopeRule("log:read"),
	"GET /api/log/export/stream":                        accessTokenScopeRule("log:read"),
	"GET /api/log/self/export":                          accessTokenScopeRule("usage:read"),
	"GET /api/log/self/export/stream":                   accessTokenScopeRule("usage:read"),
	"GET /api/data/":                                    accessTokenScopeRule("log:read"),
	"GET /api/data/activity":                            accessTokenScopeRule("user:read"),
	"GET /api/data/users":                               accessTokenScopeRule("log:read"),
	"GET /api/data/self":                                accessTokenScopeRule("usage:read"),
	"GET /api/data/flow":                                accessTokenScopeRule("log:read"),
	"GET /api/data/flow/self":                           accessTokenScopeRule("usage:read"),
	"GET /api/data/groups":                              accessTokenScopeRule("log:read"),
	"GET /api/image-generation-logs":                    accessTokenScopeRule("log:read"),
	"GET /api/image-generation-logs/":                   accessTokenScopeRule("log:read"),
	"GET /api/image-generation-logs/:id/task":           accessTokenScopeRule("log:read"),
	"GET /api/image-generation-logs/:id/images/:index":  accessTokenScopeRule("log:read"),
	"GET /api/error-reports":                            accessTokenScopeRule("ops:read"),
	"GET /api/error-reports/":                           accessTokenScopeRule("ops:read"),
	"GET /api/error-reports/:id":                        accessTokenScopeRule("ops:read"),
	"POST /api/error-reports":                           accessTokenAnyRule,
	"GET /api/support-contacts/qrcode/:filename":        accessTokenAnyRule,
	"GET /api/support-contacts":                         accessTokenScopeRule("profile:read"),
	"PUT /api/support-contacts":                         accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsSupportContactsWrite)),
	"POST /api/site-assets/logo":                        accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("site.system-info"))),
	"DELETE /api/site-assets/logo/:filename":            accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("site.system-info"))),
	"GET /api/site-assets/logo/:filename":               accessTokenAnyRule,
	"GET /api/email-campaigns":                          accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"GET /api/email-campaigns/stats":                    accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"GET /api/email-campaigns/users":                    accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"POST /api/email-campaigns/users/resolve":           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"POST /api/email-campaigns":                         accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"POST /api/email-campaigns/preview":                 accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"GET /api/email-campaigns/:id":                      accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"PUT /api/email-campaigns/:id":                      accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"DELETE /api/email-campaigns/:id":                   accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"POST /api/email-campaigns/:id/activate":            accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"POST /api/email-campaigns/:id/pause":               accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"POST /api/email-campaigns/:id/retry":               accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"GET /api/email-campaigns/:id/deliveries":           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-campaigns"))),
	"GET /api/email-settings":                           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"PUT /api/email-settings":                           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"GET /api/email-settings/config":                    accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"PUT /api/email-settings/config":                    accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"GET /api/email-settings/recipients":                accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/recipients/resolve":       accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"GET /api/email-settings/monitored-users":           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/monitored-users/resolve":  accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/channel-anomaly/test":     accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/dashboard-report/test":    accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/risk-user/test":           accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/user-presence/test":       accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"GET /api/email-settings/templates":                 accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"PUT /api/email-settings/templates/:event":          accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/templates/:event/preview": accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),
	"POST /api/email-settings/templates/:event/reset":   accessTokenScopeRule(service.AccessTokenScopeOf(authz.SystemSettingsPermission("operations.email-templates"))),

	// router/api-router.go: /api/group, /api/prefill_group
	"GET /api/group/":               accessTokenScopeRule("group:read"),
	"GET /api/prefill_group/":       accessTokenScopeRule("group:read"),
	"POST /api/prefill_group/":      accessTokenScopeRule("group:write"),
	"PUT /api/prefill_group/":       accessTokenScopeRule("group:write"),
	"DELETE /api/prefill_group/:id": accessTokenScopeRule("group:write"),

	// router/api-router.go: /api/mj, /api/task
	"GET /api/mj/self":                 accessTokenScopeRule("usage:read"),
	"GET /api/mj/":                     accessTokenScopeRule("log:read"),
	"GET /api/task/self":               accessTokenScopeRule("usage:read"),
	"GET /api/task":                    accessTokenScopeRule("log:read"),
	"GET /api/task/:task_id/artifacts": accessTokenScopeRule("usage:read"),
	"GET /api/task/:task_id/request":   accessTokenScopeRule("usage:read"),
	"GET /api/task/:task_id/detail":    accessTokenScopeRule("usage:read"),

	// router/api-router.go: /api/media-assets
	"GET /api/media-assets":              accessTokenScopeRule("log:read"),
	"DELETE /api/media-assets/:asset_id": accessTokenAnyRule,

	// router/api-router.go: /api/vendors, /api/models (admin)
	"POST /api/vendors/operations/preview":  accessTokenScopeRule("model:read"),
	"POST /api/vendors/operations":          accessTokenScopeRule("model:write"),
	"GET /api/vendors/":                     accessTokenScopeRule("model:read"),
	"GET /api/vendors/search":               accessTokenScopeRule("model:read"),
	"GET /api/vendors/:id":                  accessTokenScopeRule("model:read"),
	"POST /api/vendors/":                    accessTokenScopeRule("model:write"),
	"PUT /api/vendors/":                     accessTokenScopeRule("model:write"),
	"DELETE /api/vendors/:id":               accessTokenScopeRule("model:write"),
	"GET /api/models/sync_upstream/preview": accessTokenScopeRule("model:read"),
	"POST /api/models/sync_upstream":        accessTokenScopeRule("model:write"),
	"POST /api/models/delete":               accessTokenScopeRule("model:write"),
	"GET /api/models/missing":               accessTokenScopeRule("model:read"),
	"GET /api/models/":                      accessTokenScopeRule("model:read"),
	"GET /api/models/search":                accessTokenScopeRule("model:read"),
	"GET /api/models/:id":                   accessTokenScopeRule("model:read"),
	"POST /api/models/":                     accessTokenScopeRule("model:write"),
	"PUT /api/models/":                      accessTokenScopeRule("model:write"),
	"DELETE /api/models/:id":                accessTokenScopeRule("model:write"),

	// router/api-router.go: /api/model-document-templates
	"GET /api/model-document-templates":        accessTokenScopeRule("model:read"),
	"GET /api/model-document-templates/:id":    accessTokenScopeRule("model:read"),
	"POST /api/model-document-templates":       accessTokenScopeRule("model:write"),
	"PUT /api/model-document-templates/:id":    accessTokenScopeRule("model:write"),
	"DELETE /api/model-document-templates/:id": accessTokenScopeRule("model:write"),

	// router/api-router.go: /api/deployments
	"GET /api/deployments/settings":                     accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/settings/test-connection":    accessTokenScopeRule("deployment:write"),
	"GET /api/deployments/":                             accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/search":                       accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/test-connection":             accessTokenScopeRule("deployment:write"),
	"GET /api/deployments/hardware-types":               accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/locations":                    accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/available-replicas":           accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/price-estimation":            accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/check-name":                   accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/":                            accessTokenScopeRule("deployment:write"),
	"GET /api/deployments/:id":                          accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/:id/logs":                     accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/:id/containers":               accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/:id/containers/:container_id": accessTokenScopeRule("deployment:read"),
	"PUT /api/deployments/:id":                          accessTokenScopeRule("deployment:write"),
	"PUT /api/deployments/:id/name":                     accessTokenScopeRule("deployment:write"),
	"POST /api/deployments/:id/extend":                  accessTokenScopeRule("deployment:write"),
	"DELETE /api/deployments/:id":                       accessTokenScopeRule("deployment:write"),

	// The handler requires wallet:read for ownership or billing:read for administrators.
	"GET /api/user/invoice/applications/:id/payment-proofs/:proof_id": accessTokenAnyRule,

	// router/relay-router.go
	"POST /pg/chat/completions": accessTokenSessionRule,
}

var (
	permissionRouteRulesMu sync.RWMutex
	permissionRouteRules   = map[string]accessTokenRouteRule{}
)

// DeclareAccessTokenPermissionRoute records that the Casbin permission guarding
// a route is also the token scope it requires. It runs while routes are
// registered and panics on conflicting declarations.
func DeclareAccessTokenPermissionRoute(method, fullPath string, permission authz.Permission) {
	key := method + " " + fullPath
	rule := accessTokenScopeRule(service.AccessTokenScopeOf(permission))
	if _, ok := accessTokenRouteRules[key]; ok {
		panic(fmt.Sprintf("access token route %s is declared twice", key))
	}
	permissionRouteRulesMu.Lock()
	defer permissionRouteRulesMu.Unlock()
	if existing, ok := permissionRouteRules[key]; ok && existing != rule {
		panic(fmt.Sprintf("access token route %s is declared with %s and %s", key, existing.scope, rule.scope))
	}
	permissionRouteRules[key] = rule
}

// AccessTokenRouteRule returns the rule for "METHOD /full/path".
func AccessTokenRouteRule(key string) (accessTokenRouteRule, bool) {
	if rule, ok := accessTokenRouteRules[key]; ok {
		return rule, true
	}
	permissionRouteRulesMu.RLock()
	defer permissionRouteRulesMu.RUnlock()
	rule, ok := permissionRouteRules[key]
	return rule, ok
}

// AccessTokenRouteRuleKeys lists every declared route key, sorted.
func AccessTokenRouteRuleKeys() []string {
	permissionRouteRulesMu.RLock()
	keys := slices.Collect(maps.Keys(permissionRouteRules))
	permissionRouteRulesMu.RUnlock()
	keys = slices.AppendSeq(keys, maps.Keys(accessTokenRouteRules))
	slices.Sort(keys)
	return keys
}

// Scope returns the token scope the rule requires, or "" for any/session rules.
func (rule accessTokenRouteRule) Scope() string {
	return rule.scope
}

// Kind names the rule category: "scope", "any" or "session".
func (rule accessTokenRouteRule) Kind() string {
	switch rule.kind {
	case accessTokenRuleScope:
		return "scope"
	case accessTokenRuleSettings:
		return "settings"
	case accessTokenRuleAny:
		return "any"
	case accessTokenRuleSession:
		return "session"
	}
	return ""
}
