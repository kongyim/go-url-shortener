package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGracefulShutdownWaitsForActiveRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	finishRequest := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)

		<-finishRequest

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("completed"))
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	server := &http.Server{
		Handler: handler,
	}

	serverDone := make(chan error, 1)

	go func() {
		serverDone <- server.Serve(listener)
	}()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	requestDone := make(chan error, 1)

	go func() {
		resp, err := client.Get(
			"http://" + listener.Addr().String(),
		)
		if err != nil {
			requestDone <- err
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			requestDone <- err
			return
		}

		if resp.StatusCode != http.StatusOK || string(body) != "completed" {
			requestDone <- io.ErrUnexpectedEOF
			return
		}

		requestDone <- nil
	}()

	select {
	case <-requestStarted:
		// The handler is now processing the request.
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	shutdownDone := make(chan error, 1)

	go func() {
		shutdownDone <- server.Shutdown(shutdownCtx)
	}()

	// While the handler is blocked, Shutdown must not complete.
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown finished before active request: %v", err)
	case <-time.After(100 * time.Millisecond):
		// Expected: Shutdown is still waiting.
	}

	// Allow the active request to finish.
	close(finishRequest)

	select {
	case err := <-requestDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("active request did not finish")
	}

	select {
	case err := <-shutdownDone:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not finish")
	}

	select {
	case err := <-serverDone:
		assert.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP server did not stop")
	}
}

func TestShutdownTimeoutFallsBackToClose(t *testing.T) {
	requestStarted := make(chan struct{})
	finishRequest := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)

		<-finishRequest

		w.WriteHeader(http.StatusOK)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &http.Server{
		Handler: handler,
	}

	serverDone := make(chan error, 1)

	go func() {
		serverDone <- server.Serve(listener)
	}()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	requestDone := make(chan error, 1)

	go func() {
		resp, err := client.Get(
			"http://" + listener.Addr().String(),
		)
		if err == nil {
			resp.Body.Close()
		}
		requestDone <- err
	}()

	// Wait until the handler has started.
	select {
	case <-requestStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}

	// Use an already-expired context for a deterministic timeout.
	shutdownCtx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	// Shutdown should fail because the active request
	// has not completed and the context is canceled.
	err = server.Shutdown(shutdownCtx)

	require.ErrorIs(t, err, context.Canceled)

	// Simulate the fallback used in main.go.
	require.NoError(t, server.Close())

	// The client should fail because the connection was closed.
	select {
	case err := <-requestDone:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("client request did not finish")
	}

	// Release the blocked handler to avoid leaking a goroutine.
	close(finishRequest)

	select {
	case err := <-serverDone:
		assert.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP server did not stop")
	}
}
