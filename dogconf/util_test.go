// Random assets shared between test code.
package dogconf

import (
	"./stable"
	"bytes"
	"io"
	"os"
)

// Used for fast and lucid handling of regression file generation.
// The main function is to record, in-memory, the bytes written to a
// result file (as to avoid re-reading them from disk) for future
// comparison against expected-output files on disk.
type resultFile struct {
	io.Writer
	io.Closer
	Byteser
	buf     bytes.Buffer
	diskOut io.WriteCloser
}

type Byteser interface {
	Bytes() []byte
}

// Open a new resultFile, failing the test should that not be
// possible.
func newResultFile(path string) (*resultFile, error) {
	destFile, err := os.OpenFile(path,
		os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		return nil, stable.Errorf(
			"Could not open results file at %v: %v",
			path, err)
	}

	rf := resultFile{}
	rf.diskOut = destFile
	rf.Writer = io.MultiWriter(destFile, &rf.buf)
	rf.Closer = rf.diskOut
	rf.Byteser = &rf.buf

	return &rf, nil
}
