package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/logger"
	"github.com/DagDigg/unpaper/backend/pkg/protocol/grpcserver"
	v1Service "github.com/DagDigg/unpaper/backend/pkg/service/v1"
	"github.com/DagDigg/unpaper/core/config"
	"github.com/DagDigg/unpaper/core/k8s"
	stripe "github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	// postgres driver
	_ "github.com/lib/pq"
)

// RunServer runs gRPC server and HTTP gateway
func RunServer() error {
	// load config
	cfg := config.Get(config.Params{
		K8sClientSet: k8s.GetClientSet(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize logger
	if err := logger.Init(cfg.LogLevel, cfg.LogTimeFormat); err != nil {
		return fmt.Errorf("failed to initialize logger: %v", err)
	}

	serverCertPEM := "pkg/certificate/server-cert.pem"
	serverKeyPEM := "pkg/certificate/server-key.pem"
	caCertPEM := "pkg/certificate/ca-cert.pem"
	tlsConf, err := loadCerts(serverCertPEM, serverKeyPEM, caCertPEM)
	if err != nil {
		logger.Log.Error("failed to load TLS key pair", zap.String("err", err.Error()))
	}

	conn, err := net.Listen("tcp", ":"+cfg.APIPort)
	if err != nil {
		logger.Log.Panic(err.Error())
	}

	// mux := cmux.New(conn)
	// grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	// httpL := mux.Match(cmux.Any())

	// go func() {
	// 	sErr := mux.Serve()
	// 	if sErr != nil {
	// 		logger.Log.Fatal("failed to serve cmux", zap.String("err", err.Error()))
	// 	}
	// }()

	// Register GRPC server
	gServer := grpcserver.Get(ctx, credentials.NewTLS(tlsConf))
	gServer.ListenForShutdown()

	v1Server, err := v1Service.NewUnpaperServiceServer(cfg)
	if err != nil {
		return err
	}
	v1API.RegisterUnpaperServiceServer(gServer.Srv, v1Server)
	reflection.Register(gServer.Srv)

	// Serve gRPC server
	go func() {
		logger.Log.Info("starting grpc server on: " + cfg.APIPort)
		err = gServer.Srv.Serve(conn)
		if err != nil {
			logger.Log.Fatal("error serving gRPC server", zap.String("err", err.Error()))
		}
	}()

	// grpcGatewayMux := restserver.NewRESTServer()
	// dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConf))}
	// // Protobuf translation from incoming HTTP requests to grpc calls
	// if err := v1API.RegisterUnpaperServiceHandlerFromEndpoint(ctx, grpcGatewayMux, "127.0.0.1:"+cfg.APIPort, dialOpts); err != nil {
	// 	logger.Log.Error("error registering grpc gateway", zap.Error(err))
	// }

	// Set stripe API Key
	stripe.Key = cfg.StripeAPIKey

	// httpS := http.Server{
	// 	Addr:      "127.0.0.1:" + cfg.APIPort,
	// 	Handler:   restserver.CustomMIME(grpcGatewayMux),
	// 	TLSConfig: tlsConf,
	// }
	// go func() {
	// 	<-ctx.Done()
	// 	logger.Log.Info("Shutting down http gateway server")
	// 	if err = httpS.Shutdown(context.Background()); err != nil {
	// 		logger.Log.Error("Error shutting down http gateway server: %v", zap.Error(err))
	// 	}
	// }()
	// if err = httpS.ServeTLS(httpL, serverCertPEM, serverKeyPEM); err != http.ErrServerClosed {
	// 	return err
	// }
	block := make(chan struct{})
	<-block
	return nil
}

func loadCerts(serverCertF, serverKeyF, caCertF string) (*tls.Config, error) {
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair(serverCertF, serverKeyF)
	if err != nil {
		return nil, err
	}

	// Load certificate of the CA who signed client's certificate
	rootCAs := x509.NewCertPool()
	caCertP, err := ioutil.ReadFile(caCertF)
	if err != nil {
		return nil, err
	}
	rootCAs.AppendCertsFromPEM(caCertP)

	// Create TLS config
	return &tls.Config{
		RootCAs:      rootCAs,
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}
