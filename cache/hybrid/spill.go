package hybrid

import (
	"bytes"
	"errors"
	"io"
)

const hybridUnknownSpillChunkSize = 32 * 1024

type spillPrefix struct {
	chunks [][]byte
	size   int64
}

func (p *spillPrefix) append(chunk []byte) {
	p.chunks = append(p.chunks, chunk)
	p.size += int64(len(chunk))
}

func (p spillPrefix) readerWithTail(tail io.Reader) io.Reader {
	readers := make([]io.Reader, 0, len(p.chunks)+1)
	for _, chunk := range p.chunks {
		readers = append(readers, bytes.NewReader(chunk))
	}
	readers = append(readers, tail)
	return io.MultiReader(readers...)
}

func (p spillPrefix) bytes() []byte {
	if len(p.chunks) == 0 {
		return nil
	}
	if len(p.chunks) == 1 {
		return p.chunks[0]
	}

	data := make([]byte, 0, int(p.size))
	for _, chunk := range p.chunks {
		data = append(data, chunk...)
	}
	return data
}

func readUntilSpillThreshold(data io.Reader, threshold int64) ([]byte, io.Reader, bool, error) {
	if threshold < 0 {
		threshold = 0
	}

	prefix := spillPrefix{chunks: make([][]byte, 0, 1)}

	emptyReads := 0
	for {
		remaining := threshold - prefix.size + 1
		if remaining <= 0 {
			return nil, prefix.readerWithTail(data), true, nil
		}

		readSize := hybridUnknownSpillChunkSize
		if remaining < int64(readSize) {
			readSize = int(remaining)
		}
		chunk := make([]byte, readSize)
		n, err := data.Read(chunk)
		if n > 0 {
			prefix.append(chunk[:n])
			emptyReads = 0
			if prefix.size > threshold {
				return nil, prefix.readerWithTail(data), true, nil
			}
		}

		if errors.Is(err, io.EOF) {
			return prefix.bytes(), nil, false, nil
		}
		if err != nil {
			return nil, nil, false, err
		}
		if n == 0 {
			emptyReads++
			if emptyReads >= 100 {
				return nil, nil, false, io.ErrNoProgress
			}
		}
	}
}
