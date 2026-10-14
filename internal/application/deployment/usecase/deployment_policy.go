package deploymentsvc

import "fmt"

func validateDeploymentPullPolicy(policy string) error {
	switch policy {
	case "always", "missing", "never":
		return nil
	default:
		return fmt.Errorf("deployment pull_policy must be always, missing or never")
	}
}
