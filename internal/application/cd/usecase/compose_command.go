package cdsvc

import (
	"fmt"
	"strings"
)

type composeCommand struct {
	Name string
	Args []string
}

func (c composeCommand) String() string {
	return strings.Join(c.argv(), " ")
}

func (c composeCommand) argv() []string {
	parts := make([]string, 0, len(c.Args)+1)
	parts = append(parts, c.Name)
	parts = append(parts, c.Args...)
	return parts
}

func composeProjectName(appCode string, envCode string, instanceKey string) string {
	return sanitizeComposeName(fmt.Sprintf("%s-%s-%s", appCode, envCode, instanceKey))
}

func sanitizeComposeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	out := strings.Trim(b.String(), "-_")
	if out == "" {
		return "app"
	}
	return out
}

func deployComposeCommand(projectName string, imagePullPolicy string, forceRecreate bool) composeCommand {
	args := []string{"compose", "-p", projectName, "-f", "docker-compose.yml", "up", "-d", "--remove-orphans", "--pull", imagePullPolicy}
	if forceRecreate {
		args = append(args, "--force-recreate")
	}
	return composeCommand{Name: "docker", Args: args}
}

func stopComposeCommand(projectName string, removeVolumes bool) composeCommand {
	args := []string{"compose", "-p", projectName, "-f", "docker-compose.yml", "down"}
	if removeVolumes {
		args = append(args, "-v")
	}
	return composeCommand{Name: "docker", Args: args}
}

func containerLogsSinceCommand(projectName string, since string) composeCommand {
	return composeCommand{Name: "docker", Args: []string{"compose", "-p", projectName, "-f", "docker-compose.yml", "logs", "--since", since}}
}

func containerLogsTailCommand(projectName string, tail string) composeCommand {
	return composeCommand{Name: "docker", Args: []string{"compose", "-p", projectName, "-f", "docker-compose.yml", "logs", "--tail", tail}}
}

func containerPsCommand(projectName string) composeCommand {
	return composeCommand{Name: "docker", Args: []string{"compose", "-p", projectName, "-f", "docker-compose.yml", "ps", "--format", "json"}}
}
