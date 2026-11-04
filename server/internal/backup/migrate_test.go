package backup

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/rpbox/server/internal/config"
)

func TestVerifyObjectMetaRequiresMatchingSizeAndCRC(t *testing.T) {
	meta := http.Header{}
	meta.Set("Content-Length", "123")
	meta.Set(oss.HTTPHeaderOssCRC64, "456")
	if err := verifyObjectMeta(meta, 123, 456); err != nil {
		t.Fatalf("matching object rejected: %v", err)
	}
	if err := verifyObjectMeta(meta, 124, 456); err == nil {
		t.Fatal("mismatched size accepted")
	}
	if err := verifyObjectMeta(meta, 123, 457); err == nil {
		t.Fatal("mismatched CRC accepted")
	}
	meta.Del(oss.HTTPHeaderOssCRC64)
	if err := verifyObjectMeta(meta, 123, 456); err == nil {
		t.Fatal("missing OSS checksum accepted")
	}
}

func TestMigrationDryRunPreservesLocalDumps(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "prod", "rpbox")
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	name := "rpbox_20261030_071200Z.dump"
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("test dump"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.Database.DBName = "rpbox"
	cfg.Backup.LocalDir = root
	cfg.Backup.Environment = "prod"
	cfg.Backup.OSS.Enabled = true
	cfg.Backup.OSS.Endpoint = "https://example.invalid"
	cfg.Backup.OSS.Bucket = "test"
	cfg.Backup.OSS.AccessKeyID = "test"
	cfg.Backup.OSS.AccessKeySecret = "test"
	result, err := MigrateLocalBackups(cfg, true, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Found != 1 || result.Removed != 0 {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("dry run removed local dump: %v", err)
	}
}

func TestRunOnceDoesNotCreateLocalDumpWithoutOSS(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{}
	cfg.Backup.LocalDir = root
	cfg.Backup.Environment = "prod"
	cfg.Database.DBName = "rpbox"
	(&Service{cfg: cfg}).RunOnce()
	if _, err := os.Stat(filepath.Join(root, "prod", "rpbox")); !os.IsNotExist(err) {
		t.Fatalf("backup directory created without OSS: %v", err)
	}
}
