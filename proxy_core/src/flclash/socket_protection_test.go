package main

import (
	"sync"
	"testing"
	"time"
)

func TestProtectionAcknowledgementWakesMatchingSocket(t *testing.T) {
	var protected, waiters sync.Map
	first, second := make(chan struct{}, 1), make(chan struct{}, 1)
	waiters.Store(int64(1), first)
	waiters.Store(int64(2), second)
	acknowledgeSocketProtection(2, &protected, &waiters)
	select {
	case <-second:
	case <-time.After(time.Second):
		t.Fatal("protected socket remained blocked")
	}
	select {
	case <-first:
		t.Fatal("acknowledgement woke a different socket")
	default:
	}
	if _, ok := protected.Load(int64(2)); !ok {
		t.Fatal("legacy acknowledgement was lost")
	}
}

func TestProtectionAcknowledgementHandlesDuplicateAndLateReplies(t *testing.T) {
	var protected, waiters sync.Map
	ch := make(chan struct{}, 1)
	waiters.Store(int64(1), ch)
	done := make(chan struct{})
	go func() {
		acknowledgeSocketProtection(1, &protected, &waiters)
		acknowledgeSocketProtection(1, &protected, &waiters)
		waiters.Delete(int64(1))
		acknowledgeSocketProtection(1, &protected, &waiters)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("duplicate or late acknowledgement blocked")
	}
}
