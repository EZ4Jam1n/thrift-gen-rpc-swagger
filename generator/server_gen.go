/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package generator

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/plugin"
	"github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger/args"
)

type ServerGenerator struct {
	IdlPath   string
	HostPort  string
	OutputDir string
}

func NewServerGenerator(ast *parser.Thrift, args *args.Arguments) *ServerGenerator {
	return &ServerGenerator{
		IdlPath:   ast.Filename,
		HostPort:  args.HostPort,
		OutputDir: args.OutputDir,
	}
}

func (g *ServerGenerator) Generate() []*plugin.Generated {
	tmpl, err := template.New("server").Delims("{{", "}}").Parse(serverTemplate)
	if err != nil {
		fmt.Sprintf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, g)
	if err != nil {
		fmt.Sprintf("failed to execute template: %v", err)
	}

	filePath := filepath.Clean(g.OutputDir)
	filePath = filepath.Join(filePath, "swagger.go")

	var ret []*plugin.Generated
	ret = append(ret, &plugin.Generated{
		Content: buf.String(),
		Name:    &filePath,
	})

	return ret
}

const serverTemplate = `package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
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
	p, err := generic.NewThriftFileProvider("{{.IdlPath}}")
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
		server.WithHostPorts("{{.HostPort}}"),
	)
	h.Use(cors.Default())

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

		// Get query parameters
		rawQueryParams := ctx.Request.URI().QueryArgs()
		formattedQueryParams := make(map[string][]string)

		rawQueryParams.VisitAll(func(key, value []byte) {
			k := string(key)
			v := string(value)
			formattedQueryParams[k] = append(formattedQueryParams[k], v)
		})

		// Format query parameters
		var newQueryParams []string
		for k, v := range formattedQueryParams {
			newQueryParams = append(newQueryParams, k+"="+strings.Join(v, ","))
		}
		queryString := strings.Join(newQueryParams, "&")

		// Get request body and Content-Type
		bodyBytes := ctx.Request.Body()
		contentType := string(ctx.Request.Header.ContentType())

		url := "http://localhost:1234/" + serviceMethod
		if len(queryString) > 0 {
			url += "?" + string(queryString)
		}
		req, err := http.NewRequest(string(ctx.Request.Method()), url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			hlog.Errorf("Failed to create HTTP request: %v", err)
			ctx.JSON(500, map[string]interface{}{
				"error": "Failed to create HTTP request",
			})
			return
		}

		// Set headers from original request to new request
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		// Set the Content-Type
		req.Header.Set("Content-Type", contentType)

		customReq, err := generic.FromHTTPRequest(req)
		if err != nil {
			hlog.Errorf("Failed to create generic request: %v", err)
			ctx.JSON(500, map[string]interface{}{
				"error": "Failed to create generic request",
			})
			return
		}

		resp, err := cli.GenericCall(c, "", customReq)
		if err != nil {
			hlog.Errorf("GenericCall error: %v", err)
			ctx.JSON(500, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		
		// Check if the response is nil or empty
		if resp == nil {
			hlog.Error("Received nil response")
			ctx.JSON(500, map[string]interface{}{
				"error": "Received nil response from the service",
			})
			return
		}

		realResp, ok := resp.(*generic.HTTPResponse)
		if !ok {
			hlog.Error("Response is not a generic.HTTPResponse:", resp)
			ctx.JSON(500, map[string]interface{}{
				"error": "Invalid response format",
			})
			return
		}
		
		if realResp.StatusCode == 0 {
			realResp.StatusCode = 200
		}

		for key, values := range realResp.Header {
			for _, value := range values {
				ctx.Response.Header.Set(key, value)
			}
		}

		respBody, err := json.Marshal(realResp.Body)
		if err != nil {
			hlog.Errorf("Failed to marshal response body: %v", err)
			ctx.JSON(500, map[string]interface{}{
				"error": "Failed to marshal response body",
			})
			return
		}

		ctx.Data(int(realResp.StatusCode), string(realResp.ContentType), respBody)
	})
	h.Spin()
}
`
