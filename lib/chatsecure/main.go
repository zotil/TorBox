package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"torbox/chatsecure/server"
	"torbox/chatsecure/storage"
)

func main() {
	var (
		nameFlag        string
		onionDomainFlag string
		devFlag         bool
		debugFlag       bool
	)

	flag.StringVar(&nameFlag, "name", "", "Onion Service Name")
	flag.StringVar(&nameFlag, "n", "", "Onion Service Name (shorthand)")
	flag.StringVar(&onionDomainFlag, "onion-domain", "", "Onion domain where TCS will be published")
	flag.StringVar(&onionDomainFlag, "od", "", "Onion domain where TCS will be published (shorthand)")
	flag.BoolVar(&devFlag, "dev", false, "Run in development mode (Default: false)")
	flag.BoolVar(&debugFlag, "debug", false, "Run in debug mode (Default: false)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -n, --name string\n        Onion Service Name\n")
		fmt.Fprintf(os.Stderr, "  -od, --onion-domain string\n        Onion domain where TCS will be published\n")
		fmt.Fprintf(os.Stderr, "  --dev\n        Run in development mode (Default: 0/false)\n")
		fmt.Fprintf(os.Stderr, "  --debug\n        Run in debug mode (Default: 0/false)\n")
	}

	flag.Parse()

	if nameFlag == "" {
		fmt.Println("Error: You must specify a name.")
		fmt.Printf("run %s --help for more info.\n", os.Args[0])
		os.Exit(1)
	}

	if onionDomainFlag == "" {
		fmt.Println("Error: Onion domain is required")
		fmt.Printf("run %s --help for more info.\n", os.Args[0])
		os.Exit(1)
	}

	instanceName := nameFlag
	onionDomain := onionDomainFlag

	_ = os.Setenv("INSTANCE_NAME", instanceName)
	_ = os.Setenv("ONION_DOMAIN", onionDomain)

	// Determine base directory
	baseDir := findBaseDir()
	pidDir := filepath.Join(baseDir, "pid")
	_ = os.MkdirAll(pidDir, 0755)
	pidFile := filepath.Join(pidDir, fmt.Sprintf("%s.pid", instanceName))

	// Check if another instance is already running
	if isProcessRunning(pidFile) {
		fmt.Printf("Error: Another instance is already using name \"%s\"\n", instanceName)
		os.Exit(1)
	}

	if debugFlag {
		log.Printf("[tcs] Starting TorBox ChatSecure (Go) instance '%s' for domain '%s'", instanceName, onionDomain)
	}

	// Initialize ephemeral in-RAM database
	store, err := storage.InitDB(instanceName, debugFlag)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create and start WebSocket Hub
	hub := server.NewHub(store, debugFlag)
	go hub.Run()

	secretKey := server.GenerateSecretKey()
	srvHandler := server.NewServer(server.ServerConfig{
		BaseDir:     baseDir,
		OnionDomain: onionDomain,
		SecretKey:   secretKey,
		Dev:         devFlag,
		Debug:       debugFlag,
	}, store, hub)

	httpServer := &http.Server{
		Handler:      srvHandler,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  180 * time.Second,
	}

	var listener net.Listener
	var socketPath string

	if devFlag {
		addr := "0.0.0.0:5000"
		l, err := net.Listen("tcp", addr)
		if err != nil {
			_ = store.Close()
			log.Fatalf("Failed to bind TCP listener on %s: %v", addr, err)
		}
		listener = l
		fmt.Printf("> Run dev server on http://%s\n", addr)
	} else {
		socketPath = fmt.Sprintf("/var/run/tcs_%s.sock", instanceName)
		_ = os.Remove(socketPath)

		l, err := net.Listen("unix", socketPath)
		if err != nil {
			// Fallback for non-root development or testing environments
			fallbackPath := filepath.Join(os.TempDir(), fmt.Sprintf("tcs_%s.sock", instanceName))
			_ = os.Remove(fallbackPath)
			l, err = net.Listen("unix", fallbackPath)
			if err != nil {
				_ = store.Close()
				log.Fatalf("Failed to bind UNIX socket %s: %v", socketPath, err)
			}
			socketPath = fallbackPath
		}
		_ = os.Chmod(socketPath, 0777)
		listener = l
		fmt.Printf("> Run production server on %s\n", socketPath)
	}

	// Write PID file
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		log.Printf("Warning: Failed to write PID file: %v", err)
	}

	// Setup graceful shutdown signal handler
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-stop
		if debugFlag {
			log.Println("[tcs] Shutdown signal received. Cleaning up ephemeral storage and sockets...")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
		_ = listener.Close()

		// Clean ephemeral resources
		if socketPath != "" {
			_ = os.Remove(socketPath)
		}
		_ = os.Remove(pidFile)
		_ = store.Close()

		if debugFlag {
			log.Println("[tcs] Graceful cleanup finished. Goodbye.")
		}
		os.Exit(0)
	}()

	// Serve incoming connections
	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// findBaseDir locates the directory where the binary or source resides
func findBaseDir() string {
	// 1. Try executable path
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "webclient")); err == nil {
			return dir
		}
	}

	// 2. Try current working directory
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "webclient")); err == nil {
			return cwd
		}
		// If running from TorBox root directory
		tcsSub := filepath.Join(cwd, "lib", "chatsecure")
		if _, err := os.Stat(filepath.Join(tcsSub, "webclient")); err == nil {
			return tcsSub
		}
	}

	// 3. Fallback default
	return "/home/torbox/torbox/lib/chatsecure"
}

// isProcessRunning checks if the PID in pidFile belongs to a running process
func isProcessRunning(pidFile string) bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return false
	}

	// On unix, kill(pid, 0) checks if process exists without sending signal
	err = syscall.Kill(pid, 0)
	if err == nil {
		return true // Process is alive
	}
	if err == syscall.EPERM {
		return true // Process is alive but owned by another user
	}

	// Process doesn't exist; remove stale pid file
	_ = os.Remove(pidFile)
	return false
}
