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
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/plugin"
	"github.com/cloudwego/thriftgo/thrift_reflection"
	"github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger/args"
	openapi "github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger/thrift"
	"github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger/utils"
)

const (
	infoURL = "https://github.com/hertz-contrib/swagger-generate/thrift-gen-rpc-swagger"
)

type OpenAPIv3Generator struct {
	fileDesc          *thrift_reflection.FileDescriptor
	ast               *parser.Thrift
	generatedSchemas  []string
	requiredSchemas   []string
	commentPattern    *regexp.Regexp
	linterRulePattern *regexp.Regexp
}

// NewOpenAPIv3Generator creates a new generator for a protoc plugin invocation.
func NewOpenAPIv3Generator(ast *parser.Thrift) *OpenAPIv3Generator {
	_, fileDesc := thrift_reflection.RegisterAST(ast)
	return &OpenAPIv3Generator{
		fileDesc:          fileDesc,
		ast:               ast,
		generatedSchemas:  make([]string, 0),
		commentPattern:    regexp.MustCompile(`//\s*(.*)|/\*([\s\S]*?)\*/`),
		linterRulePattern: regexp.MustCompile(`\(-- .* --\)`),
	}
}

func (g *OpenAPIv3Generator) BuildDocument(arguments *args.Arguments) []*plugin.Generated {
	d := &openapi.Document{}

	version := "3.0.3"
	d.Openapi = version
	d.Info = &openapi.Info{
		Title:       "API",
		Description: "API description",
		Version:     "1.0.0",
	}
	d.Paths = &openapi.Paths{}
	d.Components = &openapi.Components{
		Schemas: &openapi.SchemasOrReferences{
			AdditionalProperties: []*openapi.NamedSchemaOrReference{},
		},
	}

	var extDocument *openapi.Document
	g.getDocumentOption(&extDocument)
	if extDocument != nil {
		utils.MergeStructs(d, extDocument)
	}

	g.addPathsToDocument(d, g.ast.Services)

	// While we have required schemas left to generate, go through the files again
	// looking for the related message and adding them to the document if required.
	for len(g.requiredSchemas) > 0 {
		count := len(g.requiredSchemas)
		g.addSchemasForStructsToDocument(d, g.ast.GetStructLikes())
		g.requiredSchemas = g.requiredSchemas[count:len(g.requiredSchemas)]
	}

	// If there is only 1 service, then use it's title for the
	// document, if the document is missing it.
	if len(d.Tags) == 1 {
		if d.Info.Title == "" && d.Tags[0].Name != "" {
			d.Info.Title = d.Tags[0].Name + " API"
		}
		if d.Info.Description == "" {
			d.Info.Description = d.Tags[0].Description
		}
		d.Tags[0].Description = ""
	}

	var allServers []string

	// If paths methods has servers, but they're all the same, then move servers to path level
	for _, path := range d.Paths.Path {
		var servers []string
		// Only 1 server will ever be set, per method, by the generator

		if path.Value.Get != nil && len(path.Value.Get.Servers) == 1 {
			servers = utils.AppendUnique(servers, path.Value.Get.Servers[0].URL)
			allServers = utils.AppendUnique(allServers, path.Value.Get.Servers[0].URL)
		}
		if path.Value.Post != nil && len(path.Value.Post.Servers) == 1 {
			servers = utils.AppendUnique(servers, path.Value.Post.Servers[0].URL)
			allServers = utils.AppendUnique(allServers, path.Value.Post.Servers[0].URL)
		}
		if path.Value.Put != nil && len(path.Value.Put.Servers) == 1 {
			servers = utils.AppendUnique(servers, path.Value.Put.Servers[0].URL)
			allServers = utils.AppendUnique(allServers, path.Value.Put.Servers[0].URL)
		}
		if path.Value.Delete != nil && len(path.Value.Delete.Servers) == 1 {
			servers = utils.AppendUnique(servers, path.Value.Delete.Servers[0].URL)
			allServers = utils.AppendUnique(allServers, path.Value.Delete.Servers[0].URL)
		}
		if path.Value.Patch != nil && len(path.Value.Patch.Servers) == 1 {
			servers = utils.AppendUnique(servers, path.Value.Patch.Servers[0].URL)
			allServers = utils.AppendUnique(allServers, path.Value.Patch.Servers[0].URL)
		}

		if len(servers) == 1 {
			path.Value.Servers = []*openapi.Server{{URL: servers[0]}}

			if path.Value.Get != nil {
				path.Value.Get.Servers = nil
			}
			if path.Value.Post != nil {
				path.Value.Post.Servers = nil
			}
			if path.Value.Put != nil {
				path.Value.Put.Servers = nil
			}
			if path.Value.Delete != nil {
				path.Value.Delete.Servers = nil
			}
			if path.Value.Patch != nil {
				path.Value.Patch.Servers = nil
			}
		}
	}

	// Set all servers on API level
	if len(allServers) > 0 {
		d.Servers = []*openapi.Server{}
		for _, server := range allServers {
			d.Servers = append(d.Servers, &openapi.Server{URL: server})
		}
	}

	// If there is only 1 server, we can safely remove all path level servers
	if len(allServers) == 1 {
		for _, path := range d.Paths.Path {
			path.Value.Servers = nil
		}
	}
	// Sort the tags.
	{
		pairs := d.Tags
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Name < pairs[j].Name
		})
		d.Tags = pairs
	}
	// Sort the paths.
	{
		pairs := d.Paths.Path
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Name < pairs[j].Name
		})
		d.Paths.Path = pairs
	}
	// Sort the schemas.
	{
		pairs := d.Components.Schemas.AdditionalProperties
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Name < pairs[j].Name
		})
		d.Components.Schemas.AdditionalProperties = pairs
	}

	bytes, err := d.YAMLValue("Generated with thrift-gen-http-swagger\n" + infoURL)
	if err != nil {
		fmt.Printf("Error converting to yaml: %s\n", err)
	}
	filePath := filepath.Clean(arguments.OutputDir)
	filePath = filepath.Join(filePath, "openapi.yaml")
	var ret []*plugin.Generated
	ret = append(ret, &plugin.Generated{
		Content: string(bytes),
		Name:    &filePath,
	})

	return ret
}

