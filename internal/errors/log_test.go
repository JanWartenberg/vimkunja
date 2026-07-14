package errors

import "testing"

func TestLogAppendsAndGetAllReturnsCopy(t *testing.T) {
	before := GetAll()
	first := "test error one"
	second := "test error two"

	Log(first)
	Log(second)

	got := GetAll()
	if len(got) != len(before)+2 {
		t.Fatalf("GetAll() returned %d entries, want %d", len(got), len(before)+2)
	}
	if got[len(got)-2] != first || got[len(got)-1] != second {
		t.Fatalf("last entries = %q, %q; want %q, %q", got[len(got)-2], got[len(got)-1], first, second)
	}

	got[len(got)-1] = "modified copy"
	if again := GetAll(); again[len(again)-1] != second {
		t.Fatalf("GetAll() exposed internal slice: got %q, want %q", again[len(again)-1], second)
	}
}

func TestLogIgnoresEmptyMessages(t *testing.T) {
	before := len(GetAll())
	Log("")
	if got := len(GetAll()); got != before {
		t.Fatalf("empty message changed log length to %d, want %d", got, before)
	}
}
