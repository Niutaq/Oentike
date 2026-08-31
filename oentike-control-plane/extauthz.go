package main

import (
	"context"
	"log"
	"strings"

	"google.golang.org/grpc/codes"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	status_pb "google.golang.org/genproto/googleapis/rpc/status"
)

// ExtAuthzServer implements Envoy's ExtAuthz API.
type ExtAuthzServer struct {
	authv3.UnimplementedAuthorizationServer
	controlPlane *server
}

// Check verifies if the requested SPIFFE ID is quarantined.
func (s *ExtAuthzServer) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	agentID := ""

	// Envoy passes the client's URI SAN (SPIFFE ID) in the Source.Principal field
	principal := req.Attributes.Source.Principal
	if strings.HasPrefix(principal, "spiffe://example.org/") {
		agentID = strings.TrimPrefix(principal, "spiffe://example.org/")
	} else {
		// Fallback to headers just in case
		spiffeHeader := req.Attributes.Request.Http.Headers["x-spiffe-id"]
		if strings.HasPrefix(spiffeHeader, "spiffe://example.org/") {
			agentID = strings.TrimPrefix(spiffeHeader, "spiffe://example.org/")
		}
	}

	if agentID != "" {
		s.controlPlane.mu.RLock()
		isBanned := s.controlPlane.bannedAgents[agentID]
		s.controlPlane.mu.RUnlock()

		if isBanned {
			log.Printf("[ExtAuthz] L7 Edge DENY for quarantined agent: %s", agentID)
			return &authv3.CheckResponse{
				Status: &status_pb.Status{Code: int32(codes.PermissionDenied)},
				HttpResponse: &authv3.CheckResponse_DeniedResponse{
					DeniedResponse: &authv3.DeniedHttpResponse{
						Status: &typev3.HttpStatus{
							Code: typev3.StatusCode_Forbidden,
						},
						Body: "Forbidden: Agent is Quarantined by SecOps",
					},
				},
			}, nil
		}
	}

	// Allow by default
	return &authv3.CheckResponse{
		Status: &status_pb.Status{Code: int32(codes.OK)},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   "x-secops-authorized",
							Value: "true",
						},
					},
				},
			},
		},
	}, nil
}
