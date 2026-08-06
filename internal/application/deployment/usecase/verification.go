package deploymentsvc

import (
	"context"
	"fmt"
	"strings"
	"time"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gopkg.in/yaml.v3"
)

// VerifyDeployment compares the persisted deployment intent, rendered Compose
// configuration, and managed runtime evidence. It never calls a tool handler
// or executes an unbounded Docker operation.
func (s Service) VerifyDeployment(ctx context.Context, userId, applicationId, deploymentId string, input deploymentdto.DeploymentVerificationInput) (deploymentdto.DeploymentVerificationResult, error) {
	deployment, err := s.DeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}
	evidence := map[string]any{"deployment": deployment}
	if deployment.ApplicationId == nil || *deployment.ApplicationId != applicationId {
		return verificationResult("inconclusive", []string{"deployment does not belong to application"}, evidence), nil
	}
	if deployment.Status == status.WorkStatusFaulted || deployment.Status == status.WorkStatusCanceled {
		return verificationResult("failed", []string{"deployment reached a failed terminal state"}, evidence), nil
	}
	if deployment.Status != status.WorkStatusRanToCompletion {
		return verificationResult("inconclusive", []string{"deployment is not complete"}, evidence), nil
	}
	if deployment.OperationType != "deploy" && deployment.OperationType != "restart" {
		return verificationResult("inconclusive", []string{"stability verification only applies to deploy or restart"}, evidence), nil
	}
	if deployment.ServiceId == nil || deployment.VersionId == nil || *deployment.ServiceId == "" || *deployment.VersionId == "" {
		return verificationResult("inconclusive", []string{"deployment lacks service or version linkage"}, evidence), nil
	}

	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}
	service, err := s.service.Service(ctx, *deployment.ServiceId)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment service", err)
	}
	if service.ApplicationId != app.Id {
		return verificationResult("inconclusive", []string{"deployment service does not belong to application"}, evidence), nil
	}
	target, err := s.ResolveRuntimeTarget(ctx, userId, app.Id, service.InstanceKey, false)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}
	preview, err := s.PreviewService(ctx, userId, service.Id)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}
	config, err := s.RuntimeComposeConfig(ctx, target)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}
	ps, err := s.RuntimeComposePS(ctx, target)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}
	inspections, err := s.runtimeInspections(ctx, target, ps.Containers)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}
	stability, err := s.observeRuntimeStability(ctx, target, input)
	if err != nil {
		return deploymentdto.DeploymentVerificationResult{}, err
	}

	evidence["application"] = app
	evidence["service"] = service
	evidence["target"] = target
	evidence["preview_compose"] = preview
	evidence["compose_config"] = config.Text
	evidence["containers"] = ps.Containers
	evidence["inspections"] = inspections
	evidence["stability"] = stability
	if state, _ := stability["state"].(string); state == "inconclusive" || state == "failed" {
		issues, _ := stability["issues"].([]string)
		conclusion := "inconclusive"
		if state == "failed" {
			conclusion = "failed"
		}
		return verificationResult(conclusion, issues, evidence), nil
	}
	differences := compareRuntimeCompose(preview, config.Text, ps.Containers, inspections)
	conclusion := "consistent"
	if len(differences) > 0 {
		conclusion = "drift"
	}
	return verificationResult(conclusion, differences, evidence), nil
}

func verificationResult(conclusion string, differences []string, evidence map[string]any) deploymentdto.DeploymentVerificationResult {
	return deploymentdto.DeploymentVerificationResult{Conclusion: conclusion, Differences: differences, Evidence: evidence, Summary: verificationSummary(evidence, differences)}
}

func (s Service) runtimeInspections(ctx context.Context, target deploymentdto.RuntimeTarget, containers []deploymentdto.RuntimeContainer) (map[string]any, error) {
	items := make(map[string]any, len(containers))
	for _, container := range containers {
		if container.Id == "" {
			return nil, apperror.New(apperror.KindInternal, "docker compose ps result did not contain a container id")
		}
		inspect, err := s.RuntimeContainerInspect(ctx, target, container.Id)
		if err != nil {
			return nil, err
		}
		items[container.Id] = inspect.Data
	}
	return items, nil
}

