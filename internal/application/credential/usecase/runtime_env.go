package credentialsvc

import "regexp"

var runtimeEnvKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
