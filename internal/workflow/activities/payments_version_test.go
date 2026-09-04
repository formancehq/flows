package activities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputePaymentsVersion(t *testing.T) {
	testCases := []struct {
		name       string
		raw        string
		supportsV3 bool
	}{
		{name: "pre-v3 major", raw: "v2.4.1", supportsV3: false},
		{name: "pre-v3 major without leading v", raw: "2.4.1", supportsV3: false},
		{name: "v0 major", raw: "v0.9.9", supportsV3: false},
		{name: "exactly v3.0.0", raw: "v3.0.0", supportsV3: true},
		{name: "v3 patch release", raw: "v3.4.2", supportsV3: true},
		{name: "double digit major beyond v3", raw: "v10.0.0", supportsV3: true},
		{name: "v3 prerelease", raw: "v3.2.0-beta.1", supportsV3: true},
		{name: "major only, no minor/patch", raw: "v3", supportsV3: true},
		{name: "unparseable dev build assumes latest", raw: "deadbeef1234", supportsV3: true},
		{name: "empty string assumes latest", raw: "", supportsV3: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := computePaymentsVersion(tc.raw)
			require.Equal(t, tc.raw, v.raw)
			require.Equal(t, tc.supportsV3, v.supportsV3, "raw version %q", tc.raw)
		})
	}
}
