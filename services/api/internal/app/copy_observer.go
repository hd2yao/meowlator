package app

import (
	"context"
	"errors"
	"net"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type CopyObserver interface {
	ObserveCopyRequest()
	ObserveCopyFailure(timeout bool)
}

type observedCopyClient struct {
	next     CopyClient
	observer CopyObserver
}

func NewObservedCopyClient(next CopyClient, observer CopyObserver) CopyClient {
	if next == nil || observer == nil {
		return next
	}
	return &observedCopyClient{next: next, observer: observer}
}

func (c *observedCopyClient) Generate(ctx context.Context, result domain.InferenceResult, styleVersion string) (domain.CopyBlock, error) {
	c.observer.ObserveCopyRequest()
	out, err := c.next.Generate(ctx, result, styleVersion)
	if err != nil {
		c.observer.ObserveCopyFailure(isCopyTimeout(err))
	}
	return out, err
}

func isCopyTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