func (g *OpenAPIv3Generator) getDocumentOption(obj interface{}) error {
	serviceOrStruct, name := g.getDocumentAnnotationInWhichServiceOrStruct()
	if serviceOrStruct == "service" {
		serviceDesc := g.fileDesc.GetServiceDescriptor(name)
		utils.ParseServiceOption(serviceDesc, OpenapiDocument, obj)
	} else if serviceOrStruct == "struct" {
		structDesc := g.fileDesc.GetStructDescriptor(name)
		utils.ParseStructOption(structDesc, OpenapiDocument, obj)
	}
	return nil
}

func (g *OpenAPIv3Generator) addPathsToDocument(d *openapi.Document, services []*parser.Service) {
	for _, s := range services {

		annotationsCount := 0
		for _, f := range s.Functions {
			comment := g.filterCommentString(f.ReservedComments)

			operationID := s.GetName() + "_" + f.GetName()
			rs := utils.GetAnnotations(f.Annotations, HttpMethodAnnotations)
			if len(rs) == 0 {
				continue
			}

			var inputDesc *thrift_reflection.StructDescriptor
			if len(f.Arguments) >= 1 {
				if len(f.Arguments) > 1 {
					logs.Warnf("function '%s' has more than one argument, but only the first can be used in hertz now", f.GetName())
				}
				inputDesc = g.fileDesc.GetStructDescriptor(f.GetArguments()[0].GetType().GetName())
			}
			outputDesc := g.fileDesc.GetStructDescriptor(f.GetFunctionType().GetName())
			for methodName, path := range rs {
				if methodName != "" {
					annotationsCount++
					var host string

					hostOrNil := utils.GetAnnotation(f.Annotations, ApiBaseURL)

					if len(hostOrNil) != 0 {
						host = utils.GetAnnotation(f.Annotations, ApiBaseURL)[0]
					}

					if host == "" {
						hostOrNil = utils.GetAnnotation(s.Annotations, ApiBaseDomain)
						if len(hostOrNil) != 0 {
							host = utils.GetAnnotation(s.Annotations, ApiBaseDomain)[0]
						}
					}

					op, path2 := g.buildOperation(d, methodName, comment, operationID, s.GetName(), path[0], host, inputDesc, outputDesc)
					var methodDesc *thrift_reflection.MethodDescriptor
					methodDesc = g.fileDesc.GetMethodDescriptor(s.GetName(), f.GetName())
					newOp := &openapi.Operation{}
					utils.ParseMethodOption(methodDesc, OpenapiOperation, &newOp)
					utils.MergeStructs(op, newOp)
					//extOperationOrNil := utils.GetAnnotation(f.Annotations, OpenapiOperation)
					//if len(extOperationOrNil) > 0 {
					//	extOperation := extOperationOrNil[0]
					//
					//}
					g.addOperationToDocument(d, op, path2, methodName)
				}
			}
		}
		if annotationsCount > 0 {
			comment := g.filterCommentString(s.ReservedComments)
			d.Tags = append(d.Tags, &openapi.Tag{Name: s.GetName(), Description: comment})
		}
	}
}

