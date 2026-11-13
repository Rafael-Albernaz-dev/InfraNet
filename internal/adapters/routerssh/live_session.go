package routerssh

import (
	"context"
	"errors"
	"golang.org/x/crypto/ssh"
	"infradesktop/internal/adapters/routerssh/deviceparse"
	"infradesktop/internal/adapters/routerssh/pppoeparse"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pulse"
	"net"
	"strconv"
	"strings"
	"time"
)

type liveSession struct {
	client *ssh.Client
	conn   net.Conn
}

// OpenLive owns a separate connection with a handshake deadline, never a cumulative read budget.
func OpenLive(ctx context.Context, d equipment.Device, user, secret string) (pulse.ReadSession, pulse.ReadFault) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	endpoint := net.JoinHostPort(d.Address, strconv.Itoa(d.Port))
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, pulse.ReadLost
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	// ignored: WithTimeout always establishes a deadline.
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, pulse.ReadLost
	}
	client, fault := liveHandshake(conn, d, user, secret)
	secret = ""
	if fault != pulse.ReadOK {
		conn.Close()
		return nil, fault
	}
	if !stop() || ctx.Err() != nil {
		client.Close()
		return nil, pulse.ReadLost
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		client.Close()
		return nil, pulse.ReadLost
	}
	return &liveSession{client: client, conn: conn}, pulse.ReadOK
}
func liveHandshake(conn net.Conn, d equipment.Device, user, secret string) (*ssh.Client, pulse.ReadFault) {
	cfg := config(user)
	mismatch := false
	cfg.HostKeyCallback = func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if d.Fingerprint == "" || ssh.FingerprintSHA256(key) != d.Fingerprint {
			mismatch = true
			return errors.New("host key mismatch")
		}
		return nil
	}
	cfg.Auth = []ssh.AuthMethod{ssh.PasswordCallback(func() (string, error) { return secret, nil })}
	endpoint := net.JoinHostPort(d.Address, strconv.Itoa(d.Port))
	cc, channels, requests, err := ssh.NewClientConn(conn, endpoint, cfg)
	secret = ""
	cfg.Auth = nil
	if err != nil {
		return nil, liveConnectFault(err, mismatch)
	}
	return ssh.NewClient(cc, channels, requests), pulse.ReadOK
}
func liveConnectFault(err error, mismatch bool) pulse.ReadFault {
	if mismatch {
		return pulse.ReadKeyChanged
	}
	if strings.Contains(err.Error(), "unable to authenticate, attempted methods") {
		return pulse.ReadRefused
	}
	return pulse.ReadLost
}
func (s *liveSession) Close() { s.client.Close() }
func (s *liveSession) Read(ctx context.Context, t pulse.ReadTarget) (pulse.Reading, pulse.ReadFault) {
	command := deviceparse.LiveConcentratorScript
	switch t.Role {
	case equipment.RoleConcentrator:
	case equipment.RoleAP:
		command = deviceparse.LiveAPScript
	default:
		return pulse.Reading{}, pulse.ReadLost
	}
	started := time.Now()
	output, fault := s.runLive(ctx, command, 4096)
	if fault != pulse.ReadOK {
		return pulse.Reading{}, fault
	}
	reading, fault := deviceparse.ParseLiveReading(output)
	if fault != pulse.ReadOK {
		return reading, fault
	}
	reading.Started, reading.At = started, time.Now()
	if t.Role == equipment.RoleConcentrator {
		reading.Observation, reading.Tunnels = s.readSessions(ctx)
		reading.At = time.Now()
	}
	return reading, pulse.ReadOK
}

// readSessions never discards the device reading of the same cycle: a failed list
// returns no observation (the caller records a collection gap) and failed counters
// return no tunnels (averages wait for the next cycle).
func (s *liveSession) readSessions(ctx context.Context) (*equipment.PPObservation, []equipment.PPTunnel) {
	output, fault := s.runLive(ctx, pppoeparse.PPPoECommand, equipment.PPWireBytes)
	if fault != pulse.ReadOK {
		return nil, nil
	}
	observation, code := pppoeparse.ParsePPPoE(output)
	if code != equipment.OK || observation.Quality.Status != "complete" {
		return nil, nil
	}
	output, fault = s.runLive(ctx, pppoeparse.PulseCountersCommand, equipment.PPWireBytes)
	if fault != pulse.ReadOK {
		return observation, nil
	}
	tunnels, code := pppoeparse.ParsePulseCounters(output)
	if code != equipment.OK {
		return observation, nil
	}
	return observation, tunnels
}
func (s *liveSession) runLive(ctx context.Context, command string, limit int) (string, pulse.ReadFault) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// ignored: WithTimeout always establishes a deadline.
	deadline, _ := ctx.Deadline()
	if err := s.conn.SetDeadline(deadline); err != nil {
		return "", pulse.ReadLost
	}
	stop := context.AfterFunc(ctx, func() { s.conn.Close() })
	defer stop()
	session, err := s.client.NewSession()
	if err != nil {
		return "", pulse.ReadLost
	}
	defer session.Close()
	budget := &outputBudget{conn: s.conn, limit: limit}
	stdout, stderr := &limitedOutput{budget: budget}, &limitedOutput{budget: budget}
	session.Stdout, session.Stderr = stdout, stderr
	err = session.Run(command)
	if err != nil || ctx.Err() != nil || budget.overflow || stderr.buf.Len() != 0 {
		return "", pulse.ReadLost
	}
	if !stop() {
		return "", pulse.ReadLost
	}
	if err := s.conn.SetDeadline(time.Time{}); err != nil {
		return "", pulse.ReadLost
	}
	return stdout.buf.String(), pulse.ReadOK
}
