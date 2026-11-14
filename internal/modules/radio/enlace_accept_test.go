package radio_test

// Aceite L-ENLACE (TELAS §4.4 e decisões F1–F5 do Grill do QA 2): a CPE é achada pelo MAC
// só nos APs ligados à interface, segue o IP da sessão do login e entra numa ronda de 15–30 min.
// Escrito por Claude; não editar.

import (
	"fmt"
	"infradesktop/internal/modules/radio"
	"testing"
	"time"
)

func ap(id string, failed bool, macs ...string) radio.APSeen {
	seen := radio.APSeen{AP: id, Failed: failed}
	for i, m := range macs {
		seen.Clients = append(seen.Clients, radio.APClient{MAC: m, Signal: fmt.Sprintf("-6%d dBm", i)})
	}
	return seen
}

const macSessao = "02:00:5e:00:00:01" // caller-id do PPPoE, minúsculo

func TestEnlaceMatchFindsTheCPEInOneAP(t *testing.T) {
	um := radio.EnlaceMatch(macSessao, []radio.APSeen{ap("ap-norte", false, "02:00:5E:00:00:01"), ap("ap-sul", false, "02:00:5E:00:00:09")})
	if um.State != radio.EnlaceFound || um.AP != "ap-norte" {
		t.Fatalf("MAC em um AP só deveria achar ap-norte: %+v", um)
	}
	falhou := radio.EnlaceMatch(macSessao, []radio.APSeen{ap("ap-norte", false, "02:00:5E:00:00:01"), ap("ap-sul", true)})
	if falhou.State != radio.EnlaceFound || falhou.AP != "ap-norte" || len(falhou.Failed) != 1 || falhou.Failed[0] != "ap-sul" {
		t.Fatalf("AP que falha não bloqueia o que respondeu e aparece em Failed: %+v", falhou)
	}
}

func TestEnlaceMatchAsksWhenTheMACIsInTwoAPs(t *testing.T) {
	dois := radio.EnlaceMatch(macSessao, []radio.APSeen{ap("ap-norte", false, "02:00:5E:00:00:01"), ap("ap-sul", false, "02:00:5e:00:00:01")})
	if dois.State != radio.EnlaceTwo || dois.AP != "" || len(dois.Choices) != 2 || dois.Choices[0].Signal == "" {
		t.Fatalf("MAC em dois APs pede escolha com o sinal de cada: %+v", dois)
	}
}

func TestEnlaceMatchNeverInventsAnAP(t *testing.T) {
	nenhum := radio.EnlaceMatch(macSessao, []radio.APSeen{ap("ap-norte", false, "02:00:5E:00:00:07")})
	if nenhum.State != radio.EnlaceNone || nenhum.AP != "" {
		t.Fatalf("MAC ausente não inventa AP: %+v", nenhum)
	}
	semLeitura := radio.EnlaceMatch(macSessao, []radio.APSeen{ap("ap-norte", true), ap("ap-sul", true)})
	if semLeitura.State != radio.EnlaceUnread || len(semLeitura.Failed) != 2 {
		t.Fatalf("todos os APs falharam: %+v", semLeitura)
	}
	invalido := radio.EnlaceMatch("roteador-em-bridge", []radio.APSeen{ap("ap-norte", false, "02:00:5E:00:00:01")})
	if invalido.State != radio.EnlaceNone {
		t.Fatalf("caller-id sem MAC não casa: %+v", invalido)
	}
}

func TestFollowLoginKeepsTheCPEOnTheCurrentSessionIP(t *testing.T) {
	row := func(state, address string) radio.PPRow {
		return radio.PPRow{Login: "cliente-0001", State: state, Address: address}
	}
	if got, changed := radio.FollowLogin("100.64.0.10", row("ONLINE", "100.64.0.22")); !changed || got != "100.64.0.22" {
		t.Fatalf("reconectou com IP novo: a CPE segue o IP da sessão (%q, %v)", got, changed)
	}
	if got, changed := radio.FollowLogin("100.64.0.10", row("ONLINE", "100.64.0.10")); changed || got != "100.64.0.10" {
		t.Fatalf("mesmo IP não é mudança (%q, %v)", got, changed)
	}
	for _, r := range []radio.PPRow{row("OFFLINE", "100.64.0.99"), row("ONLINE", ""), row("ONLINE", "não-é-ip")} {
		if got, changed := radio.FollowLogin("100.64.0.10", r); changed || got != "100.64.0.10" {
			t.Fatalf("sessão OFFLINE ou sem IP válido não muda o endereço: %+v → (%q, %v)", r, got, changed)
		}
	}
}

func TestRondaDueSpreadsCPEsBetween15And30Minutes(t *testing.T) {
	agora := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	if !radio.RondaDue("cpe-nova", time.Time{}, agora) {
		t.Fatal("CPE nunca lida entra na ronda")
	}
	cedo := 0
	for i := 0; i < 200; i++ {
		id := fmt.Sprintf("cpe-%04d", i)
		if radio.RondaDue(id, agora.Add(-15*time.Minute+time.Second), agora) {
			t.Fatalf("%s lida antes de 15 min", id)
		}
		if !radio.RondaDue(id, agora.Add(-30*time.Minute), agora) {
			t.Fatalf("%s passou de 30 min sem leitura", id)
		}
		if radio.RondaDue(id, agora.Add(-16*time.Minute), agora) {
			cedo++
		}
		primeira := radio.RondaDue(id, agora.Add(-20*time.Minute), agora)
		segunda := radio.RondaDue(id, agora.Add(-20*time.Minute), agora)
		if primeira != segunda {
			t.Fatalf("%s: RondaDue não é determinística", id)
		}
	}
	if cedo == 0 || cedo == 200 {
		t.Fatalf("ronda sem espalhamento entre 15 e 30 min: %d de 200 vencidas aos 16 min", cedo)
	}
}