func (g *OpenAPIv3Generator) buildOperation(
	d *openapi.Document,
	methodName string,
	description string,
	operationID string,
	tagName string,
	path string,
	host string,
	inputDesc *thrift_reflection.StructDescriptor,
	outputDesc *thrift_reflection.StructDescriptor,
) (*openapi.Operation, string) {
	// Parameters array to hold all parameter objects
	var parameters []*openapi.ParameterOrReference

	for _, v := range inputDesc.GetFields() {
		var paramName, paramIn, paramDesc string
		var fieldSchema *openapi.SchemaOrReference
		required := false
		extOrNil := v.Annotations[ApiQuery]
		if len(extOrNil) > 0 {
			if ext := v.Annotations[ApiQuery][0]; ext != "" {
				paramIn = "query"
				paramName = ext
				paramDesc = g.filterCommentString(v.Comments)
				fieldSchema = g.schemaOrReferenceForField(v.Type)
				extPropertyOrNil := v.Annotations[OpenapiProperty]
				if len(extPropertyOrNil) > 0 {
					newFieldSchema := &openapi.Schema{}
					utils.ParseFieldOption(v, OpenapiProperty, &newFieldSchema)
					utils.MergeStructs(fieldSchema.Schema, newFieldSchema)
				}
			}
		}
		extOrNil = v.Annotations[ApiPath]
		if len(extOrNil) > 0 {
			if ext := v.Annotations[ApiPath][0]; ext != "" {
				paramIn = "path"
				paramName = ext
				paramDesc = g.filterCommentString(v.Comments)
				fieldSchema = g.schemaOrReferenceForField(v.Type)
				extPropertyOrNil := v.Annotations[OpenapiProperty]
				if len(extPropertyOrNil) > 0 {
					newFieldSchema := &openapi.Schema{}
					utils.ParseFieldOption(v, OpenapiProperty, &newFieldSchema)
					utils.MergeStructs(fieldSchema.Schema, newFieldSchema)
				}
				required = true
			}
		}
		extOrNil = v.Annotations[ApiCookie]
		if len(extOrNil) > 0 {
			if ext := v.Annotations[ApiCookie][0]; ext != "" {
				paramIn = "cookie"
				paramName = ext
				paramDesc = g.filterCommentString(v.Comments)
				fieldSchema = g.schemaOrReferenceForField(v.Type)
				extPropertyOrNil := v.Annotations[OpenapiProperty]
				if len(extPropertyOrNil) > 0 {
					newFieldSchema := &openapi.Schema{}
					utils.ParseFieldOption(v, OpenapiProperty, &newFieldSchema)
					utils.MergeStructs(fieldSchema.Schema, newFieldSchema)
				}
			}
		}
		extOrNil = v.Annotations[ApiHeader]
		if len(extOrNil) > 0 {
			if ext := v.Annotations[ApiHeader][0]; ext != "" {
				paramIn = "header"
				paramName = ext
				paramDesc = g.filterCommentString(v.Comments)
				fieldSchema = g.schemaOrReferenceForField(v.Type)
				extPropertyOrNil := v.Annotations[OpenapiProperty]
				if len(extPropertyOrNil) > 0 {
					newFieldSchema := &openapi.Schema{}
					utils.ParseFieldOption(v, OpenapiProperty, &newFieldSchema)
					utils.MergeStructs(fieldSchema.Schema, newFieldSchema)
				}
			}
		}

		parameter := &openapi.Parameter{
			Name:        paramName,
			In:          paramIn,
			Description: paramDesc,
			Required:    required,
			Schema:      fieldSchema,
		}
		var extParameter *openapi.Parameter
		utils.ParseFieldOption(v, OpenapiParameter, &extParameter)
		utils.MergeStructs(parameter, extParameter)
		// Append the parameter to the parameters array if it was set
		if paramName != "" && paramIn != "" {
			parameters = append(parameters, &openapi.ParameterOrReference{
				Parameter: parameter,
			})
		}
	}
	var RequestBody *openapi.RequestBodyOrReference
	if methodName != "GET" && methodName != "HEAD" && methodName != "DELETE" {
		bodySchema := g.getSchemaByOption(inputDesc, ApiBody)
		formSchema := g.getSchemaByOption(inputDesc, ApiForm)
		rawBodySchema := g.getSchemaByOption(inputDesc, ApiRawBody)

		var additionalProperties []*openapi.NamedMediaType
		if len(bodySchema.Properties.AdditionalProperties) > 0 {
			additionalProperties = append(additionalProperties, &openapi.NamedMediaType{
				Name: "application/json",
				Value: &openapi.MediaType{
					Schema: &openapi.SchemaOrReference{
						Schema: bodySchema,
					},
				},
			})
		}
		if len(formSchema.Properties.AdditionalProperties) > 0 {
			additionalProperties = append(additionalProperties, &openapi.NamedMediaType{
				Name: "multipart/form-data",
				Value: &openapi.MediaType{
					Schema: &openapi.SchemaOrReference{
						Schema: formSchema,
					},
				},
			})
		}

		if len(rawBodySchema.Properties.AdditionalProperties) > 0 {
			additionalProperties = append(additionalProperties, &openapi.NamedMediaType{
				Name: "application/octet-stream",
				Value: &openapi.MediaType{
					Schema: &openapi.SchemaOrReference{
						Schema: rawBodySchema,
					},
				},
			})
		}
		if len(additionalProperties) > 0 {
			RequestBody = &openapi.RequestBodyOrReference{
				RequestBody: &openapi.RequestBody{
					Description: g.filterCommentString(inputDesc.Comments),
					Content: &openapi.MediaTypes{
						AdditionalProperties: additionalProperties,
					},
				},
			}
		}

	}

	name, header, content := g.getResponseForStruct(d, outputDesc)

	desc := g.filterCommentString(outputDesc.Comments)

	if desc == "" {
		desc = "Successful response"
	}

	var headerOrEmpty *openapi.HeadersOrReferences

	if len(header.AdditionalProperties) != 0 {
		headerOrEmpty = header
	}

	var contentOrEmpty *openapi.MediaTypes

	if len(content.AdditionalProperties) != 0 {
		contentOrEmpty = content
	}
	var responses *openapi.Responses
	if headerOrEmpty != nil || contentOrEmpty != nil {
		responses = &openapi.Responses{
			ResponseOrReference: []*openapi.NamedResponseOrReference{
				{
					Name: name,
					Value: &openapi.ResponseOrReference{
						Response: &openapi.Response{
							Description: desc,
							Headers:     headerOrEmpty,
							Content:     contentOrEmpty,
						},
					},
				},
			},
		}
	}

	re := regexp.MustCompile(`:(\w+)`)
	path = re.ReplaceAllString(path, `{$1}`)

	op := &openapi.Operation{
		Tags:        []string{tagName},
		Description: description,
		OperationID: operationID,
		Parameters:  parameters,
		Responses:   responses,
		RequestBody: RequestBody,
	}
	if host != "" {
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "http://" + host
		}
		op.Servers = append(op.Servers, &openapi.Server{URL: host})
	}

	return op, path
}

