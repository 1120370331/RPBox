package backup

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/rpbox/server/internal/config"
)

const minimumBackupAge = 2 * time.Hour

// MigrationResult reports how many local database dumps were verified in OSS and removed.
type MigrationResult struct {
	Found    int
	Skipped  int
	Uploaded int
	Removed  int
	Bytes    int64
}

// MigrateLocalBackups uploads complete local dumps and removes each file only
// after OSS reports matching size and CRC64. It is safe to rerun after a failure.
func MigrateLocalBackups(cfg *config.Config, dryRun bool, limit int, report func(string)) (MigrationResult, error) {
	var result MigrationResult
	if cfg == nil || limit < 0 {
		return result, errors.New("invalid backup migration configuration")
	}
	s := &Service{cfg: cfg}
	if !s.ossEnabled() {
		return result, errors.New("OSS backup is not fully configured")
	}
	backupDir := s.backupDir()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return result, fmt.Errorf("read backup directory: %w", err)
	}
	pattern := regexp.MustCompile("^" + regexp.QuoteMeta(s.databaseName()) + `_\d{8}_\d{6}Z\.dump$`)
	var names []string
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink == 0 && !entry.IsDir() && pattern.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	result.Found = len(names)
	if report != nil {
		report(fmt.Sprintf("found %d local database dumps", result.Found))
	}
	if result.Found == 0 {
		return result, nil
	}
	if dryRun {
		return result, nil
	}

	if err := s.checkOSSReady(); err != nil {
		return result, fmt.Errorf("OSS preflight failed: %w", err)
	}
	bucket, _ := s.getOSSBucket()

	for _, name := range names {
		if limit > 0 && result.Removed >= limit {
			break
		}
		localPath := filepath.Join(backupDir, name)
		info, err := os.Lstat(localPath)
		if err != nil {
			return result, fmt.Errorf("stat %s: %w", name, err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 || time.Since(info.ModTime()) < minimumBackupAge {
			result.Skipped++
			continue
		}
		if err := validateDump(localPath); err != nil {
			return result, fmt.Errorf("invalid dump %s: %w", name, err)
		}
		checksum, err := localCRC64(localPath)
		if err != nil {
			return result, fmt.Errorf("checksum %s: %w", name, err)
		}
		key := normalizeOSSKey(s.ossBackupKey(name))
		meta, err := bucket.GetObjectDetailedMeta(key)
		if err != nil {
			var serviceErr oss.ServiceError
			if !errors.As(err, &serviceErr) || serviceErr.StatusCode != http.StatusNotFound {
				return result, fmt.Errorf("inspect OSS object %s: %w", name, err)
			}
			if report != nil {
				report("uploading " + name)
			}
			if err := bucket.PutObjectFromFile(key, localPath); err != nil {
				return result, fmt.Errorf("upload %s: %w", name, err)
			}
			result.Uploaded++
			meta, err = bucket.GetObjectDetailedMeta(key)
			if err != nil {
				return result, fmt.Errorf("verify OSS object %s: %w", name, err)
			}
		}
		if err := verifyObjectMeta(meta, info.Size(), checksum); err != nil {
			return result, fmt.Errorf("OSS object %s differs from local dump: %w", name, err)
		}
		current, err := os.Stat(localPath)
		if err != nil || !os.SameFile(info, current) || current.Size() != info.Size() || !current.ModTime().Equal(info.ModTime()) {
			return result, fmt.Errorf("local dump %s changed during verification", name)
		}
		if err := os.Remove(localPath); err != nil {
			return result, fmt.Errorf("remove verified dump %s: %w", name, err)
		}
		result.Removed++
		result.Bytes += info.Size()
		if report != nil {
			report(fmt.Sprintf("verified in OSS and removed %s (%d bytes)", name, info.Size()))
		}
	}
	return result, nil
}

func validateDump(path string) error {
	command := exec.Command("pg_restore", "--list", path)
	command.Stdout = io.Discard
	var output bytes.Buffer
	command.Stderr = &output
	err := command.Run()
	if err != nil {
		return fmt.Errorf("pg_restore --list: %w: %s", err, output.String())
	}
	return nil
}

func localCRC64(path string) (uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	checksum := crc64.New(crc64.MakeTable(crc64.ECMA))
	if _, err := io.Copy(checksum, file); err != nil {
		return 0, err
	}
	return checksum.Sum64(), nil
}

func verifyObjectMeta(meta http.Header, localSize int64, localCRC uint64) error {
	size, err := strconv.ParseInt(meta.Get("Content-Length"), 10, 64)
	if err != nil || size != localSize {
		return fmt.Errorf("size mismatch: OSS %q, local %d", meta.Get("Content-Length"), localSize)
	}
	checksum, err := strconv.ParseUint(meta.Get(oss.HTTPHeaderOssCRC64), 10, 64)
	if err != nil || checksum != localCRC {
		return fmt.Errorf("CRC64 mismatch: OSS %q, local %d", meta.Get(oss.HTTPHeaderOssCRC64), localCRC)
	}
	return nil
}

func (s *Service) checkOSSReady() error {
	bucket, err := s.getOSSBucket()
	if err != nil {
		return err
	}
	_, err = bucket.ListObjects(oss.Prefix(s.ossBackupPrefix()), oss.MaxKeys(1))
	return err
}

func (s *Service) uploadAndVerify(localPath, objectKey string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	checksum, err := localCRC64(localPath)
	if err != nil {
		return err
	}
	if err := s.uploadToOSS(localPath, objectKey); err != nil {
		return err
	}
	bucket, err := s.getOSSBucket()
	if err != nil {
		return err
	}
	meta, err := bucket.GetObjectDetailedMeta(normalizeOSSKey(objectKey))
	if err != nil {
		return err
	}
	if err := verifyObjectMeta(meta, info.Size(), checksum); err != nil {
		return err
	}
	current, err := os.Stat(localPath)
	if err != nil || !os.SameFile(info, current) || current.Size() != info.Size() || !current.ModTime().Equal(info.ModTime()) {
		return errors.New("local dump changed during upload")
	}
	return nil
}
