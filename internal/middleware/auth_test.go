package middleware

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestExtractIdentity(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func() context.Context
		wantCN     string
		wantCode   codes.Code
		wantErr    bool
	}{
		{
			name: "no peer info",
			setupCtx: context.Background,
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name: "no TLS info",
			setupCtx: func() context.Context {
				p := &peer.Peer{AuthInfo: nil}
				return peer.NewContext(context.Background(), p)
			},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name: "no certificates",
			setupCtx: func() context.Context {
				tlsInfo := credentials.TLSInfo{}
				p := &peer.Peer{AuthInfo: tlsInfo}
				return peer.NewContext(context.Background(), p)
			},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name: "valid certificate",
			setupCtx: func() context.Context {
				cert := &x509.Certificate{
					Subject: pkix.Name{CommonName: "test-client"},
				}
				tlsInfo := credentials.TLSInfo{
					State: tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert},
					},
				}
				p := &peer.Peer{AuthInfo: tlsInfo}
				return peer.NewContext(context.Background(), p)
			},
			wantCN:   "test-client",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			id, err := extractIdentity(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				if status.Code(err) != tt.wantCode {
					t.Errorf("extractIdentity() code = %v, wantCode %v", status.Code(err), tt.wantCode)
				}
				return
			}
			if id.CommonName != tt.wantCN {
				t.Errorf("extractIdentity() CN = %v, want %v", id.CommonName, tt.wantCN)
			}
		})
	}
}
