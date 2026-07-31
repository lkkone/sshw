package sshw

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	data := []byte(`
- name: Production
  children:
    - name: API
      alias: api
      host: 10.0.0.10
      user: deploy
      jump:
        - name: Bastion
          host: bastion.example.com
          user: jump
- name: Database
  alias: db
  host: 10.0.0.20
`)
	stats, err := ValidateConfig(data)
	if err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
	if stats.Hosts != 2 || stats.Groups != 1 || stats.Aliases != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestValidateConfigRejectsDuplicateAliases(t *testing.T) {
	data := []byte(`
- name: One
  alias: shared
  host: 10.0.0.1
- name: Two
  alias: shared
  host: 10.0.0.2
`)
	_, err := ValidateConfig(data)
	if err == nil || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func TestValidateConfigAllowsUnnamedJumpHost(t *testing.T) {
	data := []byte(`
- name: Internal server
  host: 10.0.0.10
  jump:
    - host: bastion.example.com
      user: jump
      port: 2222
`)
	stats, err := ValidateConfig(data)
	if err != nil {
		t.Fatalf("ValidateConfig() error = %v", err)
	}
	if stats.Hosts != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestValidateConfigRejectsUnknownFields(t *testing.T) {
	data := []byte(`
- name: One
  host: 10.0.0.1
  unexpected: value
`)
	_, err := ValidateConfig(data)
	if err == nil || !strings.Contains(err.Error(), "field unexpected") {
		t.Fatalf("expected strict YAML error, got %v", err)
	}
}
