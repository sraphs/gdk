// Package terraform provides a function to read Terraform output.
package terraform

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// ReadOutput runs `terraform output` on the given directory and returns
// the parsed result.
func ReadOutput(dir string) (map[string]Output, error) {
	c := exec.Command("terraform", "output", "-json")
	c.Dir = dir
	data, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("read terraform output: %v", err)
	}
	var parsed map[string]Output
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("read terraform output: %v", err)
	}
	return parsed, nil
}

// Output describes a single output value.
type Output struct {
	Type      string      `json:"type"` // one of "string", "list", or "map"
	Sensitive bool        `json:"sensitive"`
	Value     interface{} `json:"value"`
}
