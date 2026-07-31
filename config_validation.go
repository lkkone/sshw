package sshw

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

const MaxConfigSize = 10 << 20

type ConfigStats struct {
	Hosts   int `json:"hosts"`
	Groups  int `json:"groups"`
	Aliases int `json:"aliases"`
}

func ParseConfig(data []byte) ([]*Node, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("configuration is empty")
	}
	if len(data) > MaxConfigSize {
		return nil, fmt.Errorf("configuration exceeds %d bytes", MaxConfigSize)
	}

	var nodes []*Node
	if err := yaml.UnmarshalStrict(data, &nodes); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("configuration contains no hosts")
	}
	return nodes, nil
}

func ValidateConfig(data []byte) (ConfigStats, error) {
	nodes, err := ParseConfig(data)
	if err != nil {
		return ConfigStats{}, err
	}

	aliases := make(map[string]string)
	var stats ConfigStats
	for i, node := range nodes {
		if err := validateNode(node, fmt.Sprintf("item %d", i+1), aliases, &stats, false); err != nil {
			return ConfigStats{}, err
		}
	}
	return stats, nil
}

func validateNode(node *Node, location string, aliases map[string]string, stats *ConfigStats, jump bool) error {
	if node == nil {
		return fmt.Errorf("%s is null", location)
	}
	node.Name = strings.TrimSpace(node.Name)
	node.Alias = strings.TrimSpace(node.Alias)
	node.Host = strings.TrimSpace(node.Host)
	node.User = strings.TrimSpace(node.User)

	if node.Name == "" && !jump {
		return fmt.Errorf("%s: name is required", location)
	}
	if node.Port < 0 || node.Port > 65535 {
		return fmt.Errorf("%s (%s): port must be between 1 and 65535", location, node.Name)
	}
	if jump && node.Alias != "" {
		return fmt.Errorf("%s (%s): a jump host cannot define an alias", location, node.Name)
	}
	if node.Alias != "" {
		if previous, exists := aliases[node.Alias]; exists {
			return fmt.Errorf("%s (%s): alias %q is already used by %s", location, node.Name, node.Alias, previous)
		}
		aliases[node.Alias] = node.Name
		stats.Aliases++
	}

	if len(node.Children) > 0 {
		if jump {
			return fmt.Errorf("%s (%s): a jump host cannot be a group", location, node.Name)
		}
		if node.Host != "" {
			return fmt.Errorf("%s (%s): a group cannot also define a host", location, node.Name)
		}
		if node.Alias != "" {
			return fmt.Errorf("%s (%s): aliases can only be assigned to hosts", location, node.Name)
		}
		stats.Groups++
		for i, child := range node.Children {
			if err := validateNode(child, fmt.Sprintf("%s/%s child %d", location, node.Name, i+1), aliases, stats, false); err != nil {
				return err
			}
		}
		return nil
	}

	if node.Host == "" {
		return fmt.Errorf("%s (%s): host is required", location, node.Name)
	}
	if !jump {
		stats.Hosts++
	}
	for i, hop := range node.Jump {
		if err := validateNode(hop, fmt.Sprintf("%s/%s jump %d", location, node.Name, i+1), aliases, stats, true); err != nil {
			return err
		}
	}
	for i, callback := range node.CallbackShells {
		if callback == nil || strings.TrimSpace(callback.Cmd) == "" {
			return fmt.Errorf("%s (%s): callback %d requires a command", location, node.Name, i+1)
		}
		if callback.Delay < 0 {
			return fmt.Errorf("%s (%s): callback %d delay cannot be negative", location, node.Name, i+1)
		}
	}
	return nil
}
