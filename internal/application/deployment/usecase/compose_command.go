package deploymentsvc

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
	return fmt.Sprintf("%s-%s-%s", appCode, envCode, instanceKey)
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

func containerLogsTailCommand(projectName string, tail string, serviceNames ...string) composeCommand {
	args := []string{"compose", "-p", projectName, "-f", "docker-compose.yml", "logs", "--tail", tail}
	for _, name := range serviceNames {
		if name == "" {
			continue
		}
		args = append(args, name)
	}
	return composeCommand{Name: "docker", Args: args}
}

func containerPsCommand(projectName string) composeCommand {
	return composeCommand{Name: "docker", Args: []string{"compose", "-p", projectName, "-f", "docker-compose.yml", "ps", "--format", "json"}}
}
