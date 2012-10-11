package dogconf

import (
	"./stable"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func astRegress(name string, input string) error {
	// Set up destination file to dump test results
	destFileName := filepath.Join("ast_regress", "results", name) + ".out"
	resultOut, err := newResultFile(destFileName)
	if err != nil {
		return err
	}

	defer resultOut.Close()

	// Write the input to the top of the output file because that
	// makes it easier to skim the corresponding results
	// immediately after it.
	_, err = io.WriteString(resultOut, "INPUT<\n"+input+"\n\n")
	if err != nil {
		return stable.Errorf(
			"Could echo test input to results file: %v", err)
	}

	// Run the parser, rendering either the generated AST or the
	// resultant error as a string.
	render := func() string {
		result, err := ParseRequest(bytes.NewBuffer([]byte(input)))
		if err != nil {
			return stable.Sprintf("%v\n", err)
		}

		return stable.Sprintf("%#v\n", result)
	}()

	// Write
	_, err = io.WriteString(resultOut, "OUTPUT>\n"+render)
	if err != nil {
		return stable.Errorf(
			"Could write test output to results file: %v", err)
	}

	// Open the expected-output file
	expectedFileName := filepath.Join(
		"ast_regress", "expected", name) + ".out"
	expectedFile, err := os.OpenFile(expectedFileName, os.O_RDONLY, 0666)
	if err != nil {
		return stable.Errorf(
			"Could not open expected output file at %v: %v",
			expectedFileName, err)
	}
	defer expectedFile.Close()

	// Perform a quick comparison between the bytes in memory and
	// the bytes on disk.  It is the intention that at a later
	// date a diff can be emitted in the slow-path when there is a
	// failure, even though technically 'diff' could also be
	// expensively used to determine if the test failed or not.
	resultBytes := []byte(resultOut.Bytes())

	// Read one more byte than required to see if expected output
	// is longer than result output.
	expectedBytes := make([]byte, len(resultBytes)+1)

	n, err := io.ReadAtLeast(expectedFile, expectedBytes, len(expectedBytes))
	switch err {
	case io.EOF:
		return stable.Errorf(
			"Expected output file is empty: %v", expectedFile)

	case io.ErrUnexpectedEOF:
		// Check if the read input has the same size and
		// contents.  The test must succeed if it does.
		if n != len(resultBytes) ||
			!bytes.Equal(resultBytes, expectedBytes[0:n]) {
			return stable.Errorf(
				"Difference between results and expected: %v",
				name)
		}

		return nil
	case nil:
		return stable.Errorf(
			"Difference between results and expected: %v", name)
	default:
		return stable.Errorf(
			"ast_regress bug: unexpected error %v", err)
	}

	panic("Non-covering switch")
}

func astRegressFail(t *testing.T, name string, input string) {
	err := astRegress(name, input)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteAll(t *testing.T) {
	astRegressFail(t, "delete_all", `[route all [delete]]`)
}

func TestDeleteAt(t *testing.T) {
	astRegressFail(t, "delete_at", `[route 'foo' @ 42 [delete]]`)
}

func TestCreateRouteAtTime(t *testing.T) {
	astRegressFail(t, "create_route_at",
		`[route 'bar' @ 42 [create [addr='123.123.123.125:5445']]]`)
}

func TestCreateRoute(t *testing.T) {
	astRegressFail(t, "create_route",
		`[route 'bar' [create [addr='123.124.123.125:5445']]]`)
}

func TestPatchRoute(t *testing.T) {
	astRegressFail(t, "patch_at_address",
		`[route 'bar' @ 1 [patch [addr='123.123.123.125:5445']]]`)
}

func TestGetRoute(t *testing.T) {
	astRegressFail(t, "get_one_route", `[route 'bar' [get]]`)
}

func TestGetAllRoutes(t *testing.T) {
	astRegressFail(t, "get_all_routes", `[route all [get]]`)
}

func TestGetOcnRoute(t *testing.T) {
	// This is bogus semantically, but it does generate a valid
	// syntax tree.
	astRegressFail(t, "get_ocn_route", `[route 'bar' @ 137 [get]]`)
}

func TestQuoting(t *testing.T) {
	astRegressFail(t, "quoting",
		`[route '!xp' @ 5 [patch [dbnameIn='x'',"',lock='true']]]`)
}

// Onto some negative tests, for input that should fail in particular
// ways.

func TestUnterminated(t *testing.T) {
	// Note the two missing ']]' at the end
	astRegressFail(t, "unterminated_action", `[route 'bar' @ 137 [delete`)

	// Note the missing ']' at the end.  This parses incorrectly
	astRegressFail(t, "unterminated_toplevel", `[route 'bar' @ 137 [delete]`)

	// Unterminated string
	astRegressFail(t, "unterminated_string", `[route 'bar @ 137 [delete]`)
}

func TestExtraBrackets(t *testing.T) {
	// Extra surrounding set of brackets around all
	astRegressFail(t, "extra_brackets_top_level",
		`[[route 'bar' @ 137 [delete]]]`)

	// Extra surrounding brackets around action
	astRegressFail(t, "extra_brackets_action",
		`[route 'bar' @ 137 [[delete]]]`)

	// Extra brackets around the target
	astRegressFail(t, "extra_brackets_target",
		`[route ['bar' @ 137] [delete]]`)
}
