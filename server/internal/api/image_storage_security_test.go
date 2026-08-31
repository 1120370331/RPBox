package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

const testPublicImageIP = "8.8.8.8"

type imageTestConn struct {
	net.Conn
	remote net.Addr
}

func (c *imageTestConn) RemoteAddr() net.Addr {
	return c.remote
}

func TestReadImageSSRFRejectsNonPublicLiteralAddresses(t *testing.T) {
	tests := []string{
		"http://127.0.0.1/image.png",
		"http://10.1.2.3/image.png",
		"http://172.16.0.1/image.png",
		"http://192.168.1.1/image.png",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/image.png",
		"http://[fc00::1]/image.png",
		"http://[fe80::1]/image.png",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			dialed := false
			_, _, err := readRemoteImage(nil, raw, remoteImageNetwork{
				lookupIP: func(context.Context, string, string) ([]net.IP, error) {
					return nil, fmt.Errorf("literal address should not be resolved")
				},
				dialContext: func(context.Context, string, string) (net.Conn, error) {
					dialed = true
					return nil, fmt.Errorf("must not dial")
				},
			})
			if err == nil || !strings.Contains(err.Error(), "not publicly routable") {
				t.Fatalf("expected non-public address rejection, got %v", err)
			}
			if dialed {
				t.Fatal("non-public literal address reached the dialer")
			}
		})
	}
}

func TestReadImageSSRFRejectsEveryNonPublicDNSAnswer(t *testing.T) {
	tests := []string{
		"127.0.0.1",
		"10.1.2.3",
		"172.31.255.254",
		"192.168.1.2",
		"169.254.169.254",
		"::1",
		"fd00::1",
		"fe80::1234",
	}

	for _, resolved := range tests {
		t.Run(resolved, func(t *testing.T) {
			dialed := false
			_, _, err := readRemoteImage(nil, "http://image.example/photo.png", remoteImageNetwork{
				lookupIP: func(context.Context, string, string) ([]net.IP, error) {
					return []net.IP{net.ParseIP(testPublicImageIP), net.ParseIP(resolved)}, nil
				},
				dialContext: func(context.Context, string, string) (net.Conn, error) {
					dialed = true
					return nil, fmt.Errorf("must not dial")
				},
			})
			if err == nil || !strings.Contains(err.Error(), "not publicly routable") {
				t.Fatalf("expected DNS answer rejection, got %v", err)
			}
			if dialed {
				t.Fatal("dialer was called before every DNS answer was validated")
			}
		})
	}
}

func TestReadImageSSRFPublicResolutionUsesValidatedLiteralDial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "" || r.URL.Path != "/photo.png" {
			t.Errorf("unexpected request: host=%q path=%q", r.Host, r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png; charset=binary")
		_, _ = w.Write([]byte("public-image"))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(serverURL.Host)
	if err != nil {
		t.Fatal(err)
	}

	dialedAddress := ""
	network := mappedImageTestNetwork(t, server.Listener.Addr().String(), map[string][]net.IP{
		"public-image.example": {net.ParseIP(testPublicImageIP)},
	}, &dialedAddress, true)

	data, contentType, err := readRemoteImage(nil, "http://public-image.example:"+port+"/photo.png", network)
	if err != nil {
		t.Fatalf("read public image: %v", err)
	}
	if string(data) != "public-image" || contentType != "image/png" {
		t.Fatalf("unexpected image result: data=%q contentType=%q", data, contentType)
	}
	if host, _, splitErr := net.SplitHostPort(dialedAddress); splitErr != nil || host != testPublicImageIP {
		t.Fatalf("dial was not bound to the validated DNS answer: address=%q err=%v", dialedAddress, splitErr)
	}
}

func TestReadImageSSRFRejectsPrivateRedirectTargets(t *testing.T) {
	tests := []string{
		"http://127.0.0.1/secret",
		"http://10.0.0.1/secret",
		"http://172.16.0.1/secret",
		"http://192.168.0.1/secret",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/secret",
		"http://[fd00::1]/secret",
		"http://[fe80::1]/secret",
	}

	for _, redirectTarget := range tests {
		t.Run(redirectTarget, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Location", redirectTarget)
				w.WriteHeader(http.StatusFound)
			}))
			defer server.Close()

			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			_, port, err := net.SplitHostPort(serverURL.Host)
			if err != nil {
				t.Fatal(err)
			}
			network := mappedImageTestNetwork(t, server.Listener.Addr().String(), map[string][]net.IP{
				"redirector.example": {net.ParseIP(testPublicImageIP)},
			}, nil, true)

			_, _, err = readRemoteImage(nil, "http://redirector.example:"+port+"/start", network)
			if err == nil || !strings.Contains(err.Error(), "not publicly routable") {
				t.Fatalf("expected redirect target rejection, got %v", err)
			}
		})
	}
}

func TestReadImageSSRFReresolvesRedirectHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "http://redirect-private.example/secret")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(serverURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	dialCount := 0
	network := remoteImageNetwork{
		lookupIP: func(_ context.Context, _ string, host string) ([]net.IP, error) {
			switch host {
			case "redirector.example":
				return []net.IP{net.ParseIP(testPublicImageIP)}, nil
			case "redirect-private.example":
				return []net.IP{net.ParseIP("127.0.0.1")}, nil
			default:
				return nil, fmt.Errorf("unexpected host %q", host)
			}
		},
		dialContext: func(ctx context.Context, networkName, _ string) (net.Conn, error) {
			dialCount++
			conn, dialErr := (&net.Dialer{}).DialContext(ctx, networkName, server.Listener.Addr().String())
			if dialErr != nil {
				return nil, dialErr
			}
			return &imageTestConn{Conn: conn, remote: &net.TCPAddr{IP: net.ParseIP(testPublicImageIP), Port: portNumber}}, nil
		},
	}

	_, _, err = readRemoteImage(nil, "http://redirector.example:"+port+"/start", network)
	if err == nil || !strings.Contains(err.Error(), "not publicly routable") {
		t.Fatalf("expected redirected DNS answer rejection, got %v", err)
	}
	if dialCount != 1 {
		t.Fatalf("redirect target should be rejected before a second dial, got %d dials", dialCount)
	}
}

func TestReadImageSSRFLimitsRedirectCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", r.URL.Path)
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(serverURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	network := mappedImageTestNetwork(t, server.Listener.Addr().String(), map[string][]net.IP{
		"redirect-loop.example": {net.ParseIP(testPublicImageIP)},
	}, nil, true)

	_, _, err = readRemoteImage(nil, "http://redirect-loop.example:"+port+"/again", network)
	if err == nil || !strings.Contains(err.Error(), "too many image redirects") {
		t.Fatalf("expected redirect limit rejection, got %v", err)
	}
}

func TestReadImageSSRFRejectsReboundDialPeer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("must not be read"))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(serverURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	network := mappedImageTestNetwork(t, server.Listener.Addr().String(), map[string][]net.IP{
		"rebind.example": {net.ParseIP(testPublicImageIP)},
	}, nil, false)

	_, _, err = readRemoteImage(nil, "http://rebind.example:"+port+"/image", network)
	if err == nil || !strings.Contains(err.Error(), "peer is not publicly routable") {
		t.Fatalf("expected actual dial peer rejection, got %v", err)
	}
}

func TestReadImageSSRFLimitsRemoteResponseSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		chunk := strings.Repeat("x", 64*1024)
		remaining := remoteImageMaxBytes + 1
		for remaining > 0 {
			part := chunk
			if len(part) > remaining {
				part = part[:remaining]
			}
			if _, err := w.Write([]byte(part)); err != nil {
				return
			}
			remaining -= len(part)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(serverURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	network := mappedImageTestNetwork(t, server.Listener.Addr().String(), map[string][]net.IP{
		"large-image.example": {net.ParseIP(testPublicImageIP)},
	}, nil, true)

	_, _, err = readRemoteImage(nil, "http://large-image.example:"+port+"/image", network)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversized image rejection, got %v", err)
	}
}

func TestReadImageStorageSameHostUploadUsesLocalFastPath(t *testing.T) {
	storageRoot := t.TempDir()
	uploadDir := filepath.Join(storageRoot, uploadDirName, "images")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("local-upload")
	if err := os.WriteFile(filepath.Join(uploadDir, "avatar.png"), want, 0o644); err != nil {
		t.Fatal(err)
	}

	server := &Server{cfg: &config.Config{Storage: config.StorageConfig{Path: storageRoot}}}
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodGet, "http://rpbox.example/api/v1/images/avatar/1", nil)
	context.Request.Host = "rpbox.example"

	data, _, err := server.readImageFromURL(context, "http://rpbox.example/uploads/images/avatar.png")
	if err != nil {
		t.Fatalf("read same-host upload: %v", err)
	}
	if string(data) != string(want) {
		t.Fatalf("unexpected local upload data: %q", data)
	}

	if _, _, err := server.readImageFromURL(context, "http://user@rpbox.example/uploads/images/avatar.png"); err == nil {
		t.Fatal("same-host upload URL with userinfo should be rejected")
	}
}

func TestReadImageSSRFRejectsUserinfoAndUnsupportedSchemes(t *testing.T) {
	tests := []string{
		"http://user:password@public-image.example/image.png",
		"file:///etc/passwd",
		"ftp://public-image.example/image.png",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			_, _, err := readRemoteImage(nil, raw, remoteImageNetwork{
				lookupIP: func(context.Context, string, string) ([]net.IP, error) {
					return []net.IP{net.ParseIP(testPublicImageIP)}, nil
				},
				dialContext: func(context.Context, string, string) (net.Conn, error) {
					return nil, fmt.Errorf("must not dial")
				},
			})
			if err == nil {
				t.Fatal("expected URL rejection")
			}
		})
	}
}

func mappedImageTestNetwork(t *testing.T, target string, answers map[string][]net.IP, dialedAddress *string, publicPeer bool) remoteImageNetwork {
	t.Helper()
	return remoteImageNetwork{
		lookupIP: func(_ context.Context, _ string, host string) ([]net.IP, error) {
			answer, ok := answers[host]
			if !ok {
				return nil, fmt.Errorf("unexpected lookup host %q", host)
			}
			return answer, nil
		},
		dialContext: func(ctx context.Context, networkName, address string) (net.Conn, error) {
			if dialedAddress != nil {
				*dialedAddress = address
			}
			conn, err := (&net.Dialer{}).DialContext(ctx, networkName, target)
			if err != nil {
				return nil, err
			}
			if !publicPeer {
				return conn, nil
			}
			_, port, splitErr := net.SplitHostPort(address)
			if splitErr != nil {
				_ = conn.Close()
				return nil, splitErr
			}
			portNumber, convertErr := net.LookupPort("tcp", port)
			if convertErr != nil {
				_ = conn.Close()
				return nil, convertErr
			}
			return &imageTestConn{Conn: conn, remote: &net.TCPAddr{IP: net.ParseIP(testPublicImageIP), Port: portNumber}}, nil
		},
	}
}
