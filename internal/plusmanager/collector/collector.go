package collector

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

const defaultBatchSize = 100

type Queue interface {
	PopOldest(count int) [][]byte
}

type UsageEventStore interface {
	InsertUsageEvent(context.Context, json.RawMessage) error
}

type Config struct {
	Mode         string
	QueueName    string
	BatchSize    int
	PollInterval time.Duration
}

type Status struct {
	Collector      string     `json:"collector"`
	Mode           string     `json:"mode"`
	Queue          string     `json:"queue"`
	LastConsumedAt *time.Time `json:"lastConsumedAt"`
	LastInsertedAt *time.Time `json:"lastInsertedAt"`
	TotalInserted  int64      `json:"totalInserted"`
	TotalSkipped   int64      `json:"totalSkipped"`
	DeadLetters    int64      `json:"deadLetters"`
	LastError      string     `json:"lastError"`
}

type Collector struct {
	queue Queue
	store UsageEventStore

	mode         string
	queueName    string
	batchSize    int
	pollInterval time.Duration

	mu             sync.Mutex
	running        bool
	stopCh         chan struct{}
	doneCh         chan struct{}
	lastConsumedAt *time.Time
	lastInsertedAt *time.Time
	totalInserted  int64
	totalSkipped   int64
	deadLetters    int64
	lastError      string
}

func New(cfg Config, queue Queue, store UsageEventStore) *Collector {
	mode := strings.TrimSpace(cfg.Mode)
	if mode == "" {
		mode = "auto"
	}
	queueName := strings.TrimSpace(cfg.QueueName)
	if queueName == "" {
		queueName = "redisqueue"
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return &Collector{
		queue:        queue,
		store:        store,
		mode:         mode,
		queueName:    queueName,
		batchSize:    batchSize,
		pollInterval: pollInterval,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}
}

func (c *Collector) Start(ctx context.Context) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	go c.run(ctx)
}

func (c *Collector) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
	select {
	case <-c.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Collector) run(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		close(c.doneCh)
	}()

	_, _ = c.PollOnce(ctx)
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			_, _ = c.PollOnce(ctx)
		}
	}
}

func (c *Collector) PollOnce(ctx context.Context) (int, error) {
	if c == nil || c.queue == nil || c.store == nil {
		return 0, nil
	}
	items := c.queue.PopOldest(c.batchSize)
	if len(items) == 0 {
		return 0, nil
	}

	inserted := 0
	var lastErr error
	for _, item := range items {
		now := time.Now()
		c.mu.Lock()
		c.lastConsumedAt = &now
		c.mu.Unlock()

		var raw json.RawMessage
		if err := json.Unmarshal(item, &raw); err != nil || !json.Valid(raw) {
			c.recordSkipped("invalid usage event JSON")
			continue
		}
		if err := c.store.InsertUsageEvent(ctx, append(json.RawMessage(nil), raw...)); err != nil {
			lastErr = err
			c.recordDeadLetter(err.Error())
			continue
		}
		inserted++
		insertedAt := time.Now()
		c.mu.Lock()
		c.totalInserted++
		c.lastInsertedAt = &insertedAt
		c.lastError = ""
		c.mu.Unlock()
	}
	return inserted, lastErr
}

func (c *Collector) Status() Status {
	if c == nil {
		return Status{Collector: "unavailable", Mode: "", Queue: ""}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	collector := "stopped"
	if c.running {
		collector = "running"
	}
	return Status{
		Collector:      collector,
		Mode:           c.mode,
		Queue:          c.queueName,
		LastConsumedAt: cloneTimePtr(c.lastConsumedAt),
		LastInsertedAt: cloneTimePtr(c.lastInsertedAt),
		TotalInserted:  c.totalInserted,
		TotalSkipped:   c.totalSkipped,
		DeadLetters:    c.deadLetters,
		LastError:      c.lastError,
	}
}

func (c *Collector) recordSkipped(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalSkipped++
	c.lastError = message
}

func (c *Collector) recordDeadLetter(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deadLetters++
	c.lastError = message
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cloned := *t
	return &cloned
}
