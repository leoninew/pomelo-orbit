package environmentsvc

import (
	"context"
	"time"
)

// MonitorSession checks access and idle policy until a session must end or its
// owner cancels ctx. lastActivity must be safe to read while I/O updates it.
// The caller owns the connection and closes it after receiving a reason.
func (s *TerminalService) MonitorSession(ctx context.Context, grant TerminalGrant, lastActivity func() time.Time) string {
	ticker := time.NewTicker(terminalCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ""
		case <-ticker.C:
			if ctx.Err() != nil {
				return ""
			}
			if time.Since(lastActivity()) >= terminalIdleLimit {
				return "idle_timeout"
			}
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := s.ValidateSession(checkCtx, grant)
			cancel()
			if ctx.Err() != nil {
				return ""
			}
			if err != nil {
				return "access_or_target_changed"
			}
		}
	}
}
