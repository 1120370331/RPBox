package api

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
)

const characterCardPortraitPendingTTL = 24 * time.Hour

type characterCardPortraitStorageKind uint8

const (
	characterCardPortraitStorageInvalid characterCardPortraitStorageKind = iota
	characterCardPortraitStoragePending
	characterCardPortraitStorageCurrent
)

// uploadCharacterCardPortrait stores a validated portrait in private pending
// storage. The returned reference is intentionally not a directly accessible
// URL; it can only be consumed by updateCharacterCard for the same user.
func (s *Server) uploadCharacterCardPortrait(c *gin.Context) {
	userID := c.GetUint("userID")
	header, err := c.FormFile("image")
	if err != nil || header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择角色大图"})
		return
	}
	if header.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色大图不能为空"})
		return
	}
	if header.Size > characterCardPortraitMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色大图不能超过 20MB"})
		return
	}

	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取角色大图失败"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, characterCardPortraitMaxBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取角色大图失败"})
		return
	}
	if len(data) > characterCardPortraitMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色大图不能超过 20MB"})
		return
	}
	contentType, err := validateCharacterCardPortraitBytes(data, header.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cutoff := time.Now().UTC().Add(-characterCardPortraitPendingTTL)
	if _, err := s.cleanupExpiredCharacterCardPendingPortraits(cutoff); err != nil {
		log.Printf("[CharacterCard] cleanup expired pending portraits: %v", err)
	}

	saved, err := s.saveImageBytes(c, data, contentType, characterCardPortraitPendingSubdir(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存角色大图失败"})
		return
	}
	canonical, kind, ok := s.characterCardOwnedPortraitStoragePath(c, userID, saved)
	if !ok || kind != characterCardPortraitStoragePending {
		if key := extractUploadKey(c, saved); key != "" {
			s.deleteUploadKey(key)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存角色大图失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"portrait_image_ref": canonical})
}

func characterCardPortraitPendingSubdir(userID uint) string {
	return path.Join("character-cards", strconv.FormatUint(uint64(userID), 10), "pending")
}

func characterCardPortraitCurrentSubdir(userID uint) string {
	return path.Join("character-cards", strconv.FormatUint(uint64(userID), 10), "portrait")
}

func (s *Server) characterCardOwnedPortraitStoragePath(c *gin.Context, userID uint, raw string) (string, characterCardPortraitStorageKind, bool) {
	canonical, ok := s.characterCardInternalUploadPath(c, raw)
	if !ok {
		return "", characterCardPortraitStorageInvalid, false
	}
	policyPath := path.Clean(strings.ToLower(strings.ReplaceAll(canonical, `\`, "/")))
	pendingRoot := "/uploads/" + characterCardPortraitPendingSubdir(userID) + "/"
	currentRoot := "/uploads/" + characterCardPortraitCurrentSubdir(userID) + "/"
	switch {
	case strings.HasPrefix(policyPath, pendingRoot):
		return canonical, characterCardPortraitStoragePending, true
	case strings.HasPrefix(policyPath, currentRoot):
		return canonical, characterCardPortraitStorageCurrent, true
	default:
		return "", characterCardPortraitStorageInvalid, false
	}
}

func (s *Server) cleanupOwnedCharacterCardPendingPortrait(c *gin.Context, userID uint, raw string) {
	canonical, kind, ok := s.characterCardOwnedPortraitStoragePath(c, userID, raw)
	if !ok || kind != characterCardPortraitStoragePending {
		return
	}
	if key := uploadsKeyFromPath(canonical); key != "" {
		s.deleteUploadKey(key)
	}
}

// cleanupExpiredCharacterCardPendingPortraits removes abandoned pending files
// older than the TTL from both local storage and OSS. It runs opportunistically
// before new portrait uploads; a failed card save keeps its pending reference
// available for retry until it expires.
func (s *Server) cleanupExpiredCharacterCardPendingPortraits(cutoff time.Time) (int, error) {
	keys := make(map[string]struct{})
	localErr := s.collectExpiredLocalCharacterCardPendingPortraits(cutoff, keys)
	ossErr := s.collectExpiredOSSCharacterCardPendingPortraits(cutoff, keys)
	for key := range keys {
		s.deleteUploadKey(key)
	}
	return len(keys), errors.Join(localErr, ossErr)
}

func (s *Server) collectExpiredLocalCharacterCardPendingPortraits(cutoff time.Time, keys map[string]struct{}) error {
	if s == nil || s.cfg == nil {
		return nil
	}
	uploadRoot := filepath.Clean(filepath.Join(s.cfg.Storage.Path, uploadDirName))
	characterCardRoot := filepath.Join(uploadRoot, "character-cards")
	err := filepath.WalkDir(characterCardRoot, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.ModTime().Before(cutoff) {
			return nil
		}
		relative, err := filepath.Rel(uploadRoot, filePath)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(relative)
		if isCharacterCardPendingStorageKey(key) {
			keys[key] = struct{}{}
		}
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *Server) collectExpiredOSSCharacterCardPendingPortraits(cutoff time.Time, keys map[string]struct{}) error {
	if !s.ossEnabled() {
		return nil
	}
	bucket, err := s.getOSSBucket()
	if err != nil {
		return err
	}
	objectPrefix := strings.TrimSuffix(s.buildOSSKey("character-cards", ""), "/") + "/"
	marker := ""
	for {
		options := []oss.Option{oss.Prefix(objectPrefix), oss.MaxKeys(1000)}
		if marker != "" {
			options = append(options, oss.Marker(marker))
		}
		result, err := bucket.ListObjects(options...)
		if err != nil {
			return err
		}
		for _, object := range result.Objects {
			if !object.LastModified.Before(cutoff) {
				continue
			}
			key, ok := s.uploadKeyFromOSSObjectKey(object.Key)
			if ok && isCharacterCardPendingStorageKey(key) {
				keys[key] = struct{}{}
			}
		}
		if !result.IsTruncated || result.NextMarker == "" || result.NextMarker == marker {
			return nil
		}
		marker = result.NextMarker
	}
}

func (s *Server) uploadKeyFromOSSObjectKey(objectKey string) (string, bool) {
	key := strings.TrimPrefix(path.Clean("/"+objectKey), "/")
	if prefix := s.ossPrefix(); prefix != "" {
		prefix += "/"
		if !strings.HasPrefix(key, prefix) {
			return "", false
		}
		key = strings.TrimPrefix(key, prefix)
	}
	return key, key != "" && key != "."
}

func isCharacterCardPendingStorageKey(raw string) bool {
	cleaned := strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(raw, `\`, "/")), "/")
	parts := strings.Split(cleaned, "/")
	if len(parts) != 4 || parts[0] != "character-cards" || parts[2] != "pending" || parts[3] == "" {
		return false
	}
	userID, err := strconv.ParseUint(parts[1], 10, 32)
	return err == nil && userID > 0
}
