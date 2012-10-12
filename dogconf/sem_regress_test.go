// Semantic Analysis tests
//
// A lot of infrastructure is a blatent copy of astRegress
//
// TODO: Refactor if this remains sufficiently similar for a little
// while.  Copied and search-and-replaced on 2012-10-10
package dogconf

import (
	"./stable"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func semRegress(name string, input string) error {
	// Set up destination file to dump test results
	destFileName := filepath.Join("sem_regress", "results", name) + ".out"
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

	// Run the parser, rendering either the generated SEM or the
	// resultant error as a string.
	render := func() string {
		parsed, err := ParseRequest(bytes.NewBuffer([]byte(input)))
		if err != nil {
			return stable.Sprintf("%v\n", err)
		}

		result, err := Analyze(parsed)
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
		"sem_regress", "expected", name) + ".out"
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
			"sem_regress bug: unexpected error %v", err)
	}

	panic("Non-covering switch")
}

func semRegressFail(t *testing.T, name string, input string) {
	err := semRegress(name, input)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestSemDeleteAll(t *testing.T) {
	semRegressFail(t, "delete_all", `[route all [delete]]`)
}

func TestSemDeleteAt(t *testing.T) {
	semRegressFail(t, "delete_at", `[route 'foo' @ 42 [delete]]`)
}

func TestSemCreateRouteAtTime(t *testing.T) {
	semRegressFail(t, "create_route_at",
		`[route 'bar' @ 42 [create [addr='123.123.123.125:5445']]]`)
}

func TestSemCreateRoute(t *testing.T) {
	semRegressFail(t, "create_route",
		`[route 'bar' [create [addr='123.124.123.125:5445']]]`)
}

func TestSemPatchRoute(t *testing.T) {
	semRegressFail(t, "patch_at_address",
		`[route 'bar' @ 1 [patch [addr='123.123.123.125:5445']]]`)
}

func TestSemGetRoute(t *testing.T) {
	semRegressFail(t, "get_one_route", `[route 'bar' [get]]`)
}

func TestSemGetAllRoutes(t *testing.T) {
	semRegressFail(t, "get_all_routes", `[route all [get]]`)
}

func TestSemGetOcnRoute(t *testing.T) {
	// This is bogus semantically and should fail
	semRegressFail(t, "get_ocn_route", `[route 'bar' @ 137 [get]]`)
}

func TestSemQuoting(t *testing.T) {
	semRegressFail(t, "quoting",
		`[route '!xp' @ 5 [patch [dbnameIn='x'',"',lock='true']]]`)
}
