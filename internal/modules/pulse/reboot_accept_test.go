package pulse_test

// Aceite L-PULSO-4 (decisão de paridade de 09/11 e TELAS §3.1/§4.1): reinício é a hora de
// ligar que avança, não só o tempo ligado que diminui; contagem de 7 e 30 dias.
// Escrito por Claude; não editar.

import (
	"infradesktop/internal/modules/pulse"
	"testing"
	"time"
)

var agora = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

func TestBootTimeIsReadingTimeMinusUptime(t *testing.T) {
	if got := pulse.BootTime(agora, 10*time.Minute); !got.Equal(agora.Add(-10 * time.Minute)) {
		t.Fatalf("hora de ligar: %v", got)
	}
}

func TestRebootedComparesBootTimes(t *testing.T) {
	ligou := agora.Add(-2 * time.Hour)
	casos := []struct {
		nome       string
		antes, now time.Time
		quer       bool
	}{
		{"primeira leitura, sem hora anterior", time.Time{}, ligou, false},
		{"mesma hora de ligar", ligou, ligou, false},
		{"arredondamento e relógio até 60 s", ligou, ligou.Add(60 * time.Second), false},
		{"tempo ligado voltou a zero", ligou, agora.Add(-1 * time.Minute), true},
		{"leitura atrasada: religou e já passou o tempo anterior", ligou, agora.Add(-30 * time.Minute), true},
		{"hora de ligar recuou (relógio do equipamento)", ligou, ligou.Add(-10 * time.Minute), false},
	}
	for _, c := range casos {
		if got := pulse.Rebooted(c.antes, c.now); got != c.quer {
			t.Fatalf("%s: quer %v, veio %v", c.nome, c.quer, got)
		}
	}
}

func TestRebootCountsLastSevenAndThirtyDays(t *testing.T) {
	dia := 24 * time.Hour
	boots := []time.Time{agora.Add(-40 * dia), agora.Add(-20 * dia), agora.Add(-3 * dia), agora.Add(-time.Hour)}
	last, d7, d30 := pulse.RebootCounts(boots, agora)
	if !last.Equal(agora.Add(-time.Hour)) || d7 != 2 || d30 != 3 {
		t.Fatalf("último %v, 7 dias %d, 30 dias %d", last, d7, d30)
	}
	last, d7, d30 = pulse.RebootCounts(nil, agora)
	if !last.IsZero() || d7 != 0 || d30 != 0 {
		t.Fatal("sem reinício gravado deve dar zero")
	}
}