func (g *OpenAPIv3Generator) getDocumentAnnotationInWhichServiceOrStruct() (string, string) {
	var ret string
	for _, s := range g.ast.Services {
		v := s.Annotations.Get(OpenapiDocument)
		if len(v) > 0 {
			ret = s.GetName()
			return "service", ret
		}
	}
	for _, s := range g.ast.Structs {
		v := s.Annotations.Get(OpenapiDocument)
		if len(v) > 0 {
			ret = s.GetName()
			return "struct", ret
		}
	}
	return "", ret
}

func (g *OpenAPIv3Generator) getResponseForStruct(d *openapi.Document, desc *thrift_reflection.StructDescriptor) (string, *openapi.HeadersOrReferences, *openapi.MediaTypes) {
	headers := &openapi.HeadersOrReferences{AdditionalProperties: []*openapi.NamedHeaderOrReference{}}

	for _, field := range desc.Fields {
		if len(field.Annotations[ApiHeader]) < 1 {
			continue
		}
		if ext := field.Annotations[ApiHeader][0]; ext != "" {
			headerName := ext
			header := &openapi.Header{
				Description: g.filterCommentString(field.Comments),
				Schema:      g.schemaOrReferenceForField(field.Type),
			}
			headers.AdditionalProperties = append(headers.AdditionalProperties, &openapi.NamedHeaderOrReference{
				Name: headerName,
				Value: &openapi.HeaderOrReference{
					Header: header,
				},
			})
		}
	}

	// get api.body、api.raw_body option schema
	bodySchema := g.getSchemaByOption(desc, ApiBody)
	rawBodySchema := g.getSchemaByOption(desc, ApiRawBody)
	var additionalProperties []*openapi.NamedMediaType

	if len(bodySchema.Properties.AdditionalProperties) > 0 {
		refSchema := &openapi.NamedSchemaOrReference{
			Name:  desc.GetName(),
			Value: &openapi.SchemaOrReference{Schema: bodySchema},
		}
		ref := "#/components/schemas/" + desc.GetName()
		g.addSchemaToDocument(d, refSchema)
		additionalProperties = append(additionalProperties, &openapi.NamedMediaType{
			Name: "application/json",
			Value: &openapi.MediaType{
				Schema: &openapi.SchemaOrReference{
					Reference: &openapi.Reference{Xref: ref},
				},
			},
		})
	}

	if len(rawBodySchema.Properties.AdditionalProperties) > 0 {
		refSchema := &openapi.NamedSchemaOrReference{
			Name:  desc.GetName(),
			Value: &openapi.SchemaOrReference{Schema: rawBodySchema},
		}
		ref := "#/components/schemas/" + desc.GetName()
		g.addSchemaToDocument(d, refSchema)
		additionalProperties = append(additionalProperties, &openapi.NamedMediaType{
			Name: "application/octet-stream",
			Value: &openapi.MediaType{
				Schema: &openapi.SchemaOrReference{
					Reference: &openapi.Reference{Xref: ref},
				},
			},
		})
	}

	content := &openapi.MediaTypes{
		AdditionalProperties: additionalProperties,
	}

	return "200", headers, content
}

