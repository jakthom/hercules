package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	CommunityExtensionType = "community"
	CoreExtensionType      = "core"
)

var extensionNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type CoreExtension struct {
	Name string `json:"name"`
}

type CommunityExtension struct {
	Name string `json:"name"`
}

type Extensions struct {
	Core      []CoreExtension      `json:"core"`
	Community []CommunityExtension `json:"community"`
}

type ExtensionPolicy struct {
	Allowed        []string
	AllowCommunity bool
}

func EnsureExtensions(ctx context.Context, executor Executor, extensions Extensions, policy ExtensionPolicy) error {
	if err := ValidateExtensions(extensions, policy); err != nil {
		return err
	}
	var errs []error
	for _, extension := range extensions.Core {
		if err := ensureExtension(ctx, executor, extension.Name, CoreExtensionType); err != nil {
			errs = append(errs, err)
		}
	}
	for _, extension := range extensions.Community {
		if err := ensureExtension(ctx, executor, extension.Name, CommunityExtensionType); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func ValidateExtensions(extensions Extensions, policy ExtensionPolicy) error {
	seen := make(map[string]string, len(extensions.Core)+len(extensions.Community))
	var errs []error
	for _, extension := range extensions.Core {
		if err := validateExtension(extension.Name, CoreExtensionType, policy, seen); err != nil {
			errs = append(errs, err)
		}
	}
	for _, extension := range extensions.Community {
		if err := validateExtension(extension.Name, CommunityExtensionType, policy, seen); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateExtension(name string, extensionType string, policy ExtensionPolicy, seen map[string]string) error {
	if !extensionNamePattern.MatchString(name) {
		return fmt.Errorf("invalid DuckDB extension name %q", name)
	}
	lookupName := strings.ToLower(name)
	if previousType, exists := seen[lookupName]; exists {
		return fmt.Errorf("DuckDB extension %q is repeated as %s and %s", name, previousType, extensionType)
	}
	seen[lookupName] = extensionType
	if !slices.Contains(policy.Allowed, name) {
		return fmt.Errorf("DuckDB extension %q is not in security.allowedExtensions", name)
	}
	if extensionType == CommunityExtensionType && !policy.AllowCommunity {
		return fmt.Errorf("community DuckDB extension %q is disabled", name)
	}
	return nil
}

func ensureExtension(
	ctx context.Context,
	executor Executor,
	name string,
	extensionType string,
) error {
	install := SQL("install " + name + ";")
	if extensionType == CommunityExtensionType {
		install = SQL("install " + name + " from community;")
	}
	if err := executor.Exec(ctx, install); err != nil {
		return fmt.Errorf("install %s extension %q: %w", extensionType, name, err)
	}
	if err := executor.Exec(ctx, SQL("load "+name+";")); err != nil {
		return fmt.Errorf("load %s extension %q: %w", extensionType, name, err)
	}
	return nil
}
