package main

import (
	"fmt"
	"log"
	"net"

	"github.com/DagDigg/unpaper/core/config"
	"github.com/DagDigg/unpaper/core/k8s"
	"github.com/DagDigg/unpaper/extauth/server"
	authenvoy "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"google.golang.org/grpc"
)

func main() {
	// Initialize config
	cfg := config.Get(config.Params{
		K8sClientSet: k8s.GetClientSet(),
	})

	fmt.Printf("Starting Authorization Server on port: %v\n", cfg.AuthServerPort)

	// // Load TLS certificates
	// tlsCert, err := tls.LoadX509KeyPair("pkg/certificate/server-cert.pem", "pkg/certificate/server-key.pem")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// tlsCert.Leaf, _ = x509.ParseCertificate(tlsCert.Certificate[0]) // Can't fail if LoadX509KeyPair succeeded

	// // Create TLS config
	// tlsConf := &tls.Config{
	// 	Certificates: []tls.Certificate{tlsCert},
	// }

	// Start listening on port provided by Config
	lis, err := net.Listen("tcp", ":"+cfg.AuthServerPort)

	if err != nil {
		log.Fatalf("failed to listen to tcp network: %v", err)
	}

	// Create Authorization server
	authServer, err := server.NewAuthorizationServer(cfg.GetRDBConnURL(), cfg)
	grpcServer := grpc.NewServer()
	authenvoy.RegisterAuthorizationServer(grpcServer, authServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc server: %v", err)
	}
}
