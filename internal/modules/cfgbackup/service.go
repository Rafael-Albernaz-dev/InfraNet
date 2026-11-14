package cfgbackup

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"sync"
	"time"
)

type Service struct {
	Auth    equipment.Authorizer
	Store   Store
	Vault   Vault
	Remote  Remote
	Files   FileWriter
	IDs     equipment.IDs
	Now     func() time.Time
	mu      sync.Mutex
	running map[string]bool
	slots   chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	closed  bool
}

func New(auth equipment.Authorizer, store Store, vault Vault, remote Remote, files FileWriter, ids equipment.IDs) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{Auth: auth, Store: store, Vault: vault, Remote: remote, Files: files, IDs: ids, Now: time.Now,
		running: map[string]bool{}, slots: make(chan struct{}, 4), ctx: ctx, cancel: cancel}
}
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.wg.Wait()
}
func (s *Service) authorized(ctx context.Context, q equipment.Command, fn func() equipment.Code) equipment.Code {
	if q.Scope.Profile == "" || q.Scope.Profile != q.Scope.ExpectedProfile || q.Scope.Activation != q.Scope.ExpectedActivation {
		return equipment.Changed
	}
	return s.Auth.WithAuthorization(ctx, q.UID, q.Scope, func(version uint64) equipment.Code {
		if q.Authority != 0 && version != q.Authority {
			return equipment.Changed
		}
		return fn()
	})
}
func (s *Service) Handle(ctx context.Context, q equipment.Command, in Input) (View, equipment.Code) {
	var out View
	if q.Scope.Profile == "" || q.Scope.Profile != q.Scope.ExpectedProfile || q.Scope.Activation != q.Scope.ExpectedActivation {
		return out, equipment.Changed
	}
	code := s.Auth.WithAuthorization(ctx, q.UID, q.Scope, func(version uint64) equipment.Code {
		q.Authority = version
		return s.handleAuthorized(ctx, q, in, &out)
	})
	return out, code
}
func (s *Service) handleAuthorized(ctx context.Context, q equipment.Command, in Input, out *View) equipment.Code {
	st, code := s.Store.ReadEquipment(ctx, q.Scope.Profile)
	if code != equipment.OK {
		return code
	}
	d, ok := findDevice(st, in.DeviceID)
	if !ok {
		return equipment.Invalid
	}
	out.Format = FormatFor(d)
	out.Supported = out.Format != ""
	switch q.Op {
	case "config.save":
		if code = s.startManual(q, d); code != equipment.OK {
			return code
		}
	case "config.compare":
		return s.compare(ctx, st, d, in.VersionID, out)
	case "config.download":
		return s.download(ctx, q, st, d, in)
	case "config.list":
	default:
		return equipment.Invalid
	}
	versions, code := s.Store.ConfigVersions(ctx, q.Scope.Profile, d.ID)
	if code != equipment.OK {
		return code
	}
	out.Versions = []Version{}
	for _, v := range versions {
		out.Versions = append(out.Versions, v.Version)
	}
	out.Status, code = s.Store.ConfigStatus(ctx, q.Scope.Profile, d.ID)
	s.mu.Lock()
	out.Busy = s.running[q.Scope.Profile+"|"+d.ID]
	s.mu.Unlock()
	return code
}
func findDevice(st equipment.State, id string) (equipment.Device, bool) {
	for _, d := range st.Devices {
		if d.ID == id {
			return d, true
		}
	}
	return equipment.Device{}, false
}
func credential(st equipment.State, d equipment.Device) (equipment.SealedCredential, equipment.Code) {
	if !d.Active {
		return equipment.SealedCredential{}, equipment.Inactive
	}
	if FormatFor(d) == "" {
		return equipment.SealedCredential{}, Unsupported
	}
	if d.Transport != equipment.TransportSSH || d.Fingerprint == "" {
		return equipment.SealedCredential{}, equipment.TrustRequired
	}
	for _, cr := range st.Credentials {
		if cr.Metadata.ID == d.CredentialID {
			return cr, equipment.OK
		}
	}
	return equipment.SealedCredential{}, equipment.MissingCredential
}
func (s *Service) claim(profile, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := profile + "|" + id
	if s.closed || s.running[key] {
		return false
	}
	select {
	case s.slots <- struct{}{}:
	default:
		return false
	}
	s.running[key] = true
	s.wg.Add(1)
	return true
}
func (s *Service) release(profile, id string) {
	s.mu.Lock()
	delete(s.running, profile+"|"+id)
	s.mu.Unlock()
	<-s.slots
	s.wg.Done()
}
func (s *Service) startManual(q equipment.Command, d equipment.Device) equipment.Code {
	st, code := s.Store.ReadEquipment(s.ctx, q.Scope.Profile)
	if code != equipment.OK {
		return code
	}
	// ignored: only the result code or validation outcome is needed here.
	if _, code = credential(st, d); code != equipment.OK {
		return code
	}
	if !s.claim(q.Scope.Profile, d.ID) {
		return equipment.Code("BUSY")
	}
	go func() {
		defer s.release(q.Scope.Profile, d.ID)
		guard := func(fn func() equipment.Code) equipment.Code { return s.authorized(s.ctx, q, fn) }
		s.capture(s.ctx, q.Scope.Profile, d, Manual, guard)
	}()
	return equipment.OK
}
