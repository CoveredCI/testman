package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

const someText = "some text"

type Schemap struct {
	M schemap
}

var xxx2uint32 = map[string]uint32{
	"default": 0,
	"1XX":     1,
	"2XX":     2,
	"3XX":     3,
	"4XX":     4,
	"5XX":     5,
}

func TestMakeXXXFromOA3(t *testing.T) {
	for k, v := range xxx2uint32 {
		got := makeXXXFromOA3(k)
		require.Equal(t, v, got)
	}

	for i := 100; i < 600; i++ {
		k, v := strconv.Itoa(i), uint32(i)
		got := makeXXXFromOA3(k)
		require.Equal(t, v, got)
	}
}

func TestEncodeVersusEncodeDecodeEncode(t *testing.T) {
	jsoner := &jsonpb.Marshaler{Indent: "\t"}
	for _, docPath := range []string{
		"./misc/openapiv3.0.0_petstore.yaml",
		"./misc/openapiv3.0.0_petstore.json",
		"./misc/openapiv3.0.0_petstore-expanded.yaml",
	} {
		t.Run(docPath, func(t *testing.T) {
			blob0, err := ioutil.ReadFile(docPath)
			require.NoError(t, err)
			spec0, err := doLint(docPath, blob0, false)
			require.NoError(t, err)
			require.NotNil(t, spec0)
			require.IsType(t, &SpecIR{}, spec0)
			bin0, err := proto.Marshal(spec0)
			require.NoError(t, err)
			require.NotNil(t, bin0)
			jsn0, err := jsoner.MarshalToString(spec0)
			require.NoError(t, err)
			require.NotEmpty(t, jsn0)

			var spec1 SpecIR
			err = proto.Unmarshal(bin0, &spec1)
			require.NoError(t, err)
			require.NotNil(t, &spec1)
			doc := specToOA3(&spec1)
			blob1, err := json.MarshalIndent(doc, "", "  ")
			require.NoError(t, err)
			log.Println("here we go again")
			spec2, err := doLint("bla.json", blob1, false)
			require.NoError(t, err)
			require.NotNil(t, spec2)
			require.IsType(t, &SpecIR{}, spec2)
			jsn1, err := jsoner.MarshalToString(spec2)
			require.NoError(t, err)
			require.NotEmpty(t, jsn1)

			require.JSONEq(t, jsn0, jsn1)
		})
	}
}

func specToOA3(spec *SpecIR) (doc openapi3.Swagger) {
	doc.OpenAPI = "3.0.0"
	doc.Info = openapi3.Info{
		Title:   someText,
		Version: "1.42.3",
	}
	sm := &Schemap{M: spec.GetSchemas().GetJson()}
	sm.schemasToOA3(&doc)
	sm.endpointsToOA3(&doc, spec.GetEndpoints())
	return
}

func (sm *Schemap) schemasToOA3(doc *openapi3.Swagger) {
	seededSchemas := make(map[string]*openapi3.SchemaRef, len(sm.M))
	for _, refOrSchema := range sm.M {
		if schemaPtr := refOrSchema.GetPtr(); schemaPtr != nil {
			if ref := schemaPtr.GetRef(); ref != "" {
				name := strings.TrimPrefix(ref, "#/components/schemas/")
				refd := sm.M[schemaPtr.GetUID()]
				seededSchemas[name] = sm.schemaToOA3(refd.GetSchema())
			}
		}
	}
	doc.Components.Schemas = seededSchemas
}

func (sm *Schemap) endpointsToOA3(doc *openapi3.Swagger, es []*Endpoint) {
	doc.Paths = make(openapi3.Paths, len(es))
	for _, e := range es {
		endpoint := e.GetJson()
		url := pathToOA3(endpoint.GetPathPartials())
		reqBody, params := sm.inputsToOA3(endpoint.GetInputs())
		op := &openapi3.Operation{
			RequestBody: reqBody,
			Parameters:  params,
			Responses:   sm.outputsToOA3(endpoint.GetOutputs()),
		}
		if doc.Paths[url] == nil {
			doc.Paths[url] = &openapi3.PathItem{}
		}
		methodToOA3(endpoint.GetMethod(), op, doc.Paths[url])
	}
}

