package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/MrPark97/tages/pb"
	"github.com/MrPark97/tages/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		return nil, err
	}

	// Load certificate of the CA who signed client's certificate
	pemClientCA, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, errors.New("failed to add client's CA certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	return credentials.NewTLS(config), nil
}

func main() {
	port := flag.Int("port", 8080, "the server port")
	flag.Parse()
	log.Printf("start server on port %d", *port)

	imageStore := service.NewDiskImageStore("img")

	imageServer := service.NewImageServer(imageStore)

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(tlsCredentials),
		grpc.UnaryInterceptor(service.UnaryServerInterceptor),
		grpc.StreamInterceptor(service.StreamServerInterceptor),
	)
	pb.RegisterImageServiceServer(grpcServer, imageServer)

	address := fmt.Sprintf("0.0.0.0:%d", *port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("cannot start server: ", err)
	}

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("cannot start server: ", err)
	}
}
