package collector

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeQueue struct{ batches [][][]byte }

func (q *fakeQueue) PopOldest(count int) [][]byte {
	if len(q.batches) == 0 {
		return nil
	}
	batch := q.batches[0]
	q.batches = q.batches[1:]
	if count > 0 && len(batch) > count {
		remain := append([][]byte(nil), batch[count:]...)
		q.batches = append([][][]byte{remain}, q.batches...)
		batch = batch[:count]
	}
	return batch
}

type fakeStore struct {
	events []json.RawMessage
	err    error
}

func (s *fakeStore) InsertUsageEvent(ctx context.Context, event json.RawMessage) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, append(json.RawMessage(nil), event...))
	return nil
}

func TestPlusCollectorPollOnceInsertsUsageEventsAndUpdatesStatus(t *testing.T) {
	q := &fakeQueue{batches: [][][]byte{{
		[]byte(`{"request_id":"one","tokens":{"total_tokens":3}}`),
		[]byte(`{"request_id":"two","tokens":{"total_tokens":5}}`),
	}}}
	s := &fakeStore{}
	c := New(Config{Mode: "auto", QueueName: "test", BatchSize: 10, PollInterval: time.Hour}, q, s)

	inserted, err := c.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if inserted != 2 {
		t.Fatalf("inserted = %d, want 2", inserted)
	}
	if len(s.events) != 2 {
		t.Fatalf("store events = %d, want 2", len(s.events))
	}
	status := c.Status()
	if status.Collector != "stopped" || status.Mode != "auto" || status.Queue != "test" {
		t.Fatalf("status identity = %#v", status)
	}
	if status.TotalInserted != 2 || status.TotalSkipped != 0 || status.DeadLetters != 0 {
		t.Fatalf("status counters = %#v", status)
	}
	if status.LastConsumedAt == nil || status.LastInsertedAt == nil {
		t.Fatalf("expected consumed and inserted timestamps, got %#v", status)
	}
	if status.LastError != "" {
		t.Fatalf("last error = %q, want empty", status.LastError)
	}
}

func TestPlusCollectorPollOnceSkipsInvalidJSONAndCountsStoreFailuresAsDeadLetters(t *testing.T) {
	q := &fakeQueue{batches: [][][]byte{{[]byte(`not-json`), []byte(`{"request_id":"ok"}`)}}}
	s := &fakeStore{err: errors.New("insert failed")}
	c := New(Config{Mode: "auto", QueueName: "test", BatchSize: 10, PollInterval: time.Hour}, q, s)

	inserted, err := c.PollOnce(context.Background())
	if err == nil {
		t.Fatalf("PollOnce() error = nil, want insert failure")
	}
	if inserted != 0 {
		t.Fatalf("inserted = %d, want 0", inserted)
	}
	status := c.Status()
	if status.TotalInserted != 0 || status.TotalSkipped != 1 || status.DeadLetters != 1 {
		t.Fatalf("status counters = %#v", status)
	}
	if status.LastConsumedAt == nil {
		t.Fatalf("expected consumed timestamp, got %#v", status)
	}
	if status.LastInsertedAt != nil {
		t.Fatalf("last inserted timestamp = %v, want nil", status.LastInsertedAt)
	}
	if status.LastError == "" {
		t.Fatalf("last error empty, want failure details")
	}
}
