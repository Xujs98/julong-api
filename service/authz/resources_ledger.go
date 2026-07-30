package authz

const (
	ResourceLedger = "ledger"
	ActionDelete   = "delete"
)

var (
	LedgerRead   = Permission{Resource: ResourceLedger, Action: ActionRead}
	LedgerWrite  = Permission{Resource: ResourceLedger, Action: ActionWrite}
	LedgerDelete = Permission{Resource: ResourceLedger, Action: ActionDelete}
)

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceLedger,
		LabelKey: "Ledger",
		Actions: []ActionDefinition{
			{
				Action:         ActionRead,
				LabelKey:       "View ledger",
				DescriptionKey: "View ledger entries, summaries, and downloads.",
			},
			{
				Action:         ActionWrite,
				LabelKey:       "Edit ledger",
				DescriptionKey: "Create and edit ledger entries.",
			},
			{
				Action:         ActionDelete,
				LabelKey:       "Delete ledger entries",
				DescriptionKey: "Delete one or multiple ledger entries after confirmation.",
			},
		},
	})
}
