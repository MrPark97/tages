package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/MrPark97/tages/pb"
	"github.com/MrPark97/tages/service"
	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 8080, "the server port")
	flag.Parse()
	log.Printf("start server on port %d", *port)

	imageStore := service.NewDiskImageStore("img")

	imageServer := service.NewImageServer(imageStore)

	grpcServer := grpc.NewServer(
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
