package pppoe_test

// Aceite L-PPP (TELAS §4.2 e decisão E3 do Grill do QA 2): a coleta de 60 s mantém o contexto
// da sessão que continua (plano, VLAN, serviço) e o nome do cliente vem do comentário do secret.
// Escrito por Claude; não editar.

import (
	"infradesktop/internal/modules/pppoe"
	"strings"
	"testing"
)

func vlan(n int) *int { return &n }

func linha(remote, sid, login string, ctx *pppoe.PPContext) pppoe.PPRow {
	return pppoe.PPRow{RemoteID: remote, SessionID: sid, Login: login, Context: ctx}
}

func TestCarryContextKeepsPlanAndVLANOfTheSameSession(t *testing.T) {
	cheio := &pppoe.PPContext{Association: "associated", Interface: "<pppoe-ana>", Service: "painel-04",
		VLAN: vlan(1004), Profile: "Plano 100M", SecretState: "disabled", DownloadLimit: "100M", UploadLimit: "20M"}
	antes := []pppoe.PPRow{
		linha("*1", "0x81", "ana", cheio),
		linha("*2", "0x82", "bia", cheio),
	}
	so := &pppoe.PPContext{Association: "associated", Interface: "<pppoe-ana>"}
	proprio := &pppoe.PPContext{Association: "associated", Interface: "<pppoe-caio>", Profile: "Plano 50M"}
	depois := []pppoe.PPRow{
		linha("*1", "0x81", "ana", so),       // mesma sessão: herda
		linha("*9", "0x99", "bia", nil),      // reconectou: sessão nova, não herda
		linha("*3", "0x83", "caio", proprio), // já veio com contexto próprio: fica
	}
	pppoe.CarryContext(antes, depois)
	if c := depois[0].Context; c == nil || c.Profile != "Plano 100M" || c.VLAN == nil || *c.VLAN != 1004 ||
		c.Service != "painel-04" || c.DownloadLimit != "100M" || c.Interface != "<pppoe-ana>" {
		t.Fatalf("sessão que continua perdeu o contexto: %+v", c)
	}
	if depois[1].Context != nil && depois[1].Context.Profile != "" {
		t.Fatalf("sessão nova herdou contexto de outra sessão: %+v", depois[1].Context)
	}
	if c := depois[2].Context; c == nil || c.Profile != "Plano 50M" {
		t.Fatalf("contexto lido agora foi trocado pelo antigo: %+v", c)
	}
}

func TestClientNameFromSecretComment(t *testing.T) {
	casos := map[string]string{
		"SGP: Ana Maria da Silva": "Ana Maria da Silva",
		"SGP:Bia":                 "Bia",
		"sgp:  Caio ":             "Caio",
		"Cliente antigo":          "Cliente antigo",
		"":                        "",
		"SGP:":                    "",
		"SGP: Ana\tMaria\n":       "Ana Maria",
	}
	for comentario, quer := range casos {
		if got := pppoe.ClientName(comentario); got != quer {
			t.Fatalf("ClientName(%q) = %q, quer %q", comentario, got, quer)
		}
	}
	if got := pppoe.ClientName("SGP: " + strings.Repeat("x", 200)); len([]rune(got)) != 120 {
		t.Fatalf("nome acima de 120 caracteres: %d", len([]rune(got)))
	}
}
