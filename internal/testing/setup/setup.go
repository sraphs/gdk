package setup

import (
	"os"
)

// HasDockerTestEnvironment returns true when either:
// 1) Not on Github Actions.
// 2) On Github's Linux environment, where Docker is available.
func HasDockerTestEnvironment() bool {
	s := os.Getenv("RUNNER_OS")
	return s == "" || s == "Linux"
}
