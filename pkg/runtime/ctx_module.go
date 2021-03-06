package runtime

import (
	"errors"
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"go.starlark.net/starlark"
)

type (
	ctxctor2 func(*fm.Clt_CallResponseRaw_Output) ctxctor1
	ctxctor1 func(starlark.Value) starlark.Value
)

func ctxCurry(callInput *fm.Clt_CallRequestRaw_Input) ctxctor2 {
	request := inputAsValue(callInput)
	request.Freeze()
	return func(callOutput *fm.Clt_CallResponseRaw_Output) ctxctor1 {
		response := outputAsValue(callOutput)
		response.Freeze()
		return func(state starlark.Value) starlark.Value {
			// state is mutated through checks
			return &ctxModule{
				request:  request,
				response: response,
				state:    state,
			}
		}
	}
}

// Modified https://github.com/google/starlark-go/blob/ebe61bd709bf/starlarkstruct/module.go
type ctxModule struct {
	accessedState     bool
	request, response starlark.Value
	state             starlark.Value
	//TODO: specs             starlark.Value
	//TODO: CLI filter `--only="starlark.expr(ctx.specs)"`
	//TODO: ctx.specs stops being accessible on first ctx.state access
}

// TODO? easy access to generated parameters. For instance:
// post_id = ctx.request["parameters"]["path"]["{id}"] (note decoded int)

var _ starlark.HasAttrs = (*ctxModule)(nil)

func (m *ctxModule) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *ctxModule) String() string        { return "ctx" }
func (m *ctxModule) Truth() starlark.Bool  { return true }
func (m *ctxModule) Type() string          { return "ctx" }
func (m *ctxModule) AttrNames() []string   { return []string{"request", "response", "state"} }

func (m *ctxModule) Freeze() {
	m.request.Freeze()
	m.response.Freeze()
	m.state.Freeze()
}

func (m *ctxModule) Attr(name string) (starlark.Value, error) {
	switch name {
	case "request":
		if m.accessedState {
			return nil, errors.New("cannot access ctx.request after accessing ctx.state")
		}
		return m.request, nil
	case "response":
		if m.accessedState {
			return nil, errors.New("cannot access ctx.response after accessing ctx.state")
		}
		return m.response, nil
	case "state":
		m.accessedState = true
		return m.state, nil
	default:
		return nil, nil // no such method
	}
}
