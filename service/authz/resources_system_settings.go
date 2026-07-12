package authz

const (
	ResourceSystemSettings     = "system_settings"
	ActionSupportContactsWrite = "support_contacts_write"
)

var SystemSettingsSupportContactsWrite = Permission{
	Resource: ResourceSystemSettings,
	Action:   ActionSupportContactsWrite,
}

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceSystemSettings,
		LabelKey: "System Settings",
		Actions: []ActionDefinition{{
			Action:         ActionSupportContactsWrite,
			LabelKey:       "Configure support contacts",
			DescriptionKey: "Manage the QQ, WeChat, and phone contacts shown to users.",
		}},
	})
}