func (sm *Schemap) inputsToOA3(inputs *ParamsJSON) (
	reqBodyRef *openapi3.RequestBodyRef,
	params openapi3.Parameters,
) {
	if body := inputs.GetBody(); body != nil {
		reqBody := &openapi3.RequestBody{
			Content:     sm.contentToOA3(body.GetPtr()),
			Required:    body.GetRequired(),
			Description: someText,
		}
		reqBodyRef = &openapi3.RequestBodyRef{Value: reqBody}
	}
	for name, input := range inputs.GetHeader() {
		//FIXME: handle ParameterInCookie
		param := &openapi3.Parameter{
			Name:        name,
			Required:    input.GetRequired(),
			In:          openapi3.ParameterInHeader,
			Description: someText,
			Schema:      sm.derefSchemaPtr(input.GetPtr()),
		}
		params = append(params, &openapi3.ParameterRef{Value: param})
	}
	for name, input := range inputs.GetPath() {
		param := &openapi3.Parameter{
			Name:        name,
			Required:    input.GetRequired(),
			In:          openapi3.ParameterInPath,
			Description: someText,
			Schema:      sm.derefSchemaPtr(input.GetPtr()),
		}
		params = append(params, &openapi3.ParameterRef{Value: param})
	}
	for name, input := range inputs.GetQuery() {
		param := &openapi3.Parameter{
			Name:        name,
			Required:    input.GetRequired(),
			In:          openapi3.ParameterInQuery,
			Description: someText,
			Schema:      sm.derefSchemaPtr(input.GetPtr()),
		}
		params = append(params, &openapi3.ParameterRef{Value: param})
	}
	return
}

func (sm *Schemap) outputsToOA3(outs map[uint32]*SchemaPtr) openapi3.Responses {
	responses := make(openapi3.Responses, len(outs))
	for xxx, schema := range outs {
		XXX := xxx2XXX(xxx)
		responses[XXX] = &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: someText,
				Content:     sm.contentToOA3(schema),
			},
		}
	}
	return responses
}

func (sm *Schemap) contentToOA3(schemaPtr *SchemaPtr) openapi3.Content {
	schemaRef := sm.derefSchemaPtr(schemaPtr)
	return openapi3.NewContentWithJSONSchemaRef(schemaRef)
}

func (sm *Schemap) derefSchemaPtr(schemaPtr *SchemaPtr) *openapi3.SchemaRef {
	s, ok := sm.M[schemaPtr.GetUID()]
	if !ok {
		panic(`schemaptr's UID must be in schemap`)
	}

	if ss := s.GetSchema(); ss != nil {
		if sp := s.GetPtr(); sp != nil {
			panic(`sub schemaptr must not be set`)
		}
		schema := sm.schemaToOA3(ss)
		schema.Ref = schemaPtr.GetRef()
		return schema
	}
	return sm.derefSchemaPtr(s.GetPtr())
}

