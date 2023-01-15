// package service provides grpc methods to operate with images
package service

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/MrPark97/tages/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// maximum 1 megabyte
const maxImageSize = 1 << 20

// 1 kilobyte (just for test, optimal size is near 128 kB)
const bufferSize = 1024

// LaptopServer is the server that provides laptop services
type ImageServer struct {
	pb.UnimplementedImageServiceServer
	imageStore ImageStore
}

// NewImageServer returns a new ImageServer
func NewImageServer(imageStore ImageStore) *ImageServer {
	return &ImageServer{
		imageStore: imageStore,
	}
}

// UploadImage is a client-streaming RPC to upload an image
func (server *ImageServer) UploadImage(stream pb.ImageService_UploadImageServer) error {
	// trying to receive image info
	req, err := stream.Recv()
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot receive image info"))
	}

	// get image info from request
	imageName := req.GetInfo().GetName()
	imageType := req.GetInfo().GetType()
	imageUpdateTime := req.GetInfo().GetUpdatedAt().AsTime()
	log.Printf("receive an upload-image request with name %s and image type %s", imageName, imageType)

	// init variables for bytes of image and image size
	imageData := bytes.Buffer{}
	imageSize := 0

	for {
		// if there is context error returns it
		if err := contextError(stream.Context()); err != nil {
			return err
		}

		log.Print("waiting to receive more data")

		// trying to get data chunk by chunk, break if there is no more data
		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive chunk data: %v", err))
		}

		// getting chunk of data and calculating size
		chunk := req.GetChunkData()
		size := len(chunk)

		log.Printf("received a chunk with size: %d", size)

		// counting image size
		imageSize += size

		// if image size is too large return error
		if imageSize > maxImageSize {
			return logError(status.Errorf(codes.InvalidArgument, "image is too large: %d > %d", imageSize, maxImageSize))
		}

		// trying to write data to buffer
		_, err = imageData.Write(chunk)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data: %v", err))
		}
	}

	// trying to save image to disk and in-memory store
	imageName, err = server.imageStore.Save(imageName, imageType, imageData, imageUpdateTime)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot save image to the store: %v", err))
	}

	// forming response
	res := &pb.UploadImageResponse{
		Name: imageName,
		Size: uint32(imageSize),
	}

	// trying to send response
	err = stream.SendAndClose(res)
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot send response: %v", err))
	}

	log.Printf("saved the image with name: %s, size: %d", imageName, imageSize)

	return nil
}

// DownloadImage is a server-streaming RPC to download an image
func (server *ImageServer) DownloadImage(req *pb.DownloadImageRequest, stream pb.ImageService_DownloadImageServer) error {
	// get image name from request
	imageName := req.GetName()
	log.Printf("receive name of an image from request: %v", imageName)

	// trying to send image from storage
	err := server.imageStore.Send(
		stream,
		imageName,
		func(chunkData []byte) error {
			// forming response with chunk data
			res := &pb.DownloadImageResponse{
				Data: &pb.DownloadImageResponse_ChunkData{
					ChunkData: chunkData,
				},
			}

			// trying to send response
			err := stream.Send(res)
			if err != nil {
				return err
			}

			log.Printf("sent chunk of data of size %d bytes:", len(chunkData))
			return nil
		},
	)

	// if there is unexpected error return it
	if err != nil {
		return status.Errorf(codes.Internal, "unxepected error: %v", err)
	}

	return nil
}

// function to log and return error
func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}

// function to check if context canceled or deadline exceeded
func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Error(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}
