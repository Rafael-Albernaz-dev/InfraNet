package routerssh

import (
	"bufio"
	"context"
	"infradesktop/internal/adapters/routerssh/internal/routeros"
	"infradesktop/internal/modules/equipment"
	"strings"
)

// stream owns exactly one SSH channel. Cancellation closes it even when the
// remote peer stops producing output; no goroutine is left waiting on Scanner.
func (r *ppCommandRunner) stream(command string, consume func(string) equipment.Code) equipment.Code {
	if code := r.check(); code != equipment.OK {
		return code
	}
	session, code := openPPCommandSession(r.client)
	if code != equipment.OK {
		return code
	}
	defer session.Close()
	stop := context.AfterFunc(r.ctx, func() { session.Close() })
	defer stop()
	output, err := session.StdoutPipe()
	if err != nil {
		return equipment.RadioCodeSSHConnectionLost
	}
	stderr := &limitedOutput{budget: r.budget}
	session.Stderr = stderr
	r.budget.mu.Lock()
	(*r.commands)++
	r.budget.mu.Unlock()
	if err = session.Start(command); err != nil {
		return equipment.RadioCodeSSHConnectionLost
	}
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 1024), 1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if code := r.checkStreamLine(line); code != equipment.OK {
			return code
		}
		if code := consume(line); code != equipment.OK {
			return code
		}
	}
	if scanner.Err() != nil {
		return equipment.Partial
	}
	err = session.Wait()
	if r.budget.overflowed() {
		return equipment.Limit
	}
	if permissionDiagnostic(stderr.buf.String()) {
		return equipment.RadioCodeRemotePermissionDenied
	}
	if routeros.CollectorSyntaxDiagnostic(stderr.buf.String()) {
		return equipment.CollectorDefect
	}
	if err != nil || stderr.buf.Len() != 0 {
		return equipment.RadioCodeSSHConnectionLost
	}
	return equipment.OK
}

func (r *ppCommandRunner) checkStreamLine(line string) equipment.Code {
	measured := &limitedOutput{budget: r.budget}
	// ignored: only the bounded writer error is needed; no transcript copy retained.
	if _, err := measured.Write([]byte(line + "\n")); err != nil {
		return equipment.Limit
	}
	if permissionDiagnostic(line) {
		return equipment.RadioCodeRemotePermissionDenied
	}
	if routeros.CollectorSyntaxDiagnostic(line) {
		return equipment.CollectorDefect
	}
	return equipment.OK
}
