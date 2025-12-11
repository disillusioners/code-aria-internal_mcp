package main

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Mock os.Hostname for testing
var mockHostname = func() (string, error) {
	return "test-hostname", nil
}

// Mock net.DialTimeout for testing
var mockDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
	// Mock implementation that always succeeds for 8.8.8.8:53
	if address == "8.8.8.8:53" {
		return &mockConn{}, nil
	}
	// Fail for other addresses
	return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
}

// Mock net.InterfaceAddrs for testing
var mockInterfaceAddrs = func() ([]net.Addr, error) {
	// Mock implementation that returns a non-loopback IPv4 address
	_, ipNet, _ := net.ParseCIDR("192.168.1.100/24")
	return []net.Addr{ipNet}, nil
}

// Mock net.Interfaces for testing
var mockInterfaces = func() ([]net.Interface, error) {
	// Mock implementation that returns a non-loopback interface
	return []net.Interface{
		{
			Name:  "eth0",
			Flags: net.FlagUp,
		},
	}, nil
}

// Mock connection implementation
type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, error error) { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, error error) { return 0, nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// TestGetNetworkInfo tests the getNetworkInfo function
func TestGetNetworkInfo(t *testing.T) {
	// Save original functions
	originalHostname := os.Hostname
	originalDialTimeout := net.DialTimeout
	originalInterfaceAddrs := net.InterfaceAddrs
	originalInterfaces := net.Interfaces
	
	// Restore after test
	defer func() {
		os.Hostname = originalHostname
		net.DialTimeout = originalDialTimeout
		net.InterfaceAddrs = originalInterfaceAddrs
		net.Interfaces = originalInterfaces
	}()

	// Set mock functions
	os.Hostname = mockHostname
	net.DialTimeout = mockDialTimeout
	net.InterfaceAddrs = mockInterfaceAddrs
	net.Interfaces = mockInterfaces

	got, err := getNetworkInfo()
	if err != nil {
		t.Errorf("getNetworkInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getNetworkInfo() returned nil")
		return
	}

	// Verify hostname
	if got.Hostname != "test-hostname" {
		t.Errorf("getNetworkInfo() hostname = %v, want test-hostname", got.Hostname)
	}

	// Verify IP address
	if got.IPAddress == "" {
		t.Error("getNetworkInfo() IP address is empty")
	}

	// Verify connected status
	if !got.Connected {
		t.Error("getNetworkInfo() connected = false, want true")
	}

	// Verify internet access
	if !got.InternetAccess {
		t.Error("getNetworkInfo() internetAccess = false, want true")
	}
}

// TestGetNetworkAddresses tests the getNetworkAddresses function
func TestGetNetworkAddresses(t *testing.T) {
	// Save original functions
	originalInterfaceAddrs := net.InterfaceAddrs
	
	// Restore after test
	defer func() {
		net.InterfaceAddrs = originalInterfaceAddrs
	}()

	tests := []struct {
		name            string
		mockInterfaceAddrs func() ([]net.Addr, error)
		wantIP           string
		wantMAC          string
	}{
		{
			name: "Valid IPv4 address",
			mockInterfaceAddrs: func() ([]net.Addr, error) {
				_, ipNet, _ := net.ParseCIDR("192.168.1.100/24")
				return []net.Addr{ipNet}, nil
			},
			wantIP:  "192.168.1.100",
			wantMAC: "",
		},
		{
			name: "Loopback address only",
			mockInterfaceAddrs: func() ([]net.Addr, error) {
				_, ipNet, _ := net.ParseCIDR("127.0.0.1/8")
				return []net.Addr{ipNet}, nil
			},
			wantIP:  "",
			wantMAC: "",
		},
		{
			name: "IPv6 address only",
			mockInterfaceAddrs: func() ([]net.Addr, error) {
				_, ipNet, _ := net.ParseCIDR("2001:db8::1/128")
				return []net.Addr{ipNet}, nil
			},
			wantIP:  "",
			wantMAC: "",
		},
		{
			name: "Error getting interfaces",
			mockInterfaceAddrs: func() ([]net.Addr, error) {
				return nil, &net.OpError{Op: "route", Net: "ip+net", Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			wantIP:  "",
			wantMAC: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net.InterfaceAddrs = tt.mockInterfaceAddrs
			
			gotIP, gotMAC := getNetworkAddresses()
			
			if gotIP != tt.wantIP {
				t.Errorf("getNetworkAddresses() IP = %v, want %v", gotIP, tt.wantIP)
			}
			
			if gotMAC != tt.wantMAC {
				t.Errorf("getNetworkAddresses() MAC = %v, want %v", gotMAC, tt.wantMAC)
			}
		})
	}
}

// TestGetGateway tests the getGateway function
func TestGetGateway(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	tests := []struct {
		name     string
		platform string
		want     string
	}{
		{
			name:     "Windows platform",
			platform: "windows",
			want:     "",
		},
		{
			name:     "Linux platform",
			platform: "linux",
			want:     "",
		},
		{
			name:     "MacOS platform",
			platform: "darwin",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test for platform-specific gateway detection
			// In a real implementation, we would mock the command execution
			// and verify the parsing logic
			
			// Test the function - it will use the actual runtime.GOOS
			got := getGateway()
			_ = got // We can't predict the output without mocking
			
			// Verify it doesn't panic
		})
	}
}

// TestGetDNSServers tests the getDNSServers function
func TestGetDNSServers(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	tests := []struct {
		name     string
		platform string
	}{
		{
			name:     "Windows platform",
			platform: "windows",
		},
		{
			name:     "Linux platform",
			platform: "linux",
		},
		{
			name:     "MacOS platform",
			platform: "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test for platform-specific DNS server detection
			// In a real implementation, we would mock the command execution
			// or file reading and verify the parsing logic
			
			// Test the function - it will use the actual runtime.GOOS
			got := getDNSServers()
			_ = got // We can't predict the output without mocking
			
			// Verify it doesn't panic and returns a slice
			if got == nil {
				t.Error("getDNSServers() returned nil")
			}
		})
	}
}

