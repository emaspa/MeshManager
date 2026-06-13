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
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"github.com/emaspa/meshmanager/amtd/internal/amt"
	"github.com/emaspa/meshmanager/amtd/internal/api"
)

// Version is overridden at build time via -ldflags "-X main.Version=...".
var Version = "0.1.0-dev"

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:0", "address to listen on (port 0 = OS-assigned)")
		token      = flag.String("token", "", "bearer token required on every request; empty disables auth (dev only)")
		debug      = flag.Bool("debug", false, "enable debug logging, including raw AMT message tracing")
		logDir     = flag.String("log-dir", "", "directory for rotating log files; logs to stderr only if empty")
		logMaxAge  = flag.Int("log-max-age", 30, "days to retain rotated log files (0 = keep until count/size limits)")
		logBackups = flag.Int("log-max-backups", 20, "max number of rotated log files to keep")
	)
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}

	// Always log to stderr (captured by the Tauri shell). When a log dir is
	// given, also write a rotating file so testers can attach logs to reports.
	var w io.Writer = os.Stderr
	var logFile string
	if *logDir != "" {
		if err := os.MkdirAll(*logDir, 0o755); err == nil {
			logFile = filepath.Join(*logDir, "amtd.log")
			w = io.MultiWriter(os.Stderr, &lumberjack.Logger{
				Filename:   logFile,
				MaxSize:    5, // megabytes per file
				MaxBackups: *logBackups,
				MaxAge:     *logMaxAge, // days
				Compress:   true,
			})
		} else {
			fmt.Fprintf(os.Stderr, "log dir %q unusable: %v (stderr only)\n", *logDir, err)
		}
	}
	logger := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
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
	logger.Info("amtd started",
		"version", Version,
		"addr", ln.Addr().String(),
		"auth", *token != "",
		"pid", os.Getpid(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"go", runtime.Version(),
		"logFile", logFile,
	)

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
