package db

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var (
	macroNamePattern        = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	macroDeclarationPattern = regexp.MustCompile(`(?i)^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
)

type Macro struct {
	Name string `json:"name"`
	SQL  SQL    `json:"sql"`
}

func (m Macro) CreateOrReplaceSQL() SQL {
	return SQL("create or replace macro " + string(m.SQL))
}

func (m Macro) Validate() error {
	if !macroNamePattern.MatchString(m.Name) {
		return fmt.Errorf("invalid macro name %q", m.Name)
	}
	if strings.TrimSpace(string(m.SQL)) == "" {
		return fmt.Errorf("macro %q has empty sql", m.Name)
	}
	declaration := macroDeclarationPattern.FindStringSubmatch(string(m.SQL))
	if len(declaration) != 2 || !strings.EqualFold(declaration[1], m.Name) {
		return fmt.Errorf("macro %q sql must declare the same macro name", m.Name)
	}
	return nil
}

func EnsureMacros(ctx context.Context, executor Executor, macros []Macro) error {
	for _, macro := range macros {
		if err := macro.Validate(); err != nil {
			return err
		}
		if err := executor.Exec(ctx, macro.CreateOrReplaceSQL()); err != nil {
			return fmt.Errorf("create macro %q: %w", macro.Name, err)
		}
	}
	return nil
}
