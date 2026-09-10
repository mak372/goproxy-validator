package config

import (
	"encoding/json"
	"os"
)

type Contract struct {
	Endpoint  string                 `json:"endpoint"`
	Method    string                 `json:"method"`
	Target    string                 `json:"target"`
	Request   map[string]interface{} `json:"request"`
	Response  map[string]interface{} `json:"response"`
	CreatedAt string                 `json:"createdAt"`

	// Protocol selects the upstream transport: "http" (default, existing
	// behavior) or "grpc". For "grpc" contracts, Target is a host:port gRPC
	// address (no scheme) and the fields below describe which RPC to call.
	// Request/Response above are still used to validate the JSON the gateway
	// itself sees, in the same way regardless of the upstream transport.
	Protocol    string `json:"protocol,omitempty"`
	ProtoSource string `json:"protoSource,omitempty"`
	GRPCService string `json:"grpcService,omitempty"`
	GRPCMethod  string `json:"grpcMethod,omitempty"`
}

// IsGRPC reports whether this contract proxies to a gRPC upstream.
func (c *Contract) IsGRPC() bool {
	return c.Protocol == "grpc"
}

func LoadContract(path string) (*Contract, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var contract Contract
	err = json.Unmarshal(file, &contract)
	if err != nil {
		return nil, err
	}

	return &contract, nil
}
