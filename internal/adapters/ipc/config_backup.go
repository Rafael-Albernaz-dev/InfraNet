package ipc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/foundation"
)

var configFields = map[string][]string{
	"config.list": {"device_id"}, "config.save": {"device_id"},
	"config.compare":  {"device_id", "version_id"},
	"config.download": {"device_id", "version_id", "path", "confirmed"},
}

// ignored: only the result code or validation outcome is needed here.
func isConfig(op string) bool                             { _, ok := configFields[op]; return ok }
func (s *Server) AttachConfig(service *cfgbackup.Service) { s.config = service }
func validateConfig(b []byte, q Request, fields map[string]json.RawMessage) (Request, foundation.Code) {
	if q.ContractRevision != ProductRevision {
		return q, foundation.Version
	}
	keys := []string{"version", "request_id", "op", "capability", "contract_revision", "session", "profile_id",
		"activation_id", "expected_profile", "expected_activation", "config"}
	if len(fields) != len(keys) || q.Config == nil || !access.ValidCompanyID(q.ExpectedProfile) || !validID(q.ExpectedActivation) {
		return q, foundation.Invalid
	}
	for _, key := range keys {
		// ignored: only the result code or validation outcome is needed here.
		if _, ok := fields[key]; !ok {
			return q, foundation.Invalid
		}
	}
	if !strictPPObject(fields["config"], configFields[q.Op]) {
		return q, foundation.Invalid
	}
	var in cfgbackup.Input
	if strict(fields["config"], &in) != nil || !access.ValidCompanyID(in.DeviceID) {
		return q, foundation.Invalid
	}
	if !validConfigInput(q.Op, in) {
		return q, foundation.Invalid
	}
	q.Config = &in
	return q, foundation.OK
}
func (s *Server) dispatchConfig(ctx context.Context, uid uint32, q Request, owner string) Response {
	r := Response{Version: 1, RequestID: q.RequestID, Code: foundation.Invalid}
	if s.config == nil || q.Config == nil {
		return r
	}
	if subtle.ConstantTimeCompare([]byte(q.Capability), []byte(s.capability)) != 1 {
		r.Code = foundation.Unauthenticated
		return r
	}
	s.account.Reap(ownerAlive)
	scope := access.Command{Session: q.Session, Owner: owner, Profile: q.Profile, Activation: q.Activation,
		ExpectedProfile: q.ExpectedProfile, ExpectedActivation: q.ExpectedActivation}
	view, code := s.config.Handle(ctx, equipment.Command{Op: q.Op, UID: uid, Scope: scope}, *q.Config)
	r.Code = foundation.Code(code)
	if code != equipment.OK {
		return r
	}
	encoded, err := json.Marshal(view)
	if err != nil || len(encoded) > clientFrameLimit(q.Op, true)-4096 {
		r.Code = foundation.Code(equipment.Limit)
		return r
	}
	r.Data = &Data{ContractRevision: ProductRevision, ProductVersion: ProductVersion, Instance: s.identity.Instance,
		Profile: q.Profile, Activation: q.Activation, Authenticated: true, Config: &view}
	return r
}

func validConfigInput(op string, in cfgbackup.Input) bool {
	if op == "config.compare" || op == "config.download" {
		if !access.ValidCompanyID(in.VersionID) {
			return false
		}
	}
	if op == "config.download" && (!in.Confirmed || len(in.Path) == 0 || len(in.Path) > 4096) {
		return false
	}
	return true
}
