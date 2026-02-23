package utils

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// session fatigue state — reset at the start of each play session.
var (
	sessionMu    sync.RWMutex
	sessionStart time.Time
)

// SetSessionStart records the start of a new play session. Call once each
// time the bot enters a game. Sleep will then apply a progressive fatigue
// multiplier that rises from 1.0 to 1.25 over the first 3 hours, modelling
// the mild reaction-time slowdown observed in extended human play sessions.
func SetSessionStart() {
	sessionMu.Lock()
	sessionStart = time.Now()
	sessionMu.Unlock()
}

// sessionFatigue returns a multiplier in [1.0, 1.25] that grows linearly over
// the first 3 hours of a session and then plateaus. Returns 1.0 when no
// session has been started (e.g. outside a play sequence).
func sessionFatigue() float64 {
	sessionMu.RLock()
	start := sessionStart
	sessionMu.RUnlock()
	if start.IsZero() {
		return 1.0
	}
	f := time.Since(start).Hours() / 3.0
	if f > 1.0 {
		f = 1.0
	}
	return 1.0 + 0.25*f
}

// sampleGamma returns a sample from the Gamma(shape, scale) distribution using
// the Marsaglia-Tsang squeeze method. shape must be >= 1.
func sampleGamma(shape, scale float64) float64 {
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	for {
		x := rand.NormFloat64()
		v := 1.0 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		x2 := x * x
		u := rand.Float64()
		// Fast accept path
		if u < 1.0-0.0331*(x2*x2) {
			return d * v * scale
		}
		// Slow accept path
		if math.Log(u) < 0.5*x2+d*(1.0-v+math.Log(v)) {
			return d * v * scale
		}
	}
}

// Sleep pauses for a duration drawn from a Gamma(4, 0.25) distribution centred
// on the requested millisecond value (mean multiplier = 1.0). This produces a
// right-skewed distribution that resembles empirical human reaction-time data,
// unlike the previous flat ±30 % uniform jitter. The multiplier is clamped to
// [0.4, 2.5] to prevent pathological extremes.
func Sleep(milliseconds int) {
	const shape = 4.0
	const scale = 0.25 // mean = shape*scale = 1.0
	multiplier := sampleGamma(shape, scale)
	if multiplier < 0.4 {
		multiplier = 0.4
	}
	if multiplier > 2.5 {
		multiplier = 2.5
	}
	sleepMs := int(float64(milliseconds) * multiplier * sessionFatigue())
	time.Sleep(time.Duration(sleepMs) * time.Millisecond)
}

// RandLogNormal returns a duration in milliseconds sampled from a log-normal
// distribution parameterised by the given mean and standard deviation (both in
// ms). Log-normal is right-skewed, matching empirical human idle-time data
// (e.g. between-game gaps) far better than a flat uniform range.
func RandLogNormal(meanMs, stdMs float64) int {
	variance := stdMs * stdMs
	mu := math.Log(meanMs * meanMs / math.Sqrt(variance+meanMs*meanMs))
	sigma := math.Sqrt(math.Log(1.0 + variance/(meanMs*meanMs)))
	sample := math.Exp(mu + rand.NormFloat64()*sigma)
	if sample < 1 {
		sample = 1
	}
	return int(sample)
}

// RandGammaDurationMs returns a time.Duration sampled from a
// Gamma(shape, mean/shape) distribution with the requested mean in milliseconds.
// Higher shape → narrower spread; shape=3 gives a moderate right-skewed
// distribution that matches empirical human walk-click intervals better than
// the narrow uniform windows previously used.
func RandGammaDurationMs(meanMs float64, shape float64) time.Duration {
	scale := meanMs / shape
	sample := sampleGamma(shape, scale)
	if sample < 1 {
		sample = 1
	}
	return time.Duration(sample) * time.Millisecond
}
