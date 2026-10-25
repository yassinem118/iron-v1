// Package grid models only observed game state. It deliberately does not infer
// unrevealed outcomes or manufacture a "safe" choice from random/hidden data.
package grid

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

const Columns = 5

// Cell represents an explicitly observed cell in a five-column row.
// Status is source-provided text such as "apple", "bomb", or "unknown".
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

// Store is safe for concurrent Telegram handlers. It stores the latest
// observed snapshot, not a predicted future board.
type Store struct {
	mu       sync.RWMutex
	snapshot Snapshot
	hasState bool
}

func NewStore() *Store { return &Store{} }

func (s *Store) Replace(snapshot Snapshot) error {
	if len(snapshot.Rows) == 0 {
		return errors.New("payload contains no explicitly observed rows")
	}
	seenRows := make(map[int]bool, len(snapshot.Rows))
	for _, row := range snapshot.Rows {
		if row.Number < 1 || row.Number > 100 {
			return fmt.Errorf("row %d outside supported range 1..100", row.Number)
		}
		if seenRows[row.Number] {
			return fmt.Errorf("duplicate row %d", row.Number)
		}
		seenRows[row.Number] = true
		seenCols := make(map[int]bool, len(row.Cells))
		for _, cell := range row.Cells {
			if cell.Column < 1 || cell.Column > Columns {
				return fmt.Errorf("column %d outside 1..%d", cell.Column, Columns)
			}
			if seenCols[cell.Column] {
				return fmt.Errorf("duplicate column %d in row %d", cell.Column, row.Number)
			}
			seenCols[cell.Column] = true
			if len(cell.Status) > 40 {
				return fmt.Errorf("status too long in row %d column %d", row.Number, cell.Column)
			}
		}
	}
	if snapshot.SeenAt.IsZero() {
		snapshot.SeenAt = time.Now().UTC()
	}
	sort.Slice(snapshot.Rows, func(i, j int) bool { return snapshot.Rows[i].Number < snapshot.Rows[j].Number })
	for i := range snapshot.Rows {
		sort.Slice(snapshot.Rows[i].Cells, func(a, b int) bool { return snapshot.Rows[i].Cells[a].Column < snapshot.Rows[i].Cells[b].Column })
	}
	s.mu.Lock()
	s.snapshot, s.hasState = snapshot, true
	s.mu.Unlock()
	return nil
}

func (s *Store) Latest() (Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasState {
		return Snapshot{}, false
	}
	// Return a defensive copy so callers cannot mutate shared state.
	out := s.snapshot
	out.Rows = append([]Row(nil), s.snapshot.Rows...)
	for i := range out.Rows {
		out.Rows[i].Cells = append([]Cell(nil), s.snapshot.Rows[i].Cells...)
	}
	return out, true
}

func (s *Store) Row(number int) (Row, bool) {
	if number < 1 || number > 100 {
		return Row{}, false
	}
	snapshot, ok := s.Latest()
	if !ok {
		return Row{}, false
	}
	for _, row := range snapshot.Rows {
		if row.Number == number {
			return row, true
		}
	}
	return Row{}, false
}
