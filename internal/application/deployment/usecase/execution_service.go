package deploymentsvc

import (
	"log/slog"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type ExecutionService struct {
	executionStore    deploymentport.ExecutionStore
	logger            *slog.Logger
	targetResolver    environmentport.TargetResolver
	runtime           deploymentport.ExecutionWorkspace
	observer          deploymentport.ContainerObserver
	lifecycle         deploymentport.ServiceLifecycle
	logStore          deploymentport.ExecutionLogWriter
	pollInterval      time.Duration
	executionTimeout  time.Duration
	cancelTimeout     time.Duration
	transactionRunner deploymentport.TransactionRunner
	gatewayReadiness  deploymentport.GatewayReadinessChecker
	mountInventory    deploymentport.MountInventory
}

// NewExecutionService constructs the worker's execution and cleanup owner.
func NewExecutionService(store deploymentport.ExecutionStore, logger *slog.Logger, targetResolver environmentport.TargetResolver, runtime deploymentport.ExecutionWorkspace, logs deploymentport.ExecutionLogWriter, pollInterval, executionTimeout, cancelTimeout time.Duration, transaction deploymentport.TransactionRunner, readiness deploymentport.GatewayReadinessChecker, mounts deploymentport.MountInventory, observer deploymentport.ContainerObserver, lifecycle deploymentport.ServiceLifecycle) ExecutionService {
	return ExecutionService{executionStore: store, logger: logger, targetResolver: targetResolver, runtime: runtime, logStore: logs, pollInterval: pollInterval, executionTimeout: executionTimeout, cancelTimeout: cancelTimeout, transactionRunner: transaction, gatewayReadiness: readiness, mountInventory: mounts, observer: observer, lifecycle: lifecycle}
}
