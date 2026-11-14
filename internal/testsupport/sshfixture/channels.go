package sshfixture

import "golang.org/x/crypto/ssh"

func (s *Server) handleChannels(channels <-chan ssh.NewChannel) {
	for channel := range channels {
		if channel.ChannelType() != "session" {
			channel.Reject(ssh.Prohibited, "only session")
			continue
		}
		ch, reqs, err := channel.Accept()
		if err != nil {
			continue
		}
		s.handleSession(ch, reqs)
	}
}
func (s *Server) handleSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	for req := range reqs {
		if req.Type != "exec" {
			req.Reply(false, nil)
			continue
		}
		var exec struct{ Command string }
		if ssh.Unmarshal(req.Payload, &exec) != nil {
			req.Reply(false, nil)
			return
		}
		s.Commands.Add(1)
		req.Reply(true, nil)
		if s.config.Stream != nil && s.config.Stream(exec.Command, ch) {
			return
		}
		if s.config.Hold != nil {
			select {
			case <-s.config.Hold:
			case <-s.closed:
				return
			}
		}
		if s.config.DisconnectCommand != nil && s.config.DisconnectCommand(exec.Command) {
			return
		}
		s.writeCommandResult(ch, exec.Command)
		return
	}
}
func (s *Server) writeCommandResult(ch ssh.Channel, command string) {
	out, ok := s.output(command)
	status := s.config.ExitStatus
	if !ok {
		status = 1
	}
	// ignored: the cancellation under test may have closed the synthetic peer.
	_, err := ch.Stderr().Write([]byte(s.config.Stderr))
	if err != nil {
		return
	}
	// ignored: the cancellation under test may have closed the synthetic peer.
	_, err = ch.Write([]byte(out))
	if err != nil {
		return
	}
	// ignored: no receiver remains after cancellation; session cleanup still runs.
	_, err = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
	if err != nil {
		return
	}
}
