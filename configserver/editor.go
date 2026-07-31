package configserver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yinheli/sshw"
	"gopkg.in/yaml.v2"
)

type Document struct {
	Nodes []*EditorNode `json:"nodes"`
}

type EditorNode struct {
	Kind           string                `json:"kind"`
	Name           string                `json:"name"`
	Alias          string                `json:"alias,omitempty"`
	Host           string                `json:"host,omitempty"`
	User           string                `json:"user,omitempty"`
	Port           int                   `json:"port,omitempty"`
	KeyPath        string                `json:"keypath,omitempty"`
	AgentPath      string                `json:"agentpath,omitempty"`
	Passphrase     string                `json:"passphrase,omitempty"`
	Password       string                `json:"password,omitempty"`
	CallbackShells []*sshw.CallbackShell `json:"callback-shells,omitempty"`
	Children       []*EditorNode         `json:"children,omitempty"`
	Jump           []*EditorNode         `json:"jump,omitempty"`
}

func EmptyDocument() Document {
	return Document{Nodes: []*EditorNode{}}
}

func DecodeDocument(data []byte) (Document, error) {
	var document Document
	if err := json.Unmarshal(data, &document); err != nil {
		return Document{}, err
	}
	if document.Nodes == nil {
		document.Nodes = []*EditorNode{}
	}
	return document, nil
}

func EncodeDocument(document Document) ([]byte, error) {
	if document.Nodes == nil {
		document.Nodes = []*EditorNode{}
	}
	return json.Marshal(document)
}

func RenderDocument(document Document) ([]byte, sshw.ConfigStats, error) {
	if len(document.Nodes) == 0 {
		return []byte("[]\n"), sshw.ConfigStats{}, fmt.Errorf("configuration contains no hosts")
	}
	nodes := make([]*sshw.Node, 0, len(document.Nodes))
	for i, node := range document.Nodes {
		converted, err := node.toConfigNode(fmt.Sprintf("item %d", i+1), false)
		if err != nil {
			return nil, sshw.ConfigStats{}, err
		}
		nodes = append(nodes, converted)
	}
	data, err := yaml.Marshal(nodes)
	if err != nil {
		return nil, sshw.ConfigStats{}, err
	}
	stats, err := sshw.ValidateConfig(data)
	if err != nil {
		return data, sshw.ConfigStats{}, err
	}
	return data, stats, nil
}

func DocumentFromYAML(data []byte) (Document, error) {
	nodes, err := sshw.ParseConfig(data)
	if err != nil {
		return Document{}, err
	}
	document := Document{Nodes: make([]*EditorNode, 0, len(nodes))}
	for _, node := range nodes {
		document.Nodes = append(document.Nodes, editorNodeFromConfig(node))
	}
	return document, nil
}

func (node *EditorNode) toConfigNode(location string, jump bool) (*sshw.Node, error) {
	if node == nil {
		return nil, fmt.Errorf("%s is null", location)
	}
	kind := strings.ToLower(strings.TrimSpace(node.Kind))
	if kind == "" {
		if len(node.Children) > 0 {
			kind = "group"
		} else {
			kind = "host"
		}
	}
	if kind != "host" && kind != "group" {
		return nil, fmt.Errorf("%s: kind must be host or group", location)
	}
	if jump && kind != "host" {
		return nil, fmt.Errorf("%s: jump entries must be hosts", location)
	}

	result := &sshw.Node{
		Name:           strings.TrimSpace(node.Name),
		Alias:          strings.TrimSpace(node.Alias),
		Host:           strings.TrimSpace(node.Host),
		User:           strings.TrimSpace(node.User),
		Port:           node.Port,
		KeyPath:        strings.TrimSpace(node.KeyPath),
		AgentPath:      strings.TrimSpace(node.AgentPath),
		Passphrase:     node.Passphrase,
		Password:       node.Password,
		CallbackShells: node.CallbackShells,
	}
	if kind == "group" {
		if len(node.Children) == 0 {
			return nil, fmt.Errorf("%s (%s): group must contain at least one host or group", location, result.Name)
		}
		result.Children = make([]*sshw.Node, 0, len(node.Children))
		for i, child := range node.Children {
			converted, err := child.toConfigNode(fmt.Sprintf("%s child %d", location, i+1), false)
			if err != nil {
				return nil, err
			}
			result.Children = append(result.Children, converted)
		}
		return result, nil
	}
	for i, hop := range node.Jump {
		converted, err := hop.toConfigNode(fmt.Sprintf("%s jump %d", location, i+1), true)
		if err != nil {
			return nil, err
		}
		result.Jump = append(result.Jump, converted)
	}
	return result, nil
}

func editorNodeFromConfig(node *sshw.Node) *EditorNode {
	kind := "host"
	if len(node.Children) > 0 {
		kind = "group"
	}
	result := &EditorNode{
		Kind:           kind,
		Name:           node.Name,
		Alias:          node.Alias,
		Host:           node.Host,
		User:           node.User,
		Port:           node.Port,
		KeyPath:        node.KeyPath,
		AgentPath:      node.AgentPath,
		Passphrase:     node.Passphrase,
		Password:       node.Password,
		CallbackShells: node.CallbackShells,
	}
	for _, child := range node.Children {
		result.Children = append(result.Children, editorNodeFromConfig(child))
	}
	for _, hop := range node.Jump {
		result.Jump = append(result.Jump, editorNodeFromConfig(hop))
	}
	return result
}
