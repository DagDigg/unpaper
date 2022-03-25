package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

func loadTLSCreds() (credentials.TransportCredentials, error) {
	serverCA, err := ioutil.ReadFile("pkg/certificate/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(serverCA) {
		return nil, fmt.Errorf("cannot create certificate pool")
	}

	config := &tls.Config{
		RootCAs: certPool,
	}

	return credentials.NewTLS(config), nil
}

func runTLS() {
	creds, err := loadTLSCreds()
	if err != nil {
		log.Fatalf("error contructing TLS credentials: %s", err)
	}

	conn, err := grpc.Dial("localhost:10000", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("error dialing to localhost:10000: %s", err)
	}

	client := v1API.NewUnpaperServiceClient(conn)
	var header metadata.MD
	ping, err := client.Ping(context.Background(), &v1API.PingRequest{}, grpc.Header(&header))

	if err != nil {
		log.Fatalf("error during healthcheck call: %s", err)
	}

	log.Fatalf("response: %s", ping.String())
}

func runHTTPS() {
	caCert, err := ioutil.ReadFile("pkg/certificate/localhost-cert.pem")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	r, err := client.Get("https://localhost:8443/h1/v1/ping")
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	_, err = ioutil.ReadAll(r.Body)
}

func main() {
	runTLS()
}
