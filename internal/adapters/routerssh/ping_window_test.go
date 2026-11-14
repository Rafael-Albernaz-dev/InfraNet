package routerssh_test

import (
	"context"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/adapters/routerssh/pppoeparse"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/pppoe"
	"infradesktop/internal/testsupport/sshfixture"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPingWindowStreamsUntilCancelClosesSSHChannel(t *testing.T) {
	closed := make(chan struct{})
	published := make(chan string, 2)
	peer, err := sshfixture.Start(sshfixture.Config{Stream: func(command string, ch io.ReadWriteCloser) bool {
		if !strings.Contains(command, "/ping address=198.51.100.1") {
			return false
		}
		if strings.Contains(command, "count=") {
			t.Error("finite count in window command")
		}
		if _, err := io.WriteString(ch, "SEQ HOST SIZE TTL TIME STATUS\n0 198.51.100.1 56 64 3ms\n"); err != nil {
			t.Error(err)
		}
		_, err := io.Copy(io.Discard, ch)
		if err != nil {
			t.Error(err)
		}
		close(closed)
		return true
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan equipment.Code, 1)
	go func() {
		_, code := (routerssh.Adapter{}).ReadPP(ctx, equipment.PPReadRequest{
			Device: equipment.Device{Address: "127.0.0.1", Port: peer.Port(), Fingerprint: peer.Fingerprint()},
			User:   "reader", Secret: sshfixture.Password, Scope: equipment.PingWindowScope,
			Target: &equipment.PPReadTarget{Ping: pingTarget("v7")}, Check: func() equipment.Code { return equipment.OK },
			Publish: func(result *equipment.PPReadResult) equipment.Code {
				published <- result.PingSummary
				return equipment.OK
			},
		})
		done <- code
	}()
	select {
	case summary := <-published:
		if !strings.Contains(summary, "sent=1 received=1 packet-loss=0.0%") {
			t.Fatal(summary)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no incremental publication while channel open")
	}
	cancel()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("remote channel still running")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("adapter did not return")
	}
	if peer.Auth.Load() != 1 || peer.Commands.Load() != 2 {
		t.Fatal("expected one SSH and prelude plus ping", peer.Auth.Load(), peer.Commands.Load())
	}
}

func TestPingWindowTimeoutGoldenRetainsSessionLossAndSummary(t *testing.T) {
	body, err := os.ReadFile("testdata/ping-window-timeout.txt")
	if err != nil {
		t.Fatal(err)
	}
	window := pppoe.PingWindow{Started: time.Now()}
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n")[1:5] {
		probe, ok := pppoeparse.ParsePingWindowLine(line, "198.51.100.1")
		if !ok || window.Add(line, probe) != equipment.OK {
			t.Fatal("rejected synthetic RouterOS line", line)
		}
	}
	result := window.Result(time.Now())
	if len(result.Rows) != 4 || !strings.HasSuffix(result.Rows[3].Value, "timeout") ||
		result.PingSummary != "sent=4 received=2 packet-loss=50.0% min-rtt=3ms avg-rtt=5ms max-rtt=7ms" {
		t.Fatal(result)
	}
}

func TestPingWindowRefusesUnsafeContextBeforeStartingPing(t *testing.T) {
	for _, counts := range []string{"1\t0\t0", "0\t1\t0", "0\t0\t1"} {
		peer, err := sshfixture.Start(sshfixture.Config{Command: func(command string) (string, bool) {
			if command == pppoeparse.PDPingContextV7 {
				return "C\tv7\t" + counts + "\n", true
			}
			t.Error("unsafe context reached ping")
			return "", false
		}})
		if err != nil {
			t.Fatal(err)
		}
		_, code := (routerssh.Adapter{}).ReadPP(context.Background(), equipment.PPReadRequest{
			Device: equipment.Device{Address: "127.0.0.1", Port: peer.Port(), Fingerprint: peer.Fingerprint()},
			User:   "reader", Secret: sshfixture.Password, Scope: equipment.PingWindowScope,
			Target: &equipment.PPReadTarget{Ping: pingTarget("v7")}, Check: func() equipment.Code { return equipment.OK },
		})
		peer.Close()
		if code != "PING_CONTEXT_UNAVAILABLE" || peer.Commands.Load() != 1 {
			t.Fatal(code, peer.Commands.Load())
		}
	}
}
