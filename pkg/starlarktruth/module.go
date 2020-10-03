package starlarktruth

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
)

const module = "AssertThat"

var (
	_ starlark.Value    = (*T)(nil)
	_ starlark.HasAttrs = (*T)(nil)
)

// NewModule registers a Starlark module of https://truth.dev/
func NewModule(predeclared starlark.StringDict) {
	predeclared[module] = starlark.NewBuiltin(module, func(
		thread *starlark.Thread,
		b *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		//TODO: store closedness in thread? bltn.Receiver() to check closedness
		var target starlark.Value
		if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &target); err != nil {
			return nil, err
		}
		return newT(target), nil
	})
}

func newT(target starlark.Value) *T { return &T{actual: target} }

func (t *T) String() string                           { return fmt.Sprintf("%s(%s)", module, t.actual.String()) }
func (t *T) Type() string                             { return module }
func (t *T) Freeze()                                  { t.actual.Freeze() }
func (t *T) Truth() starlark.Bool                     { return t.actual.Truth() }
func (t *T) Hash() (uint32, error)                    { return t.actual.Hash() }
func (t *T) Attr(name string) (starlark.Value, error) { return builtinAttr(t, name) }
func (t *T) AttrNames() []string                      { return attrNames }

type (
	attr  func(t *T, args ...starlark.Value) (starlark.Value, error)
	attrs map[string]attr
)

// TODO: turn all builtins matching *InOrder* into closedness-aware .inOrder()s

var (
	methods0args = attrs{
		"containsNoDuplicates": containsNoDuplicates,
		"isCallable":           isCallable,
		"isEmpty":              isEmpty,
		"isFalse":              isFalse,
		"isFalsy":              isFalsy,
		"isNone":               isNone,
		"isNotCallable":        isNotCallable,
		"isNotEmpty":           isNotEmpty,
		"isNotNone":            isNotNone,
		"isOrdered":            isOrdered,
		"isStrictlyOrdered":    isStrictlyOrdered,
		"isTrue":               isTrue,
		"isTruthy":             isTruthy,
	}

	methods1arg = attrs{
		"contains":                         contains,
		"containsAllIn":                    containsAllIn,
		"containsAllInOrderIn":             containsAllInOrderIn,
		"containsAnyIn":                    containsAnyIn,
		"containsExactlyElementsIn":        containsExactlyElementsIn,
		"containsExactlyElementsInOrderIn": containsExactlyElementsInOrderIn,
		"containsExactlyItemsIn":           containsExactlyItemsIn,
		"containsKey":                      containsKey,
		"containsMatch":                    containsMatch,
		"containsNoneIn":                   containsNoneIn,
		"doesNotContain":                   doesNotContain,
		"doesNotContainKey":                doesNotContainKey,
		"doesNotContainMatch":              doesNotContainMatch,
		"doesNotHaveAttribute":             doesNotHaveAttribute,
		"doesNotMatch":                     doesNotMatch,
		"endsWith":                         endsWith,
		"hasAttribute":                     hasAttribute,
		"hasLength":                        hasLength,
		"hasSize":                          hasSize,
		"isAtLeast":                        isAtLeast,
		"isAtMost":                         isAtMost,
		"isEqualTo":                        isEqualTo,
		"isGreaterThan":                    isGreaterThan,
		"isIn":                             isIn,
		"isLessThan":                       isLessThan,
		"isNotEqualTo":                     isNotEqualTo,
		"isNotIn":                          isNotIn,
		"isOrderedAccordingTo":             isOrderedAccordingTo,
		"isStrictlyOrderedAccordingTo":     isStrictlyOrderedAccordingTo,
		"matches":                          matches,
		"named":                            named,
		"startsWith":                       startsWith,
	}

	methods2args = attrs{
		"containsItem":       containsItem,
		"doesNotContainItem": doesNotContainItem,
	}

	methodsNargs = attrs{
		"containsAllOf":          containsAllOf,
		"containsAllOfInOrder":   containsAllOfInOrder,
		"containsAnyOf":          containsAnyOf,
		"containsExactly":        containsExactly,
		"containsExactlyInOrder": containsExactlyInOrder,
		"containsNoneOf":         containsNoneOf,
		"isAnyOf":                isAnyOf,
		"isNoneOf":               isNoneOf,
	}

	methods = []attrs{
		methodsNargs,
		methods0args,
		methods1arg,
		methods2args,
	}

	attrNames = func() []string {
		count := 0
		for _, ms := range methods {
			count += len(ms)
		}
		names := make([]string, 0, count)
		for _, ms := range methods {
			for name := range ms {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		return names
	}()
)

func findAttr(name string) (attr, int) {
	for i, ms := range methods[1:] {
		if m, ok := ms[name]; ok {
			return m, i
		}
	}
	if m, ok := methodsNargs[name]; ok {
		return m, -1
	}
	return nil, 0
}

func builtinAttr(t *T, name string) (starlark.Value, error) {
	method, nArgs := findAttr(name)
	if method == nil {
		return nil, nil // no such method
	}
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		closeness := 0
		if c, ok := thread.Local("closeness").(int); ok {
			// thread.Print(thread, fmt.Sprintf(">>> closeness = %d", c))
			closeness = c
		}
		defer thread.SetLocal("closeness", 1+closeness)

		if err := t.registerValues(thread); err != nil {
			return nil, err
		}

		var argz []starlark.Value
		switch nArgs {
		case -1:
			if len(kwargs) > 0 {
				return nil, fmt.Errorf("%s: unexpected keyword arguments", b.Name())
			}
			argz = []starlark.Value(args)
		case 0:
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
				return nil, err
			}
		case 1:
			var arg1 starlark.Value
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &arg1); err != nil {
				return nil, err
			}
			argz = append(argz, arg1)
		case 2:
			var arg1, arg2 starlark.Value
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &arg1, &arg2); err != nil {
				return nil, err
			}
			argz = append(argz, []starlark.Value{arg1, arg2}...)
		default:
			panic("unreachable")
		}

		ret, err := method(t, argz...)
		switch err {
		case nil:
			return ret, nil
		case errUnhandled:
			return nil, t.unhandled(b.Name(), argz...)
		default:
			return nil, err
		}
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(t), nil
}