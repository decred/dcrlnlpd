package server

import (
	"testing"
	"time"
)

func TestChannelHadActivity(t *testing.T) {
	day := time.Hour * 24

	tests := []struct {
		name string
		ra   int64
		ta   int64
		ri   time.Duration
		lf   time.Duration
		want bool
	}{{
		name: "1000 atoms / day, exact amount and time",
		ra:   1000,
		ta:   1000,
		ri:   day,
		lf:   day,
		want: true,
	}, {
		name: "1000 atoms / day, day + 23 hours, 59 seconds",
		ra:   1000,
		ta:   1000,
		ri:   day,
		lf:   day + time.Hour*23 + time.Second*59,
		want: true,
	}, {
		name: "1000 atoms / day, 2 days",
		ra:   1000,
		ta:   1000,
		ri:   day,
		lf:   day * 2,
		want: false,
	}, {
		name: "1000 atoms / day, 1999 atoms in 2 days",
		ra:   1000,
		ta:   1999,
		ri:   day,
		lf:   day * 2,
		want: false,
	}, {
		name: "1000 atoms / day, 2000 atoms in 2 days",
		ra:   1000,
		ta:   2000,
		ri:   day,
		lf:   day * 2,
		want: true,
	}, {
		name: "1000 atoms / day, 39999 atoms in 40 days",
		ra:   1000,
		ta:   39999,
		ri:   day,
		lf:   day * 40,
		want: false,
	}, {
		name: "1000 atoms / day, 40000 atoms in 40 days",
		ra:   1000,
		ta:   40000,
		ri:   day,
		lf:   day * 40,
		want: true,
	}}

	for _, tc := range tests {
		got := channelHadActivity(tc.ra, tc.ta, tc.ri, tc.lf)
		if tc.want != got {
			t.Fatalf("test case %q: unexpected result: got %v, want %v",
				tc.name, got, tc.want)
		}
	}
}
