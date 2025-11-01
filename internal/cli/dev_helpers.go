package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

func runDevSimple(mainPath, port, debug string) error {
	fmt.Println("\n🚀 Starting server...")
	
	args := []string{"run", mainPath}
	if debug == "true" {
		args = append([]string{"run", "-race"}, args[1:]...)
	}
	
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", port))
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	
	fmt.Printf("✅ Server started on port %s (PID: %d)\n", port, cmd.Process.Pid)
	fmt.Println("Press Ctrl+C to stop...")
	
	<-sigChan
	fmt.Println("\n🛑 Shutting down server...")
	
	if err := cmd.Process.Kill(); err != nil {
		fmt.Printf("Error killing process: %v\n", err)
	}
	
	cmd.Wait()
	fmt.Println("Server stopped")
	
	return nil
}

func runDevWithHotReload(mainPath, port, debug string) error {
	fmt.Println("\n🔥 Starting development server with hot reload...")
	
	watcher := &FileWatcher{
		paths:    []string{".", "internal", "cmd", "pkg"},
		excludes: []string{".git", "vendor", "node_modules", ".tmp", "*.log"},
		debounce: 1 * time.Second,
	}
	
	server := &DevServer{
		mainPath: mainPath,
		port:     port,
		debug:    debug,
		watcher:  watcher,
	}
	
	return server.Start()
}

type FileWatcher struct {
	paths    []string
	excludes []string
	debounce time.Duration
	mu       sync.Mutex
	lastChange time.Time
}

type DevServer struct {
	mainPath string
	port     string
	debug    string
	watcher  *FileWatcher
	cmd      *exec.Cmd
	mu       sync.Mutex
	stopping bool
}

func (ds *DevServer) Start() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	changeChan := make(chan bool, 1)
	go ds.watcher.Watch(changeChan)
	
	if err := ds.startServer(); err != nil {
		return err
	}
	
	fmt.Printf("✅ Development server started on port %s\n", ds.port)
	fmt.Println("📁 Watching for file changes...")
	fmt.Println("Press Ctrl+C to stop...") 
	
	for {
		select {
		case <-sigChan:
			fmt.Println("\n🛑 Shutting down development server...")
			ds.stopServer()
			return nil
			
		case <-changeChan:
			if !ds.stopping {
				fmt.Println("\n🔄 File changes detected, restarting server...")
				ds.restartServer()
			}
		}
	}
}

func (ds *DevServer) startServer() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	args := []string{"run", ds.mainPath}
	if ds.debug == "true" {
		args = append([]string{"run", "-race"}, args[1:]...)
	}
	
	ds.cmd = exec.Command("go", args...)
	ds.cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", ds.port))
	
	stdout, err := ds.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	
	stderr, err := ds.cmd.StderrPipe()
	if err != nil {
		return err
	}
	
	if err := ds.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	
	go ds.streamOutput(stdout, "OUT")
	go ds.streamOutput(stderr, "ERR")
	
	return nil
}

func (ds *DevServer) stopServer() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	ds.stopping = true
	
	if ds.cmd != nil && ds.cmd.Process != nil {
		ds.cmd.Process.Signal(syscall.SIGTERM)
		
		done := make(chan error, 1)
		go func() {
			done <- ds.cmd.Wait()
		}()
		
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			ds.cmd.Process.Kill()
			<-done
		}
		
		ds.cmd = nil
	}
}

func (ds *DevServer) restartServer() {
	ds.stopServer()
	time.Sleep(500 * time.Millisecond) 
	
	ds.mu.Lock()
	ds.stopping = false
	ds.mu.Unlock()
	
	if err := ds.startServer(); err != nil {
		fmt.Printf("❌ Failed to restart server: %v\n", err)
	} else {
		fmt.Println("✅ Server restarted successfully")
	}
}

func (ds *DevServer) streamOutput(reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Printf("[%s] %s\n", prefix, scanner.Text())
	}
}

