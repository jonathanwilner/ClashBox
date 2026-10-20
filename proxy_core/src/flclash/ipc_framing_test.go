package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type fragmentedIPCReader struct{ io.Reader }

func (r fragmentedIPCReader) Read(p []byte) (int, error) {
	if len(p) > 17 {
		p = p[:17]
	}
	return r.Reader.Read(p)
}

func TestDecodeIPCFragmentedNetworkProperties(t *testing.T) {
	want := struct {
		Method int      `json:"method"`
		Params []string `json:"params"`
	}{28, []string{`[{"ifaceName":"wlan0","domain":"` + strings.Repeat("network-properties-", 2048) + `"}]`}}
	payload, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Method int      `json:"method"`
		Params []string `json:"params"`
	}
	if err := decodeIPCRequest(fragmentedIPCReader{strings.NewReader(string(payload))}, &got); err != nil {
		t.Fatalf("fragmented request larger than former 10KB buffer: %v", err)
	}
	if got.Method != want.Method || len(got.Params) != 1 || got.Params[0] != want.Params[0] {
		t.Fatal("network-properties request changed during decoding")
	}
}

func TestDecodeIPCRejectsIncompleteAndOversizedRequests(t *testing.T) {
	for _, payload := range []string{`{"method":28,"params":[`, `{"params":["` + strings.Repeat("x", maxIPCRequestSize) + `"]}`} {
		var got any
		if err := decodeIPCRequest(strings.NewReader(payload), &got); err == nil {
			t.Fatal("accepted incomplete or oversized request")
		}
	}
}

func TestDecodeIPCReadinessProbe(t *testing.T) {
	var got any
	if err := decodeIPCRequest(strings.NewReader(""), &got); err != io.EOF {
		t.Fatalf("empty readiness probe: got %v, want EOF", err)
	}
}
