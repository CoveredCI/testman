package starlarktruth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func helper(t *testing.T, program string) (starlark.StringDict, error) {
	// Enabled so they can be tested
	resolve.AllowFloat = true
	resolve.AllowSet = true
	// resolve.AllowLambda = true

	predeclared := starlark.StringDict{}
	NewModule(predeclared)
	thread := &starlark.Thread{
		Name: t.Name(),
		Print: func(_ *starlark.Thread, msg string) {
			t.Logf("--> %s", msg)
		},
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			return nil, errors.New("load() unsupported")
		},
	}
	return starlark.ExecFile(thread, t.Name()+".star", program, predeclared)
	// if err != nil {
	// 	if evalErr, ok := err.(*starlark.EvalError); ok {
	// 		log.Fatal(evalErr.Backtrace())
	// 	}
	// 	log.Fatal(err)
	// }
	// require.NoError(t,err)

	// for _, name := range globals.Keys() {
	// 	v := globals[name]
	// 	t.Logf("%s (%s) = %s\n", name, v.Type(), v.String())
	// }
	// require.Len(t, globals, 0)
}

func testEach(t *testing.T, m map[string]error) {
	for code, expectedErr := range m {
		t.Run(code, func(t *testing.T) {
			globals, err := helper(t, code)
			require.Empty(t, globals)
			if expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, expectedErr.Error())
				require.True(t, errors.As(err, &expectedErr))
				require.IsType(t, expectedErr, err)
			}
		})
	}
}

func fail(value, expected string, suffixes ...string) error {
	suffix := ""
	if len(suffixes) == 1 {
		suffix = suffixes[0]
	}
	msg := "Not true that <" + value + "> " + expected + "." + suffix
	return newTruthAssertion(msg)
}

func TestTrue(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(True).isTrue()`:  nil,
		`AssertThat(True).isFalse()`: fail("True", "is False"),
	})
}

func TestFalse(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(False).isFalse()`: nil,
		`AssertThat(False).isTrue()`:  fail("False", "is True"),
	})
}

func TestTruthyThings(t *testing.T) {
	values := []string{
		`1`,
		`True`,
		`2.5`,
		`"Hi"`,
		`[3]`,
		`{4: "four"}`,
		`("my", "tuple")`,
		`set([5])`,
		`-1`,
	}
	m := make(map[string]error, 4*len(values))
	for _, v := range values {
		m[`AssertThat(`+v+`).isTruthy()`] = nil
		m[`AssertThat(`+v+`).isFalsy()`] = fail(v, "is falsy")
		m[`AssertThat(`+v+`).isFalse()`] = fail(v, "is False")
		if v != `True` {
			m[`AssertThat(`+v+`).isTrue()`] = fail(v, "is True",
				" However, it is truthy. Did you mean to call .isTruthy() instead?")
		}
	}
	testEach(t, m)
}

func TestFalsyThings(t *testing.T) {
	values := []string{
		`None`,
		`False`,
		`0`,
		`0.0`,
		`""`,
		`()`, // tuple
		`[]`,
		`{}`,
		`set()`,
	}
	m := make(map[string]error, 4*len(values))
	for _, v := range values {
		vv := v
		if v == `0.0` {
			vv = `0`
		}
		if v == `set()` {
			vv = `set([])`
		}
		m[`AssertThat(`+v+`).isFalsy()`] = nil
		m[`AssertThat(`+v+`).isTruthy()`] = fail(vv, "is truthy")
		m[`AssertThat(`+v+`).isTrue()`] = fail(vv, "is True")
		if v != `False` {
			m[`AssertThat(`+v+`).isFalse()`] = fail(vv, "is False",
				" However, it is falsy. Did you mean to call .isFalsy() instead?")
		}
	}
	testEach(t, m)
}

func TestIsAtLeast(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isAtLeast(3)`: nil,
		`AssertThat(5).isAtLeast(5)`: nil,
		`AssertThat(5).isAtLeast(8)`: fail("5", "is at least <8>"),
	})
}

func TestIsAtMost(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isAtMost(5)`: nil,
		`AssertThat(5).isAtMost(8)`: nil,
		`AssertThat(5).isAtMost(3)`: fail("5", "is at most <3>"),
	})
}

