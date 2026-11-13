package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/testsupport/sshfixture"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type equipmentIDs struct{}

func (equipmentIDs) ID() (string, equipment.Code) { return uuid.NewString(), equipment.OK }
func newEquipment(t *testing.T, j *journey) *equipment.Service {
	t.Helper()
	s := equipment.New(j.svc, j.st, vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil }), routerssh.Adapter{}, equipmentIDs{}, time.Now)
	t.Cleanup(s.Close)
	return s
}
func equipmentScope(j *journey) equipment.Command {
	q := j.q
	q.ExpectedProfile = q.Profile
	q.ExpectedActivation = q.Activation
	return equipment.Command{Scope: q, UID: 1000}
}
func equipmentCall(t *testing.T, s *equipment.Service, q equipment.Command, want equipment.Code) equipment.View {
	t.Helper()
	v, c := s.Handle(context.Background(), q)
	if c != want {
		t.Fatalf("%s got %s want %s", q.Op, c, want)
	}
	return v
}

// setupEquipment moved to equipment_vault_deadline_test.go.

// awaitJob moved to equipment_vault_deadline_test.go.

// trustEquipment moved to equipment_vault_deadline_test.go.

// TestEquipmentQueriesRevocationSaturationAndRestart moved to equipment_job_replay_test.go.

// TestEquipmentConcurrentOptimisticUpdateAndMissingVault moved to equipment_vault_deadline_test.go.

func TestEquipmentCoherentRestoreAndSchemaThree(t *testing.T) {
	j := newJourney(t)
	j.create(companyX, "X", access.OK)
	j.selectCompany(companyX, access.OK)
	s := newEquipment(t, j)
	setupEquipment(t, j, s, 22)
	st, _ := j.st.ReadEquipment(context.Background(), companyX)
	installation := st.Installation
	assertSchemaThreeRollbackAndUpgrade(t, j)
	credential, _ := setupEquipment(t, j, s, 22)
	s.Close()
	source := j.st.dir
	j.st.Close()
	// Copy only after clean close: no omitted active WAL is presented as backup.
	restored := privateDir(t)
	if e := copyUnitWithoutLock(source, restored); e != nil {
		t.Fatal(e)
	}
	restoredStore, e := Open(restored)
	if e != nil {
		t.Fatal(e)
	}
	defer restoredStore.Close()
	st, c := restoredStore.ReadEquipment(context.Background(), companyX)
	if c != equipment.OK || st.Installation != installation || st.Company != companyX {
		t.Fatal("restored identities differ")
	}
	v := vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{37}, 32), nil })
	plain, c := v.Open(st.Installation, st.Company, credential.ID, 1, st.Credentials[0].Envelope, st.WrappedKey)
	if c != equipment.OK || plain != sshfixture.Password {
		t.Fatal("coherent set failed to restore")
	}
	missing := vault.New(func() ([]byte, error) { return nil, fmt.Errorf("missing") })
	if missing.Ready(st.Installation, st.Company, st.WrappedKey) {
		t.Fatal("missing key reported ready")
	}
	wrong := vault.New(func() ([]byte, error) { return bytes.Repeat([]byte{38}, 32), nil })
	if wrong.Ready(st.Installation, st.Company, st.WrappedKey) {
		t.Fatal("wrong master reported ready")
	}
}

// assertSchemaThreeRollbackAndUpgrade simulates an exact V2 unit, then exercises
// the additive V3 rollback and upgrade.
func assertSchemaThreeRollbackAndUpgrade(t *testing.T, j *journey) {
	t.Helper()
	dropPPPoEFixtureSchema(t, j)
	stmt := "DROP TABLE devices;DROP TABLE credentials;DROP TABLE vault_key;DROP TABLE equipment_operations;" +
		"DELETE FROM schema_migrations WHERE version>=3;CREATE TABLE devices(conflict TEXT)"
	if _, e := j.st.unit.Exec(stmt); e != nil {
		t.Fatal(e)
	}
	if j.st.validateUnit(j.st.unit, companyX) == nil {
		t.Fatal("conflicting schema accepted")
	}
	var count int
	j.st.unit.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&count)
	if count != 2 {
		t.Fatal("failed migration advanced ledger")
	}
	j.st.unit.Exec("DROP TABLE devices")
	if e := j.st.validateUnit(j.st.unit, companyX); e != nil {
		t.Fatal(e)
	}
}

