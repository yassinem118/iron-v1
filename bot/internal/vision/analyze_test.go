package vision

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, color.White)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestAnalyzeMapsGridWithoutInventingPredictions(t *testing.T) {
	raw := testPNG(t, 1000, 500)
	res, err := Analyze(raw, Request{SessionID: "s1", Rows: 5, Columns: 5, Board: Rect{X: .1, Y: .2, Width: .8, Height: .6}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "insufficient_evidence" || res.PredictiveModelAvailable || len(res.Recommendations) != 0 {
		t.Fatalf("must not invent predictions: %#v", res)
	}
	if len(res.Cells) != 25 || res.Cells[0].Bounds.X != 100 || res.Cells[0].Bounds.Y != 100 {
		t.Fatalf("unexpected geometry: %#v", res.Cells[0])
	}
	if res.Cells[0].Observed != "unknown" {
		t.Fatal("unobserved cell must remain unknown")
	}
}

func TestAnalyzeRejectsBadROIAndDimensions(t *testing.T) {
	raw := testPNG(t, 20, 20)
	_, err := Analyze(raw, Request{Rows: 5, Columns: 5, Board: Rect{X: .8, Y: .8, Width: .5, Height: .5}})
	if err == nil {
		t.Fatal("expected ROI validation error")
	}
	_, err = Analyze(raw, Request{Rows: 50, Columns: 50, Board: Rect{X: 0, Y: 0, Width: 1, Height: 1}})
	if err == nil {
		t.Fatal("expected cell limit error")
	}
}

func TestBase64CanRepresentImage(t *testing.T) {
	raw := testPNG(t, 2, 2)
	if base64.StdEncoding.EncodeToString(raw) == "" {
		t.Fatal("base64 encoding empty")
	}
}
