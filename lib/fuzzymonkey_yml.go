package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	// "net/url"
	"os"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	// yaml "gopkg.in/yaml.v2"
)

const (
	// lastCfgVersion = 1
	defaultCfgHost = "http://localhost:3000"
)

// FIXME: use new Modeler intf struc to pass these
var addHeaderAuthorization, addHost *string

// ModelerFunc TODO
type ModelerFunc func(d starlark.StringDict) (Modeler, *ModelerError)
type slBuiltin func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

// ModelerError TODO
type ModelerError struct {
	FieldRead, Want, Got string
}

func (me *ModelerError) Error(modelerName string) error {
	return fmt.Errorf("%s(%s = ...) must be %s, got: %s",
		modelerName, me.FieldRead, me.Want, me.Got)
}

func RegisterModeler(name string, fn ModelerFunc) {
	if _, ok := registeredIRModels[name]; ok {
		panic(fmt.Sprintf("modeler %q is already registered", name))
	}
	registeredIRModels[name] = fn
}

func modelerOpenAPIv3(d starlark.StringDict) (Modeler, *ModelerError) {
	mo := &ModelOpenAPIv3{}
	var (
		found              bool
		field              string
		file, host, hAuthz starlark.Value
	)

	field = "file"
	if file, found = d[field]; !found || file.Type() != "string" {
		e := &ModelerError{FieldRead: field, Want: "a string", Got: file.Type()}
		return nil, e
	}
	mo.File = file.(starlark.String).GoString()

	field = "host"
	if host, found = d[field]; found && host.Type() != "string" {
		e := &ModelerError{FieldRead: field, Want: "a string", Got: host.Type()}
		return nil, e
	}
	if found {
		h := host.(starlark.String).GoString()
		mo.Host = h
		addHost = &h
	}

	field = "header_authorization"
	if hAuthz, found = d[field]; found && hAuthz.Type() != "string" {
		e := &ModelerError{FieldRead: field, Want: "a string", Got: hAuthz.Type()}
		return nil, e
	}
	if found {
		authz := hAuthz.(starlark.String).GoString()
		mo.HeaderAuthorization = authz
		addHeaderAuthorization = &authz
	}

	return mo, nil
}

func (mnk *monkey) modelMaker(modelName string, modeler ModelerFunc) slBuiltin {
	return func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (ret starlark.Value, err error) {
		ret = starlark.None
		fname := b.Name()
		if args.Len() != 0 {
			err = fmt.Errorf("%s(...) does not take positional arguments", fname)
			return
		}

		u := make(starlark.StringDict, len(kwargs))
		r := make(starlark.StringDict, len(kwargs))
		for _, kv := range kwargs {
			k, v := kv.Index(0), kv.Index(1)
			key := k.(starlark.String).GoString()
			reserved := false
			if err = printableASCII(key); err != nil {
				err = errors.Wrap(err, "illegal field")
				log.Println("[ERR]", err)
				return
			}
			for i, c := range key {
				if i == 0 && unicode.IsUpper(c) {
					reserved = true
					break
				}
			}
			if !reserved {
				u[key] = v
			} else {
				r[key] = v
			}
		}
		mo, modelerErr := modeler(u)
		if modelerErr != nil {
			err = modelerErr.Error(modelName)
			log.Println("[ERR]", err)
			return
		}
		var resetter SUTResetter
		if resetter, err = newSUTResetter(fname, r); err != nil {
			return
		}
		mo.SetSUTResetter(resetter)

		mnk.modelers = append(mnk.modelers, mo)
		return
	}
}

func newSUTResetter(modelerName string, r starlark.StringDict) (SUTResetter, error) {
	var (
		ok bool
		v  starlark.Value
		vv starlark.String
		t  string
		// TODO: other SUTResetter.s
		resetter = &SUTShell{}
	)
	t = tExecStart
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.start = vv.GoString()
	}
	t = tExecReset
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.reset = vv.GoString()
	}
	t = tExecStop
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		resetter.stop = vv.GoString()
	}
	if len(r) != 0 {
		return nil, fmt.Errorf("unexpected arguments to %s(): %s", modelerName, strings.Join(r.Keys(), ", "))
	}
	return resetter, nil
}

var _ Modeler = (*ModelOpenAPIv3)(nil)