func (g *OpenAPIv3Generator) getSchemaByOption(inputDesc *thrift_reflection.StructDescriptor, option string) *openapi.Schema {
	definitionProperties := &openapi.Properties{
		AdditionalProperties: make([]*openapi.NamedSchemaOrReference, 0),
	}
	var allRequired []string
	var extSchema *openapi.Schema
	utils.ParseStructOption(inputDesc, OpenapiSchema, &extSchema)
	if extSchema != nil {
		if extSchema.Required != nil {
			allRequired = extSchema.Required
		}
	}
	var required []string

	for _, field := range inputDesc.GetFields() {
		if field.Annotations[option] != nil {
			extName := field.GetName()
			if field.Annotations[option] != nil {
				if field.Annotations[option][0] != "" {
					extName = field.Annotations[option][0]
				}
			}
			if utils.Contains(allRequired, extName) {
				required = append(required, extName)
			}
			// Get the field description from the comments.
			description := g.filterCommentString(field.Comments)
			fieldSchema := g.schemaOrReferenceForField(field.Type)
			if fieldSchema == nil {
				continue
			}
			if fieldSchema.IsSetSchema() {
				fieldSchema.Schema.Description = description
				newFieldSchema := &openapi.Schema{}
				utils.ParseFieldOption(field, OpenapiProperty, &newFieldSchema)
				utils.MergeStructs(fieldSchema.Schema, newFieldSchema)
			}
			definitionProperties.AdditionalProperties = append(
				definitionProperties.AdditionalProperties,
				&openapi.NamedSchemaOrReference{
					Name:  extName,
					Value: fieldSchema,
				},
			)
		}
	}
	schema := &openapi.Schema{
		Type:       "object",
		Properties: definitionProperties,
	}
	if extSchema != nil {
		utils.MergeStructs(schema, extSchema)
	}
	schema.Required = required
	return schema
}

func (g *OpenAPIv3Generator) getStructLikeByName(name string) *parser.StructLike {
	for _, s := range g.ast.GetStructLikes() {
		if s.GetName() == name {
			return s
		}
	}
	return nil
}

