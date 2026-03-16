package handler

import (
	"context"
	"io"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func ImportVolunteers(ctx context.Context, c *app.RequestContext) {
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
	defer func() {
		_ = file.Close()
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewImportService(ctx, c).ImportVolunteers(fileHeader.Filename, content)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func ImportActivities(ctx context.Context, c *app.RequestContext) {
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
	defer func() {
		_ = file.Close()
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewImportService(ctx, c).ImportActivities(fileHeader.Filename, content)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}
