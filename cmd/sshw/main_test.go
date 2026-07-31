package main

import (
	"testing"

	"github.com/yinheli/sshw"
)

func TestFindAliasContinuesAfterUnmatchedGroup(t *testing.T) {
	target := &sshw.Node{Name: "MasterGo-Poc", Alias: "poc"}
	nodes := []*sshw.Node{
		{
			Name: "jump",
			Children: []*sshw.Node{
				{Name: "jump-host", Alias: "jump-host"},
			},
		},
		{
			Name:     "Mastergo",
			Children: []*sshw.Node{target},
		},
	}

	if got := findAlias(nodes, "poc"); got != target {
		t.Fatalf("findAlias() = %#v, want %#v", got, target)
	}
}

func TestFindAliasSearchesNestedGroups(t *testing.T) {
	target := &sshw.Node{Name: "production", Alias: "prod"}
	nodes := []*sshw.Node{
		{
			Name: "region",
			Children: []*sshw.Node{
				{
					Name:     "team",
					Children: []*sshw.Node{target},
				},
			},
		},
	}

	if got := findAlias(nodes, "prod"); got != target {
		t.Fatalf("findAlias() = %#v, want %#v", got, target)
	}
}

func TestFindAliasReturnsNilWhenMissing(t *testing.T) {
	nodes := []*sshw.Node{
		{
			Name: "group",
			Children: []*sshw.Node{
				{Name: "server", Alias: "dev"},
			},
		},
	}

	if got := findAlias(nodes, "missing"); got != nil {
		t.Fatalf("findAlias() = %#v, want nil", got)
	}
}
