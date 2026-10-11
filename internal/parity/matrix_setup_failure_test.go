package parity

import (
	"testing"

	"github.com/unraid/apprise-go/internal/notify"
	"github.com/unraid/apprise-go/internal/testutil"
)

// TestMatrixSetupFailureRequestParity proves setup fallback uses upstream's
// request flow; a failed account lookup leaves device-key setup unavailable.
func TestMatrixSetupFailureRequestParity(t *testing.T) {
	const url = "matrixs://token@example.com:1/!room?discovery=no"
	pythonSpecs, pythonSuccess := testutil.CapturePythonRequestsWithFailuresResult(t, url, "body", "title", 1)
	parsed, err := notify.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	restore := testutil.FailNextRequests(1)
	t.Cleanup(restore)
	goSpecs, err := testutil.CaptureGoRequestsResult(t, func() error {
		if err := notify.StoreFor(parsed).Clear(); err != nil {
			return err
		}
		target, err := notify.NewMatrixTarget(parsed)
		if err != nil {
			return err
		}
		return target.Send("body", "title", notify.NotifyInfo)
	})
	if pythonSuccess == nil || !*pythonSuccess || err != nil {
		t.Fatalf("setup fallback failed: python=%v go=%v", pythonSuccess, err)
	}
	assertRequestSpecSequenceMatches(t, pythonSpecs, goSpecs)
}
