package server

// The V1 API is just a simple json encoded, REST-like API that runs on top of
// an HTTP server.

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/decred/dcrlnlpd/rpc/lprpc_v1"
	"github.com/gorilla/mux"
)

var errWrongKey = errors.New("wrong key")

type V1Handler struct {
	*Server
}

func NewV1Handler(s *Server, router *mux.Router) *V1Handler {
	v1 := &V1Handler{Server: s}
	router.Path("/api/v1/invoice").HandlerFunc(v1.handlerFunc(v1.handleInvoice))
	router.Path("/api/v1/policy").HandlerFunc(v1.handlerFunc(v1.handlePolicy))
	return v1
}

func (v1 *V1Handler) handleInvoice(req *http.Request) (interface{}, error) {
	var reqInv lprpc_v1.InvoiceRequest
	if err := lprpc_v1.Decode(req.Body, &reqInv); err != nil {
		return nil, err
	}

	if v1.cfg.CreateKey != nil && subtle.ConstantTimeCompare(v1.cfg.CreateKey, []byte(reqInv.Key)) != 1 {
		return nil, errWrongKey
	}

	var res lprpc_v1.InvoiceResponse
	var err error
	res.Invoice, err = v1.Server.CreateInvoice(req.Context(), reqInv.TargetNode, reqInv.ChannelSize)

	return res, err
}

func (v1 *V1Handler) handlePolicy(req *http.Request) (interface{}, error) {

	res := lprpc_v1.PolicyResponse{
		Node:               v1.nodeID,
		NodeAddresses:      v1.cfg.LNNodeAddrs,
		MinChanSize:        v1.cfg.MinChanSize,
		MaxChanSize:        v1.cfg.MaxChanSize,
		MaxNbChannels:      v1.cfg.MaxNbChannels,
		MinChanLifetime:    v1.cfg.MinChanLifetime,
		ChanInvoiceFeeRate: v1.cfg.ChanInvoiceFeeRate,
	}
	return res, nil
}

func (v1 *V1Handler) handlerFunc(f func(*http.Request) (interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		res, err := f(req)

		// Encode response if error is nil. Some errors are handled
		// specially, otherwise an internal server error is returned.
		switch {
		case errors.Is(err, lprpc_v1.ErrDecode), errors.Is(err, errPolicy):
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))

		case errors.Is(err, errWrongKey):
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(err.Error()))

		case err != nil:
			w.WriteHeader(http.StatusInternalServerError)
			v1.log.Errorf("Error processing request: %v", err)

		default:
			// No Error on request!
			if err := lprpc_v1.Encode(w, res); err != nil {
				v1.log.Errorf("Unable to encode full reply: %v", err)
			}
		}
	}
}
