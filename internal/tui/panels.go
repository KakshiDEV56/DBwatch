package tui

// PanelKind identifies one of the sidebar monitoring sections. Each maps
// 1:1 to a collector.
type PanelKind int

const (
	PanelConnections PanelKind = iota
	PanelCache
	PanelQueries
	PanelLocks
	PanelTransactions
)

var allPanels = []PanelKind{
	PanelConnections,
	PanelCache,
	PanelQueries,
	PanelLocks,
	PanelTransactions,
}

func (k PanelKind) Title() string {
	switch k {
	case PanelConnections:
		return "Connections"
	case PanelCache:
		return "Cache Hit Ratio"
	case PanelQueries:
		return "Top Queries"
	case PanelLocks:
		return "Locks & Blocking"
	case PanelTransactions:
		return "Long-Running Transactions"
	default:
		return "?"
	}
}
