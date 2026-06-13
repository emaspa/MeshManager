// Command amtd is the AMT protocol daemon for MeshManager.
//
// It exposes a local HTTP + WebSocket API on 127.0.0.1 that the Tauri/React
// frontend uses to talk to Intel AMT / vPro devices. All AMT protocol work
// (WS-MAN over Digest/TLS, and the binary redirection protocols for
// Serial-over-LAN, KVM and IDE-R) lives here so the UI never needs raw TCP.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emaspa/meshmanager/amtd/internal/amt"
	"github.com/emaspa/meshmanager/amtd/internal/api"
)

// Version is overridden at build time via -ldflags "-X main.Version=...".
var Version = "0.1.0-dev"

func main() {
	var (
		addr  = flag.String("addr", "127.0.0.1:0", "address to listen on (port 0 = OS-assigned)")
		token = flag.String("token", "", "bearer token required on every request; empty disables auth (dev only)")
		debug = flag.Bool("debug", false, "enable debug logging, including raw AMT message tracing")
	)
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	// Bind first so we can print the chosen port before serving. Tauri reads
	// this line from stdout to learn where the sidecar landed.
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Error("failed to bind", "addr", *addr, "err", err)
		os.Exit(1)
	}

	sessions := amt.NewSessionManager(*debug)
	srv := api.NewServer(sessions, *token, Version)

	httpServer := &http.Server{
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Contract with the Tauri launcher: the very first stdout line is the
	// machine-readable endpoint announcement.
	fmt.Printf("AMTD_LISTENING %s\n", ln.Addr().String())
	os.Stdout.Sync()
	logger.Info("amtd listening", "addr", ln.Addr().String(), "version", Version, "auth", *token != "")

	go func() {
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on Ctrl-C / SIGTERM (Tauri kills the sidecar on exit).
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	sessions.CloseAll()
}
