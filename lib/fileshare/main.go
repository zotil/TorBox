package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"torbox/fileshare/server"
)

func printHelp(progName string) {
	fmt.Printf("Usage: %s [OPTIONS]\n\n", progName)
	fmt.Println("Options:")
	fmt.Println("  -n, --name TEXT           Onion Service Name")
	fmt.Println("  -od, --onion-domain TEXT  Onion domain where TFS will be published")
	fmt.Println("  -fp, --file-path TEXT     Path to share")
	fmt.Println("  --dev BOOLEAN             Run in development mode (Default: 0)")
	fmt.Println("  --help                    Show this message and exit.")
}

func parseCLIArgs() (name, onionDomain, filePath string, dev bool, help bool) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			help = true
			return
		case arg == "-n" || arg == "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-n="):
			name = strings.TrimPrefix(arg, "-n=")
		case strings.HasPrefix(arg, "--name="):
			name = strings.TrimPrefix(arg, "--name=")
		case arg == "-od" || arg == "--onion-domain":
			if i+1 < len(args) {
				onionDomain = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-od="):
			onionDomain = strings.TrimPrefix(arg, "-od=")
		case strings.HasPrefix(arg, "--onion-domain="):
			onionDomain = strings.TrimPrefix(arg, "--onion-domain=")
		case arg == "-fp" || arg == "--file-path":
			if i+1 < len(args) {
				filePath = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-fp="):
			filePath = strings.TrimPrefix(arg, "-fp=")
		case strings.HasPrefix(arg, "--file-path="):
			filePath = strings.TrimPrefix(arg, "--file-path=")
		case arg == "--dev" || arg == "-dev":
			if i+1 < len(args) && (args[i+1] == "1" || args[i+1] == "true" || args[i+1] == "0" || args[i+1] == "false") {
				dev = (args[i+1] == "1" || args[i+1] == "true")
				i++
			} else {
				dev = true
			}
		case strings.HasPrefix(arg, "--dev="):
			val := strings.TrimPrefix(arg, "--dev=")
			dev = (val == "1" || val == "true")
		}
	}
	return
}

func getBaseDir() string {
	exePath, err := os.Executable()
	if err == nil {
		resolved, err := filepath.EvalSymlinks(exePath)
		if err == nil {
			dir := filepath.Dir(resolved)
			if _, err := os.Stat(filepath.Join(dir, "webclient")); err == nil {
				return dir
			}
		}
	}
	cwd, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(cwd, "webclient")); err == nil {
		return cwd
	}
	if _, err := os.Stat(filepath.Join(cwd, "lib", "fileshare", "webclient")); err == nil {
		return filepath.Join(cwd, "lib", "fileshare")
	}
	return cwd
}

func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}

func main() {
	progName := os.Args[0]
	name, onionDomain, filePath, dev, help := parseCLIArgs()

	if help {
		printHelp(progName)
		os.Exit(0)
	}

	if name == "" {
		fmt.Println("Error: You must specify a name.")
		fmt.Printf("run %s --help for more info.\n", progName)
		os.Exit(1)
	}

	if onionDomain == "" {
		fmt.Println("Error: Onion domain is required")
		fmt.Printf("run %s --help for more info.\n", progName)
		os.Exit(1)
	}

	if filePath == "" {
		fmt.Println("Error: files-path must be set.")
		fmt.Printf("run %s --help for more info.\n", progName)
		os.Exit(1)
	}

	baseDir := getBaseDir()
	pidDir := filepath.Join(baseDir, "pid")
	_ = os.MkdirAll(pidDir, 0755)
	pidFile := filepath.Join(pidDir, name+".pid")

	// Check if another instance is already running
	if content, err := os.ReadFile(pidFile); err == nil {
		oldPID, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err == nil && isProcessAlive(oldPID) {
			fmt.Printf("Error: Another instance is already using name \"%s\"\n", name)
			os.Exit(1)
		}
		// Stale PID file, clean it up
		_ = os.Remove(pidFile)
	}

	// Write PID file
	currPID := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(currPID)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PID file: %v\n", err)
		os.Exit(1)
	}

	var sockPath string
	var listener net.Listener
	var err error

	if dev {
		listener, err = net.Listen("tcp", "0.0.0.0:8888")
		if err != nil {
			_ = os.Remove(pidFile)
			fmt.Fprintf(os.Stderr, "Error starting TCP listener: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("TorBox FileShare (TFS) running in DEV mode on http://0.0.0.0:8888\n")
	} else {
		sockPath = fmt.Sprintf("/var/run/tfs_%s.sock", name)
		_ = os.Remove(sockPath)
		listener, err = net.Listen("unix", sockPath)
		if err != nil {
			_ = os.Remove(pidFile)
			fmt.Fprintf(os.Stderr, "Error creating UNIX socket %s: %v\n", sockPath, err)
			os.Exit(1)
		}
		_ = os.Chmod(sockPath, 0666)
	}

	cleanup := func() {
		if !dev && sockPath != "" {
			_ = os.Remove(sockPath)
		}
		_ = os.Remove(pidFile)
	}
	defer cleanup()

	// Handle graceful shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	appHandler := server.NewServer(filePath, baseDir, onionDomain)
	httpServer := &http.Server{
		Handler: appHandler,
	}

	go func() {
		sig := <-sigChan
		_ = sig
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
		_ = listener.Close()
		cleanup()
		os.Exit(0)
	}()

	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		cleanup()
		os.Exit(1)
	}
}
