package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func (s *Server) ossEnabled() bool {
	if s == nil || s.cfg == nil {
		return false
	}
	cfg := s.cfg.OSS
	if !cfg.Enabled {
		return false
	}
	return cfg.Endpoint != "" &&
		cfg.Bucket != "" &&
		cfg.AccessKeyID != "" &&
		cfg.AccessKeySecret != ""
}

func (s *Server) ossPrefix() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	prefix := strings.TrimSpace(s.cfg.OSS.Prefix)
	return strings.Trim(prefix, "/")
}

func (s *Server) buildOSSKey(subdir, filename string) string {
	cleanSubdir := strings.Trim(subdir, "/")
	relative := cleanSubdir
	if filename != "" {
		if relative == "" {
			relative = filename
		} else {
			relative = path.Join(relative, filename)
		}
	}
	if relative == "" {
		return ""
	}

	if prefix := s.ossPrefix(); prefix != "" {
		return path.Join(prefix, relative)
	}
	return relative
}

func (s *Server) getOSSBucket() (*oss.Bucket, error) {
	if !s.ossEnabled() {
		return nil, errors.New("oss is not configured")
	}

	s.ossInitOnce.Do(func() {
		client, err := oss.New(
			s.cfg.OSS.Endpoint,
			s.cfg.OSS.AccessKeyID,
			s.cfg.OSS.AccessKeySecret,
		)
		if err != nil {
			s.ossInitErr = err
			return
		}

		bucket, err := client.Bucket(s.cfg.OSS.Bucket)
		if err != nil {
			s.ossInitErr = err
			return
		}
		s.ossBucket = bucket
	})

	if s.ossInitErr != nil {
		return nil, s.ossInitErr
	}
	if s.ossBucket == nil {
		return nil, errors.New("oss bucket not initialized")
	}
	return s.ossBucket, nil
}

func (s *Server) uploadToOSS(key string, data []byte, contentType string) error {
	normalized := normalizeOSSKey(key)
	if normalized == "" {
		return errors.New("empty object key")
	}

	bucket, err := s.getOSSBucket()
	if err != nil {
		return err
	}

	options := make([]oss.Option, 0, 1)
	if contentType != "" {
		options = append(options, oss.ContentType(contentType))
	}
	return bucket.PutObject(normalized, bytes.NewReader(data), options...)
}

func (s *Server) readImageFromOSS(key string) ([]byte, string, error) {
	normalized := normalizeOSSKey(key)
	if normalized == "" {
		return nil, "", errors.New("empty object key")
	}

	bucket, err := s.getOSSBucket()
	if err != nil {
		return nil, "", err
	}

	meta, err := bucket.GetObjectMeta(normalized)
	contentType := ""
	if err == nil {
		contentType = strings.TrimSpace(strings.Split(meta.Get("Content-Type"), ";")[0])
	}

	body, err := bucket.GetObject(normalized)
	if err != nil {
		return nil, "", err
	}
	defer body.Close()

	limit := io.LimitReader(body, remoteImageMaxBytes)
	data, err := io.ReadAll(limit)
	if err != nil {
		return nil, "", err
	}

	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func normalizeOSSKey(key string) string {
	cleaned := path.Clean("/" + key)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}
