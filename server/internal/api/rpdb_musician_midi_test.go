package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
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
	"github.com/rpbox/server/internal/model"
)

func TestUploadRPDBMusicianMIDIStoresValidStandardMIDI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	storageRoot := t.TempDir()
	server := &Server{cfg: &config.Config{Storage: config.StorageConfig{Path: storageRoot}}}
	body, contentType := musicianMIDIMultipartBody(t, "song.mid", append([]byte{},
		'M', 'T', 'h', 'd', 0, 0, 0, 6, 0, 1, 0, 1, 0, 96,
		'M', 'T', 'r', 'k', 0, 0, 0, 4, 0, 0xFF, 0x2F, 0,
	))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rpdb/uploads/musician-midi", body)
	context.Request.Header.Set("Content-Type", contentType)

	server.uploadRPDBMusicianMIDI(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		URL  string `json:"url"`
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	parsedURL, err := url.Parse(response.URL)
	if err != nil {
		t.Fatalf("parse upload URL: %v", err)
	}
	if response.Name != "song.mid" || response.Size <= 0 || !strings.HasPrefix(parsedURL.Path, "/uploads/rpdb/musician-midi/") {
		t.Fatalf("unexpected response: %+v", response)
	}
	storedName := filepath.Base(parsedURL.Path)
	if _, err := os.Stat(filepath.Join(storageRoot, uploadDirName, "rpdb", "musician-midi", storedName)); err != nil {
		t.Fatalf("expected stored MIDI file: %v", err)
	}
}

func TestUploadRPDBMusicianMIDIRejectsSpoofedFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{cfg: &config.Config{Storage: config.StorageConfig{Path: t.TempDir()}}}
	body, contentType := musicianMIDIMultipartBody(t, "not-midi.mid", []byte("not a midi file"))
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/rpdb/uploads/musician-midi", body)
	context.Request.Header.Set("Content-Type", contentType)

	server.uploadRPDBMusicianMIDI(context)

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "Standard MIDI") {
		t.Fatalf("expected spoofed MIDI rejection, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestValidateRPDBWriteRequestSupportsMusicianMIDI(t *testing.T) {
	valid := rpdbWorkWriteRequest{
		Type:   model.RPDBWorkTypeMusicianMIDI,
		Title:  "Moonlight Sonata",
		Status: model.RPDBStatusPublished,
		Extra:  json.RawMessage(`{"midi_url":"/uploads/rpdb/musician-midi/abc.mid","midi_name":"Moonlight.mid","midi_size":1024}`),
	}
	if err := validateRPDBWriteRequest(valid, true); err != nil {
		t.Fatalf("expected Musician MIDI request to be valid: %v", err)
	}

	valid.Extra = json.RawMessage(`{}`)
	if err := validateRPDBWriteRequest(valid, true); err == nil || !strings.Contains(err.Error(), "请上传") {
		t.Fatalf("expected missing MIDI upload to be rejected, got %v", err)
	}

	musicianCode := base64.StdEncoding.EncodeToString([]byte{'M', 'U', 'S', '8', 0, 0, 0x10, 0, 0, 0, 0})
	valid.Extra = json.RawMessage(`{"musician_code":"` + musicianCode + `"}`)
	if err := validateRPDBWriteRequest(valid, true); err != nil {
		t.Fatalf("expected Musician code-only request to be valid: %v", err)
	}

	valid.Extra = json.RawMessage(`{"musician_code":"not-base64"}`)
	if err := validateRPDBWriteRequest(valid, true); err == nil || !strings.Contains(err.Error(), "音乐代码无效") {
		t.Fatalf("expected invalid Musician code to be rejected, got %v", err)
	}
}

func TestMusicianMIDIDraftAutosavesPartialCodeAndPublishesCompleteCode(t *testing.T) {
	server, _, token := newRPDBAuthoringTestServer(t)
	partialCode := "TVVTOA"
	createResponse := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/drafts",
		map[string]interface{}{
			"payload": map[string]interface{}{
				"type":       model.RPDBWorkTypeMusicianMIDI,
				"title":      "Editing Musician song",
				"status":     model.RPDBStatusDraft,
				"visibility": model.RPDBVisibilityPublic,
				"extra": map[string]interface{}{
					"musician_code": partialCode,
				},
			},
		},
		token,
	)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("partial Musician draft should autosave, got %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		Draft model.RPDBDraft `json:"draft"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode draft response: %v", err)
	}
	publishPath := "/api/v1/rpdb/drafts/" + strconv.FormatUint(uint64(created.Draft.ID), 10) + "/publish"
	incompletePublish := performRequest(server.router, http.MethodPost, publishPath, nil, token)
	if incompletePublish.Code != http.StatusBadRequest || !strings.Contains(incompletePublish.Body.String(), "音乐代码无效") {
		t.Fatalf("incomplete Musician code should fail only at publish, got %d body=%s", incompletePublish.Code, incompletePublish.Body.String())
	}

	title := []byte("Moonlight")
	songData := append([]byte{'M', 'U', 'S', '8', 0, byte(len(title))}, title...)
	songData = append(songData, 0x10, 0, 0, 0, 0)
	completeCode := base64.RawStdEncoding.EncodeToString(songData)
	updateResponse := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/drafts/"+strconv.FormatUint(uint64(created.Draft.ID), 10),
		map[string]interface{}{
			"payload": map[string]interface{}{
				"type":       model.RPDBWorkTypeMusicianMIDI,
				"title":      "Published Musician song",
				"status":     model.RPDBStatusDraft,
				"visibility": model.RPDBVisibilityPublic,
				"extra": map[string]interface{}{
					"musician_code": completeCode,
				},
			},
		},
		token,
	)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("complete Musician draft should save, got %d body=%s", updateResponse.Code, updateResponse.Body.String())
	}

	published := performRequest(server.router, http.MethodPost, publishPath, nil, token)
	if published.Code != http.StatusCreated {
		t.Fatalf("complete Musician draft should publish, got %d body=%s", published.Code, published.Body.String())
	}
	if !strings.Contains(published.Body.String(), "Published Musician song") {
		t.Fatalf("published Musician response lost content: %s", published.Body.String())
	}
}

func musicianMIDIMultipartBody(t *testing.T, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}
