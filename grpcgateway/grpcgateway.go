// Package grpcgateway lets the proxy speak gRPC to an upstream while still
// accepting and returning plain JSON over HTTP, mirroring the JD's
// "HTTP-to-gRPC reverse proxy" shape: REST/JSON in -> Protobuf message ->
// gRPC call -> Protobuf response -> JSON out.
//
// A contract's Protobuf schema (service/method + .proto source) is supplied
// at contract-registration time, just like the existing JSON request/response
// schemas. It is compiled in memory with protocompile (the same compiler
// used inside the `buf` CLI — no protoc binary, no files on disk) and the
// resulting descriptors drive dynamic message construction
// (google.golang.org/protobuf/types/dynamicpb) and RPC dispatch via
// grpc.ClientConn.Invoke. Everything here is built on the standard,
// non-deprecated protobuf-go and grpc-go APIs — no generated stubs — so a
// new gRPC contract can be registered and served immediately, with no
// gateway rebuild or restart, the same guarantee the existing JSON contract
// path already makes.
package grpcgateway

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go_project/config"

	"github.com/bufbuild/protocompile"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// entry caches everything needed to invoke one gRPC contract so its .proto
// is compiled and its connection dialed only once, not on every request.
type entry struct {
	conn       *grpc.ClientConn
	fullMethod string // e.g. "/kyc.KycService/Verify"
	inputDesc  protoreflect.MessageDescriptor
	outputDesc protoreflect.MessageDescriptor
}

var (
	mu    sync.RWMutex
	cache = make(map[string]*entry)
)

// Invoke sends reqJSON to the gRPC method described by the contract and
// returns the upstream's response re-encoded as JSON, so the rest of the
// gateway (contract validation, violation logging, HTTP response writing)
// can treat a gRPC upstream exactly like an HTTP/JSON one.
func Invoke(ctx context.Context, key string, c *config.Contract, reqJSON []byte) ([]byte, error) {
	e, err := getEntry(key, c)
	if err != nil {
		return nil, err
	}

	reqMsg := dynamicpb.NewMessage(e.inputDesc)
	if err := protojson.Unmarshal(reqJSON, reqMsg); err != nil {
		return nil, fmt.Errorf("request does not match proto schema: %w", err)
	}

	respMsg := dynamicpb.NewMessage(e.outputDesc)
	if err := e.conn.Invoke(ctx, e.fullMethod, reqMsg, respMsg); err != nil {
		return nil, fmt.Errorf("upstream gRPC call failed: %w", err)
	}

	return protojson.Marshal(respMsg)
}

// getEntry returns the cached descriptor+connection for a contract,
// building it on first use and reusing it for every subsequent request.
func getEntry(key string, c *config.Contract) (*entry, error) {
	mu.RLock()
	e, ok := cache[key]
	mu.RUnlock()
	if ok {
		return e, nil
	}

	mu.Lock()
	defer mu.Unlock()
	if e, ok := cache[key]; ok { // re-check: another request may have built it first
		return e, nil
	}

	inputDesc, outputDesc, fullMethod, err := resolveMethod(c)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(c.Target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s: %w", c.Target, err)
	}

	e = &entry{conn: conn, fullMethod: fullMethod, inputDesc: inputDesc, outputDesc: outputDesc}
	cache[key] = e
	return e, nil
}

// resolveMethod compiles the contract's embedded .proto source and locates
// the requested service/method within it, returning the input/output
// message descriptors and the fully-qualified gRPC method path.
func resolveMethod(c *config.Contract) (inputDesc, outputDesc protoreflect.MessageDescriptor, fullMethod string, err error) {
	if c.ProtoSource == "" || c.GRPCService == "" || c.GRPCMethod == "" {
		return nil, nil, "", fmt.Errorf("grpc contract requires protoSource, grpcService, and grpcMethod")
	}

	const filename = "contract.proto"
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				filename: c.ProtoSource,
			}),
		}),
	}

	files, err := compiler.Compile(context.Background(), filename)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to compile contract proto: %w", err)
	}
	fd := files[0]

	shortService := strings.TrimPrefix(c.GRPCService, string(fd.Package())+".")
	svc := fd.Services().ByName(protoreflect.Name(shortService))
	if svc == nil {
		return nil, nil, "", fmt.Errorf("service %q not found in contract proto", c.GRPCService)
	}
	method := svc.Methods().ByName(protoreflect.Name(c.GRPCMethod))
	if method == nil {
		return nil, nil, "", fmt.Errorf("method %q not found on service %q", c.GRPCMethod, c.GRPCService)
	}

	fullMethod = fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())
	return method.Input(), method.Output(), fullMethod, nil
}

// Invalidate drops any cached connection/descriptor for a contract key. Call
// it when a contract is deleted or re-registered with a different schema so
// the next request rebuilds against the new definition instead of reusing a
// stale connection or descriptor.
func Invalidate(key string) {
	mu.Lock()
	defer mu.Unlock()
	if e, ok := cache[key]; ok {
		e.conn.Close()
		delete(cache, key)
	}
}
