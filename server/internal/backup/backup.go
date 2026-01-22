package backup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/rpbox/server/internal/config"
)

const (
	defaultIntervalMinutes = 60
	defaultRetentionDays   = 30
	defaultTimeoutMinutes  = 60
)

type Service struct {
	cfg       *config.Config
	ticker    *time.Ticker
	stopCh    chan struct{}
	running   int32
	ossBucket *oss.Bucket
	ossOnce   sync.Once
	ossErr    error
}

func Start(cfg *config.Config) *Service {
	if cfg == nil || !cfg.Backup.Enabled {
		return nil
	}

	intervalMinutes := cfg.Backup.IntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = defaultIntervalMinutes
	}
	interval := time.Duration(intervalMinutes) * time.Minute

	s := &Service{
		cfg:    cfg,
		ticker: time.NewTicker(interval),
		stopCh: make(chan struct{}),
	}

	log.Printf("[Backup] enabled interval=%s retention=%dd env=%s",
		interval, s.retentionDays(), s.environment(),
	)

	if cfg.Backup.RunOnStart {
		go s.RunOnce()
	}

	go s.loop()
	return s
}

func (s *Service) Stop() {
	if s == nil {
		return
	}
	close(s.stopCh)
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

func (s *Service) loop() {
	for {
		select {
		case <-s.ticker.C:
			s.RunOnce()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) RunOnce() {
	if s == nil || s.cfg == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		log.Printf("[Backup] skipped: previous backup still running")
		return
	}
	defer atomic.StoreInt32(&s.running, 0)

	startedAt := time.Now()
	filePath, objectKey, err := s.dumpDatabase()
	if err != nil {
		log.Printf("[Backup] dump failed: %v", err)
		return
	}
	log.Printf("[Backup] dump created: %s", filePath)

	if s.ossEnabled() {
		if err := s.uploadToOSS(filePath, objectKey); err != nil {
			log.Printf("[Backup] upload failed: %v", err)
		} else {
			log.Printf("[Backup] uploaded to oss: %s", objectKey)
		}
	}

	s.cleanupLocal()
	if s.ossEnabled() {
		s.cleanupOSS()
	}

	log.Printf("[Backup] finished in %s", time.Since(startedAt).Truncate(time.Second))
}

func (s *Service) dumpDatabase() (string, string, error) {
	if s == nil || s.cfg == nil {
		return "", "", errors.New("backup config not available")
	}

	localDir := s.backupDir()
	if localDir == "" {
		return "", "", errors.New("backup local_dir is empty")
	}
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", "", fmt.Errorf("create backup dir: %w", err)
	}

	filename := s.backupFileName()
	fullPath := filepath.Join(localDir, filename)
	objectKey := s.ossBackupKey(filename)

	pgDumpPath := strings.TrimSpace(s.cfg.Backup.PGDumpPath)
	if pgDumpPath == "" {
		pgDumpPath = "pg_dump"
	}

	timeout := s.dumpTimeout()
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	args := []string{
		"-h", s.cfg.Database.Host,
		"-p", s.cfg.Database.Port,
		"-U", s.cfg.Database.User,
		"-d", s.cfg.Database.DBName,
		"-F", "c",
		"-Z", "9",
		"--no-owner",
		"--no-acl",
		"-f", fullPath,
	}

	cmd := exec.CommandContext(ctx, pgDumpPath, args...)
	if s.cfg.Database.Password != "" {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+s.cfg.Database.Password)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(fullPath)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", "", fmt.Errorf("pg_dump timeout after %s", timeout)
		}
		return "", "", fmt.Errorf("pg_dump failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return "", "", fmt.Errorf("backup file missing: %w", err)
	}
	if info.Size() == 0 {
		return "", "", errors.New("backup file is empty")
	}

	return fullPath, objectKey, nil
}

func (s *Service) cleanupLocal() {
	retention := s.retentionDays()
	if retention <= 0 {
		return
	}

	root := s.backupDir()
	if root == "" {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retention)
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("[Backup] cleanup local: %v", err)
			return nil
		}
		if d.IsDir() || !isBackupFile(p) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			log.Printf("[Backup] cleanup local: %v", infoErr)
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if removeErr := os.Remove(p); removeErr != nil {
				log.Printf("[Backup] cleanup local: remove %s failed: %v", p, removeErr)
			}
		}
		return nil
	})
}

