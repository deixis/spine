package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"net/http"

	"github.com/pkg/errors"
	lcontext "github.com/deixis/spine/context"
)

const (
	transitHeader   = "context-transit-bin"
	shipmentsHeader = "context-shipments-bin"
)

func encodeContext(ctx context.Context, req *http.Request) error {
	tr := lcontext.TransitFromContext(ctx)
	if tr != nil {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(tr); err != nil {
			return errors.Wrap(err, "failed to encode transit")
		}
		text := base64.StdEncoding.EncodeToString(buf.Bytes())
		req.Header.Add(transitHeader, text)
	}

	// Encode shipments
	var shipments []shipment
	lcontext.ShipmentRange(ctx, func(k string, v interface{}) bool {
		shipments = append([]shipment{{k, v}}, shipments...) // prepend
		return true
	})
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&shipments); err != nil {
		return errors.Wrap(err, "failed to encode shipment")
	}
	text := base64.StdEncoding.EncodeToString(buf.Bytes())
	req.Header.Add(shipmentsHeader, text)
	return nil
}

func decodeContext(parent context.Context, req *http.Request) (context.Context, error) {
	header := req.Header.Get(transitHeader)
	data, err := base64.StdEncoding.DecodeString(string(header))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode transit")
	}
	var ctx context.Context
	if len(data) > 0 {
		tr := lcontext.TransitFactory()
		if err := gob.NewDecoder(bytes.NewReader(data)).Decode(tr); err != nil {
			return nil, err
		}
		ctx = lcontext.TransitWithContext(parent, tr)
	} else {
		ctx, _ = lcontext.NewTransitWithContext(parent)
	}

	// Decode shipments
	header = req.Header.Get(shipmentsHeader)
	data, err = base64.StdEncoding.DecodeString(string(header))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode shipments")
	}
	if len(data) > 0 {
		var shipments []shipment
		if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&shipments); err != nil {
			return nil, err
		}
		for _, s := range shipments {
			ctx = lcontext.WithShipment(ctx, s.Key, s.Value)
		}
	}
	return ctx, nil
}

type shipment struct {
	Key   string
	Value interface{}
}
