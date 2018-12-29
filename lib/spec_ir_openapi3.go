package lib

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	mimeJSON = "application/json"

	// For testing
	someDescription = "some description"
)

// For testing
var xxx2uint32 = map[string]uint32{
	"default": 0,
	"1XX":     1,
	"2XX":     2,
	"3XX":     3,
	"4XX":     4,
	"5XX":     5,
}

func newSpecFromOA3(doc *openapi3.Swagger) (vald *Validator, err error) {
	log.Println("[DBG] normalizing spec from OpenAPIv3")

	docPaths, docSchemas := doc.Paths, doc.Components.Schemas
	vald = newValidator(len(docPaths), len(docSchemas))
	log.Println("[DBG] seeding schemas")
	//TODO: use docPath as root of base
	if err = vald.schemasFromOA3(docSchemas); err != nil {
		return
	}

	basePath, err := basePathFromOA3(doc.Servers)
	if err != nil {
		return
	}
	log.Println("[DBG] going through endpoints")
	vald.endpointsFromOA3(basePath, docPaths)
	return
}

func (vald *Validator) schemasFromOA3(docSchemas map[string]*openapi3.SchemaRef) error {
	schemas := make(schemasJSON, len(docSchemas))
	for name, docSchema := range docSchemas {
		schemas[name] = vald.schemaFromOA3(docSchema.Value)
	}
	return vald.seed("#/components/schemas/", schemas)
}

// For testing
func (sm schemap) schemasToOA3(doc *openapi3.Swagger) {
	seededSchemas := make(map[string]*openapi3.SchemaRef, len(sm))
	for _, refOrSchema := range sm {
		if schemaPtr := refOrSchema.GetPtr(); schemaPtr != nil {
			if ref := schemaPtr.GetRef(); ref != "" {
				name := strings.TrimPrefix(ref, "#/components/schemas/")
				seededSchemas[name] = sm.schemaToOA3(schemaPtr.GetSID())
			}
		}
	}
	doc.Components.Schemas = seededSchemas
}

func (vald *Validator) endpointsFromOA3(basePath string, docPaths openapi3.Paths) {
	i, paths := 0, make([]string, len(docPaths))
	for path := range docPaths {
		paths[i] = path
		i++
	}
	sort.Strings(paths)

	for j := 0; j != i; j++ {
		path := paths[j]
		docOps := docPaths[path].Operations()
		k, methods := 0, make([]string, len(docOps))
		for docMethod := range docOps {
			methods[k] = docMethod
			k++
		}
		sort.Strings(methods)

		for l := 0; l != k; l++ {
			docMethod := methods[l]
			log.Println("[DBG] through", docMethod, path)
			docOp := docOps[docMethod]
			inputs := make([]*ParamJSON, 0, 1+len(docOp.Parameters))
			vald.inputBodyFromOA3(&inputs, docOp.RequestBody)
			vald.inputsFromOA3(&inputs, docOp.Parameters)
			outputs := vald.outputsFromOA3(docOp.Responses)
			method := methodFromOA3(docMethod)
			vald.Spec.Endpoints[uint32(1+j+l)] = &Endpoint{
				Endpoint: &Endpoint_Json{
					&EndpointJSON{
						Method:       method,
						PathPartials: pathFromOA3(basePath, path),
						Inputs:       inputs,
						Outputs:      outputs,
					},
				},
			}
		}
	}
}

