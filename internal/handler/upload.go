package handler

import (
	"context"

	"volunteer-system/internal/response"
	"volunteer-system/internal/service"
	"volunteer-system/pkg/storage"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func Upload(ctx context.Context, c *app.RequestContext) {
	uploadType := c.PostForm("type")
	if uploadType == "" {
		response.Fail(c, service.ErrInvalidUploadType)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, err)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer file.Close()

	svc := service.NewUploadService(
		storage.GetStorage(),
		storage.GetMaxFileSizeMB(),
		storage.GetAllowedExtensions(),
	)
	url, err := svc.Upload(ctx, file, fileHeader.Filename, fileHeader.Size, uploadType)
	if err != nil {
		response.FailWithCode(c, consts.StatusBadRequest, err)
		return
	}
	response.Success(c, map[string]string{"url": url})
}
