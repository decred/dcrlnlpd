package server

import "time"

// ChanActivityScore holds "how much activity" a channel received through some
// period of time.
//
// The basic equation for determining the "activity" score for a channel is
//
//	atoms_sent / capacity / lifetime_hours
//
// The interpretation for this equation is that the activity score is the
// percentage of the channel capacity sent through the channel during its
// entire lifetime.
type ChanActivityScore float64

// toPercent returns the activity score as a percentage.
func (s ChanActivityScore) ToPercent() float64 {
	return float64(s) * 100
}

// channelActivity returns the "activity" score for a channel, which measures
// the total amount of atoms sent through the channel during its lifetime.
func channelActivity(totalSentAtoms, channelSizeAtoms int64,
	lifetime time.Duration) ChanActivityScore {

	if lifetime <= 0 {
		panic("lifetime cannot be <= 0")
	}
	if channelSizeAtoms <= 0 {
		panic("channel size cannot be <= 0")
	}

	// Consider channels with < 1 hour to be 1 hour long.
	hours := float64(lifetime / time.Hour)
	if hours <= 0 {
		hours = 1
	}

	return ChanActivityScore(float64(totalSentAtoms) / float64(channelSizeAtoms) / hours)
}