func TestIsGreaterThan(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isGreaterThan(3)`: nil,
		`AssertThat(5).isGreaterThan(5)`: fail("5", "is greater than <5>"),
		`AssertThat(5).isGreaterThan(8)`: fail("5", "is greater than <8>"),
	})
}

func TestIsLessThan(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isLessThan(8)`: nil,
		`AssertThat(5).isLessThan(5)`: fail("5", "is less than <5>"),
		`AssertThat(5).isLessThan(3)`: fail("5", "is less than <3>"),
	})
}

func TestCannotCompareToNone(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isAtLeast(None)`:     newInvalidAssertion("isAtLeast"),
		`AssertThat(5).isAtMost(None)`:      newInvalidAssertion("isAtMost"),
		`AssertThat(5).isGreaterThan(None)`: newInvalidAssertion("isGreaterThan"),
		`AssertThat(5).isLessThan(None)`:    newInvalidAssertion("isLessThan"),
	})
}

func TestIsEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isEqualTo(5)`: nil,
		`AssertThat(5).isEqualTo(3)`: fail("5", "is equal to <3>"),
	})
}

func TestIsEqualToFailsButFormattedRepresentationsAreEqual(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(0.3).isEqualTo(0.1+0.2)`: fail("0.3", "is equal to <0.3>",
			" However, their str() representations are equal."),
		`AssertThat(0.1+0.2).isEqualTo(0.3)`: fail("0.3", "is equal to <0.3>",
			" However, their str() representations are equal."),
	})
}

func TestIsNotEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isNotEqualTo(3)`: nil,
		`AssertThat(5).isNotEqualTo(5)`: fail("5", "is not equal to <5>"),
	})
}

