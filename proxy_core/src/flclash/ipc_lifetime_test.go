package main

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestIPCAsyncResponseKeepsConnection(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	callback := make(chan func([]byte) error, 1)
	done := make(chan struct{})
	go func() {
		serveIPCResponses(server, false, func(write func([]byte) error) { callback <- write })
		close(done)
	}()
	write := <-callback
	select {
	case <-done:
		t.Fatal("connection closed before async reply")
	default:
	}
	go func() { _ = write([]byte("replyEOF")) }()
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	result, err := io.ReadAll(client)
	if err != nil || string(result) != "replyEOF" {
		t.Fatalf("reply=%q err=%v", result, err)
	}
	<-done
}

func TestIPCProtectSubscriptionSurvivesMultipleCallbacks(t *testing.T) {
	server, client := net.Pipe()
	callback := make(chan func([]byte) error, 1)
	done := make(chan struct{})
	go func() {
		serveIPCResponses(server, true, func(write func([]byte) error) { callback <- write })
		close(done)
	}()
	write := <-callback
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	for _, want := range []string{"protect1EOF", "protect2EOF"} {
		go func(value string) { _ = write([]byte(value)) }(want)
		result := make([]byte, len(want))
		if _, err := io.ReadFull(client, result); err != nil || string(result) != want {
			t.Fatalf("reply=%q err=%v", result, err)
		}
	}
	select {
	case <-done:
		t.Fatal("protect subscription closed before caller disconnect")
	default:
	}
	_ = client.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscription did not close with caller")
	}
}
