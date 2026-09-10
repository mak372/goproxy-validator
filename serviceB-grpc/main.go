package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	"go_project/protos/kyc/kycpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// kycServer is a normal, statically generated gRPC service implementation —
// this stands in for a downstream team's own microservice, which in
// practice is always built against compiled protobuf stubs rather than
// dynamic messages. The gateway (proxy/) is what handles contracts
// dynamically; this service just implements one.
type kycServer struct {
	kycpb.UnimplementedKycServiceServer
}

// In-memory identity registry: documentNumber -> fullName (mirrors serviceB's HTTP demo)
var registry = map[string]string{
	"DL1234567":  "Amit Sharma",
	"PAN9876543": "Priya Mehta",
	"PASS112233": "Rahul Verma",
}

func (s *kycServer) Verify(ctx context.Context, req *kycpb.VerifyRequest) (*kycpb.VerifyResponse, error) {
	status := "rejected"
	riskScore := 85.0 + rand.Float64()*15 // high risk 85-100

	registeredName, exists := registry[req.GetDocumentNumber()]
	if exists && registeredName == req.GetFullName() {
		status = "verified"
		riskScore = rand.Float64() * 30 // low risk 0-30
	} else if exists {
		status = "pending"
		riskScore = 40 + rand.Float64()*30 // medium risk 40-70
	}

	return &kycpb.VerifyResponse{
		CustomerId:     req.GetCustomerId(),
		VerificationId: fmt.Sprintf("VER-%d", time.Now().UnixNano()),
		Status:         status,
		RiskScore:      riskScore,
		VerifiedAt:     time.Now().Format(time.RFC3339),
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":9002")
	if err != nil {
		fmt.Println("failed to listen:", err)
		return
	}

	server := grpc.NewServer()
	kycpb.RegisterKycServiceServer(server, &kycServer{})
	reflection.Register(server)

	fmt.Println("Identity Registry gRPC Service running on :9002")
	if err := server.Serve(lis); err != nil {
		fmt.Println("server error:", err)
	}
}