// filterCommentString removes linter rules from comments.
func (g *OpenAPIv3Generator) filterCommentString(str string) string {
	var comments []string
	matches := g.commentPattern.FindAllStringSubmatch(str, -1)
	for _, match := range matches {
		var comment string
		if match[1] != "" {
			// One-line comment
			comment = strings.TrimSpace(match[1])
		} else if match[2] != "" {
			// Multiline comment
			multiLineComment := match[2]
			lines := strings.Split(multiLineComment, "\n")
			for i, line := range lines {
				lines[i] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
			}
			comment = strings.Join(lines, "\n")
		}
		comments = append(comments, comment)
	}

	return strings.Join(comments, "\n")
}

func (g *OpenAPIv3Generator) addSchemasForStructsToDocument(d *openapi.Document, structs []*parser.StructLike) {
	// Handle nested structs
	for _, s := range structs {
		var sls []*parser.StructLike
		for _, f := range s.GetFields() {
			if f.GetType().GetCategory().IsStruct() {
				sls = append(sls, g.getStructLikeByName(f.GetType().GetName()))
			}
		}
		g.addSchemasForStructsToDocument(d, sls)

		schemaName := s.GetName()
		// Only generate this if we need it and haven't already generated it.
		if !utils.Contains(g.requiredSchemas, schemaName) ||
			utils.Contains(g.generatedSchemas, schemaName) {
			continue
		}

		structDesc := g.fileDesc.GetStructDescriptor(s.GetName())

		// Get the description from the comments.
		messageDescription := g.filterCommentString(structDesc.Comments)

		// Build an array holding the fields of the message.
		definitionProperties := &openapi.Properties{
			AdditionalProperties: make([]*openapi.NamedSchemaOrReference, 0),
		}

		for _, field := range structDesc.Fields {
			// Get the field description from the comments.
			description := g.filterCommentString(field.Comments)
			fieldSchema := g.schemaOrReferenceForField(field.Type)
			if fieldSchema == nil {
				continue
			}
			if fieldSchema.IsSetSchema() {
				fieldSchema.Schema.Description = description
				newFieldSchema := &openapi.Schema{}
				utils.ParseFieldOption(field, OpenapiProperty, &newFieldSchema)
				utils.MergeStructs(fieldSchema.Schema, newFieldSchema)
			}
			extName := field.GetName()
			options := []string{ApiHeader, ApiBody, ApiForm, ApiRawBody}
			for _, option := range options {
				if field.Annotations[option] != nil {
					if field.Annotations[option][0] != "" {
						extName = field.Annotations[option][0]
					}
				}
			}
			definitionProperties.AdditionalProperties = append(
				definitionProperties.AdditionalProperties,
				&openapi.NamedSchemaOrReference{
					Name:  extName,
					Value: fieldSchema,
				},
			)
		}
		schema := &openapi.Schema{
			Type:        "object",
			Description: messageDescription,
			Properties:  definitionProperties,
		}
		var extSchema *openapi.Schema
		utils.ParseStructOption(structDesc, OpenapiSchema, &extSchema)
		if extSchema != nil {
			utils.MergeStructs(schema, extSchema)
		}
		// Add the schema to the components.schema list.
		g.addSchemaToDocument(d, &openapi.NamedSchemaOrReference{
			Name: schemaName,
			Value: &openapi.SchemaOrReference{
				Schema: schema,
			},
		})
	}

}

// addSchemaToDocument adds the schema to the document if required
func (g *OpenAPIv3Generator) addSchemaToDocument(d *openapi.Document, schema *openapi.NamedSchemaOrReference) {
	if utils.Contains(g.generatedSchemas, schema.Name) {
		return
	}
	g.generatedSchemas = append(g.generatedSchemas, schema.Name)
	d.Components.Schemas.AdditionalProperties = append(d.Components.Schemas.AdditionalProperties, schema)
}

func (g *OpenAPIv3Generator) addOperationToDocument(d *openapi.Document, op *openapi.Operation, path string, methodName string) {
	var selectedPathItem *openapi.NamedPathItem
	for _, namedPathItem := range d.Paths.Path {
		if namedPathItem.Name == path {
			selectedPathItem = namedPathItem
			break
		}
	}
	// If we get here, we need to create a path item.
	if selectedPathItem == nil {
		selectedPathItem = &openapi.NamedPathItem{Name: path, Value: &openapi.PathItem{}}
		d.Paths.Path = append(d.Paths.Path, selectedPathItem)
	}
	// Set the operation on the specified method.
	switch methodName {
	case "GET":
		selectedPathItem.Value.Get = op
	case "POST":
		selectedPathItem.Value.Post = op
	case "PUT":
		selectedPathItem.Value.Put = op
	case "DELETE":
		selectedPathItem.Value.Delete = op
	case "PATCH":
		selectedPathItem.Value.Patch = op
	case "OPTIONS":
		selectedPathItem.Value.Options = op
	case "HEAD":
		selectedPathItem.Value.Head = op
	}

}