// TestGetProxySettings tests the getProxySettings function
func TestGetProxySettings(t *testing.T) {
	// Save original environment variables
	originalHTTPProxy := os.Getenv("HTTP_PROXY")
	originalHTTPSProxy := os.Getenv("HTTPS_PROXY")
	originalFTPProxy := os.Getenv("FTP_PROXY")
	originalNoProxy := os.Getenv("NO_PROXY")
	originalHttpProxy := os.Getenv("http_proxy")
	originalHttpsProxy := os.Getenv("https_proxy")
	originalFtpProxy := os.Getenv("ftp_proxy")
	originalNoProxyLower := os.Getenv("no_proxy")
	
	// Restore after test
	defer func() {
		os.Setenv("HTTP_PROXY", originalHTTPProxy)
		os.Setenv("HTTPS_PROXY", originalHTTPSProxy)
		os.Setenv("FTP_PROXY", originalFTPProxy)
		os.Setenv("NO_PROXY", originalNoProxy)
		os.Setenv("http_proxy", originalHttpProxy)
		os.Setenv("https_proxy", originalHttpsProxy)
		os.Setenv("ftp_proxy", originalFtpProxy)
		os.Setenv("no_proxy", originalNoProxyLower)
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		want     ProxyInfo
	}{
		{
			name: "All proxy variables set",
			envVars: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://proxy.example.com:8443",
				"FTP_PROXY":   "ftp://proxy.example.com:2121",
				"NO_PROXY":    "localhost,127.0.0.1",
			},
			want: ProxyInfo{
				HTTP:   "http://proxy.example.com:8080",
				HTTPS:  "https://proxy.example.com:8443",
				FTP:    "ftp://proxy.example.com:2121",
				NoProxy: []string{"localhost", "127.0.0.1"},
			},
		},
		{
			name: "Lowercase proxy variables only",
			envVars: map[string]string{
				"http_proxy":  "http://proxy.example.com:8080",
				"https_proxy": "https://proxy.example.com:8443",
				"ftp_proxy":   "ftp://proxy.example.com:2121",
				"no_proxy":    "localhost,127.0.0.1",
			},
			want: ProxyInfo{
				HTTP:   "http://proxy.example.com:8080",
				HTTPS:  "https://proxy.example.com:8443",
				FTP:    "ftp://proxy.example.com:2121",
				NoProxy: []string{"localhost", "127.0.0.1"},
			},
		},
		{
			name: "Mixed case proxy variables",
			envVars: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"https_proxy": "https://proxy.example.com:8443",
				"FTP_PROXY":   "ftp://proxy.example.com:2121",
				"no_proxy":    "localhost,127.0.0.1",
			},
			want: ProxyInfo{
				HTTP:   "http://proxy.example.com:8080",
				HTTPS:  "https://proxy.example.com:8443",
				FTP:    "ftp://proxy.example.com:2121",
				NoProxy: []string{"localhost", "127.0.0.1"},
			},
		},
		{
			name: "No proxy variables",
			envVars: map[string]string{},
			want: ProxyInfo{
				HTTP:   "",
				HTTPS:  "",
				FTP:    "",
				NoProxy: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all proxy environment variables
			os.Unsetenv("HTTP_PROXY")
			os.Unsetenv("HTTPS_PROXY")
			os.Unsetenv("FTP_PROXY")
			os.Unsetenv("NO_PROXY")
			os.Unsetenv("http_proxy")
			os.Unsetenv("https_proxy")
			os.Unsetenv("ftp_proxy")
			os.Unsetenv("no_proxy")
			
			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			
			got := getProxySettings()
			
			if got.HTTP != tt.want.HTTP {
				t.Errorf("getProxySettings() HTTP = %v, want %v", got.HTTP, tt.want.HTTP)
			}
			
			if got.HTTPS != tt.want.HTTPS {
				t.Errorf("getProxySettings() HTTPS = %v, want %v", got.HTTPS, tt.want.HTTPS)
			}
			
			if got.FTP != tt.want.FTP {
				t.Errorf("getProxySettings() FTP = %v, want %v", got.FTP, tt.want.FTP)
			}
			
			if tt.want.NoProxy == nil {
				if got.NoProxy != nil {
					t.Errorf("getProxySettings() NoProxy = %v, want nil", got.NoProxy)
				}
			} else {
				if len(got.NoProxy) != len(tt.want.NoProxy) {
					t.Errorf("getProxySettings() NoProxy length = %v, want %v", len(got.NoProxy), len(tt.want.NoProxy))
				} else {
					for i, noProxy := range got.NoProxy {
						if noProxy != tt.want.NoProxy[i] {
							t.Errorf("getProxySettings() NoProxy[%d] = %v, want %v", i, noProxy, tt.want.NoProxy[i])
						}
					}
				}
			}
		})
	}
}

