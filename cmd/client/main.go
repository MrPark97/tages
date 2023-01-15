package main

import (
	"bufio"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/MrPark97/tages/pb"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

func uploadImage(imageClient pb.ImageServiceClient, imageName string, imagePath string) {
	// trying to open image
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open image file: ", err)
	}
	defer file.Close()

	// setting up 5 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// trying to call UploadImage method
	stream, err := imageClient.UploadImage(ctx)
	if err != nil {
		log.Fatal("cannot upload image: ", err)
	}

	// forming request
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.Info{
				Name:      imageName,
				Type:      filepath.Ext(imagePath),
				UpdatedAt: ptypes.TimestampNow(),
			},
		},
	}

	// send image info request
	err = stream.Send(req)
	if err != nil {
		log.Fatal("cannot send image info: ", err, stream.RecvMsg(nil))
	}

	// init reader and buffer
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	for {
		// trying to read n bytest to buffer
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer: ", err)
		}

		// forming request
		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		// trying to send request
		err = stream.Send(req)
		if err != nil {
			log.Fatal("cannot send chunk to server: ", err, stream.RecvMsg(nil))
		}
	}

	// trying to get response
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive response: ", err)
	}

	log.Printf("image uploaded with name: %s, size: %d", res.GetName(), res.GetSize())
}

// function to upload test image
func testUploadImage(imageClient pb.ImageServiceClient) {
	uploadImage(imageClient, "laptop", "tmp/laptop.jpeg")
}

func main() {
	serverAddress := flag.String("address", "", "the server address")
	flag.Parse()
	log.Printf("dial server %s", *serverAddress)

	conn, err := grpc.Dial(*serverAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}

	imageClient := pb.NewImageServiceClient(conn)
	testUploadImage(imageClient)
}
