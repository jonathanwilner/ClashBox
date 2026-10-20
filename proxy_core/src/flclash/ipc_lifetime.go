package main

import (
	"io"
	"net"
	"sync"
)

// Async replies own the connection until their first response. Subscriptions
// (especially TUN socket protection) own it until the caller disconnects.
func serveIPCResponses(conn net.Conn, streaming bool, dispatch func(func([]byte) error)) {
	defer conn.Close()
	peerClosed := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, conn)
		close(peerClosed)
	}()
	replied := make(chan struct{})
	var once sync.Once
	var writeMu sync.Mutex
	dispatch(func(data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_, err := conn.Write(data)
		if !streaming || err != nil {
			once.Do(func() { close(replied) })
		}
		return err
	})
	select {
	case <-replied:
	case <-peerClosed:
	}
}
