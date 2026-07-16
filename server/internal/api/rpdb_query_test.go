package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	internalcache "github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func newRPDBQueryTestServer(t *testing.T) (*Server, model.User) {
	t.Helper()

	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Tag{},
		&model.RPDBWork{},
		&model.RPDBReference{},
		&model.RPDBMedia{},
		&model.RPDBGuideStep{},
		&model.RPDBTransmogSlot{},
		&model.RPDBTag{},
		&model.RPDBLike{},
		&model.RPDBFavorite{},
		&model.RPDBViewEvent{},
		&model.RPDBList{},
		&model.RPDBListEntry{},
		&model.GuildMember{},
		&model.UserHiddenContent{},
	)
	author := model.User{Username: "rp-author", Email: "rp-author@example.com", PassHash: "hash"}
	if err := db.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}

	return newTestServer(t, db), author
}

func TestRPDBPublicListUsesRedisCacheAndVersionInvalidation(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	server.cache = internalcache.NewRedisCache(redisClient, internalcache.Options{})
	t.Cleanup(func() { _ = server.cache.Close() })

	work := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "缓存前标题",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	first := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works", nil, "")
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), "缓存前标题") {
		t.Fatalf("first list failed: code=%d body=%s", first.Code, first.Body.String())
	}
	if err := database.DB.Model(&work).Update("title", "数据库新标题").Error; err != nil {
		t.Fatalf("update work: %v", err)
	}

	second := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works", nil, "")
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), "缓存前标题") {
		t.Fatalf("expected cached response, code=%d body=%s", second.Code, second.Body.String())
	}

	server.bumpRPDBListCache(context.Background())
	third := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works", nil, "")
	if third.Code != http.StatusOK || !strings.Contains(third.Body.String(), "数据库新标题") {
		t.Fatalf("expected invalidated response, code=%d body=%s", third.Code, third.Body.String())
	}
}

func TestRPDBQueryEnforcesPrivateAndGuildVisibility(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	member := model.User{Username: "guild-member", Email: "member@example.com", PassHash: "hash"}
	secondMember := model.User{Username: "second-guild-member", Email: "second-member@example.com", PassHash: "hash"}
	outsider := model.User{Username: "outsider", Email: "outsider@example.com", PassHash: "hash"}
	if err := database.DB.Create(&[]model.User{member, secondMember, outsider}).Error; err != nil {
		t.Fatalf("create viewers: %v", err)
	}
	if err := database.DB.Where("username = ?", member.Username).First(&member).Error; err != nil {
		t.Fatalf("reload member: %v", err)
	}
	if err := database.DB.Where("username = ?", outsider.Username).First(&outsider).Error; err != nil {
		t.Fatalf("reload outsider: %v", err)
	}
	if err := database.DB.Where("username = ?", secondMember.Username).First(&secondMember).Error; err != nil {
		t.Fatalf("reload second member: %v", err)
	}

	guildID := uint(42)
	privateWork := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "作者私稿",
		Status: model.RPDBStatusDraft, ReviewStatus: model.RPDBReviewNone,
		Visibility: model.RPDBVisibilityPrivate,
	}
	guildWork := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "公会档案",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityGuild, GuildID: &guildID, GuildIDs: []uint{guildID, 43},
	}
	if err := database.DB.Create(&[]*model.RPDBWork{&privateWork, &guildWork}).Error; err != nil {
		t.Fatalf("create works: %v", err)
	}
	if err := database.DB.Create(&model.GuildMember{GuildID: guildID, UserID: member.ID, Role: "member"}).Error; err != nil {
		t.Fatalf("create guild membership: %v", err)
	}
	if err := database.DB.Create(&model.GuildMember{GuildID: 43, UserID: secondMember.ID, Role: "member"}).Error; err != nil {
		t.Fatalf("create second guild membership: %v", err)
	}

	authorResp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works/"+strconv.FormatUint(uint64(privateWork.ID), 10), nil, newTestToken(t, author))
	if authorResp.Code != http.StatusOK {
		t.Fatalf("expected author to read private draft, got %d body=%s", authorResp.Code, authorResp.Body.String())
	}
	memberResp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works/"+strconv.FormatUint(uint64(guildWork.ID), 10), nil, newTestToken(t, member))
	if memberResp.Code != http.StatusOK {
		t.Fatalf("expected guild member access, got %d body=%s", memberResp.Code, memberResp.Body.String())
	}
	secondMemberResp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works/"+strconv.FormatUint(uint64(guildWork.ID), 10), nil, newTestToken(t, secondMember))
	if secondMemberResp.Code != http.StatusOK {
		t.Fatalf("expected second guild member access, got %d body=%s", secondMemberResp.Code, secondMemberResp.Body.String())
	}
	outsiderResp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works/"+strconv.FormatUint(uint64(guildWork.ID), 10), nil, newTestToken(t, outsider))
	if outsiderResp.Code != http.StatusNotFound {
		t.Fatalf("expected guild outsider 404, got %d body=%s", outsiderResp.Code, outsiderResp.Body.String())
	}
}

