package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recordingSleeper struct {
	durations []time.Duration
	cancelAt  int
}

func (s *recordingSleeper) Sleep(_ context.Context, d time.Duration) error {
	s.durations = append(s.durations, d)
	if s.cancelAt > 0 && len(s.durations) == s.cancelAt {
		return ErrCanceled
	}
	return nil
}

func TestPasteTextNormalizesAndBatches(t *testing.T) {
	rec := &Recorder{}
	sleep := &recordingSleeper{}
	err := PasteTextWithSleeper(context.Background(), "a\r\nbc", Options{
		StartDelay:    3 * time.Second,
		InterKeyDelay: 8 * time.Millisecond,
		BatchSize:     2,
		BatchPause:    20 * time.Millisecond,
	}, rec, sleep)
	if err != nil {
		t.Fatalf("PasteTextWithSleeper: %v", err)
	}
	if got := rec.String(); got != "a\nbc" {
		t.Fatalf("typed = %q", got)
	}
	if rec.Starts != 1 {
		t.Fatalf("starts = %d", rec.Starts)
	}
	wantSleeps := []time.Duration{3 * time.Second, 8 * time.Millisecond, 8 * time.Millisecond, 20 * time.Millisecond, 8 * time.Millisecond, 8 * time.Millisecond}
	if len(sleep.durations) != len(wantSleeps) {
		t.Fatalf("sleep count = %d, want %d: %v", len(sleep.durations), len(wantSleeps), sleep.durations)
	}
	for i := range wantSleeps {
		if sleep.durations[i] != wantSleeps[i] {
			t.Fatalf("sleep[%d] = %v, want %v", i, sleep.durations[i], wantSleeps[i])
		}
	}
}

func TestPasteTextEmptyNotifiesError(t *testing.T) {
	rec := &Recorder{}
	err := PasteTextWithSleeper(context.Background(), "\r", Options{}, rec, &recordingSleeper{})
	if !errors.Is(err, ErrEmptyText) {
		t.Fatalf("err = %v, want ErrEmptyText", err)
	}
	if rec.Errors != 1 {
		t.Fatalf("errors = %d", rec.Errors)
	}
}

func TestPasteTextCancellation(t *testing.T) {
	rec := &Recorder{}
	sleep := &recordingSleeper{cancelAt: 2}
	err := PasteTextWithSleeper(context.Background(), "abc", Options{
		InterKeyDelay: time.Millisecond,
	}, rec, sleep)
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("err = %v, want ErrCanceled", err)
	}
	if got := rec.String(); got != "a" {
		t.Fatalf("typed = %q", got)
	}
}