// ModelOpenAPIv3 describes OpenAPIv3 models
type ModelOpenAPIv3 struct {
	resetter SUTResetter

	/// Fields editable on initial run
	// File is a path within current directory pointing to a YAML spec
	File string
	// Host superseeds the spec's base URL
	Host string
	// HeaderAuthorization if non-empty is added to requests as bearer token
	HeaderAuthorization string

	// FIXME? tcap *tCapHTTP
}

// ToProto TODO
func (m *ModelOpenAPIv3) ToProto() isClt_Msg_Fuzz_Model_Model {
	return &Clt_Msg_Fuzz_Model_Openapiv3{&Clt_Msg_Fuzz_Model_OpenAPIv3{
		File:                m.File,
		Host:                m.Host,
		HeaderAuthorization: m.HeaderAuthorization,
	}}
}

// SetSUTResetter TODO
func (m *ModelOpenAPIv3) SetSUTResetter(sr SUTResetter) { m.resetter = sr }

// GetSUTResetter TODO
func (m *ModelOpenAPIv3) GetSUTResetter() SUTResetter { return m.resetter }

// Pretty TODO
func (m *ModelOpenAPIv3) Pretty(w io.Writer) (int, error) { return fmt.Fprintf(w, "%+v\n", m) }

var (
	_ starlark.Value           = (*modelState)(nil)
	_ starlark.HasAttrs        = (*modelState)(nil)
	_ starlark.HasSetKey       = (*modelState)(nil)
	_ starlark.IterableMapping = (*modelState)(nil)
	_ starlark.Sequence        = (*modelState)(nil)
	_ starlark.Comparable      = (*modelState)(nil)
)

type modelState struct {
	d *starlark.Dict
}

func newModelState(size int) *modelState {
	return &modelState{d: starlark.NewDict(size)}
}
func (s *modelState) Clear() error { return s.d.Clear() }
func (s *modelState) Delete(k starlark.Value) (starlark.Value, bool, error) {
	if err := slValuePrintableASCII(k); err != nil {
		return nil, false, err
	}
	return s.d.Delete(k)
}
func (s *modelState) Get(k starlark.Value) (starlark.Value, bool, error) {
	if err := slValuePrintableASCII(k); err != nil {
		return nil, false, err
	}
	return s.d.Get(k)
}
func (s *modelState) Items() []starlark.Tuple    { return s.d.Items() }
func (s *modelState) Keys() []starlark.Value     { return s.d.Keys() }
func (s *modelState) Len() int                   { return s.d.Len() }
func (s *modelState) Iterate() starlark.Iterator { return s.d.Iterate() }
func (s *modelState) SetKey(k, v starlark.Value) error {
	if err := slValuePrintableASCII(k); err != nil {
		return err
	}
	return s.d.SetKey(k, v)
}
func (s *modelState) String() string                           { return s.d.String() }
func (s *modelState) Type() string                             { return "ModelState" }
func (s *modelState) Freeze()                                  { s.d.Freeze() }
func (s *modelState) Truth() starlark.Bool                     { return s.d.Truth() }
func (s *modelState) Hash() (uint32, error)                    { return s.d.Hash() }
func (s *modelState) Attr(name string) (starlark.Value, error) { return s.d.Attr(name) }
func (s *modelState) AttrNames() []string                      { return s.d.AttrNames() }
func (s *modelState) CompareSameType(op syntax.Token, ss starlark.Value, depth int) (bool, error) {
	return s.d.CompareSameType(op, ss, depth)
}

// TODO: turn these into methods of *monkey
const (
	tEnv                     = "Env"
	tExecReset               = "ExecReset"
	tExecStart               = "ExecStart"
	tExecStop                = "ExecStop"
	tState                   = "State"
	tTriggerActionAfterProbe = "TriggerActionAfterProbe"
)

func (mnk *monkey) bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var env, def starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}
	envStr := env.GoString()
	// FIXME: actually maybe read env from Exec shell? These shells should inherit user env anyway?
	read, ok := os.LookupEnv(envStr)
	if !ok {
		return def, nil
	}
	mnk.envRead[envStr] = read
	return starlark.String(read), nil
}

type triggerActionAfterProbe struct {
	Name              starlark.String
	Probe             starlark.Tuple
	Predicate, Action *starlark.Function
}

