package service

import (
	"context"
	"strings"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor is an unary server interceptor that performs concurrency limiting.
func UnaryServerInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	// parse method name
	methodName := strings.Split(info.FullMethod, "/")[2]
	// write to the channel (blocking if reaching limit)
	if methodName == "GetUploadedImagesTableString" {
		info.Server.(*ImageServer).GetUploadedImagesTableStringLimitChannel <- struct{}{}
	}

	// calls the handler
	h, err := handler(ctx, req)

	// read from the channel (freeing one spot)
	if methodName == "GetUploadedImagesTableString" {
		<-info.Server.(*ImageServer).GetUploadedImagesTableStringLimitChannel
	}

	return h, err
}

// StreamServerInterceptor is a stream server interceptor that performs concurrency limiting.
func StreamServerInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	// parse method name
	methodName := strings.Split(info.FullMethod, "/")[2]
	// write to the channel (blocking if reaching limit)
	if methodName == "DownloadImage" {
		srv.(*ImageServer).DownloadImageLimitChannel <- struct{}{}
	} else if methodName == "UploadImage" {
		srv.(*ImageServer).UploadImageLimitChannel <- struct{}{}
	}

	// calls the handler
	err := handler(srv, ss)

	// read from the channel (freeing one spot)
	if methodName == "DownloadImage" {
		<-srv.(*ImageServer).DownloadImageLimitChannel
	} else if methodName == "UploadImage" {
		<-srv.(*ImageServer).UploadImageLimitChannel
	}

	return err
}
