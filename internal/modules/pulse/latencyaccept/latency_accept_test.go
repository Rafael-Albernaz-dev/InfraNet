package pulse_test

// Aceite L-PULSO-5 (TELAS §3, colunas Latência e Perda "do ping do coletor"): janela dos
// últimos 5 min de pings; latência = média das respostas; perda = % sem resposta.
// Escrito por Claude; não editar.

import (
	"infradesktop/internal/modules/pulse"
	"testing"
	"time"
)

var agora = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

func amostra(segundosAtras int, o pulse.Outcome, rttMs int) pulse.Sample {
	return pulse.Sample{At: agora.Add(-time.Duration(segundosAtras) * time.Second), Outcome: o, RTT: time.Duration(rttMs) * time.Millisecond}
}

func TestLatencyLossAveragesRepliesInTheLastFiveMinutes(t *testing.T) {
	s := []pulse.Sample{
		amostra(400, pulse.Reply, 900), // fora da janela de 5 min
		amostra(30, pulse.Reply, 4),
		amostra(20, pulse.NoReply, 0),
		amostra(10, pulse.Reply, 8),
		amostra(0, pulse.Reply, 6),
	}
	got := pulse.LatencyLoss(s, agora)
	if !got.Known || got.Latency != 6*time.Millisecond || got.LossPercent != 25 {
		t.Fatalf("quer 6 ms e 25%% de perda, veio %+v", got)
	}
}

func TestLatencyLossNeedsThreeProbes(t *testing.T) {
	got := pulse.LatencyLoss([]pulse.Sample{amostra(10, pulse.Reply, 5), amostra(0, pulse.Reply, 5)}, agora)
	if got.Known {
		t.Fatalf("com menos de 3 pings a coluna mostra —: %+v", got)
	}
}

func TestLatencyLossAllLostIsHundredPercentWithoutLatency(t *testing.T) {
	s := []pulse.Sample{amostra(20, pulse.NoReply, 0), amostra(10, pulse.NoReply, 0), amostra(0, pulse.NoReply, 0)}
	got := pulse.LatencyLoss(s, agora)
	if !got.Known || got.LossPercent != 100 || got.Latency != 0 {
		t.Fatalf("todos perdidos: 100%% e sem latência, veio %+v", got)
	}
}

func TestLatencyLossIgnoresProbeUnavailable(t *testing.T) {
	s := []pulse.Sample{
		amostra(30, pulse.ProbeUnavailable, 0), amostra(20, pulse.Reply, 3),
		amostra(10, pulse.ProbeUnavailable, 0), amostra(0, pulse.Reply, 5)}
	got := pulse.LatencyLoss(s, agora)
	if got.Known {
		t.Fatalf("ping sem permissão não conta como perda nem como amostra: %+v", got)
	}
}
