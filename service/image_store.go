package service

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"
)

// ImageStore is an interface to store images
type ImageStore interface {
	// Save saves a new image to the store
	Save(imageName string, imageType string, imageData bytes.Buffer, imageUpdateTime time.Time) (string, error)
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

	// upsert image info
	store.mutex.Lock()
	defer store.mutex.Unlock()

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

	store.images[imageName] = imageInfo

	return imageName, nil
}
