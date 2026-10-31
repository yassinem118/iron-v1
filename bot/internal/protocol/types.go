package protocol

import "time"

// Cell is a source-reported observation; it is not a prediction.
type Cell struct {
	Column int    `json:"column"`
	Status string `json:"status"`
}
type Row struct {
	Number int    `json:"row"`
	Cells  []Cell `json:"cells"`
}
type Snapshot struct {
	Event  string    `json:"event,omitempty"`
	GameID string    `json:"gameId,omitempty"`
	Rows   []Row     `json:"rows"`
	SeenAt time.Time `json:"seenAt"`
}