func TestSequenceIsEqualToUsesContainsExactlyElementsInPlusInOrder(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat((3,5,[])).isEqualTo((3, 5, []))`: nil,
		`AssertThat((3,5,[])).isEqualTo(([],3,5))`: fail("(3, 5, [])",
			"contains exactly these elements in order <([], 3, 5)>"),
		`AssertThat((3,5,[])).isEqualTo((3,5,[],9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, [], 9)>. It is missing <9>"),
		`AssertThat((3,5,[])).isEqualTo((9,3,5,[],10))`: fail("(3, 5, [])",
			"contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>"),
		`AssertThat((3,5,[])).isEqualTo((3,5))`: fail("(3, 5, [])",
			"contains exactly <(3, 5)>. It has unexpected items <[]>"),
		`AssertThat((3,5,[])).isEqualTo(([],3))`: fail("(3, 5, [])",
			"contains exactly <([], 3)>. It has unexpected items <5>"),
		`AssertThat((3,5,[])).isEqualTo((3,))`: fail("(3, 5, [])",
			"contains exactly <(3,)>. It has unexpected items <5, []>"),
		`AssertThat((3,5,[])).isEqualTo((4,4,3,[],5))`: fail("(3, 5, [])",
			"contains exactly <(4, 4, 3, [], 5)>. It is missing <4 [2 copies]>"),
		`AssertThat((3,5,[])).isEqualTo((4,4))`: fail("(3, 5, [])",
			"contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>"),
		`AssertThat((3,5,[])).isEqualTo((3,5,9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>"),
		`AssertThat((3,5,[])).isEqualTo(())`: fail("(3, 5, [])", "is empty"),
	})
}

func TestSetIsEqualToUsesContainsExactlyElementsIn(t *testing.T) {
	s := `AssertThat(set([3, 5, 8]))`
	testEach(t, map[string]error{
		s + `.isEqualTo(set([3, 5, 8]))`: nil,
		s + `.isEqualTo(set([8, 3, 5]))`: nil,
		s + `.isEqualTo(set([3, 5, 8, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 8, 9])>. It is missing <9>"),
		s + `.isEqualTo(set([9, 3, 5, 8, 10]))`: fail("set([3, 5, 8])",
			"contains exactly <set([9, 3, 5, 8, 10])>. It is missing <9, 10>"),
		s + `.isEqualTo(set([3, 5]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5])>. It has unexpected items <8>"),
		s + `.isEqualTo(set([8, 3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([8, 3])>. It has unexpected items <5>"),
		s + `.isEqualTo(set([3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3])>. It has unexpected items <5, 8>"),
		s + `.isEqualTo(set([4]))`: fail("set([3, 5, 8])",
			"contains exactly <set([4])>. It is missing <4> and has unexpected items <3, 5, 8>"),
		s + `.isEqualTo(set([3, 5, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 9])>. It is missing <9> and has unexpected items <8>"),
		s + `.isEqualTo(set([]))`: fail("set([3, 5, 8])", "is empty"),
	})
}

func TestSequenceIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat((3, 5, [])).isEqualTo(3)`: fail("(3, 5, [])", "is equal to <3>"),
	})
}

func TestSetIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(set([3, 5, 8])).isEqualTo(3)`: fail("set([3, 5, 8])", "is equal to <3>"),
	})
}

func TestOrderedDictIsEqualToUsesContainsExactlyItemsInPlusInOrder(t *testing.T) {
	d1 := `((2, "two"), (4, "four"))`
	d2 := `((2, "two"), (4, "four"))`
	d3 := `((4, "four"), (2, "two"))`
	d4 := `((2, "two"), (4, "for"))`
	d5 := `((2, "two"), (4, "four"), (5, "five"))`
	s := `AssertThat(` + d1 + `).isEqualTo(`
	testEach(t, map[string]error{
		s + d2 + `)`: nil,
		s + d3 + `)`: fail(d1, "contains exactly these elements in order <"+d3+">"),

		// TODO: *expected
		// with self.Failure(
		//     "contains exactly <((2, 'two'),)>",
		//     "has unexpected items <[(4, 'four')]>",
		//     'often not the correct thing to do'):
		//   s.IsEqualTo(collections.OrderedDict(((2, 'two'),)))

		s + d4 + `)`: fail(d1,
			"contains exactly <"+d4+`>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),
		s + d5 + `)`: fail(d1,
			"contains exactly <"+d5+`>. It is missing <(5, "five")>`),
	})
}

func TestDictIsEqualToUsesContainsExactlyItemsIn(t *testing.T) {
	d := `{2: "two", 4: "four"}`
	dd := `{2: "two", 4: "for"}`
	ddd := `{2: "two", 4: "four", 5: "five"}`
	s := `AssertThat(` + d + `).isEqualTo(`
	testEach(t, map[string]error{
		s + d + `)`: nil,

		// TODO: *expected
		//     with self.Failure(
		//         "contains exactly <((2, 'two'),)>",
		//         "has unexpected items <[(4, 'four')]>",
		//         'often not the correct thing to do'):
		//       s.IsEqualTo({2: 'two'})

		s + dd + `)`: fail(d,
			"contains exactly <"+dd+`>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),
		//     expected = {2: 'two', 4: 'for'}
		//     with self.Failure(
		//         'contains exactly <{0!r}>'.format(tuple(expected.items())),
		//         "missing <[(4, 'for')]>",
		//         "has unexpected items <[(4, 'four')]>"):
		//       s.IsEqualTo(expected)

		s + ddd + `)`: fail(d,
			"contains exactly <"+ddd+`>. It is missing <(5, "five")>`),
	})
}

func TestIsEqualToComparedWithNonDictionary(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat({2:"two",4:"four"}).isEqualTo(3)`: fail(
			`{2: "two", 4: "four"}`,
			`is equal to <3>`,
		),
	})
}

// func (t *testing.T) {
// 	testEach(t, map[string]error{
// //Named
//   def testNamedMultilineString(self):
//     s = truth._StringSubject('line1\nline2').Named('string-name')
//     self.assertEqual(s._GetSubject(), 'actual string-name')

//   def testIsEqualToVerifiesEquality(self):
//     s = truth._StringSubject('line1\nline2\n')
//     s.IsEqualTo('line1\nline2\n')

// 	})
// }

func TestIsEqualToRaisesErrorWithVerboseDiff(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat("line1\nline2\nline3\nline4\nline5\n") \
         .isEqualTo("line1\nline2\nline4\nline6\n")`: newTruthAssertion(
			`Not true that actual is equal to expected, found diff:
*** Expected
--- Actual
***************
*** 1,5 ****
  line1
  line2
  line4
! line6
  
--- 1,6 ----
  line1
  line2
+ line3
  line4
! line5
  
`),
	})
}
