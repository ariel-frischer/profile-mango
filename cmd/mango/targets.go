package main

import (
	"fmt"
	"strings"
)

// targetList splits only selector flags. target=value bindings intentionally use
// StringArray and retain commas in paths and values.
func targetList(values []string) ([]string, error) {
	var selected []string
	seen := make(map[string]struct{})
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				return nil, fmt.Errorf("--target %q contains an empty target", value)
			}
			if _, exists := seen[part]; !exists {
				selected = append(selected, part)
				seen[part] = struct{}{}
			}
		}
	}
	return selected, nil
}
