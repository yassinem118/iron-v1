package protocol

import "testing"

func TestParseFrameWrappedGrid(t *testing.T) {
	input := []byte(`{"event":"board:update","gameId":"g-1","data":{"grid":[{"row":1,"cells":[{"column":1,"status":"apple","revealed":true},{"column":2,"status":"bomb","revealed":false}]}]}}`)
	snapshot, err := ParseFrame(input)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Event != "board:update" || snapshot.GameID != "g-1" {
		t.Fatalf("unexpected metadata: %#v", snapshot)
	}
	if len(snapshot.Rows) != 1 || len(snapshot.Rows[0].Cells) != 1 {
		t.Fatalf("unrevealed cell should be omitted: %#v", snapshot.Rows)
	}
	if snapshot.Rows[0].Cells[0].Status != "apple" {
		t.Fatalf("unexpected cell: %#v", snapshot.Rows[0].Cells[0])
	}
}

func TestParseFrameRejectsNonJSON(t *testing.T) {
	if _, err := ParseFrame([]byte("not-json")); err == nil {
		t.Fatal("expected parse error")
	}
}
