package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func newRPDBInteractionTestServer(t *testing.T) (*Server, model.User, model.RPDBWork, string) {
	t.Helper()

	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Notification{},
		&model.RPDBWork{},
		&model.RPDBLike{},
		&model.RPDBFavorite{},
		&model.RPDBComment{},
		&model.RPDBCommentLike{},
		&model.RPDBVerification{},
		&model.RPDBList{},
		&model.RPDBListEntry{},
		&model.RPDBReference{},
		&model.RPDBGuideStep{},
	)
	user := model.User{Username: "collector", Email: "collector@example.com", PassHash: "hash"}
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&user, &author}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	work := model.RPDBWork{
		AuthorID:     author.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "路线作品",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		Version:      1,
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	if err := db.Create(&model.RPDBGuideStep{
		WorkID:    work.ID,
		SortOrder: 1,
		Title:     "第一站",
		MapID:     "84",
		X:         42.1,
		Y:         65.3,
		Label:     "旧城区",
	}).Error; err != nil {
		t.Fatalf("create guide step: %v", err)
	}
	return newTestServer(t, db), user, work, newTestToken(t, user)
}

func TestRPDBInteractionLikeAndFavoriteAreIdempotent(t *testing.T) {
	server, _, work, token := newRPDBInteractionTestServer(t)
	base := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10)

	for i := 0; i < 2; i++ {
		if resp := performRequest(server.router, http.MethodPost, base+"/like", nil, token); resp.Code != http.StatusOK {
			t.Fatalf("like attempt %d returned %d body=%s", i+1, resp.Code, resp.Body.String())
		}
		if resp := performRequest(server.router, http.MethodPost, base+"/favorite", nil, token); resp.Code != http.StatusOK {
			t.Fatalf("favorite attempt %d returned %d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.LikeCount != 1 || stored.FavoriteCount != 1 {
		t.Fatalf("expected counters 1/1, got likes=%d favorites=%d", stored.LikeCount, stored.FavoriteCount)
	}
}

func TestRPDBInteractionCreatesCommentAndDefaultList(t *testing.T) {
	server, user, work, token := newRPDBInteractionTestServer(t)
	base := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10)

	commentResp := performRequest(server.router, http.MethodPost, base+"/comments", map[string]interface{}{
		"content": "这条路线很适合调查员角色。",
	}, token)
	if commentResp.Code != http.StatusCreated {
		t.Fatalf("expected comment 201, got %d body=%s", commentResp.Code, commentResp.Body.String())
	}

	listResp := performRequest(server.router, http.MethodPost, base+"/list", map[string]interface{}{
		"status": model.RPDBListStatusFarming,
	}, token)
	if listResp.Code != http.StatusCreated {
		t.Fatalf("expected list 201, got %d body=%s", listResp.Code, listResp.Body.String())
	}

	var list model.RPDBList
	if err := database.DB.Where("user_id = ? AND is_default = ?", user.ID, true).First(&list).Error; err != nil {
		t.Fatalf("load default list: %v", err)
	}
	if list.Name != "默认收集清单" {
		t.Fatalf("expected default collection checklist name, got %q", list.Name)
	}
	var entry model.RPDBListEntry
	if err := database.DB.Where("list_id = ? AND work_id = ?", list.ID, work.ID).First(&entry).Error; err != nil {
		t.Fatalf("load list entry: %v", err)
	}
	if entry.Status != model.RPDBListStatusFarming {
		t.Fatalf("expected farming status, got %q", entry.Status)
	}
}

func TestRPDBListEndpointCreatesDefaultCollectionChecklist(t *testing.T) {
	server, user, _, token := newRPDBInteractionTestServer(t)

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/lists", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected list endpoint 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Lists []rpdbListResponse `json:"lists"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode lists: %v", err)
	}
	if len(payload.Lists) != 1 {
		t.Fatalf("expected one default list, got %d", len(payload.Lists))
	}
	if payload.Lists[0].Name != "默认收集清单" || !payload.Lists[0].IsDefault {
		t.Fatalf("unexpected default list: %#v", payload.Lists[0].RPDBList)
	}

	var favoriteCount int64
	if err := database.DB.Model(&model.RPDBFavorite{}).Where("user_id = ?", user.ID).Count(&favoriteCount).Error; err != nil {
		t.Fatalf("count favorites: %v", err)
	}
	if favoriteCount != 0 {
		t.Fatalf("default collection checklist must not create favorites, got %d", favoriteCount)
	}
}