func TestRPDBQueryPublicListFiltersUnpublishedWorks(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)

	works := []model.RPDBWork{
		{
			AuthorID:           author.ID,
			Type:               model.RPDBWorkTypeItemShowcase,
			Title:              "月光灯笼",
			Summary:            "适合酒馆与夜间巡逻 RP",
			Status:             model.RPDBStatusPublished,
			ReviewStatus:       model.RPDBReviewApproved,
			IsPublic:           true,
			BindType:           "yes",
			VerificationStatus: model.RPDBVerificationVerified,
		},
		{
			AuthorID:     author.ID,
			Type:         model.RPDBWorkTypeTransmog,
			Title:        "未审核幻化",
			Status:       model.RPDBStatusPending,
			ReviewStatus: model.RPDBReviewPending,
			IsPublic:     true,
		},
	}
	if err := database.DB.Create(&works).Error; err != nil {
		t.Fatalf("create works: %v", err)
	}
	if err := database.DB.Create(&model.RPDBReference{
		WorkID:       works[0].ID,
		ExternalType: "toy",
		ExternalID:   "rpbox-1",
		Name:         "月光灯笼",
		IsPrimary:    true,
	}).Error; err != nil {
		t.Fatalf("create primary item reference: %v", err)
	}

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works?search=月光&type=item_showcase", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Works []struct {
			ID       uint   `json:"id"`
			Title    string `json:"title"`
			Type     string `json:"type"`
			ItemType string `json:"item_type"`
			BindType string `json:"bind_type"`
		} `json:"works"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || len(payload.Works) != 1 {
		t.Fatalf("expected one public work, got total=%d works=%d", payload.Total, len(payload.Works))
	}
	if payload.Works[0].Title != "月光灯笼" {
		t.Fatalf("unexpected title %q", payload.Works[0].Title)
	}
	if payload.Works[0].ItemType != "toy" || payload.Works[0].BindType != "yes" {
		t.Fatalf("expected toy and bound traits, got item_type=%q bind_type=%q", payload.Works[0].ItemType, payload.Works[0].BindType)
	}
}

func TestRPDBQueryFiltersItemsByBindingStatus(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	works := []model.RPDBWork{
		{AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "账号绑定物品", BindType: "account", Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved, IsPublic: true},
		{AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "拾取绑定物品", BindType: "pickup", Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved, IsPublic: true},
		{AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "不绑定物品", BindType: "no", Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved, IsPublic: true},
	}
	if err := database.DB.Create(&works).Error; err != nil {
		t.Fatalf("create binding works: %v", err)
	}

	var payload struct {
		Works []rpdbWorkCard `json:"works"`
		Total int64          `json:"total"`
	}
	boundResp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works?type=item_showcase&bind_type=yes", nil, "")
	if boundResp.Code != http.StatusOK {
		t.Fatalf("expected bound filter 200, got %d body=%s", boundResp.Code, boundResp.Body.String())
	}
	if err := json.Unmarshal(boundResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bound response: %v", err)
	}
	if payload.Total != 2 || len(payload.Works) != 2 {
		t.Fatalf("expected all bound variants, got total=%d works=%d", payload.Total, len(payload.Works))
	}

	payload.Works = nil
	unboundResp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works?type=item_showcase&bind_type=no", nil, "")
	if unboundResp.Code != http.StatusOK {
		t.Fatalf("expected unbound filter 200, got %d body=%s", unboundResp.Code, unboundResp.Body.String())
	}
	if err := json.Unmarshal(unboundResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode unbound response: %v", err)
	}
	if payload.Total != 1 || len(payload.Works) != 1 || payload.Works[0].Title != "不绑定物品" {
		t.Fatalf("unexpected unbound filter result: total=%d works=%#v", payload.Total, payload.Works)
	}
}

func TestRPDBQueryPublicListCapsPageSizeAtTwelve(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)

	works := make([]model.RPDBWork, 0, 13)
	for i := 0; i < 13; i++ {
		works = append(works, model.RPDBWork{
			AuthorID:     author.ID,
			Type:         model.RPDBWorkTypeItemShowcase,
			Title:        "分页作品 " + strconv.Itoa(i+1),
			Status:       model.RPDBStatusPublished,
			ReviewStatus: model.RPDBReviewApproved,
			IsPublic:     true,
		})
	}
	if err := database.DB.Create(&works).Error; err != nil {
		t.Fatalf("create works: %v", err)
	}

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works?page=1&page_size=50", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Works    []rpdbWorkCard `json:"works"`
		Total    int64          `json:"total"`
		PageSize int            `json:"page_size"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.PageSize != 12 {
		t.Fatalf("expected page_size 12, got %d", payload.PageSize)
	}
	if len(payload.Works) != 12 || payload.Total != 13 {
		t.Fatalf("expected 12 works of total 13, got works=%d total=%d", len(payload.Works), payload.Total)
	}
}

