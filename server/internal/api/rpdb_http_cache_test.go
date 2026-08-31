package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

func TestRPDBHTTPCachePublicEndpointsReturnETagAnd304(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	current := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "可缓存的公共作品",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	related := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeTransmog, Title: "可缓存的相关推荐",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	if err := database.DB.Create(&[]*model.RPDBWork{&current, &related}).Error; err != nil {
		t.Fatalf("create public works: %v", err)
	}

	workID := strconv.FormatUint(uint64(current.ID), 10)
	tests := []struct {
		name string
		path string
	}{
		{name: "list", path: "/api/v1/rpdb/works?page=1&page_size=12"},
		{name: "hot", path: "/api/v1/rpdb/works/hot?limit=3"},
		{name: "detail", path: "/api/v1/rpdb/works/" + workID},
		{name: "preview", path: "/api/v1/rpdb/works/" + workID + "/preview"},
		{name: "recommendations", path: "/api/v1/rpdb/works/" + workID + "/recommendations?limit=6"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := performRPDBCacheRequest(server, test.path, "", "")
			if first.Code != http.StatusOK {
				t.Fatalf("first request: got %d body=%s", first.Code, first.Body.String())
			}
			if got := first.Header().Get("Cache-Control"); got != rpdbPublicCacheControl {
				t.Fatalf("Cache-Control=%q, want %q", got, rpdbPublicCacheControl)
			}
			if vary := first.Header().Get("Vary"); !strings.Contains(vary, "Authorization") {
				t.Fatalf("Vary=%q, want Authorization", vary)
			}
			etag := first.Header().Get("ETag")
			if etag == "" {
				t.Fatal("expected ETag on anonymous public response")
			}
			if want := fmt.Sprintf("\"%x\"", sha256.Sum256(first.Body.Bytes())); etag != want {
				t.Fatalf("ETag=%q, want hash of actual response bytes %q", etag, want)
			}

			conditional := performRPDBCacheRequest(server, test.path, etag, "")
			if conditional.Code != http.StatusNotModified {
				t.Fatalf("conditional request: got %d body=%s", conditional.Code, conditional.Body.String())
			}
			if conditional.Body.Len() != 0 {
				t.Fatalf("304 body must be empty, got %q", conditional.Body.String())
			}
			if got := conditional.Header().Get("ETag"); got != etag {
				t.Fatalf("304 ETag=%q, want %q", got, etag)
			}
			if got := conditional.Header().Get("Cache-Control"); got != rpdbPublicCacheControl {
				t.Fatalf("304 Cache-Control=%q, want %q", got, rpdbPublicCacheControl)
			}
		})
	}
}

func TestRPDBHTTPCacheETagChangesWithRepresentation(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	work := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "ETag 变更前",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	path := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10) + "/preview"
	first := performRPDBCacheRequest(server, path, "", "")
	if first.Code != http.StatusOK || first.Header().Get("ETag") == "" {
		t.Fatalf("first response: code=%d ETag=%q body=%s", first.Code, first.Header().Get("ETag"), first.Body.String())
	}

	if err := database.DB.Model(&work).Update("title", "ETag 变更后").Error; err != nil {
		t.Fatalf("update title: %v", err)
	}
	changed := performRPDBCacheRequest(server, path, first.Header().Get("ETag"), "")
	if changed.Code != http.StatusOK {
		t.Fatalf("changed representation must return 200, got %d body=%s", changed.Code, changed.Body.String())
	}
	if got := changed.Header().Get("ETag"); got == "" || got == first.Header().Get("ETag") {
		t.Fatalf("ETag did not change: before=%q after=%q", first.Header().Get("ETag"), got)
	}
	if !strings.Contains(changed.Body.String(), "ETag 变更后") {
		t.Fatalf("changed body missing new content: %s", changed.Body.String())
	}
}

func TestRPDBHTTPCacheAuthorizedRequestsArePrivateAndIgnoreAnonymousETag(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	viewer := model.User{Username: "cache-viewer", Email: "cache-viewer@example.com", PassHash: "hash"}
	if err := database.DB.Create(&viewer).Error; err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	work := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "带用户态字段的作品",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	if err := database.DB.Create(&model.RPDBLike{WorkID: work.ID, UserID: viewer.ID}).Error; err != nil {
		t.Fatalf("create like: %v", err)
	}

	path := "/api/v1/rpdb/works?page=1&page_size=12"
	anonymous := performRPDBCacheRequest(server, path, "", "")
	if anonymous.Code != http.StatusOK || anonymous.Header().Get("ETag") == "" {
		t.Fatalf("anonymous response: code=%d ETag=%q", anonymous.Code, anonymous.Header().Get("ETag"))
	}

	token := newTestToken(t, viewer)
	authorized := performRPDBCacheRequest(server, path, anonymous.Header().Get("ETag"), "Bearer "+token)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized request reused anonymous ETag: code=%d body=%s", authorized.Code, authorized.Body.String())
	}
	assertRPDBPrivateNoStore(t, authorized)
	var payload rpdbWorkListResponse
	if err := json.Unmarshal(authorized.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode authorized list: %v", err)
	}
	if len(payload.Works) != 1 || !payload.Works[0].IsLiked {
		t.Fatalf("authorized user state missing or shared: %#v", payload.Works)
	}

	invalidToken := performRPDBCacheRequest(server, path, anonymous.Header().Get("ETag"), "Bearer invalid-token")
	if invalidToken.Code != http.StatusOK {
		t.Fatalf("request carrying invalid Authorization reused anonymous ETag: code=%d", invalidToken.Code)
	}
	assertRPDBPrivateNoStore(t, invalidToken)
}

func TestRPDBHTTPCacheErrorsAreNoStore(t *testing.T) {
	server, _ := newRPDBQueryTestServer(t)

	badRequest := performRPDBCacheRequest(server, "/api/v1/rpdb/works/not-an-id/preview", "", "")
	if badRequest.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", badRequest.Code)
	}
	assertRPDBPrivateNoStore(t, badRequest)

	notFound := performRPDBCacheRequest(server, "/api/v1/rpdb/works/999999", "", "")
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", notFound.Code)
	}
	assertRPDBPrivateNoStore(t, notFound)

	if err := database.DB.Migrator().DropTable(&model.RPDBWork{}); err != nil {
		t.Fatalf("drop works table: %v", err)
	}
	serverError := performRPDBCacheRequest(server, "/api/v1/rpdb/works", "", "")
	if serverError.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", serverError.Code, serverError.Body.String())
	}
	assertRPDBPrivateNoStore(t, serverError)
}

func performRPDBCacheRequest(server *Server, path, ifNoneMatch, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)
	return recorder
}

func assertRPDBPrivateNoStore(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	cacheControl := response.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") || strings.Contains(cacheControl, "public") {
		t.Fatalf("Cache-Control=%q, want private no-store", cacheControl)
	}
	if etag := response.Header().Get("ETag"); etag != "" {
		t.Fatalf("private/error response must not expose public ETag, got %q", etag)
	}
}
