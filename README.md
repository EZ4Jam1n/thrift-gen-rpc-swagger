# thrift-gen-rpc-swagger

English | [中文](README_CN.md)

HTTP Swagger document generation plugin for cloudwego/cwgo & hertz.

## Supported hz Annotations

### Request Specification

1. Interface request fields need to be associated with a certain type of HTTP parameter and parameter name using annotations. Fields without annotations will not be processed.
2. Generate the `parameters` and `requestBody` of the `operation` in Swagger according to the request `message` in the `method`.
3. If the HTTP request uses the `GET`, `HEAD`, or `DELETE` methods, the `api.body` annotation in the `request` definition is invalid, and only `api.query`, `api.path`, `api.cookie`, `api.header` are valid.

#### Annotation Explanation

| Annotation     | Explanation                                                                            |  
|----------------|----------------------------------------------------------------------------------------|
| `api.query`    | `api.query` corresponds to `parameter` with `in: query`                                |  
| `api.path`     | `api.path` corresponds to `parameter` with `in: path`, required is true                |
| `api.header`   | `api.header` corresponds to `parameter` with `in: header`                              |       
| `api.cookie`   | `api.cookie` corresponds to `parameter` with `in: cookie`                              |
| `api.body`     | `api.body` corresponds to `requestBody` with `content`: `application/json`             | 
| `api.form`     | `api.form` corresponds to `requestBody` with `content`: `multipart/form-data`          | 
| `api.raw_body` | `api.raw_body` corresponds to `requestBody` with `content`: `application/octet-stream` | 

### Response Specification

1. Interface response fields need to be associated with a certain type of HTTP parameter and parameter name using annotations. Fields without annotations will not be processed.
2. Generate the `responses` of the `operation` in Swagger according to the response `message` in the `method`.

#### Annotation Explanation

| Annotation     | Explanation                                                                         |  
|----------------|-------------------------------------------------------------------------------------|
| `api.header`   | `api.header` corresponds to `response` with `header`                                |
| `api.body`     | `api.body` corresponds to `response` with `content`: `application/json`             |
| `api.raw_body` | `api.raw_body` corresponds to `response` with `content`: `application/octet-stream` |

### Method Specification

1. Each `method` is associated with a `pathItem` through an annotation.

#### Annotation Explanation

| Annotation    | Explanation                                                   |  
|---------------|---------------------------------------------------------------|
| `api.get`     | `api.get` corresponds to GET request, only `parameters `      |
| `api.put`     | `api.put` corresponds to PUT request                          |
| `api.post`    | `api.post` corresponds to POST request                        |
| `api.patch`   | `api.patch` corresponds to PATCH request                      |
| `api.delete`  | `api.delete` corresponds to DELETE request, only `parameters` |
| `api.options` | `api.options` corresponds to OPTIONS request                  |
| `api.head`    | `api.head` corresponds to HEAD request, only `parameters`     |
| `api.baseurl` | `api.baseurl` corresponds to `server` `url` of `pathItem`     |

### Service Specification

#### Annotation Explanation

| Annotation        | Explanation                                     |  
|-------------------|-------------------------------------------------|
| `api.base_domain` | `api.base_domain` corresponds to `server` `url` |

## openapi Annotations

| Annotation          | Component | Explanation                                                                        |  
|---------------------|-----------|------------------------------------------------------------------------------------|
| `openapi.operation` | Method    | Used to supplement the `operation` of `pathItem`                                   |
| `openapi.property`  | Field     | Used to supplement the `property` of `schema`                                      |
| `openapi.schema`    | Struct    | Used to supplement the `schema` of `requestBody` and `response`                    |
| `openapi.document`  | Service   | Used to supplement the Swagger document, simply add this annotation in any service |
| `openapi.parameter` | Field     | Used to supplement the `parameter`                                                 |

For more usage, please refer to [Example](example/hello.thrift).

## Installation

```sh
# Install from the official repository

git clone https://github.com/hertz-contrib/swagger-generate
cd thrift-gen-rpc-swagger
go install

# Install directly
go install github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger@latest

# Verify the installation
thrift-gen-rpc-swagger --version
```

## Usage

```sh

thriftgo -g go -p rpc-swagger:OutputDir={swagger documentation output directory} {idl}.thrift

```

## More info

See [examples](example/hello.thrift)