// copyUnitWithoutLock copies a closed unit directory, skipping the process lock.
func copyUnitWithoutLock(source, restored string) error {
	return filepath.WalkDir(source, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		relative, _ := filepath.Rel(source, path)
		if relative == "." {
			return nil
		}
		target := filepath.Join(restored, relative)
		if d.IsDir() {
			return os.Mkdir(target, 0700)
		}
		if filepath.Base(path) == "database.lock" {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		return os.WriteFile(target, b, 0600)
	})
}

// TestEquipmentInspectNonblockingAndRealTotalDeadline moved to equipment_vault_deadline_test.go.

// TestEquipmentJobRetentionAndRecordLimit moved to equipment_job_replay_test.go.

// TestFocalMissingCompanyKeyPreservesInventory moved to equipment_vault_deadline_test.go.

func TestFocalWriteReplayPreservesJobs(t *testing.T) {
	for _, kind := range []string{"device", "credential"} {
		t.Run(kind, func(t *testing.T) {
			j := newJourney(t)
			j.create(companyX, "X", access.OK)
			j.selectCompany(companyX, access.OK)
			s := newEquipment(t, j)
			hold := make(chan struct{})
			peer, e := sshfixture.Start(sshfixture.Config{Family: "v7", Hold: hold})
			if e != nil {
				t.Fatal(e)
			}
			defer peer.Close()
			defer func() {
				select {
				case <-hold:
				default:
					close(hold)
				}
			}()
			cr, d := setupEquipment(t, j, s, peer.Port())
			q := trustEquipment(t, s, equipmentScope(j), d)
			replay := d
			if kind == "credential" {
				replay = cr
			}
			var second equipment.Command
			if kind == "credential" {
				secondPeer, e := sshfixture.Start(sshfixture.Config{Family: "v7", Hold: hold})
				if e != nil {
					t.Fatal(e)
				}
				defer secondPeer.Close()
				d2 := d
				d2.Port = secondPeer.Port()
				d2.ID = uuid.NewString()
				d2.Operation = uuid.NewString()
				equipmentCall(t, s, d2, equipment.OK)
				second = trustEquipment(t, s, equipmentScope(j), d2)
			}
			v := equipmentCall(t, s, q, equipment.OK)
			var secondID string
			if kind == "credential" {
				secondID = equipmentCall(t, s, second, equipment.OK).Job.ID
			}
			equipmentCall(t, s, replay, equipment.OK)
			if secondID != "" {
				x := second
				x.Op = "device.query-status"
				x.JobID = secondID
				if equipmentCall(t, s, x, equipment.OK).Job.State != "RUNNING" {
					t.Fatal("shared replay invalidated other device")
				}
			}
			status := q
			status.Op = "device.query-status"
			status.JobID = v.Job.ID
			if job := equipmentCall(t, s, status, equipment.OK).Job; job.State != "RUNNING" {
				t.Fatal("replay invalidated running query", job)
			}
			close(hold)
			job := awaitJob(t, s, q, v.Job.ID)
			if job.Code != equipment.OK {
				t.Fatal(job.Code)
			}
			if secondID != "" {
				if awaitJob(t, s, second, secondID).Code != equipment.OK {
					t.Fatal("shared query failed")
				}
			}
			equipmentCall(t, s, replay, equipment.OK)
			if secondID != "" {
				x := second
				x.Op = "device.query-status"
				x.JobID = secondID
				if got := equipmentCall(t, s, x, equipment.OK).Job; got.State != "SUCCEEDED" || got.Result == nil {
					t.Fatal("shared replay erased completed result")
				}
			}
			if got := equipmentCall(t, s, status, equipment.OK).Job; got.State != "SUCCEEDED" || got.Result == nil {
				t.Fatal("replay invalidated completed result", got)
			}
		})
	}
}