func (s *Service) cleanupOSS() {
	retention := s.retentionDays()
	if retention <= 0 {
		return
	}

	bucket, err := s.getOSSBucket()
	if err != nil {
		log.Printf("[Backup] cleanup oss: %v", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retention)
	prefix := s.ossBackupPrefix()
	marker := ""

	for {
		result, listErr := bucket.ListObjects(
			oss.Prefix(prefix),
			oss.Marker(marker),
			oss.MaxKeys(1000),
		)
		if listErr != nil {
			log.Printf("[Backup] cleanup oss: %v", listErr)
			return
		}

		var expired []string
		for _, obj := range result.Objects {
			if !isBackupObject(obj.Key) {
				continue
			}
			if obj.LastModified.Before(cutoff) {
				expired = append(expired, obj.Key)
			}
		}

		if len(expired) > 0 {
			_, delErr := bucket.DeleteObjects(expired, oss.DeleteObjectsQuiet(true))
			if delErr != nil {
				log.Printf("[Backup] cleanup oss: %v", delErr)
			}
		}

		if !result.IsTruncated {
			break
		}
		marker = result.NextMarker
	}
}

func (s *Service) ossEnabled() bool {
	if s == nil || s.cfg == nil {
		return false
	}
	cfg := s.cfg.Backup.OSS
	if !cfg.Enabled {
		return false
	}
	return s.ossEndpoint() != "" &&
		cfg.Bucket != "" &&
		cfg.AccessKeyID != "" &&
		cfg.AccessKeySecret != ""
}

func (s *Service) ossEndpoint() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	cfg := s.cfg.Backup.OSS
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if cfg.UseInternal && cfg.InternalEndpoint != "" {
		endpoint = strings.TrimSpace(cfg.InternalEndpoint)
	}
	if endpoint == "" {
		return ""
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	scheme := "http"
	if cfg.UseHTTPS {
		scheme = "https"
	}
	return scheme + "://" + endpoint
}

func (s *Service) getOSSBucket() (*oss.Bucket, error) {
	if !s.ossEnabled() {
		return nil, errors.New("oss backup not configured")
	}

	s.ossOnce.Do(func() {
		options := make([]oss.ClientOption, 0, 1)
		if s.cfg.Backup.OSS.UseCname {
			options = append(options, oss.UseCname(true))
		}
		client, err := oss.New(
			s.ossEndpoint(),
			s.cfg.Backup.OSS.AccessKeyID,
			s.cfg.Backup.OSS.AccessKeySecret,
			options...,
		)
		if err != nil {
			s.ossErr = err
			return
		}

		bucket, err := client.Bucket(s.cfg.Backup.OSS.Bucket)
		if err != nil {
			s.ossErr = err
			return
		}
		s.ossBucket = bucket
	})

	if s.ossErr != nil {
		return nil, s.ossErr
	}
	if s.ossBucket == nil {
		return nil, errors.New("oss bucket not initialized")
	}
	return s.ossBucket, nil
}

func (s *Service) uploadToOSS(filePath, objectKey string) error {
	if filePath == "" || objectKey == "" {
		return errors.New("empty backup upload input")
	}
	bucket, err := s.getOSSBucket()
	if err != nil {
		return err
	}

	return bucket.PutObjectFromFile(normalizeOSSKey(objectKey), filePath)
}

func (s *Service) backupDir() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	base := strings.TrimSpace(s.cfg.Backup.LocalDir)
	if base == "" {
		base = strings.TrimSpace(s.cfg.Storage.Path)
		if base == "" {
			base = "storage"
		}
		base = filepath.Join(base, "backups")
	}

	return filepath.Join(base, s.environment(), s.databaseName())
}

func (s *Service) backupFileName() string {
	dbName := s.databaseName()
	if dbName == "" {
		dbName = "database"
	}
	timestamp := time.Now().UTC().Format("20060102_150405Z")
	return fmt.Sprintf("%s_%s.dump", dbName, timestamp)
}

func (s *Service) ossBackupKey(filename string) string {
	parts := make([]string, 0, 4)
	if prefix := s.ossPrefix(); prefix != "" {
		parts = append(parts, prefix)
	}
	parts = append(parts, s.environment(), s.databaseName(), filename)
	return path.Join(parts...)
}

func (s *Service) ossBackupPrefix() string {
	parts := make([]string, 0, 3)
	if prefix := s.ossPrefix(); prefix != "" {
		parts = append(parts, prefix)
	}
	parts = append(parts, s.environment(), s.databaseName())
	if len(parts) == 0 {
		return ""
	}
	return path.Join(parts...) + "/"
}

func (s *Service) ossPrefix() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	prefix := strings.TrimSpace(s.cfg.Backup.OSS.Prefix)
	return strings.Trim(prefix, "/")
}

func (s *Service) environment() string {
	if s == nil || s.cfg == nil {
		return "default"
	}
	env := strings.TrimSpace(s.cfg.Backup.Environment)
	if env == "" {
		env = strings.TrimSpace(s.cfg.Server.Mode)
	}
	return sanitizeSegment(env, "default")
}

func (s *Service) databaseName() string {
	if s == nil || s.cfg == nil {
		return "database"
	}
	return sanitizeSegment(s.cfg.Database.DBName, "database")
}

func (s *Service) retentionDays() int {
	if s == nil || s.cfg == nil {
		return 0
	}
	if s.cfg.Backup.RetentionDays == 0 {
		return 0
	}
	if s.cfg.Backup.RetentionDays > 0 {
		return s.cfg.Backup.RetentionDays
	}
	return defaultRetentionDays
}

func (s *Service) dumpTimeout() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	timeoutMinutes := s.cfg.Backup.TimeoutMinutes
	if timeoutMinutes == 0 {
		return 0
	}
	if timeoutMinutes < 0 {
		timeoutMinutes = defaultTimeoutMinutes
	}
	return time.Duration(timeoutMinutes) * time.Minute
}

func sanitizeSegment(value, fallback string) string {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, "\\", "_")
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	if cleaned == "" {
		return fallback
	}
	return cleaned
}

func isBackupFile(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, ".dump") || strings.HasSuffix(name, ".dump.gz")
}

func isBackupObject(key string) bool {
	lower := strings.ToLower(key)
	return strings.HasSuffix(lower, ".dump") || strings.HasSuffix(lower, ".dump.gz")
}

func normalizeOSSKey(key string) string {
	cleaned := path.Clean("/" + key)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}
