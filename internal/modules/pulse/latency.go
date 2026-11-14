package pulse

import (
	"context"
	"time"
)

const SampleWindow = 5 * time.Minute

type Sample struct {
	At      time.Time
	Outcome Outcome
	RTT     time.Duration
}
type LatencyMetrics struct {
	Known       bool          `json:"known"`
	Latency     time.Duration `json:"latency_ns"`
	LossPercent float64       `json:"loss_percent"`
}

func LatencyLoss(samples []Sample, now time.Time) LatencyMetrics {
	probes, replies := 0, 0
	var total time.Duration
	for _, sample := range samples {
		if sample.At.After(now) || now.Sub(sample.At) > SampleWindow {
			continue
		}
		switch sample.Outcome {
		case Reply:
			probes++
			replies++
			total += sample.RTT
		case NoReply:
			probes++
		}
	}
	if probes < 3 {
		return LatencyMetrics{}
	}
	result := LatencyMetrics{Known: true, LossPercent: float64(probes-replies) * 100 / float64(probes)}
	if replies > 0 {
		result.Latency = total / time.Duration(replies)
	}
	return result
}

// TimedProber supplies a measured RTT; the original port remains usable by callers
// that only need presence. Missing RTT never becomes a fabricated zero latency.
type TimedProber interface {
	ProbeSample(context.Context, string) Sample
}
