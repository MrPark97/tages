package service

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/MrPark97/tages/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ImageStore is an interface to store images
type ImageStore interface {
	// Save saves a new image to the store
	Save(imageName string, imageType string, imageData bytes.Buffer, imageUpdateTime time.Time) (string, error)
	// Send sends an existing image to the client from the store
	Send(stream pb.ImageService_DownloadImageServer, imageName string, send func(chunkData []byte) error) error
	// String forms an uploaded images table string from the store
	String(limit uint32) string
}

// DiskImageStore is a struct to store images on disk
type DiskImageStore struct {
	mutex       sync.RWMutex
	imageFolder string
	images      map[string]*ImageInfo
}

// ImageInfo is a struct to operate information about image
type ImageInfo struct {
	ImageName string
	Type      string
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewDiskImageStore returns a new DiskImageStore
func NewDiskImageStore(imageFolder string) *DiskImageStore {
	return &DiskImageStore{
		imageFolder: imageFolder,
		images:      make(map[string]*ImageInfo),
	}
}

// Save saves a new image to the store
func (store *DiskImageStore) Save(imageName string, imageType string, imageData bytes.Buffer, imageUpdateTime time.Time) (string, error) {
	// formatting image path string
	imagePath := fmt.Sprintf("%s/%s%s", store.imageFolder, imageName, imageType)

	// lock store
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// trying to create file
	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("cannot create image file: %w", err)
	}

	// trying to write data to file
	_, err = imageData.WriteTo(file)
	if err != nil {
		return "", fmt.Errorf("cannot write image to file: %w", err)
	}

	// prepare image info to update in-memory storage value
	imageInfo := &ImageInfo{
		ImageName: imageName,
		Type:      imageType,
		Path:      imagePath,
		UpdatedAt: imageUpdateTime,
	}

	// check if image not exists
	if store.images[imageName] == nil {
		imageInfo.CreatedAt = imageUpdateTime
	} else {
		imageInfo.CreatedAt = store.images[imageName].CreatedAt
	}

	// update in-memory storage value
	store.images[imageName] = imageInfo

	return imageName, nil
}

// Send sends image info first, then sends chunks of data one by one via send function
func (store *DiskImageStore) Send(stream pb.ImageService_DownloadImageServer, imageName string, send func(chunkData []byte) error) error {
	// lock in-memory storage
	store.mutex.Lock()
	defer store.mutex.Unlock()

	ctx := stream.Context()

	// check for context error
	if err := contextError(ctx); err != nil {
		return err
	}

	// if no image with such name return invalid argument error
	imageInfo := store.images[imageName]
	if imageInfo == nil {
		return logError(status.Errorf(codes.InvalidArgument, "image doesn't exists: %v", imageName))
	}

	// forming response with image info
	res := &pb.DownloadImageResponse{
		Data: &pb.DownloadImageResponse_Info{
			Info: &pb.Info{
				Name:      imageName,
				Type:      imageInfo.Type,
				UpdatedAt: timestamppb.New(imageInfo.UpdatedAt),
			},
		},
	}

	// trying to send response
	err := stream.Send(res)
	if err != nil {
		return err
	}

	// formatting image path string
	imagePath := fmt.Sprintf("%s/%s%s", store.imageFolder, imageName, imageInfo.Type)

	// trying to open image
	file, err := os.Open(imagePath)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot open image file: %v", imagePath))
	}
	defer file.Close()

	// init reader and buffer
	reader := bufio.NewReader(file)
	buffer := make([]byte, bufferSize)
	for {
		// if there is context error returns it
		if err := contextError(stream.Context()); err != nil {
			return err
		}

		// trying to read n bytest to buffer
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			logError(status.Errorf(codes.Internal, "cannot read chunk to buffer: %v", err))
		}

		err = send(buffer[:n])
		if err != nil {
			return err
		}
	}

	return nil
}

// String forms an uploaded images table string from the store
func (store *DiskImageStore) String(limit uint32) string {
	// lock in-memory storage
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// if there is no limit set it as store length
	storeLen := len(store.images)
	if limit == uint32(0) {
		limit = uint32(storeLen)
	}

	// find string with maximal length (for space padding)
	baseFullNameLen := utf8.RuneCountInString("?????? ?????????? ")
	maxFullNameLen := baseFullNameLen
	i := uint32(0)
	for imageName, imageInfo := range store.images {
		if i >= limit {
			break
		}
		if curFullNameLen := utf8.RuneCountInString(imageName) + utf8.RuneCountInString(imageInfo.Type); curFullNameLen > maxFullNameLen {
			maxFullNameLen = curFullNameLen
		}
		i += uint32(1)
	}

	// generate table string using calculated value
	imagesTableString := "?????? ?????????? " + strings.Repeat(" ", maxFullNameLen-baseFullNameLen) + "| ???????? ????????????????       | ???????? ????????????????????\n"
	i = uint32(0)
	fullName := ""
	for imageName, imageInfo := range store.images {
		if i >= limit {
			break
		}

		fullName = imageName + imageInfo.Type

		imagesTableString += fullName + strings.Repeat(" ", maxFullNameLen-utf8.RuneCountInString(fullName)) + "| " + fmt.Sprintf(imageInfo.CreatedAt.Local().Format("02.01.2006 15:04:05")) + " | " + fmt.Sprintf(imageInfo.UpdatedAt.Local().Format("02.01.2006 15:04:05")) + "\n"
		i += uint32(1)
	}

	return imagesTableString
}
