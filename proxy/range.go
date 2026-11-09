package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reservoir/cache"
	"reservoir/proxy/fetch"
	"reservoir/proxy/headers"
	"reservoir/proxy/responder"
)

func (p *Proxy) handleRangeRequest(r responder.Responder, req *http.Request, cached *cache.Entry[fetch.CacheMetadata], key cache.CacheKey, clientHd *headers.HeaderDirectives) error {
	rangeHeader := clientHd.Range.Value()
	start, end, err := rangeHeader.SliceSize(cached.Metadata.Size)
	if err != nil {
		slog.Error("Error slicing Range header", "url", req.URL, "key", key, "error", err, "range_header", rangeHeader, "file_size", cached.Metadata.Size)

		if !p.cfg.Proxy.RetryOnInvalidRange.Read() {
			slog.Error("Sending 416 Range Not Satisfiable due to invalid Range header from client.", "url", req.URL, "key", key, "range_header", rangeHeader)

			r.SetHeader("Accept-Ranges", "bytes")
			r.SetHeader("Content-Range", fmt.Sprintf("bytes */%d", cached.Metadata.Size))
			p.writeError(r, "invalid Range header", http.StatusRequestedRangeNotSatisfiable)

			return ErrRangeNotSatisfiable
		}

		slog.Info("Retrying request without Range header due to invalid Range header from client.", "url", req.URL, "key", key, "range_header", rangeHeader)

		clientHd.Range.SyncRemove(req.Header)
		fetched, err := p.fetcher.Fetch(req, key, clientHd)
		if err != nil {
			slog.Error("Error fetching resource without Range header", "url", req.URL, "key", key, "error", err)
			return err
		}

		data, header, status := fetched.Response()
		defer data.Close()

		r.SetHeaders(header)
		return finalizeAndRespond(p.metrics, r, data, status, req)
	}

	if clientHd.IfRange.IsPresent() {
		ifRange := clientHd.IfRange.Value()
		if ifRange.IsLeft() {
			// IfRange is ETag
			etagIfRange := ifRange.ForceUnwrapLeft()
			if etagIfRange != cached.Metadata.Object.ETag {
				slog.Info("If-Range does not match cached ETag. Sending full 200 response.", "url", req.URL, "key", key)
				return ErrIfRangeMismatch
			}
		} else {
			// IfRange is Time
			timeIfRange := ifRange.ForceUnwrapRight()
			if timeIfRange.Before(cached.Metadata.Object.LastModified) {
				slog.Info("If-Range does not match cached Last-Modified. Sending full 200 response.", "url", req.URL, "key", key)
				return ErrIfRangeMismatch
			}
		}
	}

	length := end - start + 1
	slog.Debug("Serving Range request from cache", "url", req.URL, "key", key, "range_header", rangeHeader, "start", start, "end", end, "length", length)

	r.SetHeaders(cached.Metadata.Object.Header)
	r.SetHeader("Accept-Ranges", "bytes")
	r.SetHeader("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, cached.Metadata.Size))
	r.SetHeader("Content-Length", fmt.Sprintf("%d", length))
	r.SetHeader("Last-Modified", cached.Metadata.Object.LastModified.Format(http.TimeFormat))

	sections := io.NewSectionReader(cached.Data, start, length)
	return finalizeAndRespond(p.metrics, r, sections, http.StatusPartialContent, req)
}
