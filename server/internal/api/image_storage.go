package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	uploadDirName          = "uploads"
	remoteImageMaxBytes    = 30 << 20
	remoteImageTimeout     = 15 * time.Second
	remoteImageMaxRedirect = 5
)

type remoteImageNetwork struct {
	lookupIP    func(context.Context, string, string) ([]net.IP, error)
	dialContext func(context.Context, string, string) (net.Conn, error)
}

var nonPublicImageAddressPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:10::/28"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("fec0::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

var defaultRemoteImageHTTPClient = newRemoteImageHTTPClient(defaultRemoteImageNetwork())

var imageExtMap = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func (s *Server) saveUploadedImage(c *gin.Context, header *multipart.FileHeader, subdir string) (string, error) {
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	contentType := header.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("unsupported file type")
	}

	ext := imageExtension(contentType, header.Filename)
	if ext == "" {
		return "", fmt.Errorf("unsupported image format")
	}

	cleanSubdir := cleanUploadSubdir(subdir)
	name, err := randomHex(16)
	if err != nil {
		return "", err
	}
	filename := name + ext
	relativePath := path.Join(cleanSubdir, filename)

	if s.ossEnabled() {
		objectKey := s.buildOSSKey(cleanSubdir, filename)
		if err := s.uploadToOSS(objectKey, data, contentType); err != nil {
			return "", err
		}
		urlPath := path.Join("/", uploadDirName, relativePath)
		return buildPublicURL(c, urlPath), nil
	}

	baseDir := filepath.Join(s.cfg.Storage.Path, uploadDirName)
	targetDir := baseDir
	if cleanSubdir != "" {
		targetDir = filepath.Join(baseDir, filepath.FromSlash(cleanSubdir))
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return "", err
	}

	urlPath := path.Join("/", uploadDirName, relativePath)
	return buildPublicURL(c, urlPath), nil
}

func (s *Server) saveImageBytes(c *gin.Context, data []byte, contentType, subdir string) (string, error) {
	if contentType != "" {
		contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("unsupported image content type")
	}

	ext := imageExtension(contentType, "")
	if ext == "" {
		return "", fmt.Errorf("unsupported image format")
	}

	cleanSubdir := cleanUploadSubdir(subdir)
	name, err := randomHex(16)
	if err != nil {
		return "", err
	}
	filename := name + ext
	relativePath := path.Join(cleanSubdir, filename)

	if s.ossEnabled() {
		objectKey := s.buildOSSKey(cleanSubdir, filename)
		if err := s.uploadToOSS(objectKey, data, contentType); err != nil {
			return "", err
		}
		urlPath := path.Join("/", uploadDirName, relativePath)
		return buildPublicURL(c, urlPath), nil
	}

	baseDir := filepath.Join(s.cfg.Storage.Path, uploadDirName)
	targetDir := baseDir
	if cleanSubdir != "" {
		targetDir = filepath.Join(baseDir, filepath.FromSlash(cleanSubdir))
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return "", err
	}

	urlPath := path.Join("/", uploadDirName, relativePath)
	return buildPublicURL(c, urlPath), nil
}

// normalizeAndStoreImageValue converts inline/base64 image value into uploaded URL path.
func (s *Server) normalizeAndStoreImageValue(c *gin.Context, raw, subdir string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "/uploads/") || strings.HasPrefix(value, "uploads/") {
		return value, nil
	}

	if strings.HasPrefix(value, "data:") {
		data, contentType, err := decodeDataURI(value)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(contentType, "image/") {
			return "", fmt.Errorf("unsupported image format")
		}
		return s.saveImageBytes(c, data, contentType, subdir)
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		contentType := http.DetectContentType(decoded)
		if strings.HasPrefix(contentType, "image/") {
			return s.saveImageBytes(c, decoded, contentType, subdir)
		}
	}

	return value, nil
}

func buildPublicURL(c *gin.Context, urlPath string) string {
	if urlPath == "" {
		return urlPath
	}
	if strings.HasPrefix(urlPath, "http://") || strings.HasPrefix(urlPath, "https://") {
		return urlPath
	}
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}

	host := c.Request.Host
	if forwardedHost := c.GetHeader("X-Forwarded-Host"); forwardedHost != "" {
		host = strings.TrimSpace(strings.Split(forwardedHost, ",")[0])
	}
	if host == "" {
		return urlPath
	}

	return scheme + "://" + host + urlPath
}