// TestIsNetworkConnected tests the isNetworkConnected function
func TestIsNetworkConnected(t *testing.T) {
	// Save original functions
	originalDialTimeout := net.DialTimeout
	originalInterfaces := net.Interfaces
	
	// Restore after test
	defer func() {
		net.DialTimeout = originalDialTimeout
		net.Interfaces = originalInterfaces
	}()

	tests := []struct {
		name             string
		mockDialTimeout  func(network, address string, timeout time.Duration) (net.Conn, error)
		mockInterfaces   func() ([]net.Interface, error)
		want             bool
	}{
		{
			name: "Connected network",
			mockDialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
				if address == "8.8.8.8:53" {
					return &mockConn{}, nil
				}
				return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			mockInterfaces: func() ([]net.Interface, error) {
				return []net.Interface{
					{
						Name:  "eth0",
						Flags: net.FlagUp,
					},
				}, nil
			},
			want: true,
		},
		{
			name: "Disconnected network - dial fails",
			mockDialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			mockInterfaces: func() ([]net.Interface, error) {
				return []net.Interface{
					{
						Name:  "eth0",
						Flags: net.FlagUp,
					},
				}, nil
			},
			want: false,
		},
		{
			name: "Disconnected network - no interfaces",
			mockDialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
				if address == "8.8.8.8:53" {
					return &mockConn{}, nil
				}
				return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			mockInterfaces: func() ([]net.Interface, error) {
				return []net.Interface{
					{
						Name:  "lo",
						Flags: net.FlagLoopback,
					},
				}, nil
			},
			want: false,
		},
		{
			name: "Error getting interfaces",
			mockDialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
				if address == "8.8.8.8:53" {
					return &mockConn{}, nil
				}
				return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			mockInterfaces: func() ([]net.Interface, error) {
				return nil, &net.OpError{Op: "route", Net: "ip+net", Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net.DialTimeout = tt.mockDialTimeout
			net.Interfaces = tt.mockInterfaces
			
			got := isNetworkConnected()
			
			if got != tt.want {
				t.Errorf("isNetworkConnected() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHasInternetAccess tests the hasInternetAccess function
func TestHasInternetAccess(t *testing.T) {
	// Save original function
	originalDialTimeout := net.DialTimeout
	
	// Restore after test
	defer func() {
		net.DialTimeout = originalDialTimeout
	}()

	tests := []struct {
		name            string
		mockDialTimeout func(network, address string, timeout time.Duration) (net.Conn, error)
		want            bool
	}{
		{
			name: "Internet access available",
			mockDialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
				if address == "8.8.8.8:53" {
					return &mockConn{}, nil
				}
				return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			want: true,
		},
		{
			name: "Internet access unavailable",
			mockDialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net.DialTimeout = tt.mockDialTimeout
			
			got := hasInternetAccess()
			
			if got != tt.want {
				t.Errorf("hasInternetAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetPublicIP tests the getPublicIP function
func TestGetPublicIP(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	tests := []struct {
		name     string
		platform string
	}{
		{
			name:     "Windows platform",
			platform: "windows",
		},
		{
			name:     "Linux platform",
			platform: "linux",
		},
		{
			name:     "MacOS platform",
			platform: "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test for platform-specific public IP detection
			// In a real implementation, we would mock the command execution
			// and verify the parsing logic
			
			// Test the function - it will use the actual runtime.GOOS
			got := getPublicIP()
			_ = got // We can't predict the output without mocking
			
			// Verify it doesn't panic
		})
	}
}

// TestNetworkInfoStruct tests the NetworkInfo struct
func TestNetworkInfoStruct(t *testing.T) {
	proxyInfo := ProxyInfo{
		HTTP:   "http://proxy.example.com:8080",
		HTTPS:  "https://proxy.example.com:8443",
		FTP:    "ftp://proxy.example.com:2121",
		NoProxy: []string{"localhost", "127.0.0.1"},
	}

	networkInfo := &NetworkInfo{
		Hostname:      "test-hostname",
		Domain:        "example.com",
		IPAddress:     "192.168.1.100",
		PublicIP:      "203.0.113.1",
		MAC:           "00:11:22:33:44:55",
		Gateway:       "192.168.1.1",
		DNS:           []string{"8.8.8.8", "8.8.4.4"},
		Proxy:         proxyInfo,
		Connected:     true,
		InternetAccess: true,
	}

	// Verify all fields are set correctly
	if networkInfo.Hostname != "test-hostname" {
		t.Errorf("NetworkInfo.Hostname = %v, want test-hostname", networkInfo.Hostname)
	}

	if networkInfo.Domain != "example.com" {
		t.Errorf("NetworkInfo.Domain = %v, want example.com", networkInfo.Domain)
	}

	if networkInfo.IPAddress != "192.168.1.100" {
		t.Errorf("NetworkInfo.IPAddress = %v, want 192.168.1.100", networkInfo.IPAddress)
	}

	if networkInfo.PublicIP != "203.0.113.1" {
		t.Errorf("NetworkInfo.PublicIP = %v, want 203.0.113.1", networkInfo.PublicIP)
	}

	if networkInfo.MAC != "00:11:22:33:44:55" {
		t.Errorf("NetworkInfo.MAC = %v, want 00:11:22:33:44:55", networkInfo.MAC)
	}

	if networkInfo.Gateway != "192.168.1.1" {
		t.Errorf("NetworkInfo.Gateway = %v, want 192.168.1.1", networkInfo.Gateway)
	}

	if len(networkInfo.DNS) != 2 {
		t.Errorf("NetworkInfo.DNS length = %v, want 2", len(networkInfo.DNS))
	} else {
		if networkInfo.DNS[0] != "8.8.8.8" {
			t.Errorf("NetworkInfo.DNS[0] = %v, want 8.8.8.8", networkInfo.DNS[0])
		}
		if networkInfo.DNS[1] != "8.8.4.4" {
			t.Errorf("NetworkInfo.DNS[1] = %v, want 8.8.4.4", networkInfo.DNS[1])
		}
	}

	if networkInfo.Proxy.HTTP != "http://proxy.example.com:8080" {
		t.Errorf("NetworkInfo.Proxy.HTTP = %v, want http://proxy.example.com:8080", networkInfo.Proxy.HTTP)
	}

	if !networkInfo.Connected {
		t.Errorf("NetworkInfo.Connected = %v, want true", networkInfo.Connected)
	}

	if !networkInfo.InternetAccess {
		t.Errorf("NetworkInfo.InternetAccess = %v, want true", networkInfo.InternetAccess)
	}
}

// TestProxyInfoStruct tests the ProxyInfo struct
func TestProxyInfoStruct(t *testing.T) {
	proxyInfo := ProxyInfo{
		HTTP:   "http://proxy.example.com:8080",
		HTTPS:  "https://proxy.example.com:8443",
		FTP:    "ftp://proxy.example.com:2121",
		NoProxy: []string{"localhost", "127.0.0.1"},
	}

	// Verify all fields are set correctly
	if proxyInfo.HTTP != "http://proxy.example.com:8080" {
		t.Errorf("ProxyInfo.HTTP = %v, want http://proxy.example.com:8080", proxyInfo.HTTP)
	}

	if proxyInfo.HTTPS != "https://proxy.example.com:8443" {
		t.Errorf("ProxyInfo.HTTPS = %v, want https://proxy.example.com:8443", proxyInfo.HTTPS)
	}

	if proxyInfo.FTP != "ftp://proxy.example.com:2121" {
		t.Errorf("ProxyInfo.FTP = %v, want ftp://proxy.example.com:2121", proxyInfo.FTP)
	}

	if len(proxyInfo.NoProxy) != 2 {
		t.Errorf("ProxyInfo.NoProxy length = %v, want 2", len(proxyInfo.NoProxy))
	} else {
		if proxyInfo.NoProxy[0] != "localhost" {
			t.Errorf("ProxyInfo.NoProxy[0] = %v, want localhost", proxyInfo.NoProxy[0])
		}
		if proxyInfo.NoProxy[1] != "127.0.0.1" {
			t.Errorf("ProxyInfo.NoProxy[1] = %v, want 127.0.0.1", proxyInfo.NoProxy[1])
		}
	}
}

// TestPlatformSpecificNetworkInfo tests platform-specific network information gathering
func TestPlatformSpecificNetworkInfo(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	tests := []struct {
		name     string
		platform string
	}{
		{
			name:     "Windows platform",
			platform: "windows",
		},
		{
			name:     "Linux platform",
			platform: "linux",
		},
		{
			name:     "MacOS platform",
			platform: "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test for platform-specific network information gathering
			// In a real implementation, we would mock the command execution
			// and verify the parsing logic
			
			// Test the functions - they will use the actual runtime.GOOS
			gateway := getGateway()
			dns := getDNSServers()
			publicIP := getPublicIP()
			
			// Verify they don't panic
			_ = gateway
			_ = dns
			_ = publicIP
		})
	}
}

// TestTimeoutHandling tests timeout handling for network operations
func TestTimeoutHandling(t *testing.T) {
	// Save original function
	originalDialTimeout := net.DialTimeout
	
	// Restore after test
	defer func() {
		net.DialTimeout = originalDialTimeout
	}()

	// Mock a function that always times out
	mockTimeoutDial := func(network, address string, timeout time.Duration) (net.Conn, error) {
		time.Sleep(timeout + time.Millisecond) // Sleep longer than timeout
		return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53}, Err: &net.DNSError{Err: "timeout", IsTimeout: true}}
	}

	// Test isNetworkConnected with timeout
	net.DialTimeout = mockTimeoutDial
	got := isNetworkConnected()
	
	// Should return false when connection times out
	if got {
		t.Error("isNetworkConnected() returned true when connection times out")
	}

	// Test hasInternetAccess with timeout
	got = hasInternetAccess()
	
	// Should return false when connection times out
	if got {
		t.Error("hasInternetAccess() returned true when connection times out")
	}
}

// TestErrorHandling tests error handling in network functions
func TestErrorHandling(t *testing.T) {
	// Save original functions
	originalHostname := os.Hostname
	originalInterfaceAddrs := net.InterfaceAddrs
	originalInterfaces := net.Interfaces
	
	// Restore after test
	defer func() {
		os.Hostname = originalHostname
		net.InterfaceAddrs = originalInterfaceAddrs
		net.Interfaces = originalInterfaces
	}()

	// Mock functions that return errors
	mockErrorHostname := func() (string, error) {
		return "", &net.OpError{Op: "lookup", Net: "tcp", Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53}, Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
	}

	mockErrorInterfaceAddrs := func() ([]net.Addr, error) {
		return nil, &net.OpError{Op: "route", Net: "ip+net", Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
	}

	mockErrorInterfaces := func() ([]net.Interface, error) {
		return nil, &net.OpError{Op: "route", Net: "ip+net", Err: &net.DNSError{Err: "no such host", IsNotFound: true}}
	}

	// Test with error conditions
	os.Hostname = mockErrorHostname
	net.InterfaceAddrs = mockErrorInterfaceAddrs
	net.Interfaces = mockErrorInterfaces

	got, err := getNetworkInfo()
	if err != nil {
		t.Errorf("getNetworkInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getNetworkInfo() returned nil")
		return
	}

	// Verify that the function handles errors gracefully
	// (should still return a NetworkInfo object with default values)
}