package router

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const swaggerSpecPath = "/swagger/openapi.yaml"

func swaggerIndexHTML(specURL string) []byte {
	return []byte(fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Volunteer System API Docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "%s",
      dom_id: "#swagger-ui",
      deepLinking: true,
      docExpansion: "list"
    });
  </script>
</body>
</html>`, specURL))
}

// RegisterSwaggerUI registers Swagger UI routes backed by docs/openapi.yaml.
func RegisterSwaggerUI(r *server.Hertz) {
	r.GET("/swagger", func(ctx context.Context, c *app.RequestContext) {
		c.Redirect(consts.StatusMovedPermanently, []byte("/swagger/"))
	})
	r.GET("/swagger/", func(ctx context.Context, c *app.RequestContext) {
		c.Data(consts.StatusOK, "text/html; charset=utf-8", swaggerIndexHTML(swaggerSpecPath))
	})
	r.GET("/swagger/index.html", func(ctx context.Context, c *app.RequestContext) {
		c.Data(consts.StatusOK, "text/html; charset=utf-8", swaggerIndexHTML(swaggerSpecPath))
	})
	r.GET(swaggerSpecPath, func(ctx context.Context, c *app.RequestContext) {
		c.File("docs/openapi.yaml")
	})
}