func (s *Server) loadImageBytes(c *gin.Context, value string) ([]byte, string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, "", errors.New("empty image value")
	}

	if strings.HasPrefix(raw, "data:") {
		return decodeDataURI(raw)
	}

	if isImageURL(raw) {
		return s.readImageFromURL(c, raw)
	}

	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, "", err
	}
	return data, http.DetectContentType(data), nil
}

func decodeDataURI(dataURI string) ([]byte, string, error) {
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid data uri")
	}
	meta := parts[0]
	dataPart := parts[1]

	contentType := "application/octet-stream"
	if strings.HasPrefix(meta, "data:") {
		meta = strings.TrimPrefix(meta, "data:")
		metaParts := strings.Split(meta, ";")
		if len(metaParts) > 0 && metaParts[0] != "" {
			contentType = metaParts[0]
		}
	}

	data, err := base64.StdEncoding.DecodeString(dataPart)
	if err != nil {
		return nil, "", err
	}

	if contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func (s *Server) readImageFromURL(c *gin.Context, raw string) ([]byte, string, error) {
	if strings.HasPrefix(raw, "/uploads/") || strings.HasPrefix(raw, "uploads/") {
		if s.ossEnabled() {
			if key := uploadsKeyFromPath(raw); key != "" {
				ossKey := s.buildOSSKey(key, "")
				if data, contentType, err := s.readImageFromOSS(ossKey); err == nil {
					return data, contentType, nil
				}
			}
		}
		return s.readImageFromLocalPath(raw)
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return nil, "", err
		}
		if err := validateRemoteImageURLSyntax(parsed); err != nil {
			return nil, "", err
		}
		if parsed.Path != "" && strings.HasPrefix(parsed.Path, "/uploads/") && isSameHost(c, parsed.Host) {
			if s.ossEnabled() {
				if key := uploadsKeyFromPath(parsed.Path); key != "" {
					ossKey := s.buildOSSKey(key, "")
					if data, contentType, err := s.readImageFromOSS(ossKey); err == nil {
						return data, contentType, nil
					}
				}
			}
			return s.readImageFromLocalPath(parsed.Path)
		}

		return readRemoteImageWithClient(c, raw, defaultRemoteImageHTTPClient)
	}

	return s.readImageFromLocalPath(raw)
}

func defaultRemoteImageNetwork() remoteImageNetwork {
	dialer := &net.Dialer{Timeout: remoteImageTimeout, KeepAlive: 30 * time.Second}
	return remoteImageNetwork{
		lookupIP:    net.DefaultResolver.LookupIP,
		dialContext: dialer.DialContext,
	}
}

func readRemoteImage(c *gin.Context, raw string, network remoteImageNetwork) ([]byte, string, error) {
	if network.lookupIP == nil || network.dialContext == nil {
		return nil, "", errors.New("remote image network is not configured")
	}
	client := newRemoteImageHTTPClient(network)
	if transport, ok := client.Transport.(*http.Transport); ok {
		defer transport.CloseIdleConnections()
	}
	return readRemoteImageWithClient(c, raw, client)
}

func readRemoteImageWithClient(c *gin.Context, raw string, client *http.Client) ([]byte, string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, "", err
	}
	if err := validateRemoteImageURL(parsed); err != nil {
		return nil, "", err
	}
	requestContext := context.Background()
	if c != nil && c.Request != nil {
		requestContext = c.Request.Context()
	}
	req, err := http.NewRequestWithContext(requestContext, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("image fetch failed: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, remoteImageMaxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > remoteImageMaxBytes {
		return nil, "", fmt.Errorf("remote image exceeds %d byte limit", remoteImageMaxBytes)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return data, contentType, nil
}

func newRemoteImageHTTPClient(network remoteImageNetwork) *http.Client {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           validatedRemoteImageDialer(network),
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: remoteImageTimeout,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   remoteImageTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= remoteImageMaxRedirect {
				return fmt.Errorf("too many image redirects")
			}
			return validateRemoteImageURL(req.URL)
		},
	}
}

func validateRemoteImageURL(parsed *url.URL) error {
	if err := validateRemoteImageURLSyntax(parsed); err != nil {
		return err
	}
	if address, err := netip.ParseAddr(parsed.Hostname()); err == nil && !isPublicImageAddress(address) {
		return fmt.Errorf("remote image host is not publicly routable")
	}
	return nil
}

func validateRemoteImageURLSyntax(parsed *url.URL) error {
	if parsed == nil {
		return errors.New("empty remote image URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported remote image URL scheme")
	}
	if parsed.User != nil {
		return errors.New("remote image URL userinfo is not allowed")
	}
	host := parsed.Hostname()
	if host == "" || strings.Contains(host, "%") {
		return errors.New("invalid remote image host")
	}
	if port := parsed.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return errors.New("invalid remote image port")
		}
	}
	return nil
}

