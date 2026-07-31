package labels

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
)

var labelNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Labels map[string]string

func (l *Labels) UnmarshalJSON(payload []byte) error {
	var mapped map[string]string
	if err := json.Unmarshal(payload, &mapped); err == nil {
		*l = mapped
		return nil
	}
	var legacy []map[string]string
	if err := json.Unmarshal(payload, &legacy); err != nil {
		return errors.New("labels must be an object of string values")
	}
	result := make(Labels)
	for _, item := range legacy {
		for name, value := range item {
			if _, exists := result[name]; exists {
				return fmt.Errorf("duplicate label %q", name)
			}
			result[name] = value
		}
	}
	*l = result
	return nil
}

func FromNames(names []string) Labels {
	result := make(Labels, len(names))
	for _, name := range names {
		result[name] = ""
	}
	return result
}

func (l Labels) ResolveEnvironment() (Labels, error) {
	resolved := make(Labels, len(l))
	for name, value := range l {
		if strings.HasPrefix(value, "$") {
			environmentName := strings.TrimPrefix(value, "$")
			if environmentName == "" {
				return nil, fmt.Errorf("label %q contains an empty environment reference", name)
			}
			environmentValue, exists := os.LookupEnv(environmentName)
			if !exists {
				return nil, fmt.Errorf("environment variable %q required by label %q is not set", environmentName, name)
			}
			resolved[name] = environmentValue
			continue
		}
		resolved[name] = value
	}
	return resolved, nil
}

func (l Labels) ValidateNames() error {
	var errs []error
	for name := range l {
		if !labelNamePattern.MatchString(name) {
			errs = append(errs, fmt.Errorf("invalid Prometheus label name %q", name))
		}
	}
	return errors.Join(errs...)
}

func (l Labels) LabelNames() []string {
	names := make([]string, 0, len(l))
	for name := range l {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func MergeStrict(left Labels, right Labels) (Labels, error) {
	merged := make(Labels, len(left)+len(right))
	maps.Copy(merged, left)
	for name, value := range right {
		if _, exists := merged[name]; exists {
			return nil, fmt.Errorf("duplicate label %q", name)
		}
		merged[name] = value
	}
	return merged, nil
}