func (sm *Schemap) schemaToOA3(s *Schema_JSON) *openapi3.SchemaRef {
	schema := openapi3.NewSchema()

	// "enum"
	//FIXME

	// "type", "nullable"
	for _, t := range s.GetType() {
		if t == Schema_JSON_UNKNOWN {
			panic(`no way this is ever zero`)
		}
		if t == Schema_JSON_null {
			schema.Nullable = true
		} else {
			schema.Type = Schema_JSON_Type_name[int32(t)]
		}
	}

	// "format"
	schema.Format = s.GetFormat()
	// "minLength"
	schema.MinLength = s.GetMinLength()
	// "maxLength"
	if s.GetHasMaxLength() {
		v := s.GetMaxLength()
		schema.MaxLength = &v
	}
	// "pattern"
	schema.Pattern = s.GetPattern()

	// "minimum"
	if s.GetHasMinimum() {
		v := s.GetMinimum()
		schema.Min = &v
	}
	// "maximum"
	if s.GetHasMaximum() {
		v := s.GetMaximum()
		schema.Max = &v
	}
	// "exclusiveMinimum", "exclusiveMaximum"
	schema.ExclusiveMin = s.GetExclusiveMinimum()
	schema.ExclusiveMax = s.GetExclusiveMaximum()
	// "multipleOf"
	if mulOf := s.GetTranslatedMultipleOf(); mulOf != 0.0 {
		v := mulOf + 1.0
		schema.MultipleOf = &v
	}

	// "uniqueItems"
	schema.UniqueItems = s.GetUniqueItems()
	// "minItems"
	schema.MinItems = s.GetMinItems()
	// "maxItems"
	if s.GetHasMaxItems() {
		v := s.GetMaxItems()
		schema.MaxItems = &v
	}
	// "items"
	if sItems := s.GetItems(); len(sItems) == 1 {
		schema.Items = sm.derefSchemaPtr(sItems[0])
	}

	// "minProperties"
	schema.MinProps = s.GetMinProperties()
	// "maxProperties"
	if s.GetHasMaxProperties() {
		v := s.GetMaxProperties()
		schema.MaxProps = &v
	}
	// "required"
	schema.Required = s.GetRequired()
	// "properties"
	if sProps := s.GetProperties(); len(sProps) != 0 {
		schema.Properties = make(map[string]*openapi3.SchemaRef, len(sProps))
		for propName, propSchema := range sProps {
			schema.Properties[propName] = sm.derefSchemaPtr(propSchema)
		}
	}

	// "allOf"
	if sAllOf := s.GetAllOf(); len(sAllOf) != 0 {
		schema.AllOf = make([]*openapi3.SchemaRef, len(sAllOf))
		for i, sOf := range sAllOf {
			schema.AllOf[i] = sm.derefSchemaPtr(sOf)
		}
	}

	// "AnyOf"
	if sAnyOf := s.GetAnyOf(); len(sAnyOf) != 0 {
		schema.AnyOf = make([]*openapi3.SchemaRef, len(sAnyOf))
		for i, sOf := range sAnyOf {
			schema.AnyOf[i] = sm.derefSchemaPtr(sOf)
		}
	}

	// "OneOf"
	if sOneOf := s.GetOneOf(); len(sOneOf) != 0 {
		schema.OneOf = make([]*openapi3.SchemaRef, len(sOneOf))
		for i, sOf := range sOneOf {
			schema.OneOf[i] = sm.derefSchemaPtr(sOf)
		}
	}

	// "Not"
	if sNot := s.GetNot(); nil != sNot {
		schema.Not = sm.derefSchemaPtr(sNot)
	}

	return schema.NewRef()
}

func xxx2XXX(xxx uint32) string {
	for k, v := range xxx2uint32 {
		if v == xxx {
			return k
		}
	}
	return strconv.FormatUint(uint64(xxx), 10)
}

func pathToOA3(partials []*PathPartial) (s string) {
	for _, p := range partials {
		part := p.GetPart()
		if part != "" {
			s += part
		} else {
			s += "{" + p.GetPtr() + "}"
		}
	}
	return
}

func methodToOA3(m Method, op *openapi3.Operation, p *openapi3.PathItem) {
	switch m {
	case Method_CONNECT:
		p.Connect = op
	case Method_DELETE:
		p.Delete = op
	case Method_GET:
		p.Get = op
	case Method_HEAD:
		p.Head = op
	case Method_OPTIONS:
		p.Options = op
	case Method_PATCH:
		p.Patch = op
	case Method_POST:
		p.Post = op
	case Method_PUT:
		p.Put = op
	case Method_TRACE:
		p.Trace = op
	default:
		panic(`no such method`)
	}
}
