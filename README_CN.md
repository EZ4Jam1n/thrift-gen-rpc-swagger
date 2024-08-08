# thrift-gen-rpc-swagger

[English](README.md) | 中文

适用于 cloudwego/cwgo & kitex 的 rpc swagger 文档生成插件。

## 支持的注解

### Request 规范

1. 接口请求字段需要使用注解关联到 HTTP 的某类参数和参数名称, 没有注解的字段不做处理。
2. 根据 `method` 中的请求 `message` 生成 swagger 中 `operation` 的 `parameters` 和 `requestBody`。
3. 如果 HTTP 请求是采用 `GET`、`HEAD`、`DELETE` 方式的，那么 `request` 定义中出现的 `api.body` 注解无效，只有`api.query`, `api.path`, `api.cookie`, `api.header` 有效。

#### 注解说明

| 注解           | 说明                                                                        |  
|--------------|---------------------------------------------------------------------------|
| `api.query`  | `api.query` 对应 `parameter` 中 `in: query` 参数, 支持基本类型和list(object, map暂不支持） |  
| `api.path`   | `api.path` 对应 `parameter` 中 `in: path` 参数, `required` 为 `true`, 支持基本类型    |
| `api.header` | `api.header` 对应 `parameter` 中 `in: header` 参数, 支持基本类型                     |       
| `api.cookie` | `api.cookie` 对应 `parameter` 中 `in: cookie` 参数, 支持基本类型                     |
| `api.body`   | `api.body` 对应 `requestBody` 中 `content` 为 `application/json`              |

### Response 规范

1. 接口响应字段需要使用注解关联到 HTTP 的某类参数和参数名称, 没有注解的字段不做处理。
2. 根据 `method` 中的响应 `message` 生成 swagger 中 `operation` 的 `responses`。

#### 注解说明

| 注解             | 说明                                                                        |  
|----------------|---------------------------------------------------------------------------|
| `api.header`   | `api.header` 对应 `response` 中 `header`, 只支持基本类型和逗号分隔的list                  |
| `api.body`     | `api.body` 对应 `response` 中 `content` 为 `application/json`                 |
| `api.raw_body` | `api.raw_body` 对应 `response` 中 `content` 为 `application/octet-stream`,待检验https://www.cloudwego.io/zh/docs/kitex/tutorials/advanced-feature/generic-call/thrift_idl_annotation_standards/ |

### Method 规范

1. 每个 `method` 通过注解来关联 `pathItem`

#### 注解说明

| 注解            | 说明                                             |  
|---------------|------------------------------------------------|
| `api.get`     | `api.get` 对应 `GET` 请求，只有 `parameter`           |
| `api.put`     | `api.put` 对应 `PUT` 请求                          |
| `api.post`    | `api.post` 对应 `POST` 请求                        |
| `api.patch`   | `api.patch` 对应 `PATCH` 请求                      |
| `api.delete`  | `api.delete` 对应 `DELETE` 请求，只有 `parameter`     |
| `api.baseurl` | `api.baseurl` 对应 `pathItem` 的 `server` 的 `url` |

### Service 规范

#### 注解说明

| 注解                | 说明                                    |  
|-------------------|---------------------------------------|
| `api.base_domain` | `api.base_domain` 对应 `server` 的 `url` |

## openapi 注解

| 注解                  | 使用组件    | 说明                                         |  
|---------------------|---------|--------------------------------------------|
| `openapi.operation` | Method  | 用于补充 `pathItem` 的 `operation`              |
| `openapi.property`  | Field   | 用于补充 `schema` 的 `property`                 |
| `openapi.schema`    | Struct  | 用于补充 `requestBody` 和 `response` 的 `schema` |
| `openapi.document`  | Service | 用于补充 swagger 文档，任意service中添加该注解即可          |
| `openapi.parameter` | Field   | 用于补充 `parameter`                           |

更多的使用方法请参考 [示例](example/hello.thrift)

## 安装

```sh

# 官方仓库安装

git clone https://github.com/hertz-contrib/swagger-generate
cd thrift-gen-rpc-swagger
go install

# 直接安装
go install github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger@latest

# 验证安装
thrift-gen-rpc-swagger --version
```

## 使用

```sh

thriftgo -g go -p rpc-swagger:OutputDir={swagger文档输出目录} {idl}.thrift

```

## 更多信息

查看 [示例](example/hello.thrift)




