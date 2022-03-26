package handler

import (
	"net/http"

	"github.com/goccy/go-json"
	"github.com/roadrunner-server/api/v2/payload"
)

func (h *Handler) writeStreamHeader(pld *payload.Payload, w http.ResponseWriter) (int, error) {
	rsp := h.getRsp()
	defer h.putRsp(rsp)

	// unmarshal context into response
	err := json.Unmarshal(pld.Context, rsp)
	if err != nil {
		return 0, err
	}

	// handle push headers
	if len(rsp.Headers[HTTP2Push]) != 0 {
		push := rsp.Headers[HTTP2Push]

		if pusher, ok := w.(http.Pusher); ok {
			for i := 0; i < len(push); i++ {
				err = pusher.Push(rsp.Headers[HTTP2Push][i], nil)
				if err != nil {
					return 0, err
				}
			}
		}
	}

	if len(rsp.Headers[Trailer]) != 0 {
		handleTrailers(rsp.Headers)
	}

	// write all headers from the response to the writer
	for k := range rsp.Headers {
		for kk := range rsp.Headers[k] {
			w.Header().Add(k, rsp.Headers[k][kk])
		}
	}

	w.WriteHeader(rsp.Status)

	// copy, rsp.Status will be destroyed in the sync.Pool
	status := rsp.Status
	return status, nil
}

func (h *Handler) writeStream(pld *payload.Payload, w http.ResponseWriter, once bool) error {
	if !once {
		_, errWr := h.writeStreamHeader(pld, w)
		if errWr != nil {
			return errWr
		}
	}

	_, err := w.Write(pld.Body)
	if err != nil {
		return err
	}

	// do not use buffers, flush immediately
	flusher := w.(http.Flusher)
	flusher.Flush()

	return nil
}

func (h *Handler) putErrCh(c chan error) {
	select {
	case <-c:
		break
	default:
		break
	}

	h.errPool.Put(c)
}

func (h *Handler) getErrCh() chan error {
	return h.errPool.Get().(chan error)
}