func validatedRemoteImageDialer(network remoteImageNetwork) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, networkName, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid remote image address: %w", err)
		}

		addresses, err := resolvePublicImageAddresses(ctx, network, host)
		if err != nil {
			return nil, err
		}

		var lastErr error
		for _, resolved := range addresses {
			conn, dialErr := network.dialContext(ctx, networkName, net.JoinHostPort(resolved.String(), port))
			if dialErr != nil {
				lastErr = dialErr
				continue
			}
			if err := validateRemoteConnectionAddress(conn.RemoteAddr()); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return conn, nil
		}
		if lastErr == nil {
			lastErr = errors.New("remote image host resolved to no addresses")
		}
		return nil, lastErr
	}
}

func resolvePublicImageAddresses(ctx context.Context, network remoteImageNetwork, host string) ([]netip.Addr, error) {
	if parsed, err := netip.ParseAddr(host); err == nil {
		parsed = parsed.Unmap()
		if !isPublicImageAddress(parsed) {
			return nil, fmt.Errorf("remote image address %s is not publicly routable", parsed)
		}
		return []netip.Addr{parsed}, nil
	}

	resolved, err := network.lookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve remote image host: %w", err)
	}
	if len(resolved) == 0 {
		return nil, errors.New("remote image host resolved to no addresses")
	}

	addresses := make([]netip.Addr, 0, len(resolved))
	for _, ip := range resolved {
		address, ok := netip.AddrFromSlice(ip)
		if !ok {
			return nil, errors.New("remote image host resolved to an invalid address")
		}
		address = address.Unmap()
		if !isPublicImageAddress(address) {
			return nil, fmt.Errorf("remote image address %s is not publicly routable", address)
		}
		addresses = append(addresses, address)
	}
	return addresses, nil
}

func validateRemoteConnectionAddress(remote net.Addr) error {
	if remote == nil {
		return errors.New("remote image connection has no peer address")
	}
	host, _, err := net.SplitHostPort(remote.String())
	if err != nil {
		return fmt.Errorf("invalid remote image peer address: %w", err)
	}
	address, err := netip.ParseAddr(host)
	if err != nil || !isPublicImageAddress(address.Unmap()) {
		return fmt.Errorf("remote image connection peer is not publicly routable")
	}
	return nil
}

func isPublicImageAddress(address netip.Addr) bool {
	if !address.IsValid() {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range nonPublicImageAddressPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func (s *Server) readImageFromLocalPath(raw string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, "", fmt.Errorf("empty local image path")
	}

	clean := path.Clean(trimmed)
	if strings.HasPrefix(clean, "uploads/") {
		clean = "/" + clean
	}
	if !strings.HasPrefix(clean, "/uploads/") {
		return nil, "", fmt.Errorf("unsupported local image path")
	}

	relative := strings.TrimPrefix(clean, "/uploads/")
	baseDir := filepath.Clean(filepath.Join(s.cfg.Storage.Path, uploadDirName))
	targetPath := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(relative)))

	if targetPath != baseDir && !strings.HasPrefix(targetPath, baseDir+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("invalid local image path")
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, "", err
	}
	return data, http.DetectContentType(data), nil
}

func imageExtension(contentType, filename string) string {
	if ext, ok := imageExtMap[contentType]; ok {
		return ext
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		return ext
	}

	return ""
}

func cleanUploadSubdir(subdir string) string {
	cleaned := path.Clean("/" + subdir)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func uploadsKeyFromPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	cleaned := path.Clean("/" + trimmed)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if !strings.HasPrefix(cleaned, "uploads/") {
		return ""
	}
	return strings.TrimPrefix(cleaned, "uploads/")
}

func isImageURL(value string) bool {
	return strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") ||
		strings.HasPrefix(value, "/uploads/") ||
		strings.HasPrefix(value, "uploads/")
}

func isSameHost(c *gin.Context, host string) bool {
	if c == nil {
		return false
	}
	requestHost := normalizedHost(c.GetHeader("X-Forwarded-Host"))
	if requestHost == "" {
		requestHost = normalizedHost(c.Request.Host)
	}
	compareHost := normalizedHost(host)
	return requestHost != "" && compareHost != "" && requestHost == compareHost
}

func normalizedHost(host string) string {
	if host == "" {
		return ""
	}
	host = strings.TrimSpace(strings.Split(host, ",")[0])
	return strings.ToLower(host)
}

func randomHex(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid random length")
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
