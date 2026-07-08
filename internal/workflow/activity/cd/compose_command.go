package cdsvc

import "strings"

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

func deployComposeCommand(imagePullPolicy string, forceRecreate bool) composeCommand {
	args := []string{"compose", "-f", "docker-compose.yml", "up", "-d", "--remove-orphans", "--pull", imagePullPolicy}
	if forceRecreate {
		args = append(args, "--force-recreate")
	}
	return composeCommand{Name: "docker", Args: args}
}

func restartComposeCommand() composeCommand {
	return composeCommand{Name: "docker", Args: []string{"compose", "-f", "docker-compose.yml", "restart"}}
}

func stopComposeCommand(removeVolumes bool) composeCommand {
	args := []string{"compose", "-f", "docker-compose.yml", "down"}
	if removeVolumes {
		args = append(args, "-v")
	}
	return composeCommand{Name: "docker", Args: args}
}
