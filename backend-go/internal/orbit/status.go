package orbit

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

const (
	WorkStatusWaitingToRun    = "waiting_to_run"
	WorkStatusRunning         = "running"
	WorkStatusRanToCompletion = "ran_to_completion"
	WorkStatusFaulted         = "faulted"
	WorkStatusCanceled        = "canceled"

	ApplicationStatusDeployed     = "deployed"
	ApplicationStatusDeploying    = "deploying"
	ApplicationStatusUndeployed   = "undeployed"
	ApplicationStatusDeployFailed = "deploy_failed"
)

func NewId() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return strings.ToLower(encoded)[:26]
}
