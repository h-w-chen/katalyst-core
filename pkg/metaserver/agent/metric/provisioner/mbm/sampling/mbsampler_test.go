package sampling

import (
	"context"
	"github.com/pkg/errors"
	"testing"
)

type stubMBReader struct {
	MemBandwidthMonitor
	shutdownCalled bool
}

func (s *stubMBReader) Stop() {
	s.shutdownCalled = true
}

func (s *stubMBReader) ServePackageMB() error {
	return errors.New("no-op")
}

func (s *stubMBReader) ServeCoreMB() error {
	return errors.New("no-op")
}

func Test_mbSampler_Sample_Calls_Shutdown_on_Cancel(t *testing.T) {
	t.Parallel()

	stub := &stubMBReader{}
	testSampler := &mbSampler{
		monitor: stub,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go cancel()
	for {
		select {
		case <-ctx.Done():
			testSampler.Sample(ctx)
			goto loopExit
		default:
			testSampler.Sample(ctx)
		}
	}

loopExit:
	t.Logf("sampling loop exited after cancel")
	if !stub.shutdownCalled {
		t.Errorf("expected to shutdown; not yet")
	}
}