func (fw *FileWatcher) Watch(changeChan chan<- bool) {
	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()
	
	lastModTimes := make(map[string]time.Time)
	
	fw.scanFiles(lastModTimes)
	
	for range ticker.C {
		if fw.hasChanges(lastModTimes) {
			fw.mu.Lock()
			now := time.Now()
			if now.Sub(fw.lastChange) >= fw.debounce { 
				fw.lastChange = now
				fw.mu.Unlock()
				
				select {
				case changeChan <- true:
				default:
					// full so skip !!
				}
			} else {
				fw.mu.Unlock()
			}
		}
	}
}

func (fw *FileWatcher) scanFiles(modTimes map[string]time.Time) {
	for _, path := range fw.paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		
		filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			
			for _, exclude := range fw.excludes {
				if strings.Contains(filePath, exclude) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			
			if !info.IsDir() && strings.HasSuffix(filePath, ".go") {
				modTimes[filePath] = info.ModTime()
			}
			// maybe config too like toml or json could be watch too not sure.
			return nil
		})
	}
}

func (fw *FileWatcher) hasChanges(modTimes map[string]time.Time) bool {
	newModTimes := make(map[string]time.Time)
	fw.scanFiles(newModTimes)
	
	for filePath, newTime := range newModTimes {
		if oldTime, exists := modTimes[filePath]; !exists || !newTime.Equal(oldTime) {
			for k, v := range newModTimes {
				modTimes[k] = v
			}
			return true
		}
	}
	
	for filePath := range modTimes {
		if _, exists := newModTimes[filePath]; !exists {
			delete(modTimes, filePath)
			return true
		}
	}
	
	return false
}

// helpers ...

func runBuild(mainPath, output, target string, static, compress bool) error {
	fmt.Printf("\n🏗️  Building from %s...\n", mainPath)
	
	args := []string{"build"}
	
	args = append(args, "-o", output)
	
	var ldflags []string
	
	if target == "production" {
		ldflags = append(ldflags, "-s", "-w") 
		
		if static {
			args = append(args, "-a")
			ldflags = append(ldflags, "-extldflags", "-static")
		}
	}
	
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", strings.Join(ldflags, " "))
	}
	
	args = append(args, mainPath)
	
	env := os.Environ()
	if static {
		env = append(env, "CGO_ENABLED=0")
	}
	
	if target == "production" {
		env = append(env, "GOOS=linux") 
	}
	
	fmt.Printf("📦 Running: go %s\n", strings.Join(args, " "))
	
	cmd := exec.Command("go", args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	
	if _, err := os.Stat(output); os.IsNotExist(err) {
		return fmt.Errorf("build output not found: %s", output)
	}
	
	fileInfo, err := os.Stat(output)
	if err != nil {
		return err
	}
	
	size := fileInfo.Size()
	sizeStr := formatBytes(size)
	
	fmt.Printf("✅ Build successful!\n")
	fmt.Printf("   Binary: %s (%s)\n", output, sizeStr)
	
	if compress {
		fmt.Printf("🗜️  Compressing binary...\n")
		if err := compressBinary(output); err != nil {
			fmt.Printf("⚠️  Compression failed: %v\n", err)
		} else {
			compressedInfo, _ := os.Stat(output + ".gz")
			if compressedInfo != nil {
				compressedSize := formatBytes(compressedInfo.Size())
				savings := float64(size-compressedInfo.Size()) / float64(size) * 100
				fmt.Printf("   Compressed: %s.gz (%s, %.1f%% smaller)\n", 
					output, compressedSize, savings)
			}
		}
	}
	
	if err := os.Chmod(output, 0755); err != nil {
		fmt.Printf("⚠️  Warning: could not make binary executable: %v\n", err)
	}
	
	fmt.Printf("\n💡 Run with: ./%s\n", output)
	
	return nil
}

func compressBinary(filename string) error {
	cmd := exec.Command("gzip", "-f", filename)
	return cmd.Run()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}