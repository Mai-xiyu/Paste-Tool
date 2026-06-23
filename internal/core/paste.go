package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrEmptyText = errors.New("paste text is empty")
	ErrCanceled  = errors.New("paste canceled")
)

type Options struct {
	StartDelay    time.Duration
	InterKeyDelay time.Duration
	BatchSize     int
	BatchPause    time.Duration
}

type Typer interface {
	SendRune(context.Context, rune) error
	NotifyStart() error
	NotifyError() error
}

type Sleeper interface {
	Sleep(context.Context, time.Duration) error
}

type RealSleeper struct{}

func (RealSleeper) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ErrCanceled
	case <-timer.C:
		return nil
	}
}

func PasteText(ctx context.Context, text string, options Options, typer Typer) error {
	return PasteTextWithSleeper(ctx, text, options, typer, RealSleeper{})
}

func PasteTextWithSleeper(ctx context.Context, text string, options Options, typer Typer, sleeper Sleeper) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if typer == nil {
		return errors.New("nil typer")
	}
	runes := NormalizeText(text)
	if len(runes) == 0 {
		_ = typer.NotifyError()
		return ErrEmptyText
	}
	if err := typer.NotifyStart(); err != nil {
		return fmt.Errorf("notify paste start: %w", err)
	}
	if err := sleeper.Sleep(ctx, options.StartDelay); err != nil {
		return err
	}
	for i, r := range runes {
		if err := ctx.Err(); err != nil {
			return ErrCanceled
		}
		if err := typer.SendRune(ctx, r); err != nil {
			_ = typer.NotifyError()
			return fmt.Errorf("send rune %d: %w", i, err)
		}
		if err := sleeper.Sleep(ctx, options.InterKeyDelay); err != nil {
			return err
		}
		sent := i + 1
		if options.BatchSize > 0 && sent < len(runes) && sent%options.BatchSize == 0 {
			if err := sleeper.Sleep(ctx, options.BatchPause); err != nil {
				return err
			}
		}
	}
	return nil
}

func NormalizeText(text string) []rune {
	out := make([]rune, 0, len(text))
	for _, r := range text {
		if r == '\r' {
			continue
		}
		out = append(out, r)
	}
	return out
}

type Recorder struct {
	mu     sync.Mutex
	Runes  []rune
	Starts int
	Errors int
}

func (r *Recorder) SendRune(_ context.Context, ch rune) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Runes = append(r.Runes, ch)
	return nil
}

func (r *Recorder) NotifyStart() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Starts++
	return nil
}

func (r *Recorder) NotifyError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Errors++
	return nil
}

func (r *Recorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return string(r.Runes)
}
