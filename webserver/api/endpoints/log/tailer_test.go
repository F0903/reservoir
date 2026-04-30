package log

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLogTailerStreamsCompleteLinesAndCarriesPartial(t *testing.T) {
	path := newTestLogFile(t, "")
	tailer, err := newLogTailerAtOffset(path, 0)
	if err != nil {
		t.Fatalf("newLogTailerAtOffset failed: %v", err)
	}

	appendTestLog(t, path, "first\npart")
	assertTailerLines(t, tailer, []string{"first"})

	appendTestLog(t, path, "ial\nsecond\nthird")
	assertTailerLines(t, tailer, []string{"partial", "second"})

	appendTestLog(t, path, "\n")
	assertTailerLines(t, tailer, []string{"third"})
}

func TestLogTailerStartsAtEndByDefault(t *testing.T) {
	path := newTestLogFile(t, "existing\n")
	tailer, err := newLogTailer(path)
	if err != nil {
		t.Fatalf("newLogTailer failed: %v", err)
	}

	assertTailerLines(t, tailer, nil)

	appendTestLog(t, path, "new\n")
	assertTailerLines(t, tailer, []string{"new"})
}

func TestLogTailerReadsFromStartAfterTruncation(t *testing.T) {
	path := newTestLogFile(t, "old line\nold partial")
	tailer, err := newLogTailerAtOffset(path, 0)
	if err != nil {
		t.Fatalf("newLogTailerAtOffset failed: %v", err)
	}

	assertTailerLines(t, tailer, []string{"old line"})

	if err := os.WriteFile(path, []byte("new line\n"), 0644); err != nil {
		t.Fatalf("truncate log file: %v", err)
	}

	assertTailerLines(t, tailer, []string{"new line"})
}

func TestLogTailerReadsFromStartAfterRotation(t *testing.T) {
	path := newTestLogFile(t, "old line\n")
	tailer, err := newLogTailerAtOffset(path, 0)
	if err != nil {
		t.Fatalf("newLogTailerAtOffset failed: %v", err)
	}

	assertTailerLines(t, tailer, []string{"old line"})

	rotatedPath := path + ".1"
	if err := os.Rename(path, rotatedPath); err != nil {
		t.Fatalf("rotate log file: %v", err)
	}
	if err := os.WriteFile(path, []byte("new line\n"), 0644); err != nil {
		t.Fatalf("write new log file: %v", err)
	}

	assertTailerLines(t, tailer, []string{"new line"})
}

func TestLogTailerRetriesAfterTransientStatFailure(t *testing.T) {
	statErr := errors.New("stat failed")
	file := &fakeTailerFile{id: "log", data: []byte("line\n")}
	tailer := &logTailer{
		path:        "log",
		currentInfo: fakeFileInfo{name: "log", size: 0, sys: "log"},
		offset:      0,
		ops: logTailerOps{
			open: func(string) (logTailerFile, error) {
				return file, nil
			},
			stat: func(string) (os.FileInfo, error) {
				if statErr != nil {
					return nil, statErr
				}
				return file.Stat()
			},
			sameFile: fakeSameFile,
		},
	}

	assertTailerLines(t, tailer, nil)

	statErr = nil
	assertTailerLines(t, tailer, []string{"line"})
}

func TestLogTailerRetriesAfterTransientReadFailure(t *testing.T) {
	readErr := errors.New("read failed")
	file := &fakeTailerFile{id: "log", data: []byte("line\n"), readErr: readErr}
	tailer := &logTailer{
		path:        "log",
		currentInfo: fakeFileInfo{name: "log", size: 0, sys: "log"},
		offset:      0,
		ops: logTailerOps{
			open: func(string) (logTailerFile, error) {
				return file, nil
			},
			stat: func(string) (os.FileInfo, error) {
				return file.Stat()
			},
			sameFile: fakeSameFile,
		},
	}

	assertTailerLines(t, tailer, nil)
	if tailer.offset != 0 {
		t.Fatalf("expected failed read to preserve offset 0, got %d", tailer.offset)
	}

	file.readErr = nil
	assertTailerLines(t, tailer, []string{"line"})
}

func newTestLogFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "proxy.log")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write test log file: %v", err)
	}
	return path
}

func appendTestLog(t *testing.T, path, contents string) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open test log for append: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(contents); err != nil {
		t.Fatalf("append test log: %v", err)
	}
}

func assertTailerLines(t *testing.T, tailer *logTailer, want []string) {
	t.Helper()

	var got []string
	if err := tailer.Tick(func(line []byte) error {
		got = append(got, string(line))
		return nil
	}); err != nil {
		t.Fatalf("tailer tick failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tailer lines mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

type fakeTailerFile struct {
	id      string
	data    []byte
	readErr error
}

func (f *fakeTailerFile) ReadAt(p []byte, off int64) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	if off >= int64(len(f.data)) {
		return 0, io.EOF
	}

	read := copy(p, f.data[off:])
	if read < len(p) {
		return read, io.EOF
	}
	return read, nil
}

func (f *fakeTailerFile) Stat() (os.FileInfo, error) {
	return fakeFileInfo{name: f.id, size: int64(len(f.data)), sys: f.id}, nil
}

func (f *fakeTailerFile) Close() error {
	return nil
}

type fakeFileInfo struct {
	name string
	size int64
	sys  any
}

func (f fakeFileInfo) Name() string {
	return f.name
}

func (f fakeFileInfo) Size() int64 {
	return f.size
}

func (f fakeFileInfo) Mode() os.FileMode {
	return 0644
}

func (f fakeFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (f fakeFileInfo) IsDir() bool {
	return false
}

func (f fakeFileInfo) Sys() any {
	return f.sys
}

func fakeSameFile(a, b os.FileInfo) bool {
	return a.Sys() == b.Sys()
}
