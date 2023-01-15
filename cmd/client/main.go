package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/MrPark97/tages/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
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
				UpdatedAt: timestamppb.Now(),
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

func downloadImage(imageClient pb.ImageServiceClient, imageName string, imagePath string) {
	// setting up 5 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// prepare download image request
	req := &pb.DownloadImageRequest{Name: imageName}

	// trying to call DownloadImage method
	stream, err := imageClient.DownloadImage(ctx, req)
	if err != nil {
		log.Fatal("cannot download image: ", err)
	}

	// trying to get image info
	res, err := stream.Recv()
	if err != nil {
		log.Fatal("cannot receive response: ", err)
	}
	log.Printf("received image info - name: %s, type: %s, updated_at: %s", res.GetInfo().GetName(), res.GetInfo().GetType(), res.GetInfo().GetUpdatedAt())

	// init variables for bytes of image and image size
	imageData := bytes.Buffer{}
	imageSize := 0

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot receive response: ", err)
		}

		// getting chunk of data and calculating size
		chunk := res.GetChunkData()
		size := len(chunk)

		// counting image size
		imageSize += size

		// trying to write data to buffer
		_, err = imageData.Write(chunk)
		if err != nil {
			log.Fatal("cannot write chunk of data: ", err)
		}
	}

	// trying to create file
	file, err := os.Create(imagePath)
	if err != nil {
		log.Fatal("cannot create image file: ", err)
	}

	// trying to write data to file
	_, err = imageData.WriteTo(file)
	if err != nil {
		log.Fatal("cannot write image to file: ", err)
	}

	log.Printf("image downloaded with name: %s, size: %d", imageName, imageSize)
}

// function to upload test image
func testUploadImage(imageClient pb.ImageServiceClient) {
	uploadImage(imageClient, "laptop", "tmp/laptop.jpeg")
}

// function to download test image
func testDownloadImage(imageClient pb.ImageServiceClient) {
	uploadImage(imageClient, "laptop", "tmp/laptop.jpeg")
	downloadImage(imageClient, "laptop", "Downloads/laptop.jpeg")
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
	testDownloadImage(imageClient)
}
