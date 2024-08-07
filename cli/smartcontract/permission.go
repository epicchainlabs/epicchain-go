package smartcontract

import (
	"errors"
	"fmt"

	"github.com/epicchainlabs/epicchain-go/pkg/crypto/keys"
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract/manifest"
	"github.com/epicchainlabs/epicchain-go/pkg/util"
	"gopkg.in/yaml.v3"
)

type permission manifest.Permission

const (
	permHashKey   = "hash"
	permGroupKey  = "group"
	permMethodKey = "methods"
)

func (p permission) MarshalYAML() (any, error) {
	m := yaml.Node{Kind: yaml.MappingNode}
	switch p.Contract.Type {
	case manifest.PermissionWildcard:
	case manifest.PermissionHash:
		m.Content = append(m.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: permHashKey},
			&yaml.Node{Kind: yaml.ScalarNode, Value: p.Contract.Value.(util.Uint160).StringLE()})
	case manifest.PermissionGroup:
		m.Content = append(m.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: permGroupKey},
			&yaml.Node{Kind: yaml.ScalarNode, Value: p.Contract.Value.(*keys.PublicKey).StringCompressed()})
	default:
		return nil, fmt.Errorf("invalid permission type: %d", p.Contract.Type)
	}

	var val any = "*"
	if !p.Methods.IsWildcard() {
		val = p.Methods.Value
	}

	n := &yaml.Node{Kind: yaml.ScalarNode}
	err := n.Encode(val)
	if err != nil {
		return nil, err
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: permMethodKey},
		n)

	return m, nil
}

func (p *permission) UnmarshalYAML(unmarshal func(any) error) error {
	var m map[string]any
	if err := unmarshal(&m); err != nil {
		return err
	}

	if err := p.fillType(m); err != nil {
		return err
	}

	return p.fillMethods(m)
}

func (p *permission) fillType(m map[string]any) error {
	vh, ok1 := m[permHashKey]
	vg, ok2 := m[permGroupKey]
	switch {
	case ok1 && ok2:
		return errors.New("permission must have either 'hash' or 'group' field")
	case ok1:
		s, ok := vh.(string)
		if !ok {
			return errors.New("invalid 'hash' type")
		}

		u, err := util.Uint160DecodeStringLE(s)
		if err != nil {
			return err
		}

		p.Contract.Type = manifest.PermissionHash
		p.Contract.Value = u
	case ok2:
		s, ok := vg.(string)
		if !ok {
			return errors.New("invalid 'hash' type")
		}

		pub, err := keys.NewPublicKeyFromString(s)
		if err != nil {
			return err
		}

		p.Contract.Type = manifest.PermissionGroup
		p.Contract.Value = pub
	default:
		p.Contract.Type = manifest.PermissionWildcard
	}
	return nil
}

func (p *permission) fillMethods(m map[string]any) error {
	methods, ok := m[permMethodKey]
	if !ok {
		return errors.New("'methods' field is missing from permission")
	}

	switch mt := methods.(type) {
	case string:
		if mt == "*" {
			p.Methods.Value = nil
			return nil
		}
	case []any:
		ms := make([]string, len(mt))
		for i := range mt {
			ms[i], ok = mt[i].(string)
			if !ok {
				return errors.New("invalid permission method name")
			}
		}
		p.Methods.Value = ms
		return nil
	default:
	}
	return errors.New("'methods' field is invalid")
}
