package main

import (
	"encoding/json"
	"io"
)

// Bound requests without assuming that a stream read contains one whole JSON
// document. Responses can remain open for log and message subscriptions.
const maxIPCRequestSize = 4 * 1024 * 1024

func decodeIPCRequest(reader io.Reader, request any) error {
	return json.NewDecoder(io.LimitReader(reader, maxIPCRequestSize)).Decode(request)
}
