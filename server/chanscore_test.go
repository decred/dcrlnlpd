package server

import (
	"testing"
	"time"
)

func TestChanActivityScore(t *testing.T) {
	hour := time.Hour
	dcr := int64(1e8)

	tests := []struct {
		name string
		sent int64
		size int64
		life time.Duration
		want chanActivityScore
	}{{
		name: "1 atom sent through 1 DCR chan within 1 hour",
		sent: 1,
		size: dcr,
		life: hour,
		want: 0.00000001,
	}, {
		name: "1 atom sent through 100k DCR chan within 100k hours",
		sent: 1,
		size: 100_000 * dcr,
		life: 100_000 * hour,
		want: 1e-18,
	}, {
		name: "100MM dcr sent through a 10K atom chan within 1 hour",
		sent: 100_000_000,
		size: 10_000,
		life: hour,
		want: 10000,
	}, {
		name: "0 dcr sent through a 10 atom chan within 1 microsecond",
		sent: 0,
		size: 10,
		life: time.Microsecond,
		want: 0,
	}, {
		name: "1 dcr sent through a 1 dcr chan within 1 minute",
		sent: dcr,
		size: dcr,
		life: time.Minute,
		want: 1,
	}}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotScore := channelActivity(tc.sent, tc.size, tc.life)
			if gotScore != tc.want {
				t.Fatalf("Unexpected score: got %v, want %v",
					gotScore, tc.want)
			}
		})
	}
}