func TestRPDBQueryPublicListSearchesTagsFuzzily(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)

	work := model.RPDBWork{
		AuthorID:           author.ID,
		Type:               model.RPDBWorkTypeHomeShowcase,
		Title:              "港湾小屋",
		Summary:            "海边家宅",
		Status:             model.RPDBStatusPublished,
		ReviewStatus:       model.RPDBReviewApproved,
		IsPublic:           true,
		VerificationStatus: model.RPDBVerificationVerified,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	tag := model.Tag{Name: "库尔提拉斯风格", Color: "356A8A", Category: "rpdb", Type: "preset", IsPublic: true}
	if err := database.DB.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := database.DB.Create(&model.RPDBTag{WorkID: work.ID, TagID: tag.ID, AddedBy: author.ID}).Error; err != nil {
		t.Fatalf("create work tag: %v", err)
	}

	assertTaggedResult := func(path string) {
		t.Helper()
		resp := performRequest(server.router, http.MethodGet, path, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
		var payload struct {
			Works []struct {
				Title string `json:"title"`
			} `json:"works"`
			Total int64 `json:"total"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload.Total != 1 || len(payload.Works) != 1 || payload.Works[0].Title != "港湾小屋" {
			t.Fatalf("expected tagged work, got total=%d works=%v", payload.Total, payload.Works)
		}
	}

	assertTaggedResult("/api/v1/rpdb/works?tag_search=%E5%BA%93%E5%B0%94")
	assertTaggedResult("/api/v1/rpdb/works?search=%23%E5%BA%93%E5%B0%94")
}

func TestRPDBQueryDetailComposesUGCContent(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)

	work := model.RPDBWork{
		AuthorID:           author.ID,
		Type:               model.RPDBWorkTypeItemShowcase,
		Title:              "旧城区巡礼路线",
		Summary:            "适合侦探与城市冒险 RP",
		Content:            "<p>从运河开始。</p>",
		Status:             model.RPDBStatusPublished,
		ReviewStatus:       model.RPDBReviewApproved,
		IsPublic:           true,
		VerificationStatus: model.RPDBVerificationVerified,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	if err := database.DB.Create(&model.RPDBReference{
		WorkID:       work.ID,
		ExternalType: "item",
		ExternalID:   "12345",
		Name:         "旧城区钥匙",
		Source:       "wowhead",
		URL:          "https://www.wowhead.com/item=12345",
		IsPrimary:    true,
	}).Error; err != nil {
		t.Fatalf("create reference: %v", err)
	}
	if err := database.DB.Create(&model.RPDBMedia{
		WorkID:       work.ID,
		AuthorID:     &author.ID,
		Type:         "image",
		URL:          "/uploads/rpdb/route.jpg",
		ReviewStatus: model.RPDBReviewApproved,
	}).Error; err != nil {
		t.Fatalf("create media: %v", err)
	}
	if err := database.DB.Create(&model.RPDBGuideStep{
		WorkID:    work.ID,
		SortOrder: 1,
		Title:     "前往旧城区",
		Zone:      "暴风城",
		MapID:     "84",
		X:         42.1,
		Y:         65.3,
	}).Error; err != nil {
		t.Fatalf("create guide step: %v", err)
	}
	tag := model.Tag{Name: "城市冒险", Color: "B87333", Category: "rpdb", Type: "preset", IsPublic: true}
	if err := database.DB.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := database.DB.Create(&model.RPDBTag{WorkID: work.ID, TagID: tag.ID, AddedBy: author.ID}).Error; err != nil {
		t.Fatalf("create work tag: %v", err)
	}

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10), nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Work struct {
			ID         uint                  `json:"id"`
			AuthorName string                `json:"author_name"`
			References []model.RPDBReference `json:"references"`
			Media      []model.RPDBMedia     `json:"media"`
			GuideSteps []model.RPDBGuideStep `json:"guide_steps"`
			Tags       []model.Tag           `json:"tags"`
		} `json:"work"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Work.AuthorName != author.Username {
		t.Fatalf("expected author %q, got %q", author.Username, payload.Work.AuthorName)
	}
	if len(payload.Work.References) != 1 || len(payload.Work.Media) != 1 || len(payload.Work.GuideSteps) != 1 || len(payload.Work.Tags) != 1 {
		t.Fatalf(
			"unexpected composed detail refs=%d media=%d steps=%d tags=%d",
			len(payload.Work.References),
			len(payload.Work.Media),
			len(payload.Work.GuideSteps),
			len(payload.Work.Tags),
		)
	}
}

func TestRPDBWorkPreviewDoesNotIncreaseViewCount(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	work := model.RPDBWork{
		AuthorID:     author.ID,
		Type:         model.RPDBWorkTypeHomeShowcase,
		Title:        "海港会客厅",
		Summary:      "面向访客的家宅展示",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		ViewCount:    17,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	path := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10) + "/preview"
	resp := performRequest(server.router, http.MethodGet, path, nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var refreshed model.RPDBWork
	if err := database.DB.First(&refreshed, work.ID).Error; err != nil {
		t.Fatalf("reload work: %v", err)
	}
	if refreshed.ViewCount != 17 {
		t.Fatalf("preview must not increase views, got %d", refreshed.ViewCount)
	}
}

func TestRPDBRecommendationsWeightRelatedPlayerSignalsAndSameAuthor(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	firstPlayer := model.User{Username: "first-player", Email: "first-player@example.com", PassHash: "hash"}
	secondPlayer := model.User{Username: "second-player", Email: "second-player@example.com", PassHash: "hash"}
	otherAuthor := model.User{Username: "other-author", Email: "other-author@example.com", PassHash: "hash"}
	if err := database.DB.Create(&[]*model.User{&firstPlayer, &secondPlayer, &otherAuthor}).Error; err != nil {
		t.Fatalf("create recommendation users: %v", err)
	}

	newWork := func(workAuthor uint, workType, title string) model.RPDBWork {
		return model.RPDBWork{
			AuthorID:     workAuthor,
			Type:         workType,
			Title:        title,
			Status:       model.RPDBStatusPublished,
			ReviewStatus: model.RPDBReviewApproved,
			Visibility:   model.RPDBVisibilityPublic,
			IsPublic:     true,
		}
	}
	current := newWork(author.ID, model.RPDBWorkTypeItemShowcase, "月光灯笼")
	collaborative := newWork(otherAuthor.ID, model.RPDBWorkTypeTransmog, "暮色巡林幻化")
	sameAuthor := newWork(author.ID, model.RPDBWorkTypeHomeShowcase, "同作者旅店")
	playerCreated := newWork(firstPlayer.ID, model.RPDBWorkTypeItemShowcase, "相关玩家制作的道具")
	viewOnly := newWork(otherAuthor.ID, model.RPDBWorkTypeHomeShowcase, "浏览关联住宅")
	unrelated := newWork(otherAuthor.ID, model.RPDBWorkTypeItemShowcase, "无关作品")
	works := []*model.RPDBWork{&current, &collaborative, &sameAuthor, &playerCreated, &viewOnly, &unrelated}
	for _, work := range works {
		if err := database.DB.Create(work).Error; err != nil {
			t.Fatalf("create work %q: %v", work.Title, err)
		}
	}

	if err := database.DB.Create(&[]model.RPDBLike{
		{WorkID: current.ID, UserID: firstPlayer.ID},
		{WorkID: collaborative.ID, UserID: firstPlayer.ID},
	}).Error; err != nil {
		t.Fatalf("create recommendation likes: %v", err)
	}
	if err := database.DB.Create(&[]model.RPDBFavorite{
		{WorkID: current.ID, UserID: secondPlayer.ID},
		{WorkID: collaborative.ID, UserID: firstPlayer.ID},
		{WorkID: collaborative.ID, UserID: secondPlayer.ID},
	}).Error; err != nil {
		t.Fatalf("create recommendation favorites: %v", err)
	}
	if err := database.DB.Create(&[]model.RPDBViewEvent{
		{WorkID: current.ID, UserID: secondPlayer.ID, ViewDate: "2026-07-14"},
		{WorkID: viewOnly.ID, UserID: secondPlayer.ID, ViewDate: "2026-07-14"},
		{WorkID: viewOnly.ID, UserID: secondPlayer.ID, ViewDate: "2026-07-15"},
	}).Error; err != nil {
		t.Fatalf("create recommendation views: %v", err)
	}
	list := model.RPDBList{UserID: secondPlayer.ID, Name: "巡夜清单"}
	if err := database.DB.Create(&list).Error; err != nil {
		t.Fatalf("create recommendation list: %v", err)
	}
	if err := database.DB.Create(&[]model.RPDBListEntry{
		{ListID: list.ID, WorkID: current.ID},
		{ListID: list.ID, WorkID: collaborative.ID},
	}).Error; err != nil {
		t.Fatalf("create recommendation list entries: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(current.ID), 10)+"/recommendations?limit=6",
		nil,
		"",
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected recommendations 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload rpdbRecommendationResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode recommendations: %v", err)
	}
	if len(payload.Recommendations) != 4 {
		t.Fatalf("expected four recommendations, got %d: %#v", len(payload.Recommendations), payload.Recommendations)
	}
	gotIDs := []uint{
		payload.Recommendations[0].ID,
		payload.Recommendations[1].ID,
		payload.Recommendations[2].ID,
		payload.Recommendations[3].ID,
	}
	wantIDs := []uint{collaborative.ID, sameAuthor.ID, playerCreated.ID, viewOnly.ID}
	for index := range wantIDs {
		if gotIDs[index] != wantIDs[index] {
			t.Fatalf("unexpected recommendation order: got=%v want=%v", gotIDs, wantIDs)
		}
	}
	top := payload.Recommendations[0]
	if top.RecommendationScore != 42 ||
		top.RecommendationSignals.Favorites != 2 ||
		top.RecommendationSignals.Lists != 1 ||
		top.RecommendationSignals.Likes != 1 {
		t.Fatalf("unexpected collaborative score: %#v", top)
	}
	if !payload.Recommendations[1].RecommendationSignals.SameAuthor {
		t.Fatalf("expected same-author reason: %#v", payload.Recommendations[1])
	}
	if payload.Recommendations[3].RecommendationSignals.Views != 1 {
		t.Fatalf("daily repeat views must count once per player: %#v", payload.Recommendations[3])
	}
}

func TestRPDBRecommendationsUseAnonymousRedisCacheAndVersionInvalidation(t *testing.T) {
	server, author := newRPDBQueryTestServer(t)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	server.cache = internalcache.NewRedisCache(redisClient, internalcache.Options{})
	t.Cleanup(func() { _ = server.cache.Close() })

	current := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "当前作品",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	related := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeTransmog, Title: "缓存前推荐标题",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	if err := database.DB.Create(&[]*model.RPDBWork{&current, &related}).Error; err != nil {
		t.Fatalf("create recommendation cache works: %v", err)
	}
	path := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(current.ID), 10) + "/recommendations"
	first := performRequest(server.router, http.MethodGet, path, nil, "")
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), "缓存前推荐标题") {
		t.Fatalf("first recommendations failed: code=%d body=%s", first.Code, first.Body.String())
	}
	if err := database.DB.Model(&related).Update("title", "缓存失效后的标题").Error; err != nil {
		t.Fatalf("update related title: %v", err)
	}
	second := performRequest(server.router, http.MethodGet, path, nil, "")
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), "缓存前推荐标题") {
		t.Fatalf("expected cached recommendations: code=%d body=%s", second.Code, second.Body.String())
	}
	server.bumpRPDBListCache(context.Background())
	third := performRequest(server.router, http.MethodGet, path, nil, "")
	if third.Code != http.StatusOK || !strings.Contains(third.Body.String(), "缓存失效后的标题") {
		t.Fatalf("expected invalidated recommendations: code=%d body=%s", third.Code, third.Body.String())
	}
}
