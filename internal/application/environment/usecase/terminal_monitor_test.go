package environmentsvc

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestTerminalMonitorRechecksTargetAndIdleWithoutInput(t *testing.T) {
	for _, scenario := range []string{"target changed", "idle", "activity extends idle", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				svc, store, _, _ := newTestTerminalService(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				ticket, err := svc.IssueTicket(ctx, "actor", store.environment.ProjectId)
				if err != nil {
					t.Fatal(err)
				}
				grant, release, err := svc.RedeemTicket(ctx, store.environment.ProjectId, ticket)
				if err != nil {
					t.Fatal(err)
				}
				defer release()
				lastActivity := time.Now()
				done := make(chan string, 1)
				go func() { done <- svc.MonitorSession(ctx, grant, func() time.Time { return lastActivity }) }()
				synctest.Wait()
				assertActive := func() {
					t.Helper()
					synctest.Wait()
					select {
					case reason := <-done:
						t.Fatalf("monitor ended early: %q", reason)
					default:
					}
				}
				want := ""
				switch scenario {
				case "target changed":
					store.environment.TargetRevision++
					time.Sleep(terminalCheckInterval)
					want = "access_or_target_changed"
				case "idle", "activity extends idle":
					time.Sleep(terminalIdleLimit - terminalCheckInterval)
					assertActive()
					if scenario == "activity extends idle" {
						lastActivity = time.Now()
					}
					time.Sleep(terminalCheckInterval)
					if scenario == "activity extends idle" {
						assertActive()
						cancel()
					} else {
						want = "idle_timeout"
					}
				case "canceled":
					cancel()
				}
				synctest.Wait()
				select {
				case got := <-done:
					if got != want {
						t.Fatalf("reason=%q want=%q", got, want)
					}
				default:
					t.Fatal("monitor did not finish")
				}
			})
		})
	}
}
