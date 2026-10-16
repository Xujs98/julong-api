package dto

import (
	"fmt"
	"strings"
)

const (
	MaxTextReplacementRules = 100
	MaxTextReplacementPart  = 4096
)

type TextReplacementRule struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type TextReplacementConfig struct {
	Request  []TextReplacementRule `json:"request,omitempty"`
	Response []TextReplacementRule `json:"response,omitempty"`
}

func (c TextReplacementConfig) Empty() bool {
	return len(c.Request) == 0 && len(c.Response) == 0
}

func (c TextReplacementConfig) Validate() error {
	for name, rules := range map[string][]TextReplacementRule{
		"request":  c.Request,
		"response": c.Response,
	} {
		if len(rules) > MaxTextReplacementRules {
			return fmt.Errorf("text replacement %s rules exceed %d", name, MaxTextReplacementRules)
		}
		for i, rule := range rules {
			if strings.TrimSpace(rule.From) == "" {
				return fmt.Errorf("text replacement %s rule %d has empty from", name, i)
			}
			if len(rule.From) > MaxTextReplacementPart || len(rule.To) > MaxTextReplacementPart {
				return fmt.Errorf("text replacement %s rule %d is too long", name, i)
			}
		}
	}
	return nil
}

func (c TextReplacementConfig) Apply(value string, response bool) string {
	rules := c.Request
	if response {
		rules = c.Response
	}
	for _, rule := range rules {
		if rule.From != "" {
			value = strings.ReplaceAll(value, rule.From, rule.To)
		}
	}
	return value
}
