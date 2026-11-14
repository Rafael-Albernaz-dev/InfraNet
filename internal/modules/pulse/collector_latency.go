package pulse

import "context"

func (c *Collector) syncSamples(targets []Target, profile string) {
	next := make(map[string][]Sample, len(targets))
	old := make(map[string]string, len(c.targets))
	if profile == c.profile {
		for _, target := range c.targets {
			old[target.ID] = target.Address
		}
	}
	for _, target := range targets {
		next[target.ID] = nil
		if old[target.ID] == target.Address {
			next[target.ID] = c.samples[target.ID]
		}
	}
	c.samples = next
	c.sampleTargets = make(map[string]Target, len(targets))
	for _, target := range targets {
		c.sampleTargets[target.ID] = target
	}
}
func (c *Collector) probe(ctx context.Context, address string) Sample {
	if prober, ok := c.prober.(TimedProber); ok {
		return prober.ProbeSample(ctx, address)
	}
	return Sample{Outcome: c.prober.Probe(ctx, address)}
}
func (c *Collector) recordSample(tracker *Tracker, target Target, sample Sample) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if tracker != c.tracker || c.sampleTargets[target.ID] != target {
		return
	}
	tracker.Record(target, sample.Outcome, sample.At)
	// Permission failures invalidate the displayed metric until three new probes.
	if sample.Outcome == ProbeUnavailable {
		c.samples[target.ID] = nil
		return
	}
	samples, present := c.samples[target.ID]
	if !present {
		return
	}
	first := 0
	for first < len(samples) && sample.At.Sub(samples[first].At) > SampleWindow {
		first++
	}
	samples = append(append([]Sample(nil), samples[first:]...), sample)
	c.samples[target.ID] = samples
}