func (mnk *monkey) bTriggerActionAfterProbe(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var trigger triggerActionAfterProbe
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"name?", &trigger.Name,
		"probe", &trigger.Probe,
		"predicate", &trigger.Predicate,
		"action", &trigger.Action,
	); err != nil {
		return nil, err
	}
	// TODO: enforce arities
	log.Println("[NFO] registering", b.Name(), trigger)
	if name := trigger.Name.GoString(); name == "" {
		trigger.Name = starlark.String(trigger.Action.Name())
		// TODO: complain if trigger.Name == "lambda"
	}
	mnk.triggers = append(mnk.triggers, trigger)
	return starlark.None, nil
}

func (mnk *monkey) loadCfg(localCfg string, showCfg bool) (err error) {
	if mnk.globals, err = starlark.ExecFile(mnk.thread, localCfg, nil, mnk.globals); err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			bt := evalErr.Backtrace()
			log.Println("[ERR]", bt)
			return
		}
		log.Println("[ERR]", err)
		return
	}

	// Ensure at least one model was defined
	ColorERR.Printf(">>> modelers: %v\n", mnk.modelers)
	if len(mnk.modelers) == 0 {
		err = errors.New("no modelers are registered")
		log.Println("[ERR]", err)
		return
	}

	ColorERR.Printf(">>> envs: %+v\n", mnk.envRead)
	ColorERR.Printf(">>> trigs: %+v\n", mnk.triggers)
	delete(mnk.globals, tEnv)
	delete(mnk.globals, tTriggerActionAfterProbe)

	if state, ok := mnk.globals[tState]; ok {
		d, ok := state.(*starlark.Dict)
		if !ok {
			err = fmt.Errorf("monkey State must be a dict, got: %s", state.Type())
			log.Println("[ERR]", err)
			return
		}
		delete(mnk.globals, tState)
		mnk.modelState = newModelState(d.Len())
		for _, kd := range d.Items() {
			k, v := kd.Index(0), kd.Index(1)
			// Ensure State keys are all String.s
			if err = slValuePrintableASCII(k); err != nil {
				err = errors.Wrap(err, "illegal State key")
				log.Println("[ERR]", err)
				return
			}
			// Ensure State values are all literals
			switch v.(type) {
			case starlark.NoneType, starlark.Bool:
			case starlark.Int, starlark.Float:
			case starlark.String:
			case *starlark.List, *starlark.Dict, *starlark.Set:
			default:
				err = fmt.Errorf("all initial State values must be litterals: State[%s] is %s", k.String(), v.Type())
				log.Println("[ERR]", err)
				return
			}
			ColorERR.Printf(">>> modelState: SetKey(%v, %v)\n", k, v)
			var vv starlark.Value
			if vv, err = slValueCopy(v); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if err = mnk.modelState.SetKey(k, vv); err != nil {
				log.Println("[ERR]", err)
				return
			}
		}
	} else {
		mnk.modelState = newModelState(0)
	}
	for key := range mnk.globals {
		if err = printableASCII(key); err != nil {
			err = errors.Wrap(err, "illegal export")
			log.Println("[ERR]", err)
			return
		}
		for _, c := range key {
			if unicode.IsUpper(c) {
				err = fmt.Errorf("user defined exports must not be uppercase: %q", key)
				log.Println("[ERR]", err)
				return
			}
			break
		}
	}
	log.Println("[NFO] starlark cfg globals:", len(mnk.globals.Keys()))
	ColorERR.Printf(">>> globals: %#v\n", mnk.globals)
	return
}

// func parseCfg(config []byte, showCfg bool) (cfg *UserCfg, err error) {
// 	var vsn struct {
// 		V interface{} `yaml:"version"`
// 	}
// 	if vsnErr := yaml.Unmarshal(config, &vsn); vsnErr != nil {
// 		const errFmt = "field 'version' missing! Try `version: %d`"
// 		err = fmt.Errorf(errFmt, lastCfgVersion)
// 		log.Println("[ERR]", err)
// 		ColorERR.Println(err)
// 		return
// 	}

// 	version, ok := vsn.V.(int)
// 	if !ok || !knownVersion(version) {
// 		err = fmt.Errorf("bad version: `%#v'", vsn.V)
// 		log.Println("[ERR]", err)
// 		ColorERR.Println(err)
// 		return
// 	}