func TestRPDBInteractionAddsWorkToSelectedList(t *testing.T) {
	server, user, work, token := newRPDBInteractionTestServer(t)
	customList := model.RPDBList{
		UserID:      user.ID,
		Name:        "剧情道具清单",
		Description: "手动选择的清单",
	}
	if err := database.DB.Create(&customList).Error; err != nil {
		t.Fatalf("create custom list: %v", err)
	}

	resp := performRequest(server.router, http.MethodPost, "/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/list", map[string]interface{}{
		"status":  model.RPDBListStatusWanted,
		"list_id": customList.ID,
	}, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected selected list add 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var entry model.RPDBListEntry
	if err := database.DB.Where("list_id = ? AND work_id = ?", customList.ID, work.ID).First(&entry).Error; err != nil {
		t.Fatalf("load selected list entry: %v", err)
	}
	var defaultCount int64
	if err := database.DB.
		Table("rpdb_list_entries").
		Joins("JOIN rpdb_lists ON rpdb_lists.id = rpdb_list_entries.list_id").
		Where("rpdb_lists.user_id = ? AND rpdb_lists.is_default = ? AND rpdb_list_entries.work_id = ?", user.ID, true, work.ID).
		Count(&defaultCount).Error; err != nil {
		t.Fatalf("count default entries: %v", err)
	}
	if defaultCount != 0 {
		t.Fatalf("expected no default-list entry, got %d", defaultCount)
	}
}

func TestRPDBListTomTomExportUsesGuideCoordinates(t *testing.T) {
	server, _, work, token := newRPDBInteractionTestServer(t)
	if err := database.DB.Create(&model.RPDBGuideStep{
		WorkID:    work.ID,
		SortOrder: 2,
		Title:     "第二站",
		Zone:      "暮色森林",
		X:         48.6,
		Y:         72.4,
		Label:     "大教堂广场",
	}).Error; err != nil {
		t.Fatalf("create second guide step: %v", err)
	}
	base := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10)
	if resp := performRequest(server.router, http.MethodPost, base+"/list", map[string]interface{}{}, token); resp.Code != http.StatusCreated {
		t.Fatalf("add to list: %d body=%s", resp.Code, resp.Body.String())
	}

	var list model.RPDBList
	if err := database.DB.Where("is_default = ?", true).First(&list).Error; err != nil {
		t.Fatalf("load default list: %v", err)
	}
	resp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/lists/"+strconv.FormatUint(uint64(list.ID), 10)+"/export?format=tomtom",
		nil,
		token,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if !strings.Contains(payload.Content, "/way #84 42.10 65.30 [1/2] 路线作品 · 旧城区") ||
		!strings.Contains(payload.Content, "/way 暮色森林 48.60 72.40 [2/2] 路线作品 · 大教堂广场") {
		t.Fatalf("unexpected TomTom content %q", payload.Content)
	}
}

func TestRPDBInteractionListsFavoritesNewestFirst(t *testing.T) {
	server, user, firstWork, token := newRPDBInteractionTestServer(t)
	secondWork := model.RPDBWork{
		AuthorID:     firstWork.AuthorID,
		Type:         model.RPDBWorkTypeTransmog,
		Title:        "暮色巡林幻化",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		Version:      1,
	}
	if err := database.DB.Create(&secondWork).Error; err != nil {
		t.Fatalf("create second work: %v", err)
	}
	if err := database.DB.Create(&[]model.RPDBFavorite{
		{WorkID: firstWork.ID, UserID: user.ID, CreatedAt: time.Now().Add(-time.Hour)},
		{WorkID: secondWork.ID, UserID: user.ID, CreatedAt: time.Now()},
	}).Error; err != nil {
		t.Fatalf("create favorites: %v", err)
	}

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/my/favorites", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected favorites 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Works []rpdbWorkCard `json:"works"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode favorites: %v", err)
	}
	if len(payload.Works) != 2 {
		t.Fatalf("expected 2 favorites, got %d", len(payload.Works))
	}
	if payload.Works[0].ID != secondWork.ID || payload.Works[1].ID != firstWork.ID {
		t.Fatalf("unexpected favorite order: %#v", payload.Works)
	}
	if !payload.Works[0].IsFavorited || !payload.Works[1].IsFavorited {
		t.Fatalf("favorites missing viewer state: %#v", payload.Works)
	}
}
