package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/hertz-contrib/cors"
	"github.com/hertz-contrib/swagger"
	swaggerFiles "github.com/swaggo/files"
)

//go:embed openapi.yaml
var openapiYAML []byte

func main() {
	// Initialize Thrift file provider and generic client
	p, err := generic.NewThriftFileProvider("./hello.thrift")
	if err != nil {
		hlog.Fatal("Failed to create ThriftFileProvider:", err)
	}

	g, err := generic.HTTPThriftGeneric(p)
	if err != nil {
		hlog.Fatal("Failed to create HTTPThriftGeneric:", err)
	}

	cli, err := genericclient.NewClient("swagger", g, client.WithHostPorts("127.0.0.1:8888"))
	if err != nil {
		hlog.Fatal("Failed to create generic client:", err)
	}

	h := server.Default(
		server.WithHostPorts("127.0.0.1:8080"),
	)
	h.Use(cors.Default())
	//h.Use(cors.New(cors.Config{
	//	AllowOrigins:     []string{"*"},
	//	AllowMethods:     []string{"*"},
	//	AllowHeaders:     []string{"*"},
	//	ExposeHeaders:    []string{"Content-Length", "Authorization"},
	//	AllowCredentials: true,
	//}))
	h.GET("swagger/*any", swagger.WrapHandler(
		swaggerFiles.Handler,
		swagger.URL("/openapi.yaml"),
	))

	h.GET("/openapi.yaml", func(c context.Context, ctx *app.RequestContext) {
		ctx.Header("Content-Type", "application/x-yaml")
		ctx.Write(openapiYAML)
	})

	h.Any("/*ServiceMethod", func(c context.Context, ctx *app.RequestContext) {
		serviceMethod := ctx.Param("ServiceMethod")

		// 获取查询参数
		queryString := ctx.Request.URI().QueryString()

		// 获取请求体和Content-Type
		bodyBytes := ctx.Request.Body()
		contentType := string(ctx.Request.Header.ContentType())

		url := "http://localhost:1234/" + serviceMethod
		if len(queryString) > 0 {
			url += "?" + string(queryString)
		}
		req, err := http.NewRequest(string(ctx.Request.Method()), url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			hlog.Errorf("Failed to create HTTP request: %v", err)
			ctx.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create HTTP request",
			})
			return
		}

		// 将传入请求中的头信息设置到新请求中
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		// 设置请求的Content-Type
		req.Header.Set("Content-Type", contentType)

		customReq, err := generic.FromHTTPRequest(req)
		hlog.Info("Generic request: %v", customReq.Request)
		if err != nil {
			hlog.Errorf("Failed to create generic request: %v", err)
			ctx.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create generic request",
			})
			return
		}

		resp, err := cli.GenericCall(c, "", customReq)
		if err != nil {
			hlog.Errorf("GenericCall error: %v", err)
			ctx.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		realResp, ok := resp.(*generic.HTTPResponse)
		if !ok {
			hlog.Error("Response is not a generic.HTTPResponse:", resp)
			ctx.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"error": "Invalid response format",
			})
			return
		}
		hlog.Info("realResp:", realResp.StatusCode, realResp.ContentType, realResp.Body, realResp.Header)
		if realResp.StatusCode == 0 {
			realResp.StatusCode = consts.StatusOK
		}

		for key, values := range realResp.Header {
			for _, value := range values {
				ctx.Response.Header.Set(key, value)
			}
		}

		respBody, err := json.Marshal(realResp.Body)
		if err != nil {
			hlog.Errorf("Failed to marshal response body: %v", err)
			ctx.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to marshal response body",
			})
			return
		}

		ctx.Data(int(realResp.StatusCode), string(realResp.ContentType), respBody)
	})
	h.Spin()
}
