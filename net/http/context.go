package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"net/http"

	scontext "github.com/deixis/spine/context"
	"github.com/pkg/errors"
)

const (
	transitHeader   = "context-transit-bin"
	shipmentsHeader = "context-shipments-bin"
)

func encodeContext(ctx context.Context, req *http.Request) error {
	tr := scontext.TransitFromContext(ctx)
	if tr != nil {
		data, err := tr.MarshalBinary()
		if err != nil {
			return errors.Wrap(err, "failed to encode transit")
		}
		text := base64.StdEncoding.EncodeToString(data)
		req.Header.Add(transitHeader, text)
	}

	// Encode shipments
	var shipments []shipment
	scontext.ShipmentRange(ctx, func(k string, v interface{}) bool {
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
		tr := scontext.TransitFactory()
		if err := tr.UnmarshalBinary(data); err != nil {
			return nil, err
		}
		ctx = scontext.TransitWithContext(parent, tr)
	} else {
		ctx, _ = scontext.NewTransitWithContext(parent)
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
			ctx = scontext.WithShipment(ctx, s.Key, s.Value)
		}
	}
	return ctx, nil
}

type shipment struct {
	Key   string
	Value interface{}
}
