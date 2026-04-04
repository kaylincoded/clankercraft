package engine

// Tier represents the WorldEdit capability level of the server.
type Tier int

const (
	// TierUnknown means detection hasn't completed yet.
	TierUnknown Tier = iota
	// TierVanilla means no WorldEdit is available.
	TierVanilla
	// TierWorldEdit means standard WorldEdit is available.
	TierWorldEdit
	// TierFAWE means FastAsyncWorldEdit is available.
	TierFAWE
)

func (t Tier) String() string {
	switch t {
	case TierVanilla:
		return "vanilla"
	case TierWorldEdit:
		return "worldedit"
	case TierFAWE:
		return "fawe"
	default:
		return "unknown"
	}
}
