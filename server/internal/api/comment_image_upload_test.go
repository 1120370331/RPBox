package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

var minimalGIF = []byte{
	'G', 'I', 'F', '8', '9', 'a',
	0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xff, 0xff, 0xff,
	0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
	0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
}

type fakeOSSObject struct {
	data        []byte
	contentType string
}

type fakeOSSStore struct {
	bucket  string
	mu      sync.Mutex
	objects map[string]fakeOSSObject
}

func newFakeOSSServer(t *testing.T, bucket string) (*httptest.Server, *fakeOSSStore) {
	t.Helper()
	store := &fakeOSSStore{bucket: bucket, objects: make(map[string]fakeOSSObject)}
	server := httptest.NewServer(http.HandlerFunc(store.serveHTTP))
	t.Cleanup(server.Close)
	return server, store
}

func (s *fakeOSSStore) serveHTTP(w http.ResponseWriter, r *http.Request) {
	prefix := "/" + s.bucket + "/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, prefix)

	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read failed", http.StatusBadRequest)
			return
		}
		s.objects[key] = fakeOSSObject{data: append([]byte(nil), data...), contentType: r.Header.Get("Content-Type")}
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodHead:
		object, ok := s.objects[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", object.contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(object.data)))
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		object, ok := s.objects[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", object.contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(object.data)))
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(object.data)
	case http.MethodDelete:
		delete(s.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *fakeOSSStore) object(key string) (fakeOSSObject, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	object, ok := s.objects[key]
	object.data = append([]byte(nil), object.data...)
	return object, ok
}

func configureFakeCommentImageOSS(server *Server, endpoint, bucket string) {
	server.cfg.OSS = config.OSSConfig{
		Enabled:         true,
		Endpoint:        endpoint,
		Bucket:          bucket,
		AccessKeyID:     "fake-access-key",
		AccessKeySecret: "fake-access-secret",
		Prefix:          "images",
	}
}

func performCommentImageUpload(t *testing.T, router http.Handler, token, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/comment-image", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestCommentImageUploadRequiresOSSAndNeverWritesLocally(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{})
	database.DB = db
	user := model.User{Username: "comment-image-no-oss", Email: "comment-image-no-oss@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	server := newTestServer(t, db)
	server.cfg.Storage.Path = t.TempDir()
	resp := performCommentImageUpload(t, server.router, newTestToken(t, user), "pixel.gif", minimalGIF)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without OSS, got %d body=%s", resp.Code, resp.Body.String())
	}

	commentImageDir := filepath.Join(server.cfg.Storage.Path, uploadDirName, filepath.FromSlash(commentImageUploadSubdir))
	if _, err := os.Stat(commentImageDir); err == nil || !os.IsNotExist(err) {
		t.Fatalf("comment upload unexpectedly created local storage at %s: %v", commentImageDir, err)
	}
}

func TestCommentImageUploadStoresAndReadsOriginalGIFOnlyThroughOSS(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{})
	database.DB = db
	user := model.User{Username: "comment-image-oss", Email: "comment-image-oss@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	ossServer, store := newFakeOSSServer(t, "rpbox-comment-images")
	server := newTestServer(t, db)
	server.cfg.Storage.Path = t.TempDir()
	configureFakeCommentImageOSS(server, ossServer.URL, store.bucket)

	resp := performCommentImageUpload(t, server.router, newTestToken(t, user), "pixel.gif", minimalGIF)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected successful OSS upload, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Code int `json:"code"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if payload.Code != 0 || payload.Data.URL == "" {
		t.Fatalf("unexpected upload response: %s", resp.Body.String())
	}

	parsed, err := url.Parse(payload.Data.URL)
	if err != nil {
		t.Fatalf("parse upload URL: %v", err)
	}
	if parsed.IsAbs() || parsed.Host != "" {
		t.Fatalf("comment image upload must return a same-origin relative URL, got %q", payload.Data.URL)
	}
	key := uploadsKeyFromPath(parsed.Path)
	expectedPrefix := fmt.Sprintf("%s/%d/", commentImageUploadSubdir, user.ID)
	if !strings.HasPrefix(key, expectedPrefix) || path.Ext(key) != ".gif" || !immutableUploadFilePattern.MatchString(path.Base(key)) {
		t.Fatalf("unexpected comment image key %q", key)
	}

	objectKey := server.buildOSSKey(key, "")
	object, ok := store.object(objectKey)
	if !ok {
		t.Fatalf("expected OSS object %q", objectKey)
	}
	if !bytes.Equal(object.data, minimalGIF) {
		t.Fatalf("GIF bytes changed during upload")
	}
	if object.contentType != "image/gif" {
		t.Fatalf("expected image/gif content type, got %q", object.contentType)
	}

	getResp := performRequest(server.router, http.MethodGet, parsed.Path, nil, "")
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected OSS-backed image read, got %d body=%s", getResp.Code, getResp.Body.String())
	}
	if !bytes.Equal(getResp.Body.Bytes(), minimalGIF) {
		t.Fatalf("GIF bytes changed during OSS read")
	}
	if contentType := strings.TrimSpace(strings.Split(getResp.Header().Get("Content-Type"), ";")[0]); contentType != "image/gif" {
		t.Fatalf("expected image/gif response, got %q", contentType)
	}

	localPath := filepath.Join(server.cfg.Storage.Path, uploadDirName, filepath.FromSlash(key))
	if _, err := os.Stat(localPath); err == nil || !os.IsNotExist(err) {
		t.Fatalf("OSS comment image unexpectedly has a local copy at %s: %v", localPath, err)
	}
}

func TestCommentImageReadNeverFallsBackToLocalStorage(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{})
	database.DB = db
	ossServer, store := newFakeOSSServer(t, "rpbox-comment-read")
	server := newTestServer(t, db)
	server.cfg.Storage.Path = t.TempDir()
	configureFakeCommentImageOSS(server, ossServer.URL, store.bucket)

	key := commentImageUploadSubdir + "/1/00000000000000000000000000000001.gif"
	localPath := filepath.Join(server.cfg.Storage.Path, uploadDirName, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatalf("create local fallback directory: %v", err)
	}
	if err := os.WriteFile(localPath, minimalGIF, 0o644); err != nil {
		t.Fatalf("seed forbidden local fallback: %v", err)
	}

	requestPath := "/uploads/" + key
	resp := performRequest(server.router, http.MethodGet, requestPath, nil, "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected missing OSS object to ignore local fallback, got %d", resp.Code)
	}

	server.cfg.OSS.Enabled = false
	resp = performRequest(server.router, http.MethodGet, requestPath, nil, "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected OSS-disabled comment image read to ignore local storage, got %d", resp.Code)
	}
}

func TestAllCommentTypesRejectNonOSSAndCrossUserImageReferences(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.CommentLike{},
		&model.Item{},
		&model.ItemComment{},
		&model.RPDBWork{},
		&model.RPDBComment{},
	)
	database.DB = db
	user := model.User{Username: "comment-image-owner", Email: "comment-image-owner@example.com", PassHash: "hash"}
	other := model.User{Username: "comment-image-other", Email: "comment-image-other@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&user, &other}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	post := model.Post{AuthorID: other.ID, Title: "OSS 引用校验", Content: "body", Status: "published", ReviewStatus: "approved", IsPublic: true}
	item := model.Item{AuthorID: other.ID, Name: "OSS 引用校验", Type: "item", Status: "published", ReviewStatus: "approved", IsPublic: true}
	work := model.RPDBWork{
		AuthorID: other.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "OSS 引用校验",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true, Version: 1,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create RPDB work: %v", err)
	}

	type target struct {
		name string
		path string
		body func(string) map[string]interface{}
	}
	targets := []target{
		{
			name: "post",
			path: fmt.Sprintf("/api/v1/posts/%d/comments", post.ID),
			body: func(imageURL string) map[string]interface{} {
				return map[string]interface{}{"content": "配图", "image_url": imageURL}
			},
		},
		{
			name: "item",
			path: fmt.Sprintf("/api/v1/items/%d/comments", item.ID),
			body: func(imageURL string) map[string]interface{} {
				return map[string]interface{}{"content": "配图", "image_url": imageURL, "rating": 0}
			},
		},
		{
			name: "rpdb",
			path: fmt.Sprintf("/api/v1/rpdb/works/%d/comments", work.ID),
			body: func(imageURL string) map[string]interface{} {
				return map[string]interface{}{"content": "配图", "image_url": imageURL}
			},
		},
	}
	invalidURLs := []string{
		"https://example.com/a.gif",
		"https://example.com" + commentImageTestURL(user.ID, 1, "gif"),
		"/uploads/images/00000000000000000000000000000001.gif",
		commentImageTestURL(other.ID, 1, "gif"),
	}

	server := newTestServer(t, db)
	enableCommentImageTestOSS(server)
	token := newTestToken(t, user)
	for _, target := range targets {
		for _, imageURL := range invalidURLs {
			resp := performRequest(server.router, http.MethodPost, target.path, target.body(imageURL), token)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("%s accepted invalid image URL %q: status=%d body=%s", target.name, imageURL, resp.Code, resp.Body.String())
			}
		}
	}

	disabledServer := newTestServer(t, db)
	validOwnURL := commentImageTestURL(user.ID, 2, "gif")
	for _, target := range targets {
		resp := performRequest(disabledServer.router, http.MethodPost, target.path, target.body(validOwnURL), token)
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s accepted a comment image while OSS was disabled: status=%d body=%s", target.name, resp.Code, resp.Body.String())
		}
	}
}

func TestCommentImageCleanupWaitsUntilNoCommentTableReferencesObject(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.CommentLike{},
		&model.Item{},
		&model.ItemComment{},
		&model.RPDBComment{},
	)
	database.DB = db
	user := model.User{Username: "comment-image-cleanup", Email: "comment-image-cleanup@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	post := model.Post{AuthorID: user.ID, Title: "cleanup", Content: "body", CommentCount: 1}
	item := model.Item{AuthorID: user.ID, Name: "cleanup", Type: "item"}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	ossServer, store := newFakeOSSServer(t, "rpbox-comment-cleanup")
	server := newTestServer(t, db)
	server.cfg.Storage.Path = t.TempDir()
	configureFakeCommentImageOSS(server, ossServer.URL, store.bucket)
	imageURL := commentImageTestURL(user.ID, 10, "gif")
	key := uploadsKeyFromPath(imageURL)
	objectKey := server.buildOSSKey(key, "")
	if err := server.uploadToOSS(objectKey, minimalGIF, "image/gif"); err != nil {
		t.Fatalf("seed fake OSS object: %v", err)
	}

	postComment := model.Comment{
		PostID: post.ID, AuthorID: user.ID, Content: "post", ImageURL: imageURL, ImageReviewStatus: commentImageReviewApproved,
	}
	itemComment := model.ItemComment{
		ItemID: item.ID, UserID: user.ID, Content: "item", ImageURL: imageURL, ImageReviewStatus: commentImageReviewApproved,
	}
	if err := db.Create(&postComment).Error; err != nil {
		t.Fatalf("create post comment: %v", err)
	}
	if err := db.Create(&itemComment).Error; err != nil {
		t.Fatalf("create item comment: %v", err)
	}

	deleteResp := performRequest(
		server.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/posts/%d/comments/%d", post.ID, postComment.ID),
		nil,
		newTestToken(t, user),
	)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete first reference through API: status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	if _, ok := store.object(objectKey); !ok {
		t.Fatalf("shared OSS object was deleted while an item comment still referenced it")
	}

	if err := db.Delete(&itemComment).Error; err != nil {
		t.Fatalf("delete final reference: %v", err)
	}
	server.cleanupCommentImageURLs(nil, imageURL)
	if _, ok := store.object(objectKey); ok {
		t.Fatalf("unreferenced OSS object was not deleted")
	}
}
