package service_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrPark97/tages/pb"
	"github.com/MrPark97/tages/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestClientUploadImage(t *testing.T) {
	t.Parallel()

	testImageFolder := "../tmp"
	imageStore := service.NewDiskImageStore(testImageFolder)

	serverAddress := startTestImageServer(t, imageStore)
	imageClient := newTestImageClient(t, serverAddress)

	imagePath := fmt.Sprintf("%s/laptop.jpeg", testImageFolder)
	file, err := os.Open(imagePath)
	require.NoError(t, err)
	defer file.Close()

	stream, err := imageClient.UploadImage(context.Background())
	require.NoError(t, err)

	imageType := filepath.Ext(imagePath)

	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.Info{
				Name: "laptop",
				Type: imageType,
			},
		},
	}

	err = stream.Send(req)
	require.NoError(t, err)

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	size := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		size += n

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)
	}

	res, err := stream.CloseAndRecv()
	require.NoError(t, err)

	require.NotEmpty(t, res.GetName())
	require.EqualValues(t, size, res.GetSize())

	savedImagePath := fmt.Sprintf("%s/%s%s", testImageFolder, res.GetName(), imageType)
	require.FileExists(t, savedImagePath)
}

func TestClientDownloadImage(t *testing.T) {
	t.Parallel()

	testImageFolder := "../tmp"
	imageStore := service.NewDiskImageStore(testImageFolder)

	serverAddress := startTestImageServer(t, imageStore)
	imageClient := newTestImageClient(t, serverAddress)

	imagePath := fmt.Sprintf("%s/laptop.jpeg", testImageFolder)
	file, err := os.Open(imagePath)
	require.NoError(t, err)
	defer file.Close()

	stream, err := imageClient.UploadImage(context.Background())
	require.NoError(t, err)

	imageType := filepath.Ext(imagePath)
	imageName := "laptop"

	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.Info{
				Name: imageName,
				Type: imageType,
			},
		},
	}

	err = stream.Send(req)
	require.NoError(t, err)

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	size := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		size += n

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)
	}

	res, err := stream.CloseAndRecv()
	require.NoError(t, err)

	require.NotEmpty(t, res.GetName())
	require.EqualValues(t, size, res.GetSize())

	savedImagePath := fmt.Sprintf("%s/%s%s", testImageFolder, res.GetName(), imageType)
	require.FileExists(t, savedImagePath)

	// prepare download image request
	req1 := &pb.DownloadImageRequest{Name: imageName}

	// trying to call DownloadImage method
	stream1, err := imageClient.DownloadImage(context.Background(), req1)
	require.NoError(t, err)

	// trying to get image info
	_, err = stream1.Recv()
	require.NoError(t, err)

	// init variables for bytes of image and image size
	imageData := bytes.Buffer{}
	imageSize := 0

	for {
		res1, err := stream1.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		// getting chunk of data and calculating size
		chunk := res1.GetChunkData()
		size := len(chunk)

		// counting image size
		imageSize += size

		// trying to write data to buffer
		_, err = imageData.Write(chunk)
		require.NoError(t, err)
	}

	// trying to create file
	file, err = os.Create(imagePath)
	require.NoError(t, err)

	// trying to write data to file
	_, err = imageData.WriteTo(file)
	require.NoError(t, err)

	require.EqualValues(t, size, imageSize)
}

func TestClientGetUploadedImagesTableString(t *testing.T) {
	t.Parallel()

	imageStore := service.NewDiskImageStore("../tmp")
	serverAddress := startTestImageServer(t, imageStore)
	imageClient := newTestImageClient(t, serverAddress)

	req := &pb.GetUploadedImagesTableStringRequest{
		Limit: uint32(0),
	}

	res, err := imageClient.GetUploadedImagesTableString(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "Имя файла | Дата создания       | Дата обновления\n", res.GetTable())
}

func newTestImageClient(t *testing.T, serverAddress string) pb.ImageServiceClient {
	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
	require.NoError(t, err)
	return pb.NewImageServiceClient(conn)
}

func startTestImageServer(t *testing.T, imageStore service.ImageStore) string {
	imageServer := service.NewImageServer(imageStore)

	grpcServer := grpc.NewServer()
	pb.RegisterImageServiceServer(grpcServer, imageServer)

	listener, err := net.Listen("tcp", ":0") // random available port
	require.NoError(t, err)

	go grpcServer.Serve(listener) // block call

	return listener.Addr().String()
}
