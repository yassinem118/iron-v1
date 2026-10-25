// Package protocol decodes JSON websocket frames/log records supplied by an
// authorized operator. It intentionally performs no network connection to a
// third-party betting service and does not guess the meaning of undocumented
// binary frames. Once actual, authorized sample frames are available, a narrow
// adapter can be added here without changing the Telegram layer.
package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"iron-v1-bot/internal/grid"
)

const MaxPayloadBytes = 64 * 1024

// Envelope retains common websocket/event envelope fields while accepting
// unknown fields for forward compatibility. The parser only extracts board
// cells that are explicitly present in the payload.
type Envelope struct {
	Type    string          `json:"type"`
	Event   string          `json:"event"`
	Action  string          `json:"action"`
	GameID  string          `json:"gameId"`
	Data    json.RawMessage `json:"data"`
	Payload json.RawMessage `json:"payload"`
	Result  json.RawMessage `json:"result"`
	Rows    json.RawMessage `json:"rows"`
	Grid    json.RawMessage `json:"grid"`
}

type rawCell struct {
	Column   *int   `json:"column"`
	Col      *int   `json:"col"`
	Index    *int   `json:"index"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	State    string `json:"state"`
	Revealed *bool  `json:"revealed"`
}

type rawRow struct {
	Row    *int            `json:"row"`
	Number *int            `json:"number"`
	Index  *int            `json:"index"`
	Cells  json.RawMessage `json:"cells"`
	Items  json.RawMessage `json:"items"`
	Values json.RawMessage `json:"values"`
}

// ParseFrame recognizes common JSON event wrappers and returns a snapshot only
// when at least one row and one explicit cell value are present. It does not
// treat missing cells as safe or derive a next pick.
func ParseFrame(input []byte) (grid.Snapshot, error) {
	if len(input) == 0 || len(input) > MaxPayloadBytes {
		return grid.Snapshot{}, fmt.Errorf("payload must be between 1 and %d bytes", MaxPayloadBytes)
	}
	var root map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return grid.Snapshot{}, fmt.Errorf("invalid JSON frame: %w", err)
	}
	if err := ensureEOF(dec); err != nil {
		return grid.Snapshot{}, err
	}

	var env Envelope
	_ = json.Unmarshal(input, &env)
	base := root
	// Peel at most three familiar wrappers; this avoids unbounded recursive
	// traversal of arbitrary user-controlled JSON.
	for depth := 0; depth < 3; depth++ {
		var next map[string]json.RawMessage
		for _, key := range []string{"data", "payload", "result"} {
			if raw := base[key]; len(raw) > 0 && raw[0] == '{' {
				if json.Unmarshal(raw, &next) == nil && len(next) > 0 {
					base = next
					break
				}
			}
		}
		if next == nil {
			break
		}
		next = nil
	}

	rowsRaw := firstRaw(base["rows"], base["grid"], root["rows"], root["grid"])
	if len(rowsRaw) == 0 {
		return grid.Snapshot{}, errors.New("no explicit rows/grid array found; upload a representative JSON frame to add a precise adapter")
	}
	rows, err := parseRows(rowsRaw)
	if err != nil {
		return grid.Snapshot{}, err
	}
	name := firstText(env.Event, env.Type, env.Action)
	return grid.Snapshot{Event: name, GameID: env.GameID, Rows: rows, SeenAt: time.Now().UTC()}, nil
}

func parseRows(raw json.RawMessage) ([]grid.Row, error) {
	var array []json.RawMessage
	if err := json.Unmarshal(raw, &array); err != nil {
		return nil, errors.New("rows/grid must be a JSON array")
	}
	rows := make([]grid.Row, 0, len(array))
	for i, item := range array {
		var obj rawRow
		if err := json.Unmarshal(item, &obj); err != nil {
			return nil, fmt.Errorf("row item %d is not a JSON object/array", i)
		}
		number := chooseInt(obj.Row, obj.Number, obj.Index)
		if number == 0 {
			number = i + 1
		}
		cellsRaw := firstRaw(obj.Cells, obj.Items, obj.Values)
		if len(cellsRaw) == 0 {
			// Some formats encode each row as a direct array of cell objects.
			if len(item) > 0 && item[0] == '[' {
				cellsRaw = item
			}
		}
		if len(cellsRaw) == 0 {
			continue
		}
		cells, err := parseCells(cellsRaw)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", number, err)
		}
		if len(cells) > 0 {
			rows = append(rows, grid.Row{Number: number, Cells: cells})
		}
	}
	if len(rows) == 0 {
		return nil, errors.New("grid array contained no explicit cell statuses")
	}
	return rows, nil
}

func parseCells(raw json.RawMessage) ([]grid.Cell, error) {
	var cells []json.RawMessage
	if err := json.Unmarshal(raw, &cells); err != nil {
		return nil, errors.New("cells must be a JSON array")
	}
	out := make([]grid.Cell, 0, len(cells))
	for i, item := range cells {
		var c rawCell
		if err := json.Unmarshal(item, &c); err != nil {
			continue
		}
		col := chooseInt(c.Column, c.Col, c.Index)
		if col == 0 {
			col = i + 1
		}
		status := firstText(c.Status, c.Type, c.Value, c.State)
		if status == "" {
			continue
		}
		status = strings.ToLower(strings.TrimSpace(status))
		// Avoid reporting a hidden cell as an observed outcome.
		if c.Revealed != nil && !*c.Revealed {
			continue
		}
		out = append(out, grid.Cell{Column: col, Status: status})
	}
	return out, nil
}

func ensureEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("payload must contain exactly one JSON value")
	}
	return nil
}
func firstRaw(values ...json.RawMessage) json.RawMessage {
	for _, v := range values {
		if len(v) > 0 && string(v) != "null" {
			return v
		}
	}
	return nil
}
func firstText(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
func chooseInt(values ...*int) int {
	for _, v := range values {
		if v != nil {
			return *v
		}
	}
	return 0
}
