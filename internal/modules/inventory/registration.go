package inventory

import (
	"context"
	"time"
)

func (s *Service) startRegistration(ctx context.Context, q Command, v *View) Code {
	if code := canonicalizeMutation(&q); code != OK {
		return code
	}
	if code := ValidateDevice(Device{ID: q.ID, Name: q.Address, Address: q.Address, Port: q.Port, Transport: q.Transport,
		Role: q.Role, CredentialID: q.CredentialID, Revision: 1}); code != OK {
		return code
	}
	if replay, code := s.registrationReplay(q, v); replay {
		return code
	}
	if code := s.prepareRegistration(ctx, &q); code != OK {
		return code
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.PruneLocked()
	if s.Closed {
		return Unavailable
	}
	for _, job := range s.Jobs {
		if job.Command.Op == q.Op && SameScope(job.Command, q) && job.Command.Operation == q.Operation {
			if buildMutationStamp(job.Command, "") != buildMutationStamp(q, "") {
				return Revision
			}
			v.Job = CopyJob(job)
			return OK
		}
	}
	if len(s.Jobs) >= MaxJobs {
		return Limit
	}
	id, code := s.IDs.ID()
	if code != OK {
		return code
	}
	jobctx, cancel := context.WithCancel(context.Background())
	job := &RunningJob{Command: q, Device: Device{ID: q.ID, Address: q.Address, Port: q.Port}, Cancel: cancel,
		View: Job{ID: id, DeviceID: q.ID, Kind: q.Op, State: "RUNNING", Code: OK}}
	s.Jobs[id] = job
	s.WG.Add(1)
	go s.runRegistration(jobctx, job)
	v.Job = CopyJob(job)
	return OK
}

func (s *Service) runRegistration(ctx context.Context, job *RunningJob) {
	defer s.WG.Done()
	defer job.Cancel()
	view := View{}
	code := s.register(ctx, job.Command, &view)
	if view.ContactCode != "" {
		code = view.ContactCode
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.finishQueryJobLocked(job, nil, "", code)
	if view.Device != nil {
		job.View.Result = view.Device.Result
	}
	job.Released = true
}

func (s *Service) registrationStatus(ctx context.Context, q Command, v *View) Code {
	s.Mu.Lock()
	job := s.Jobs[q.JobID]
	if job == nil || !SameScope(job.Command, q) || job.Command.Op != "inventory.register" {
		s.Mu.Unlock()
		return Changed
	}
	v.Job = CopyJob(job)
	s.Mu.Unlock()
	if v.Job.State == "SUCCEEDED" {
		state, code := s.Store.ReadEquipment(ctx, q.Scope.Profile)
		if code != OK {
			return code
		}
		projectDeviceNames(&state)
		v.Device = DeviceByID(&state, v.Job.DeviceID)
	}
	return OK
}

func (s *Service) confirmRegistration(ctx context.Context, q Command, v *View) Code {
	s.Mu.Lock()
	job := s.Jobs[q.JobID]
	if job == nil || !SameScope(job.Command, q) || job.Command.Op != "inventory.register" ||
		job.View.State != "FAILED" || s.Now().Sub(job.Finished) > 10*time.Minute {
		s.Mu.Unlock()
		return Changed
	}
	original, outcome := job.Command, job.View.Code
	s.Mu.Unlock()
	original.Op, original.ContactCode = "inventory.create", outcome
	// The failed probe is never retried by confirmation.
	return s.mutate(ctx, original, v)
}

func (s *Service) prepareRegistration(ctx context.Context, q *Command) Code {
	state, code := s.Store.ReadEquipment(ctx, q.Scope.Profile)
	if code != OK {
		return code
	}
	if !groupExists(state.Groups, q.GroupID) {
		return Invalid
	}
	if q.User != "" || q.Secret != "" {
		if code := s.prepareInlineContact(&state, q); code != OK {
			return code
		}
	}
	if q.CredentialID == "" {
		q.CredentialID = state.CredentialDefaults[string(q.Role)]
	}
	credential := CredentialByID(&state, q.CredentialID)
	if q.PendingCredential != nil {
		credential = q.PendingCredential
	}
	if credential == nil {
		return MissingCredential
	}
	q.ContactCredentialRevision = credential.Metadata.Revision
	return OK
}

func (s *Service) registrationReceipt(q Command, v *View) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	for _, job := range s.Jobs {
		if job.Command.Op == "inventory.register" && job.Command.ID == q.ID && job.Command.Operation == q.Operation && SameScope(job.Command, q) {
			v.OperationState = job.View.State
			v.Job = CopyJob(job)
			return
		}
	}
}
