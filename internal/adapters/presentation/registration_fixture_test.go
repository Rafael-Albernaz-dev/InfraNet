package presentation

import (
	"testing"
	"time"
)

func finishRegistration(t *testing.T, app *App, profile, activation string, started EquipmentResult, confirmFailure bool) EquipmentResult {
	t.Helper()
	if started.Code != "OK" || started.Data == nil || started.Data.Job == nil {
		t.Fatal("registration admission", started.Code)
	}
	job := started.Data.Job
	deadline := time.Now().Add(3 * time.Second)
	var status EquipmentResult
	for time.Now().Before(deadline) {
		status = app.RegistrationStatus(profile, activation, job.ID)
		if status.Code != "OK" || status.Data == nil || status.Data.Job == nil {
			t.Fatal("registration status", status.Code)
		}
		if status.Data.Job.State != "RUNNING" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if status.Data.Job.State == "FAILED" && confirmFailure {
		return app.ConfirmRegistration(profile, activation, job.ID)
	}
	if status.Data.Job.State != "SUCCEEDED" || status.Data.Device == nil {
		t.Fatal("registration did not finish", status.Data.Job)
	}
	return status
}

func registerFixtureCompany(t *testing.T, app *App) Result {
	t.Helper()
	const pw = "isolated modal account password"
	if app.Register("technical", pw).Code != "OK" || app.AckRecovery().Code != "OK" || app.Login("technical", pw, false).Code != "OK" {
		t.Fatal("account setup")
	}
	c := app.CreateCompany("Modal synthetic")
	if c.Code != "OK" {
		t.Fatal(c.Code)
	}
	selected := app.SelectCompany(c.Companies[0].ID)
	if selected.Code != "OK" {
		t.Fatal(selected.Code)
	}
	return selected
}
