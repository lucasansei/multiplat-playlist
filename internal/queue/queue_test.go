package queue

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromPathMissingReturnsEmptyQueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")

	q, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if q.Size() != 0 {
		t.Fatalf("Size() = %d, want 0", q.Size())
	}
	if q.CurrentIndex() != -1 {
		t.Fatalf("CurrentIndex() = %d, want -1", q.CurrentIndex())
	}
	if q.Current() != nil {
		t.Fatalf("Current() = %#v, want nil", q.Current())
	}
}

func TestQueueAddNextCurrentClear(t *testing.T) {
	q, err := LoadFromPath(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	first := Track{Platform: "youtube", ID: "one", URL: "https://youtu.be/one"}
	second := Track{Platform: "spotify", ID: "two", URL: "spotify:track:two"}
	q.Add(first)
	q.Add(second)

	if q.Size() != 2 {
		t.Fatalf("Size() = %d, want 2", q.Size())
	}

	if got := q.Next(); got == nil || *got != first {
		t.Fatalf("first Next() = %#v, want %#v", got, first)
	}
	if got := q.Current(); got == nil || *got != first {
		t.Fatalf("Current() = %#v, want %#v", got, first)
	}
	if got := q.Next(); got == nil || *got != second {
		t.Fatalf("second Next() = %#v, want %#v", got, second)
	}
	if got := q.Next(); got != nil {
		t.Fatalf("third Next() = %#v, want nil", got)
	}

	q.Clear()
	if q.Size() != 0 || q.CurrentIndex() != -1 {
		t.Fatalf("after Clear() size/index = %d/%d, want 0/-1", q.Size(), q.CurrentIndex())
	}
}

func TestQueueSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "queue.json")
	q, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	first := Track{Platform: "youtube", ID: "one", URL: "https://youtu.be/one"}
	second := Track{Platform: "youtube", ID: "two", URL: "https://youtu.be/two"}
	q.Add(first)
	q.Add(second)
	q.Next()

	if err := q.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if got.Size() != 2 {
		t.Fatalf("loaded Size() = %d, want 2", got.Size())
	}
	if got.CurrentIndex() != 0 {
		t.Fatalf("loaded CurrentIndex() = %d, want 0", got.CurrentIndex())
	}
	if current := got.Current(); current == nil || *current != first {
		t.Fatalf("loaded Current() = %#v, want %#v", current, first)
	}
}

func TestLoadFromPathMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	_, err := LoadFromPath(path)
	if err == nil {
		t.Fatal("LoadFromPath() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "parse queue") {
		t.Fatalf("LoadFromPath() error = %q, want parse queue context", err)
	}
}

func TestLoadFromPathNormalizesOutOfRangeIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	data := []byte(`{
  "tracks": [{"platform":"youtube","id":"one","url":"https://youtu.be/one"}],
  "index": 99
}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	q, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if q.CurrentIndex() != -1 {
		t.Fatalf("CurrentIndex() = %d, want -1", q.CurrentIndex())
	}
	if q.Current() != nil {
		t.Fatalf("Current() = %#v, want nil", q.Current())
	}
}
