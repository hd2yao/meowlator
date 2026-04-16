package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type recordingCopyObserver struct {
	total    int
	failures int
	timeouts int
}

func (r *recordingCopyObserver) ObserveCopyRequest() {
	r.total++
}

func (r *recordingCopyObserver) ObserveCopyFailure(timeout bool) {
	r.failures++
	if timeout {
		r.timeouts++
	}
}

type timeoutCopyClient struct {
	err error
}

func (t timeoutCopyClient) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	_ = ctx
	_ = result
	_ = styleVersion
	return domain.CopyBlock{}, t.err
}

func TestObservedCopyClientRecordsTimeoutsAndFailures(t *testing.T) {
	observer := &recordingCopyObserver{}
	client := NewObservedCopyClient(timeoutCopyClient{err: context.DeadlineExceeded}, observer)

	_, err := client.Generate(context.Background(), domain.InferenceResult{}, "v1")
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if observer.total != 1 || observer.failures != 1 || observer.timeouts != 1 {
		t.Fatalf("unexpected observer counters: %+v", observer)
	}
}

func TestObservedCopyClientRecordsNonTimeoutFailure(t *testing.T) {
	observer := &recordingCopyObserver{}
	client := NewObservedCopyClient(timeoutCopyClient{err: errors.New("boom")}, observer)

	_, err := client.Generate(context.Background(), domain.InferenceResult{}, "v1")
	if err == nil {
		t.Fatalf("expected failure error")
	}
	if observer.total != 1 || observer.failures != 1 || observer.timeouts != 0 {
		t.Fatalf("unexpected observer counters: %+v", observer)
	}
}

func TestObservedCopyClientRecordsSuccess(t *testing.T) {
	observer := &recordingCopyObserver{}
	client := NewObservedCopyClient(timeoutCopyClient{}, observer)

	_, err := client.Generate(context.Background(), domain.InferenceResult{
		IntentTop3: []domain.IntentProb{{Label: domain.IntentWantPlay, Prob: 0.8}},
		State:      domain.State3D{Tension: domain.LevelMid, Arousal: domain.LevelMid, Comfort: domain.LevelLow},
		Confidence: 0.8,
		Source:     "CLOUD",
	}, "v1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if observer.total != 1 || observer.failures != 0 || observer.timeouts != 0 {
		t.Fatalf("unexpected observer counters: %+v", observer)
	}
}

func TestObservedCopyClientTreatsDeadlineWrappedTimeoutAsTimeout(t *testing.T) {
	observer := &recordingCopyObserver{}
	client := NewObservedCopyClient(timeoutCopyClient{err: context.DeadlineExceeded}, observer)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, _ = client.Generate(ctx, domain.InferenceResult{}, "v1")
	if observer.timeouts != 1 {
		t.Fatalf("expected timeout to be recorded")
	}
}
