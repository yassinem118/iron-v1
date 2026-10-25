// Package vision validates authorized screenshots and maps an explicitly supplied
// board region to cell geometry. It intentionally does not infer concealed game
// outcomes or label cells safe without a validated, evidence-backed model.
package vision

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const (
	MaxImageBytes = 4 << 20
	MaxDimension  = 4096
	MaxCells      = 400
)

type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Request struct {
	SessionID   string `json:"session_id"`
	ImageBase64 string `json:"image_base64"`
	Rows        int    `json:"rows"`
	Columns     int    `json:"columns"`
	Board       Rect   `json:"board"`
}

type Cell struct {
	Row        int     `json:"row"`
	Column     int     `json:"column"`
	Bounds     Rect    `json:"bounds_pixels"`
	Observed   string  `json:"observed"`
	Confidence float64 `json:"classification_confidence"`
	Status     string  `json:"status"`
}

type Result struct {
	SessionID                string   `json:"session_id,omitempty"`
	Status                   string   `json:"status"`
	ImageWidth               int      `json:"image_width"`
	ImageHeight              int      `json:"image_height"`
	Board                    Rect     `json:"board_pixels"`
	Cells                    []Cell   `json:"cells"`
	PredictiveModelAvailable bool     `json:"predictive_model_available"`
	Recommendations          []any    `json:"recommendations"`
	Limitations              []string `json:"limitations"`
}

// Analyze validates an image and maps a client-confirmed normalized board ROI
// into cells. It does not classify artwork or predict concealed outcomes.
func Analyze(imageBytes []byte, req Request) (Result, error) {
	if len(imageBytes) == 0 || len(imageBytes) > MaxImageBytes {
		return Result{}, fmt.Errorf("image must be between 1 and %d bytes", MaxImageBytes)
	}
	if req.Rows < 1 || req.Columns < 1 || req.Rows*req.Columns > MaxCells {
		return Result{}, fmt.Errorf("grid must contain 1..%d cells", MaxCells)
	}
	r := req.Board
	if r.X < 0 || r.Y < 0 || r.Width <= 0 || r.Height <= 0 ||
		r.X+r.Width > 1 || r.Y+r.Height > 1 {
		return Result{}, errors.New("board ROI must be normalized to 0..1 and fit within the image")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return Result{}, fmt.Errorf("unsupported or corrupt image: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxDimension || cfg.Height > MaxDimension {
		return Result{}, fmt.Errorf("image dimensions must be within 1..%d pixels", MaxDimension)
	}
	x, y := r.X*float64(cfg.Width), r.Y*float64(cfg.Height)
	w, h := r.Width*float64(cfg.Width), r.Height*float64(cfg.Height)
	out := Result{
		SessionID: req.SessionID, Status: "insufficient_evidence",
		ImageWidth: cfg.Width, ImageHeight: cfg.Height,
		Board:                    Rect{X: x, Y: y, Width: w, Height: h},
		PredictiveModelAvailable: false,
		Recommendations:          []any{},
		Limitations: []string{
			"Board region and grid dimensions were supplied by the caller, not automatically detected.",
			"Visual cell classification is not implemented without a validated, labeled game-specific dataset.",
			"Concealed outcomes cannot be established from this screenshot alone; no safe-cell recommendations are generated.",
		},
	}
	out.Cells = make([]Cell, 0, req.Rows*req.Columns)
	cellW, cellH := w/float64(req.Columns), h/float64(req.Rows)
	for row := 0; row < req.Rows; row++ {
		for col := 0; col < req.Columns; col++ {
			out.Cells = append(out.Cells, Cell{
				Row: row + 1, Column: col + 1,
				Bounds:   Rect{X: x + float64(col)*cellW, Y: y + float64(row)*cellH, Width: cellW, Height: cellH},
				Observed: "unknown", Confidence: 0, Status: "unclassified",
			})
		}
	}
	return out, nil
}
