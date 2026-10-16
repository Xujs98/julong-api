package authz

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	ResourceUsageLogs = "usage_logs"
	ResourceTaskLogs  = "task_logs"
	ResourcePrivacy   = "privacy"
)

var (
	UsageLogsRead   = Permission{ResourceUsageLogs, "read"}
	DrawingLogsRead = Permission{ResourceTaskLogs, "drawing_read"}
	ImageLogsRead   = Permission{ResourceTaskLogs, "image_read"}
	TaskLogsRead    = Permission{ResourceTaskLogs, "task_read"}
	MediaAssetsRead = Permission{ResourceTaskLogs, "assets_read"}
	HideRoot        = Permission{ResourcePrivacy, "hide_root"}
)

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceUsageLogs, LabelKey: "Usage Logs",
		Actions: []ActionDefinition{{Action: UsageLogsRead.Action, LabelKey: "View other accounts' usage logs", DescriptionKey: "View other accounts' usage logs", DefaultRoles: []string{managedRoleKey}}},
	})
	RegisterResource(ResourceDefinition{
		Resource: ResourceTaskLogs, LabelKey: "Task Logs",
		Actions: []ActionDefinition{
			{Action: DrawingLogsRead.Action, LabelKey: "View other accounts' drawing logs", DescriptionKey: "View other accounts' drawing logs", DefaultRoles: []string{managedRoleKey}},
			{Action: ImageLogsRead.Action, LabelKey: "View other accounts' image generation logs", DescriptionKey: "View other accounts' image generation logs", DefaultRoles: []string{managedRoleKey}},
			{Action: TaskLogsRead.Action, LabelKey: "View other accounts' task logs", DescriptionKey: "View other accounts' task logs", DefaultRoles: []string{managedRoleKey}},
			{Action: MediaAssetsRead.Action, LabelKey: "View other accounts' media assets", DescriptionKey: "View other accounts' media assets", DefaultRoles: []string{managedRoleKey}},
		},
	})
	RegisterResource(ResourceDefinition{
		Resource: ResourcePrivacy, LabelKey: "Privacy permissions",
		Actions: []ActionDefinition{{Action: HideRoot.Action, LabelKey: "Do not allow viewing Root accounts",
			DescriptionKey: "Hide Root usage, task and audit records and prevent opening Root user details."}},
	})
}

// RootHidden is a restriction, not a positive grant. Root's universal grants
// must never turn this restriction on for Root itself.
func RootHidden(userID, role int) bool {
	return role >= common.RoleAdminUser && role < common.RoleRootUser && Can(userID, role, HideRoot)
}

// LogScope evaluates permissions on each request. Resolve Root IDs on the main
// database, so no cross-database join is needed by the audit log store. A lookup
// error is returned to the caller rather than widening the query.
func LogScope(userID, role int, permission Permission) (model.UserLogScope, error) {
	scope := model.UserLogScope{}
	if role < common.RoleAdminUser || !Can(userID, role, permission) {
		scope.UserID = userID
	}
	if RootHidden(userID, role) {
		err := model.DB.Unscoped().Model(&model.User{}).Where("role = ?", common.RoleRootUser).Pluck("id", &scope.ExcludedUserIDs).Error
		if err != nil {
			return model.UserLogScope{}, err
		}
	}
	return scope, nil
}

func CanViewLogOwner(userID, role, ownerID int, permission Permission) bool {
	scope, err := LogScope(userID, role, permission)
	return err == nil && scope.Allows(ownerID)
}
