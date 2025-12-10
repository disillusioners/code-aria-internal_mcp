package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// getNetworkInfo gathers network information
func getNetworkInfo() (*NetworkInfo, error) {
	networkInfo := &NetworkInfo{}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		networkInfo.Hostname = hostname
	}

	// Get IP address
	ip, mac := getNetworkAddresses()
	networkInfo.IPAddress = ip
	networkInfo.MAC = mac

	// Get domain
	if domain := os.Getenv("USERDOMAIN"); domain != "" {
		networkInfo.Domain = domain
	}

	// Get gateway
	gateway := getGateway()
	networkInfo.Gateway = gateway

	// Get DNS servers
	dns := getDNSServers()
	networkInfo.DNS = dns

	// Get proxy settings
	proxy := getProxySettings()
	networkInfo.Proxy = proxy

	// Check internet connectivity
	networkInfo.Connected = isNetworkConnected()
	networkInfo.InternetAccess = hasInternetAccess()

	// Get public IP (optional, may take longer)
	if publicIP := getPublicIP(); publicIP != "" {
		networkInfo.PublicIP = publicIP
	}

	return networkInfo, nil
}

// getNetworkAddresses gets local IP and MAC addresses
func getNetworkAddresses() (string, string) {
	// Get IP address using Go's net package
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					return ipNet.IP.String(), ""
				}
			}
		}
	}

	// Fallback to system commands
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if runtime.GOOS == "windows" {
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-NetIPAddress -AddressFamily IPv4 | Where-Object {$_.IPAddress -notlike '127.*'} | Select-Object -First 1 IPAddress | ConvertTo-Json")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			var result map[string]interface{}
			if json.Unmarshal([]byte(stdout.String()), &result) == nil {
				if ip, ok := result["IPAddress"].(string); ok {
					return ip, ""
				}
			}
		}
	} else {
		cmd := exec.CommandContext(ctx, "hostname", "-I")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			if parts := strings.Fields(output); len(parts) > 0 {
				return parts[0], ""
			}
		}
	}

	return "", ""
}

// getGateway gets the default gateway
func getGateway() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if runtime.GOOS == "windows" {
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "(Get-NetRoute -DestinationPrefix '0.0.0.0/0').NextHop | Select-Object -First 1")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String())
		}
	} else {
		cmd := exec.CommandContext(ctx, "ip", "route", "show", "default")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			if strings.Contains(output, "default via") {
				parts := strings.Fields(output)
				for i, part := range parts {
					if part == "via" && i+1 < len(parts) {
						return parts[i+1]
					}
				}
			}
		}
	}

	return ""
}

// getDNSServers gets DNS servers
func getDNSServers() []string {
	var dnsServers []string

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if runtime.GOOS == "windows" {
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-DnsClientServerAddress -AddressFamily IPv4 | Select-Object -ExpandProperty ServerAddresses | ConvertTo-Json")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			var result []string
			if json.Unmarshal([]byte(stdout.String()), &result) == nil {
				for _, dns := range result {
					if dns != "" && !strings.Contains(dns, ":") { // IPv4 only
						dnsServers = append(dnsServers, dns)
					}
				}
			}
		}
	} else {
		// Try to read /etc/resolv.conf
		if data, err := os.ReadFile("/etc/resolv.conf"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "nameserver") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						dnsServers = append(dnsServers, parts[1])
					}
				}
			}
		}
	}

	return dnsServers
}

// getProxySettings gets proxy settings
func getProxySettings() ProxyInfo {
	proxy := ProxyInfo{}

	if http := os.Getenv("HTTP_PROXY"); http != "" {
		proxy.HTTP = http
	}
	if https := os.Getenv("HTTPS_PROXY"); https != "" {
		proxy.HTTPS = https
	}
	if ftp := os.Getenv("FTP_PROXY"); ftp != "" {
		proxy.FTP = ftp
	}
	if noProxy := os.Getenv("NO_PROXY"); noProxy != "" {
		proxy.NoProxy = strings.Split(noProxy, ",")
	}

	// Also check lowercase versions
	if http := os.Getenv("http_proxy"); http != "" && proxy.HTTP == "" {
		proxy.HTTP = http
	}
	if https := os.Getenv("https_proxy"); https != "" && proxy.HTTPS == "" {
		proxy.HTTPS = https
	}
	if ftp := os.Getenv("ftp_proxy"); ftp != "" && proxy.FTP == "" {
		proxy.FTP = ftp
	}
	if noProxy := os.Getenv("no_proxy"); noProxy != "" && len(proxy.NoProxy) == 0 {
		proxy.NoProxy = strings.Split(noProxy, ",")
	}

	return proxy
}

// isNetworkConnected checks if network is connected
func isNetworkConnected() bool {
	// Simple check by trying to connect to a local address
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	// Check if we have any non-loopback network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			return true
		}
	}

	return false
}

// hasInternetAccess checks if we have internet access
func hasInternetAccess() bool {
	// Try to connect to Google's DNS
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}

// getPublicIP gets the public IP address (optional)
func getPublicIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get public IP from a reliable service
	services := []string{
		"curl -s https://api.ipify.org",
		"curl -s https://ipinfo.io/ip",
		"curl -s https://icanhazip.com",
		"wget -qO- https://api.ipify.org",
	}

	for _, service := range services {
		if runtime.GOOS == "windows" {
			if strings.HasPrefix(service, "curl") {
				cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", strings.Replace(service, "curl -s", "Invoke-RestMethod -Uri", 1))
				var stdout strings.Builder
				cmd.Stdout = &stdout
				if err := cmd.Run(); err == nil {
					if ip := strings.TrimSpace(stdout.String()); ip != "" && net.ParseIP(ip) != nil {
						return ip
					}
				}
			}
		} else {
			parts := strings.Fields(service)
			if len(parts) > 0 {
				cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
				var stdout strings.Builder
				cmd.Stdout = &stdout
				if err := cmd.Run(); err == nil {
					if ip := strings.TrimSpace(stdout.String()); ip != "" && net.ParseIP(ip) != nil {
						return ip
					}
				}
			}
		}
	}

	return ""
}