func (s Service) observeRuntimeStability(ctx context.Context, target deploymentdto.RuntimeTarget, input deploymentdto.DeploymentVerificationInput) (map[string]any, error) {
	window := input.StabilityWindow
	if window < 0 {
		return nil, apperror.New(apperror.KindValidation, "stability_window must not be negative")
	}
	poll := input.StabilityPoll
	if poll <= 0 {
		poll = 2 * time.Second
	}
	deadline := time.Now().Add(window)
	baseline := map[string]int{}
	samples := make([]map[string]any, 0, 1)
	for {
		ps, err := s.RuntimeComposePS(ctx, target)
		if err != nil {
			return nil, err
		}
		if len(ps.Containers) == 0 {
			return map[string]any{"state": "failed", "issues": []string{"no containers are running for the managed Compose project"}, "samples": samples}, nil
		}
		inspections, err := s.runtimeInspections(ctx, target, ps.Containers)
		if err != nil {
			return nil, err
		}
		issues := make([]string, 0)
		current := make([]map[string]any, 0, len(ps.Containers))
		for _, container := range ps.Containers {
			state, health, restarts := inspectRuntimeState(inspections[container.Id], container)
			current = append(current, map[string]any{"container_id": container.Id, "status": state, "health": health, "restart_count": restarts})
			if state != "running" || health == "unhealthy" || strings.Contains(strings.ToLower(container.Status), "unhealthy") {
				issues = append(issues, fmt.Sprintf("container %s is not healthy and running", container.Id))
			}
			if initial, ok := baseline[container.Id]; ok && restarts > initial {
				issues = append(issues, fmt.Sprintf("container %s restart count increased from %d to %d", container.Id, initial, restarts))
			} else {
				baseline[container.Id] = restarts
			}
		}
		samples = append(samples, map[string]any{"containers": current})
		if len(issues) > 0 {
			return map[string]any{"state": "failed", "issues": issues, "samples": samples}, nil
		}
		if !time.Now().Before(deadline) {
			return map[string]any{"state": "stable", "issues": []string{}, "samples": samples}, nil
		}
		wait := time.Until(deadline)
		if wait > poll {
			wait = poll
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func inspectRuntimeState(value any, container deploymentdto.RuntimeContainer) (string, string, int) {
	items, _ := value.([]any)
	if len(items) == 0 {
		return strings.ToLower(container.State), "", 0
	}
	item, _ := items[0].(map[string]any)
	stateValue, _ := item["State"].(map[string]any)
	state, _ := stateValue["Status"].(string)
	if state == "" {
		state = container.State
	}
	health := ""
	if healthValue, ok := stateValue["Health"].(map[string]any); ok {
		health, _ = healthValue["Status"].(string)
	}
	restarts := 0
	if raw, ok := item["RestartCount"].(float64); ok {
		restarts = int(raw)
	}
	return strings.ToLower(state), strings.ToLower(health), restarts
}

func compareRuntimeCompose(previewYAML, configYAML string, containers []deploymentdto.RuntimeContainer, inspections map[string]any) []string {
	preview := map[string]any{}
	config := map[string]any{}
	if yaml.Unmarshal([]byte(previewYAML), &preview) != nil || yaml.Unmarshal([]byte(configYAML), &config) != nil {
		return []string{"Compose configuration could not be parsed"}
	}
	expectedServices := mapping(preview["services"])
	actualServices := mapping(config["services"])
	running := map[string]struct{}{}
	for _, container := range containers {
		running[container.Service] = struct{}{}
	}
	differences := make([]string, 0)
	for name, expectedValue := range expectedServices {
		expected := mapping(expectedValue)
		actualValue, ok := actualServices[name]
		if !ok {
			differences = append(differences, "Compose config is missing expected service "+name)
			continue
		}
		actual := mapping(actualValue)
		if _, ok := running[name]; !ok {
			differences = append(differences, "Docker has no running container for expected service "+name)
		}
		if expectedImage, ok := expected["image"].(string); ok {
			if actualImage, ok := actual["image"].(string); ok && actualImage != expectedImage {
				differences = append(differences, "service "+name+" image differs between preview and Compose config")
			}
		}
	}
	_ = inspections
	return differences
}

func mapping(value any) map[string]any { result, _ := value.(map[string]any); return result }

func verificationSummary(evidence map[string]any, differences []string) map[string]any {
	containers, _ := evidence["containers"].([]deploymentdto.RuntimeContainer)
	items := make([]map[string]any, 0, len(containers))
	for _, container := range containers {
		items = append(items, map[string]any{"id": container.Id, "service": container.Service, "state": container.State, "status": container.Status, "health": container.Health})
	}
	stability, _ := evidence["stability"].(map[string]any)
	return map[string]any{"components": items, "drift_detected": len(differences) > 0, "stability": stability}
}
