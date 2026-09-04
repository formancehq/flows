package activities

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

// paymentsVersion mirrors fctl's cmd/payments/versions/versions.go: the payments module
// reports its own release semver via GetServerInfo, and that release track is what
// determines API surface, not the "v1/v2/v3" prefix in a given endpoint's path. The v3
// payment-initiations API (InitiatePayment, ListConnectors, ...) only exists from payments
// major release v3 onward - older stacks must keep using the v1 create/transfer-initiation
// path, which is why every caller of CreateTransferInitiation needs to know which one it's
// talking to before picking an API surface.
type paymentsVersion struct {
	raw        string
	supportsV3 bool
}

// paymentsVersionTTL bounds how stale our belief about the target stack's payments version
// can get. Without it, a stack upgrade would go unnoticed until the worker process restarts.
const paymentsVersionTTL = 10 * time.Minute

var (
	paymentsVersionMu    sync.Mutex
	paymentsVersionCache *paymentsVersion
	paymentsVersionAt    time.Time
)

// getPaymentsVersion returns the target stack's payments module version, from a
// process-local cache refreshed at most every paymentsVersionTTL. A single worker process
// talks to exactly one stack (stackURL is fixed at startup - see cmd/worker.go), so this
// cache needs no per-stack keying.
func (a Activities) getPaymentsVersion(ctx context.Context) (*paymentsVersion, error) {
	paymentsVersionMu.Lock()
	if paymentsVersionCache != nil && time.Since(paymentsVersionAt) < paymentsVersionTTL {
		v := paymentsVersionCache
		paymentsVersionMu.Unlock()
		return v, nil
	}
	paymentsVersionMu.Unlock()

	resp, err := a.client.Payments.V1.PaymentsgetServerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching payments server info: %w", err)
	}
	if resp.PaymentsServerInfo == nil || resp.PaymentsServerInfo.Version == nil {
		return nil, fmt.Errorf("payments server info response missing version")
	}

	v := computePaymentsVersion(*resp.PaymentsServerInfo.Version)

	paymentsVersionMu.Lock()
	paymentsVersionCache = v
	paymentsVersionAt = time.Now()
	paymentsVersionMu.Unlock()

	return v, nil
}

func computePaymentsVersion(raw string) *paymentsVersion {
	version := "v" + strings.TrimPrefix(raw, "v")
	major := semver.Major(version)
	if major == "" {
		// Not parseable as semver (e.g. a dev build tagged with a commit SHA) - assume
		// latest, same fallback fctl's computePaymentVersion uses.
		return &paymentsVersion{raw: raw, supportsV3: true}
	}

	majorNum, err := strconv.Atoi(strings.TrimPrefix(major, "v"))
	if err != nil {
		return &paymentsVersion{raw: raw, supportsV3: true}
	}

	return &paymentsVersion{raw: raw, supportsV3: majorNum >= 3}
}
