package pppoe

import (
	"infradesktop/internal/modules/equipment"
	"strconv"
	"strings"
	"time"
)

func PPPValues(e equipment.PPPEntry) map[string]string {
	values := map[string]string{"name": e.Name, "client": e.Client, "concentrator": e.Concentrator, "service": e.Service,
		"profile": e.Profile, "state": e.State, "interface": e.Interface, "mtu": e.MTU, "mru": e.MRU, "authentication": e.Authentication,
		"rate": e.Rate, "local_address": e.LocalAddress, "remote_address": e.RemoteAddress, "only_one": e.OnlyOne,
		"caller_id": e.Caller, "last_logged_out": e.LastOut, "last_caller_id": e.LastCaller, "last_disconnect_reason": e.LastReason,
		"online": strconv.Itoa(e.Online), "registrations": strconv.Itoa(e.Registrations), "aps": strings.Join(e.APs, ", ")}
	if e.VLAN != nil {
		values["vlan"] = strconv.Itoa(*e.VLAN)
	}
	if e.Peak != nil {
		values["peak"] = strconv.Itoa(*e.Peak)
	}
	if e.Drop != nil {
		values["drop"] = strconv.FormatFloat(*e.Drop, 'f', 2, 64)
	}
	if e.Session != nil {
		sessionPPPValues(values, *e.Session)
	}
	if e.Kind == "server" {
		values["state"] = serverState(e)
	}
	if e.Kind == "secret" {
		values["registration"] = registrationState(e.Disabled)
	}
	return values
}
func sessionPPPValues(values map[string]string, r PPRow) {
	values["address"], values["caller_id"], values["session_id"] = r.Address, r.Caller, r.SessionID
	values["linked_device"] = r.LinkedDeviceName
	values["duplicates"] = "Sem duplicidade"
	if r.Duplicates > 1 {
		values["duplicates"] = "Com duplicidade"
	}
	if r.MTU != nil {
		values["mtu"] = strconv.FormatUint(*r.MTU, 10)
	}
	values["absent_end"] = r.AbsentEnd
	if r.Uptime.Known && r.State != "OFFLINE" {
		values["uptime"] = time.Duration(r.Uptime.Min).String()
	}
	if r.Download != nil {
		values["download"] = strconv.FormatFloat(r.Download.Min, 'f', 2, 64)
	}
	if r.Upload != nil {
		values["upload"] = strconv.FormatFloat(r.Upload.Min, 'f', 2, 64)
	}
}
func serverState(e equipment.PPPEntry) string {
	if (e.Disabled != nil && *e.Disabled) || (e.Invalid != nil && *e.Invalid) {
		return "Inválido"
	}
	if e.Disabled == nil || e.Invalid == nil {
		return "Não informado nesta leitura"
	}
	return "Válido"
}
func registrationState(disabled *bool) string {
	if disabled == nil {
		return "Não informado nesta leitura"
	}
	if *disabled {
		return "Desativado"
	}
	return "Ativo"
}
