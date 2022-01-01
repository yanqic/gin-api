package registry

import (
	"context"
)

type Node struct {
	Host   string
	Port   int
	Weight int //TODO support service weight
}

type RegistryConfig struct {
	Lease int64
}

// Registrar is service registrar
type Registrar interface {
	Register(ctx context.Context) error
	DeRegister(ctx context.Context) error
}

// Discovery is service discovery
type Discovery interface {
	SetServiceList(key string, val *Node)
	DelServiceList(key string)
	GetNodes() []*Node
	Close() error
}

// Encode func is encode service node info
type Encode func(node *Node) (string, error)

// Decode func is decode service node info
type Decode func(val string) (*Node, error)

var Services = make(map[string]Discovery)