// For testing
func (sm schemap) endpointsToOA3(doc *openapi3.Swagger, es map[uint32]*Endpoint) {
	doc.Paths = make(openapi3.Paths, len(es))
	for _, e := range es {
		endpoint := e.GetJson()
		url := pathToOA3(endpoint.GetPathPartials())
		inputs := endpoint.GetInputs()
		reqBody := sm.inputBodyToOA3(inputs)
		params := sm.inputsToOA3(inputs)
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

func (vald *Validator) inputBodyFromOA3(inputs *[]*ParamJSON, docReqBody *openapi3.RequestBodyRef) {
	if docReqBody != nil {
		//FIXME: handle .Ref
		docBody := docReqBody.Value
		for mime, ct := range docBody.Content {
			if mime == mimeJSON {
				docSchema := ct.Schema
				schema := vald.schemaOrRefFromOA3(docSchema)
				param := &ParamJSON{
					IsRequired: docBody.Required,
					SID:        vald.ensureMapped(docSchema.Ref, schema),
					Name:       "",
					Kind:       ParamJSON_body,
				}
				*inputs = append(*inputs, param)
				return
			}
		}
	}
}

// For testing
func (sm schemap) inputBodyToOA3(inputs []*ParamJSON) (reqBodyRef *openapi3.RequestBodyRef) {
	if len(inputs) > 0 {
		body := inputs[0]
		if body != nil && isInputBody(body) {
			reqBody := &openapi3.RequestBody{
				Content:     sm.contentToOA3(body.GetSID()),
				Required:    body.GetIsRequired(),
				Description: someDescription,
			}
			reqBodyRef = &openapi3.RequestBodyRef{Value: reqBody}
		}
	}
	return
}

func (vald *Validator) inputsFromOA3(inputs *[]*ParamJSON, docParams openapi3.Parameters) {
	paramsCount := len(docParams)
	paramap := make(map[string]*openapi3.ParameterRef, paramsCount)
	i, names := 0, make([]string, paramsCount)
	for _, docParamRef := range docParams {
		docParam := docParamRef.Value
		name := docParam.In + docParam.Name
		names[i] = name
		paramap[name] = docParamRef
		i++
	}
	sort.Strings(names)

	for j := 0; j != i; j++ {
		docParamRef := paramap[names[j]]
		//FIXME: handle .Ref
		docParam := docParamRef.Value
		kind := ParamJSON_UNKNOWN
		switch docParam.In {
		case openapi3.ParameterInPath:
			kind = ParamJSON_path
		case openapi3.ParameterInQuery:
			kind = ParamJSON_query
		case openapi3.ParameterInHeader:
			kind = ParamJSON_header
		case openapi3.ParameterInCookie:
			kind = ParamJSON_cookie
		}
		docSchema := docParam.Schema
		schema := vald.schemaOrRefFromOA3(docSchema)
		param := &ParamJSON{
			IsRequired: docParam.Required,
			SID:        vald.ensureMapped(docSchema.Ref, schema),
			Name:       docParam.Name,
			Kind:       kind,
		}
		*inputs = append(*inputs, param)
	}
}

// For testing
func (sm schemap) inputsToOA3(inputs []*ParamJSON) (params openapi3.Parameters) {
	for _, input := range inputs {
		if isInputBody(input) {
			continue
		}

		var in string
		switch input.GetKind() {
		case ParamJSON_path:
			in = openapi3.ParameterInPath
		case ParamJSON_query:
			in = openapi3.ParameterInQuery
		case ParamJSON_header:
			in = openapi3.ParameterInHeader
		case ParamJSON_cookie:
			in = openapi3.ParameterInCookie
		}

		param := &openapi3.Parameter{
			Name:        input.GetName(),
			Required:    input.GetIsRequired(),
			In:          in,
			Description: someDescription,
			Schema:      sm.schemaToOA3(input.GetSID()),
		}

		params = append(params, &openapi3.ParameterRef{Value: param})
	}
	return
}

func (vald *Validator) outputsFromOA3(docResponses openapi3.Responses) (
	outputs map[uint32]sid,
) {
	outputs = make(map[uint32]sid)
	i, codes := 0, make([]string, len(docResponses))
	for code := range docResponses {
		codes[i] = code
		i++
	}
	sort.Strings(codes)

	for j := 0; j != i; j++ {
		code := codes[j]
		responseRef := docResponses[code]
		xxx := makeXXXFromOA3(code)
		// NOTE: Responses MAY have a schema
		if len(responseRef.Value.Content) == 0 {
			outputs[xxx] = 0
		}
		//FIXME: handle .Ref
		for mime, ct := range responseRef.Value.Content {
			if mime == mimeJSON {
				docSchema := ct.Schema
				if docSchema == nil {
					outputs[xxx] = 0
				} else {
					schema := vald.schemaOrRefFromOA3(docSchema)
					outputs[xxx] = vald.ensureMapped(docSchema.Ref, schema)
				}
			}
		}
	}
	return
}

// For testing
func (sm schemap) outputsToOA3(outs map[uint32]sid) openapi3.Responses {
	responses := make(openapi3.Responses, len(outs))
	for xxx, SID := range outs {
		XXX := makeXXXToOA3(xxx)
		responses[XXX] = &openapi3.ResponseRef{
			Value: &openapi3.Response{Description: someDescription}}
		if SID != 0 {
			responses[XXX].Value.Content = sm.contentToOA3(SID)
		}
	}
	return responses
}

// For testing
func (sm schemap) contentToOA3(SID sid) openapi3.Content {
	schemaRef := sm.schemaToOA3(SID)
	return openapi3.NewContentWithJSONSchemaRef(schemaRef)
}

func (vald *Validator) schemaOrRefFromOA3(s *openapi3.SchemaRef) (schema schemaJSON) {
	if ref := s.Ref; ref != "" {
		return schemaJSON{"$ref": ref}
	}
	return vald.schemaFromOA3(s.Value)
}

func (vald *Validator) schemaFromOA3(s *openapi3.Schema) (schema schemaJSON) {
	schema = make(schemaJSON)

	// "enum"
	if sEnum := s.Enum; len(sEnum) != 0 {
		schema["enum"] = sEnum
	}

	// "nullable"
	if s.Nullable {
		schema["type"] = []string{"null"}
	}
	// "type"
	if sType := s.Type; sType != "" {
		schema["type"] = ensureSchemaType(schema["type"], sType)
	}

	// "format"
	if sFormat := s.Format; sFormat != "" {
		schema["format"] = sFormat
	}
	// "minLength"
	if sMinLength := s.MinLength; sMinLength != 0 {
		schema["minLength"] = sMinLength
	}
	// "maxLength"
	if sMaxLength := s.MaxLength; nil != sMaxLength {
		schema["maxLength"] = *sMaxLength
	}
	// "pattern"
	if sPattern := s.Pattern; sPattern != "" {
		schema["pattern"] = sPattern
	}

	// "minimum"
	if nil != s.Min {
		schema["minimum"] = *s.Min
	}
	// "maximum"
	if nil != s.Max {
		schema["maximum"] = *s.Max
	}
	// "exclusiveMinimum", "exclusiveMaximum"
	if sExMin := s.ExclusiveMin; sExMin {
		schema["exclusiveMinimum"] = sExMin
	}
	if sExMax := s.ExclusiveMax; sExMax {
		schema["exclusiveMaximum"] = sExMax
	}
	// "multipleOf"
	if nil != s.MultipleOf {
		schema["multipleOf"] = *s.MultipleOf
	}

	// "uniqueItems"
	if sUniq := s.UniqueItems; sUniq {
		schema["uniqueItems"] = sUniq
	}
	// "minItems"
	if sMinItems := s.MinItems; sMinItems != 0 {
		schema["minItems"] = sMinItems
	}
	// "maxItems"
	if nil != s.MaxItems {
		schema["maxItems"] = *s.MaxItems
	}
	// "items"
	if sItems := s.Items; nil != sItems {
		schema["type"] = ensureSchemaType(schema["type"], "array")
		if sItems.Value.IsEmpty() {
			schema["items"] = []schemaJSON{}
		} else {
			schema["items"] = []schemaJSON{vald.schemaOrRefFromOA3(sItems)}
		}
	}

	// "minProperties"
	if sMinProps := s.MinProps; sMinProps != 0 {
		schema["minProperties"] = sMinProps
	}
	// "maxProperties"
	if nil != s.MaxProps {
		schema["maxProperties"] = *s.MaxProps
	}
	// "required"
	if sRequired := s.Required; len(sRequired) != 0 {
		schema["required"] = sRequired
	}
	// "properties"
	if count := len(s.Properties); count != 0 {
		schema["type"] = ensureSchemaType(schema["type"], "object")
		properties := make(schemasJSON, count)
		i, props := 0, make([]string, count)
		for propName := range s.Properties {
			props[i] = propName
			i++
		}
		sort.Strings(props)

		for j := 0; j != i; j++ {
			propName := props[j]
			properties[propName] = vald.schemaOrRefFromOA3(s.Properties[propName])
		}
		schema["properties"] = properties
	}
	//FIXME: "additionalProperties"
	if sAddProps := s.AdditionalPropertiesAllowed; sAddProps != nil {
		schema["additionalProperties"] = sAddProps
	}

	// "allOf"
	if sAllOf := s.AllOf; len(sAllOf) != 0 {
		allOf := make([]schemaJSON, len(sAllOf))
		for i, sOf := range sAllOf {
			allOf[i] = vald.schemaOrRefFromOA3(sOf)
		}
		schema["allOf"] = allOf
	}

	// "anyOf"
	if sAnyOf := s.AnyOf; len(sAnyOf) != 0 {
		anyOf := make([]schemaJSON, len(sAnyOf))
		for i, sOf := range sAnyOf {
			anyOf[i] = vald.schemaOrRefFromOA3(sOf)
		}
		schema["anyOf"] = anyOf
	}

	// "oneOf"
	if sOneOf := s.OneOf; len(sOneOf) != 0 {
		oneOf := make([]schemaJSON, len(sOneOf))
		for i, sOf := range sOneOf {
			oneOf[i] = vald.schemaOrRefFromOA3(sOf)
		}
		schema["oneOf"] = oneOf
	}

	// "not"
	if sNot := s.Not; nil != sNot {
		schema["not"] = vald.schemaOrRefFromOA3(sNot)
	}

	return
}

// For testing
func (sm schemap) schemaToOA3(SID sid) *openapi3.SchemaRef {
	s := sm.toGo(SID)
	s = transformSchemaToOA3(s)

	sJSON, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	schema := openapi3.NewSchema()
	if err := json.Unmarshal(sJSON, &schema); err != nil {
		panic(err)
	}

	return schema.NewRef()
}

// For testing
func transformSchemaToOA3(s schemaJSON) schemaJSON {
	// "type", "nullable"
	if v, ok := s["type"]; ok {
		sTypes := v.([]string)
		sType := ""
		for _, v := range sTypes {
			switch v {
			case "":
				continue
			case Schema_JSON_Type_name[int32(Schema_JSON_null)]:
				s["nullable"] = true
			default:
				sType = v
			}
		}
		s["type"] = sType
	}

	// "items"
	if v, ok := s["items"]; ok {
		if vv := v.([]schemaJSON); len(vv) > 0 {
			s["items"] = transformSchemaToOA3(vv[0])
		}
	}

	// "properties"
	if v, ok := s["properties"]; ok {
		props := v.(schemaJSON)
		for propName, propSchema := range props {
			props[propName] = transformSchemaToOA3(propSchema.(schemaJSON))
		}
		s["properties"] = props
	}

	// "allOf"
	if v, ok := s["allOf"]; ok {
		allOf := v.([]schemaJSON)
		for i, schemaOf := range allOf {
			allOf[i] = transformSchemaToOA3(schemaOf)
		}
		s["allOf"] = allOf
	}

	// "anyOf"
	if v, ok := s["anyOf"]; ok {
		anyOf := v.([]schemaJSON)
		for i, schemaOf := range anyOf {
			anyOf[i] = transformSchemaToOA3(schemaOf)
		}
		s["anyOf"] = anyOf
	}

	// "oneOf"
	if v, ok := s["oneOf"]; ok {
		oneOf := v.([]schemaJSON)
		for i, schemaOf := range oneOf {
			oneOf[i] = transformSchemaToOA3(schemaOf)
		}
		s["oneOf"] = oneOf
	}

	// "not"
	if v, ok := s["not"]; ok {
		s["not"] = transformSchemaToOA3(v.(schemaJSON))
	}

	return s
}

func ensureSchemaType(types interface{}, t string) []string {
	if types == nil {
		return []string{t}
	}
	ts := types.([]string)
	for _, aT := range ts {
		if t == aT {
			return ts
		}
	}
	return append(ts, t)
}

func pathFromOA3(basePath, path string) (partials []*PathPartial) {
	if basePath != "/" {
		p := &PathPartial{Pp: &PathPartial_Part{basePath}}
		partials = append(partials, p)
	}

	onCurly := func(r rune) bool { return r == '{' || r == '}' }
	isCurly := '{' == path[0]
	for i, part := range strings.FieldsFunc(path, onCurly) {
		var p PathPartial
		if isCurly || i%2 != 0 {
			// TODO (vendor): ensure path params are part of inputs
			p.Pp = &PathPartial_Ptr{part}
		} else {
			p.Pp = &PathPartial_Part{part}
		}
		partials = append(partials, &p)
	}

	if length := len(partials); length > 1 {
		part1 := partials[0].GetPart()
		part2 := partials[1].GetPart()
		if part1 != "" && part2 != "" {
			partials = partials[1:]
			partials[0] = &PathPartial{Pp: &PathPartial_Part{part1 + part2}}
			return
		}
	}
	return
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

func makeXXXFromOA3(code string) uint32 {
	switch {
	case code == "default":
		return 0
	case code == "1XX":
		return 1
	case code == "2XX":
		return 2
	case code == "3XX":
		return 3
	case code == "4XX":
		return 4
	case code == "5XX":
		return 5

	case "100" <= code && code <= "599":
		i, _ := strconv.Atoi(code)
		return uint32(i)

	default:
		panic(code)
	}
}

func makeXXXToOA3(xxx uint32) string {
	for k, v := range xxx2uint32 {
		if v == xxx {
			return k
		}
	}
	return strconv.FormatUint(uint64(xxx), 10)
}

//TODO: support the whole spec on /"servers"
func basePathFromOA3(docServers openapi3.Servers) (basePath string, err error) {
	if len(docServers) == 0 {
		log.Println(`[NFO] field 'servers' empty/unset: using "/"`)
		basePath = "/"
		return
	}

	if len(docServers) != 1 {
		log.Println(`[NFO] field 'servers' has many values: using the first one`)
	}

	u, err := url.Parse(docServers[0].URL)
	if err != nil {
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}
	basePath = u.Path

	if basePath == "" || basePath[0] != '/' {
		err = errors.New(`field 'servers' has no suitable 'url'`)
		log.Println("[ERR]", err)
		ColorERR.Println(err)
	}
	return
}

func isInputBody(input *ParamJSON) bool {
	return input.GetName() == "" && input.GetKind() == ParamJSON_body
}

func methodFromOA3(docMethod string) EndpointJSON_Method {
	return EndpointJSON_Method(EndpointJSON_Method_value[docMethod])
}

func methodToOA3(m EndpointJSON_Method, op *openapi3.Operation, p *openapi3.PathItem) {
	switch m {
	case EndpointJSON_CONNECT:
		p.Connect = op
	case EndpointJSON_DELETE:
		p.Delete = op
	case EndpointJSON_GET:
		p.Get = op
	case EndpointJSON_HEAD:
		p.Head = op
	case EndpointJSON_OPTIONS:
		p.Options = op
	case EndpointJSON_PATCH:
		p.Patch = op
	case EndpointJSON_POST:
		p.Post = op
	case EndpointJSON_PUT:
		p.Put = op
	case EndpointJSON_TRACE:
		p.Trace = op
	default:
		panic(`no such method`)
	}
}