func (g *OpenAPIv3Generator) schemaReferenceForMessage(message *thrift_reflection.StructDescriptor) string {
	schemaName := message.GetName()
	if !utils.Contains(g.requiredSchemas, schemaName) {
		g.requiredSchemas = append(g.requiredSchemas, schemaName)
	}
	return "#/components/schemas/" + schemaName
}

func (g *OpenAPIv3Generator) schemaOrReferenceForField(fieldType *thrift_reflection.TypeDescriptor) *openapi.SchemaOrReference {
	var kindSchema *openapi.SchemaOrReference
	if fieldType.IsStruct() {
		structDesc, err := fieldType.GetStructDescriptor()
		if err != nil {
			logs.Errorf("Error getting struct descriptor: %s", err)
			return nil
		}
		ref := g.schemaReferenceForMessage(structDesc)

		kindSchema = &openapi.SchemaOrReference{
			Reference: &openapi.Reference{Xref: ref},
		}
	}

	if fieldType.GetName() == "string" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type: "string",
			},
		}
	}

	if fieldType.GetName() == "binary" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type:   "string",
				Format: "binary",
			},
		}
	}

	if fieldType.GetName() == "bool" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type: "boolean",
			},
		}
	}

	if fieldType.GetName() == "byte" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type:   "string",
				Format: "byte",
			},
		}
	}

	if fieldType.GetName() == "double" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type:   "number",
				Format: "double",
			},
		}
	}

	if fieldType.GetName() == "i8" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type:   "integer",
				Format: "int8",
			},
		}
	}

	if fieldType.GetName() == "i16" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type:   "integer",
				Format: "int16",
			},
		}
	}

	if fieldType.GetName() == "i32" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type:   "integer",
				Format: "int32",
			},
		}
	}

	if fieldType.GetName() == "i64" {
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type:   "integer",
				Format: "int64",
			},
		}
	}

	if fieldType.IsMap() {
		kindSchema = g.schemaOrReferenceForField(fieldType.GetValueType())
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type: "object",
				AdditionalProperties: &openapi.AdditionalPropertiesItem{
					SchemaOrReference: kindSchema,
				},
			},
		}
	}

	if fieldType.IsList() {
		kindSchema = g.schemaOrReferenceForField(fieldType.GetValueType())
		kindSchema = &openapi.SchemaOrReference{
			Schema: &openapi.Schema{
				Type: "array",
				Items: &openapi.ItemsItem{
					SchemaOrReference: []*openapi.SchemaOrReference{kindSchema},
				},
			},
		}
	}
	return kindSchema
}

const (
	ApiGet           = "api.get"
	ApiPost          = "api.post"
	ApiPut           = "api.put"
	ApiPatch         = "api.patch"
	ApiDelete        = "api.delete"
	ApiOptions       = "api.options"
	ApiHEAD          = "api.head"
	ApiAny           = "api.any"
	ApiQuery         = "api.query"
	ApiForm          = "api.form"
	ApiPath          = "api.path"
	ApiHeader        = "api.header"
	ApiCookie        = "api.cookie"
	ApiBody          = "api.body"
	ApiRawBody       = "api.raw_body"
	ApiBaseDomain    = "api.base_domain"
	ApiBaseURL       = "api.baseurl"
	OpenapiOperation = "openapi.operation"
	OpenapiProperty  = "openapi.property"
	OpenapiSchema    = "openapi.schema"
	OpenapiParameter = "openapi.parameter"
	OpenapiDocument  = "openapi.document"
)

var (
	HttpMethodAnnotations = map[string]string{
		ApiGet:     "GET",
		ApiPost:    "POST",
		ApiPut:     "PUT",
		ApiPatch:   "PATCH",
		ApiDelete:  "DELETE",
		ApiOptions: "OPTIONS",
		ApiHEAD:    "HEAD",
		ApiAny:     "ANY",
	}
)
