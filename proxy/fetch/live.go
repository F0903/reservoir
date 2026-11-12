package fetch

import (
	"errors"
	"io"
	"reservoir/cache"
	"sync"
)

// defaultLiveWindow is how many bytes of a running download are kept in memory
// for the clients that read it while it is still being cached.
const defaultLiveWindow = 4 << 20

var (
	errDownloadIncomplete = errors.New("shared download ended before the end of the body")
	errLeftWindow         = errors.New("live body can no longer supply this offset")
)

// liveBody is the body of a download that is still running. The download writes
// to it and never waits: clients read along through a window of the newest bytes,
// and a client that is too slow for the window continues from another source.
type liveBody struct {
	key    cache.CacheKey                     // Where the cache stores the download.
	entry  cache.EntryMetadata[CacheMetadata] // What the cache entry will say once the download is stored.
	window int

	mu       sync.Mutex
	cond     *sync.Cond
	buf      []byte // The newest bytes of the body, at most window of them.
	base     int64  // Offset of buf[0] in the body.
	finished bool
	err      error // Why the body is incomplete. Nil once the whole body has been downloaded.
	cached   bool  // The whole body is in the cache.
}

func newLiveBody(key cache.CacheKey, entry cache.EntryMetadata[CacheMetadata], window int) *liveBody {
	b := &liveBody{
		key:    key,
		entry:  entry,
		window: window,
		buf:    make([]byte, 0, min(entry.Size, int64(window))),
		err:    errDownloadIncomplete,
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *liveBody) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.window {
		// Over the window: keep the newest half. Dropping half at a time keeps the copying cheap.
		drop := len(b.buf) - b.window/2
		b.base += int64(drop)
		b.buf = b.buf[:copy(b.buf, b.buf[drop:])]
	}
	b.mu.Unlock()

	b.cond.Broadcast()
	return len(p), nil
}

// settle records how the download went: err is set if it stopped before the end of
// the body, cached tells whether the whole body can now be read from the cache.
// Readers act on it once finish is called.
func (b *liveBody) settle(err error, cached bool) {
	b.mu.Lock()
	b.err, b.cached = err, cached
	b.mu.Unlock()
}

// finish ends the body. Readers that are waiting for more of it carry on.
func (b *liveBody) finish() {
	b.mu.Lock()
	b.finished = true
	b.mu.Unlock()

	b.cond.Broadcast()
}

// readable returns the offset up to which the body can be read right now. The
// last byte is held back until the download has finished: a client that has the
// whole body can then rely on the next request finding it in the cache.
func (b *liveBody) readable() int64 {
	end := b.base + int64(len(b.buf))
	if !b.finished && end > 0 && end == b.entry.Size {
		return end - 1
	}
	return end
}

// read copies the bytes at off into p, waiting for them if the download has not
// got that far. It returns io.EOF at the end of a complete body, and errLeftWindow
// when the bytes at off have to come from somewhere else.
func (b *liveBody) read(p []byte, off int64) (n int, cached bool, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// A reader behind the window waits for the end too: only then is it known
	// whether the cache has the body to continue from.
	for !b.finished && (off < b.base || off >= b.readable()) {
		b.cond.Wait()
	}

	end := b.readable()
	switch {
	case off >= b.base && off < end:
		return copy(p, b.buf[off-b.base:end-b.base]), false, nil
	case off == end && b.err == nil:
		return 0, false, io.EOF
	default:
		return 0, b.cached, errLeftWindow
	}
}

// resumeFunc opens the rest of a body from offset off: from the cache if cached, otherwise from upstream.
type resumeFunc func(off int64, cached bool) (io.ReadCloser, error)

// newReader returns a reader over the whole body, or nil if the first bytes have
// already left the window.
func (b *liveBody) newReader(resume resumeFunc) *liveReader {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.base > 0 {
		return nil
	}
	return &liveReader{body: b, resume: resume}
}

// liveReader is one client's view of a liveBody. It takes the place of a cache
// entry's data while the entry is still being written.
type liveReader struct {
	body   *liveBody
	off    int64
	rest   io.ReadCloser // Where the body continues once the window could not supply it.
	resume resumeFunc
}

func (r *liveReader) Read(p []byte) (int, error) {
	if r.rest != nil {
		return r.rest.Read(p)
	}

	n, cached, err := r.body.read(p, r.off)
	if !errors.Is(err, errLeftWindow) {
		r.off += int64(n)
		return n, err
	}

	rest, err := r.resume(r.off, cached)
	if err != nil {
		return 0, err
	}
	r.rest = rest
	return rest.Read(p)
}

func (r *liveReader) Close() error {
	if r.rest != nil {
		return r.rest.Close()
	}
	return nil
}

// Seek and ReadAt only exist to satisfy cache.EntryData. Range requests are the
// only users of them, and those never wait on a shared download.
func (r *liveReader) Seek(int64, int) (int64, error) {
	return 0, errors.ErrUnsupported
}

func (r *liveReader) ReadAt([]byte, int64) (int, error) {
	return 0, errors.ErrUnsupported
}

// liveSource hands a download to its liveBody while the cache reads it.
type liveSource struct {
	src   io.Reader
	body  *liveBody
	start func(*liveBody) // Called before the first read: from then on clients can read along.
	read  int64
}

func (s *liveSource) Read(p []byte) (int, error) {
	if s.start != nil {
		s.start(s.body)
		s.start = nil
	}

	n, err := s.src.Read(p)
	s.body.Write(p[:n])
	s.read += int64(n)
	return n, err
}

// settle records the outcome once the cache is done with the download.
func (s *liveSource) settle(cached bool) {
	var err error
	if s.read != s.body.entry.Size {
		err = errDownloadIncomplete
	}
	s.body.settle(err, cached)
}
