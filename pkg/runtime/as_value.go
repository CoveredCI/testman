package runtime

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Request/Response fields loosely follow Python's `requests` API

// inputAsValue exposes request data as a Starlark value for user assertions.
func inputAsValue(i *fm.Clt_CallRequestRaw_Input) starlark.Value {
	s := make(starlark.StringDict, 5)
	switch x := i.GetInput().(type) {

	case *fm.Clt_CallRequestRaw_Input_HttpRequest_:
		reqProto := i.GetHttpRequest()
		s["method"] = starlark.String(reqProto.Method)
		s["url"] = starlark.String(reqProto.Url)
		s["content"] = starlark.String(reqProto.Body)

		headers := make(starlark.StringDict, len(reqProto.Headers))
		for key, values := range reqProto.Headers {
			vs := make([]starlark.Value, 0, len(values.GetValues()))
			for _, value := range values.GetValues() {
				vs = append(vs, starlark.String(value))
			}
			headers[key] = starlark.NewList(vs)
		}
		s["headers"] = &starlarkstruct.Module{
			Name:    "headers",
			Members: headers,
		}

		if len(reqProto.Body) != 0 {
			s["body"] = starlarkvalue.FromProtoValue(reqProto.BodyDecoded)
		}

	default:
		panic(fmt.Errorf("unhandled output %T: %+v", x, i))
	}
	return &starlarkstruct.Module{
		Name:    "request",
		Members: s,
	}
}

// outputAsValue exposes response data as a Starlark value for user assertions.
func outputAsValue(o *fm.Clt_CallResponseRaw_Output) starlark.Value {
	s := make(starlark.StringDict, 6)
	switch x := o.GetOutput().(type) {

	case *fm.Clt_CallResponseRaw_Output_HttpResponse_:
		repProto := o.GetHttpResponse()
		s["status_code"] = starlark.MakeUint(uint(repProto.StatusCode))
		s["reason"] = starlark.String(repProto.Reason)
		s["content"] = starlark.String(repProto.Body)
		s["elapsed_ns"] = starlark.MakeInt64(repProto.ElapsedNs)
		// "error": repProto.Error Checks make this unreachable
		// "history" :: []Rep (redirects)?

		headers := make(starlark.StringDict, len(repProto.Headers))
		for key, values := range repProto.Headers {
			vs := make([]starlark.Value, 0, len(values.GetValues()))
			for _, value := range values.GetValues() {
				vs = append(vs, starlark.String(value))
			}
			headers[key] = starlark.NewList(vs)
		}
		s["headers"] = &starlarkstruct.Module{
			Name:    "headers",
			Members: headers,
		}

		if len(repProto.Body) != 0 {
			s["body"] = starlarkvalue.FromProtoValue(repProto.BodyDecoded)
		}

	default:
		panic(fmt.Errorf("unhandled output %T: %+v", x, o))
	}
	return &starlarkstruct.Module{
		Name:    "response",
		Members: s,
	}
}
