package log

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
)

const logTailerMaxRead int64 = 1 << 20

type logTailerFile interface {
	io.ReaderAt
	Close() error
}

type logTailerOps struct {
	open     func(string) (logTailerFile, error)
	stat     func(string) (os.FileInfo, error)
	sameFile func(os.FileInfo, os.FileInfo) bool
}

type logTailer struct {
	partial     []byte
	offset      int64
	path        string
	currentInfo os.FileInfo
	ops         logTailerOps
}

func defaultLogTailerOps() logTailerOps {
	return logTailerOps{
		open: func(path string) (logTailerFile, error) {
			return os.Open(path)
		},
		stat:     os.Stat,
		sameFile: os.SameFile,
	}
}

func newLogTailer(path string) (*logTailer, error) {
	return newLogTailerAtOffset(path, -1)
}

func newLogTailerAtOffset(path string, offset int64) (*logTailer, error) {
	return newLogTailerWithOps(path, offset, defaultLogTailerOps())
}

func newLogTailerWithOps(path string, offset int64, ops logTailerOps) (*logTailer, error) {
	info, err := ops.stat(path)
	if err != nil {
		return nil, err
	}

	if offset < 0 {
		offset = info.Size()
	}

	return &logTailer{
		offset:      offset,
		path:        path,
		currentInfo: info,
		ops:         ops,
	}, nil
}

func (t *logTailer) Tick(writeStream func([]byte) error) error {
	lines, err := t.readLines()
	if err != nil {
		slog.Error("failed to tail log file in SSE stream", "path", t.path, "error", err)
		return nil
	}

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if err := writeStream(line); err != nil {
			return err
		}
	}

	return nil
}

func (t *logTailer) readLines() ([][]byte, error) {
	logStat, err := t.ops.stat(t.path)
	if err != nil {
		return nil, err
	}

	rotated := t.currentInfo != nil && !t.ops.sameFile(t.currentInfo, logStat)
	truncated := logStat.Size() < t.offset
	if rotated || truncated {
		slog.Debug("log file rotated or truncated", "log_file", t.path, "rotated", rotated, "truncated", truncated)
		t.offset = 0
		t.partial = nil
	}

	if logStat.Size() == t.offset {
		slog.Debug("no new data in log file", "log_file", t.path)
		t.currentInfo = logStat
		return nil, nil
	}

	readNum := min(logStat.Size()-t.offset, logTailerMaxRead)
	file, err := t.ops.open(t.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, readNum)
	readCount, err := file.ReadAt(buf, t.offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	t.offset += int64(readCount)
	t.currentInfo = logStat
	chunk := append(t.partial, buf[:readCount]...)
	return t.splitCompleteLines(chunk, readCount)
}

func (t *logTailer) splitCompleteLines(chunk []byte, readCount int) ([][]byte, error) {
	lines := bytes.Split(chunk, []byte{'\n'})
	if len(lines) == 0 {
		slog.Debug("no complete lines read", "log_file", t.path)
		return nil, nil
	}

	t.partial = nil
	last := lines[len(lines)-1]
	if readCount == 0 || (len(last) > 0 && chunk[len(chunk)-1] != '\n') {
		t.partial = last
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}