// 	type cfgParser func(config []byte, showCfg bool) (cfg *UserCfg, err error)
// 	cfgParsers := []cfgParser{
// 		parseCfgV001,
// 	}

// 	return cfgParsers[version-1](config, showCfg)
// }

// func knownVersion(v int) bool {
// 	return 0 < v && v <= lastCfgVersion
// }

// func parseCfgV001(config []byte, showCfg bool) (cfg *UserCfg, err error) {
// 	var userConf struct {
// 		Version uint32   `yaml:"version"`
// 		Start   []string `yaml:"start"`
// 		Reset   []string `yaml:"reset"`
// 		Stop    []string `yaml:"stop"`
// 		Spec    struct {
// 			File           string       `yaml:"file"`
// 			Kind           string       `yaml:"kind"`
// 			KindIdentified UserCfg_Kind `yaml:"-"`
// 			Host           string       `yaml:"host"`
// 			HeaderAuthz    *string      `yaml:"authorization"`
// 		} `yaml:"spec"`
// 	}

// 	if err = yaml.UnmarshalStrict(config, &userConf); err != nil {
// 		log.Println("[ERR]", err)
// 		ColorERR.Println("Failed to parse", LocalCfg)
// 		r := strings.NewReplacer("not found", "unknown")
// 		for _, e := range strings.Split(err.Error(), "\n") {
// 			if end := strings.Index(e, " in type struct"); end != -1 {
// 				ColorERR.Println(r.Replace(e[:end]))
// 			}
// 		}
// 		return
// 	}

// 	expectedKind := UserCfg_OpenAPIv3
// 	if userConf.Spec.Kind != expectedKind.String() {
// 		err = errors.New("spec's kind must be set to OpenAPIv3")
// 		log.Println("[ERR]", err)
// 		ColorERR.Println(err)
// 		return
// 	}
// 	userConf.Spec.KindIdentified = expectedKind

// 	if userConf.Spec.Host == "" {
// 		def := defaultCfgHost
// 		log.Printf("[NFO] field 'host' is empty/unset: using %q\n", def)
// 		userConf.Spec.Host = def
// 	}
// 	if !strings.Contains(userConf.Spec.Host, "{{") {
// 		if _, err = url.ParseRequestURI(userConf.Spec.Host); err != nil {
// 			log.Println("[ERR]", err)
// 			return
// 		}
// 	}

// 	if userConf.Spec.HeaderAuthz != nil {
// 		addHeaderAuthorization = userConf.Spec.HeaderAuthz
// 	}

// 	if showCfg {
// 		ColorNFO.Println("Config:")
// 		enc := yaml.NewEncoder(os.Stderr)
// 		defer enc.Close()
// 		if err = enc.Encode(userConf); err != nil {
// 			log.Println("[ERR]", err)
// 			ColorERR.Printf("Failed to pretty-print %s: %#v\n", LocalCfg, err)
// 			return
// 		}
// 	}

// 	cfg = &UserCfg{
// 		Version: userConf.Version,
// 		File:    userConf.Spec.File,
// 		Kind:    userConf.Spec.KindIdentified,
// 		Runtime: &UserCfg_Runtime{
// 			Host: userConf.Spec.Host,
// 		},
// 		Exec: &UserCfg_Exec{
// 			Start:  userConf.Start,
// 			Reset_: userConf.Reset,
// 			Stop:   userConf.Stop,
// 		},
// 	}
// 	return
// }

func (cfg *UserCfg) script(kind ExecKind) []string {
	return map[ExecKind][]string{
		ExecKind_start: cfg.Exec.Start,
		ExecKind_reset: cfg.Exec.Reset_,
		ExecKind_stop:  cfg.Exec.Stop,
	}[kind]
}

// FindThenReadBlob looks for configured spec and reads it
func (cfg *UserCfg) FindThenReadBlob() (docPath string, blob []byte, err error) {
	//TODO: force relative paths & nested under workdir. Watch out for links
	docPath = cfg.File
	if docPath == `` {
		err = errors.New("Path to spec is empty")
		log.Println("[ERR]", err)
		ColorERR.Println(err)
		return
	}

	log.Println("[NFO] reading spec from", docPath)
	if blob, err = ioutil.ReadFile(docPath); err != nil {
		log.Println("[ERR]", err)
		ColorERR.Printf("Could not read '%s'\n", docPath)
	}
	return
}
