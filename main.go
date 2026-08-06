package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func main() {
	loginMode := flag.Bool("login", false, "Run OAuth device login flow and add account to pool")
	captureMode := flag.Bool("capture", false, "Run interactive OAuth capture (records ALL traffic)")
	host := flag.String("host", "0.0.0.0", "Listen host (0.0.0.0 allows LAN access)")
	port := flag.Int("port", 3457, "Proxy server port")
	addAccount := flag.Bool("add-account", false, "Add a new account via OAuth to the pool")
	showList := flag.Bool("list", false, "List all accounts in the pool")
	startMode := flag.Bool("start", false, "Build, start proxy, and open admin panel in browser")
	flag.Parse()

	if *startMode {
		buildAndStart(*host, *port)
		return
	}

	if *captureMode {
		if err := doFullCapture(); err != nil {
			log.Fatalf("Capture failed: %v", err)
		}
		return
	}

	if *loginMode || *addAccount {
		acc, err := addAccountFromDeviceAuth()
		if err != nil {
			log.Fatalf("Login failed: %v", err)
		}
		fmt.Printf("Account added to pool successfully!\n")
		fmt.Printf("  Account ID: %s\n", acc.AccountID)
		fmt.Printf("  Email:      %s\n", acc.Email)
		fmt.Printf("  Status:     %s\n", acc.Status)
		fmt.Println("\nRun without flags to start the proxy with account rotation.")
		return
	}

	if *showList {
		accounts := listAccounts()
		if len(accounts) == 0 {
			fmt.Println("No accounts in pool. Use --add-account to add one.")
			return
		}
		fmt.Printf("\n=== Account Pool (%d accounts) ===\n\n", len(accounts))
		for i, a := range accounts {
			fmt.Printf("  %d. [%s] %s (status: %s, used: %d)\n",
				i+1, a.AccountID, a.Email, a.Status, a.UsageCount)
		}
		fmt.Println()
		return
	}

	if err := startProxy(*host, *port); err != nil {
		log.Fatalf("Proxy failed: %v", err)
		os.Exit(1)
	}
}

func buildAndStart(host string, port int) {
	exe := "cline-proxy.exe"
	if runtime.GOOS != "windows" {
		exe = "./cline-proxy"
	}

	fmt.Println("Building proxy...")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Build failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Build complete.")

	running := false
	if runtime.GOOS == "windows" {
		out, _ := exec.Command("powershell", "-Command",
			"Get-Process cline-proxy -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Id").Output()
		if len(out) > 0 {
			running = true
		}
	}

	if running {
		fmt.Println("Proxy is already running.")
	} else {
		fmt.Println("Starting proxy...")
		startCmd := exec.Command(exe, "-host", host, "-port", fmt.Sprintf("%d", port))
		startCmd.Stdout = os.Stdout
		startCmd.Stderr = os.Stderr
		if err := startCmd.Start(); err != nil {
			fmt.Printf("Start failed: %v\n", err)
			os.Exit(1)
		}
		time.Sleep(2 * time.Second)
		fmt.Println("Proxy started.")
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/admin/", port)
	fmt.Printf("\nAdmin panel: %s\n", url)

	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}
