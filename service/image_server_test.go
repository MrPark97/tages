package service_test

import (
	"context"
	"testing"

	"github.com/MrPark97/tages/pb"
	"github.com/MrPark97/tages/service"
	"github.com/stretchr/testify/require"
)

func TestServerGetUploadedImagesTableString(t *testing.T) {
	t.Parallel()

	imageStore := service.NewDiskImageStore("../tmp")
	server := service.NewImageServer(imageStore)
	req := &pb.GetUploadedImagesTableStringRequest{
		Limit: uint32(0),
	}
	res, err := server.GetUploadedImagesTableString(context.Background(), req)
	require.NoError(t, err)
	require.EqualValues(t, res.GetTable(), "Имя файла | Дата создания       | Дата обновления\n")
}
