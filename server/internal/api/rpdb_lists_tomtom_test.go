package api

import (
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestLoadRPDBTomTomGuideStepsUsesOneQuery(t *testing.T) {
	db := testutil.NewTestDB(t, &model.RPDBGuideStep{})
	database.DB = db
	if err := db.Create(&[]model.RPDBGuideStep{
		{WorkID: 11, SortOrder: 2, Title: "十一号第二步"},
		{WorkID: 11, SortOrder: 1, Title: "十一号第一步"},
		{WorkID: 12, SortOrder: 1, Title: "十二号第一步"},
		{WorkID: 13, SortOrder: 1, Title: "十三号第一步"},
		{WorkID: 11, SortOrder: 1, Title: "十一号同序后一步"},
	}).Error; err != nil {
		t.Fatalf("create guide steps: %v", err)
	}

	tests := []struct {
		name    string
		workIDs []uint
		want    int
	}{
		{name: "one work", workIDs: []uint{11}, want: 3},
		{name: "many works", workIDs: []uint{11, 12, 13}, want: 5},
		{name: "duplicate and zero work IDs", workIDs: []uint{0, 11, 11, 12, 0, 11}, want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryCount := countRPDBGuideStepQueries(t, db)
			entries := make([]rpdbListEntryResponse, 0, len(tt.workIDs))
			for _, workID := range tt.workIDs {
				entries = append(entries, rpdbListEntryResponse{
					RPDBListEntry: model.RPDBListEntry{WorkID: workID},
				})
			}

			steps, err := loadRPDBTomTomGuideSteps(entries)
			if err != nil {
				t.Fatalf("load guide steps: %v", err)
			}
			if *queryCount != 1 {
				t.Fatalf("expected exactly one guide-step SELECT, got %d", *queryCount)
			}
			if len(steps) != tt.want {
				t.Fatalf("expected %d unique guide steps, got %d", tt.want, len(steps))
			}
			if len(steps) >= 3 && steps[0].WorkID == 11 {
				if steps[0].Title != "十一号第一步" || steps[1].Title != "十一号同序后一步" || steps[2].Title != "十一号第二步" {
					t.Fatalf("guide steps are not ordered by work_id, sort_order, id: %#v", steps[:3])
				}
			}
		})
	}
}

func TestRPDBTomTomExportPreservesOrderingAndMissingContract(t *testing.T) {
	db := testutil.NewTestDB(t, &model.RPDBGuideStep{})
	database.DB = db
	if err := db.Create(&[]model.RPDBGuideStep{
		{WorkID: 101, SortOrder: 2, Zone: "暮色森林", X: 48.6, Y: 72.4, Label: "大教堂广场"},
		{WorkID: 101, SortOrder: 1, MapID: "84", X: 42.1, Y: 65.3, Label: "旧城区"},
		{WorkID: 202, SortOrder: 1, Zone: "无效地点", X: 0, Y: 0, Label: "无坐标"},
	}).Error; err != nil {
		t.Fatalf("create guide steps: %v", err)
	}
	entries := []rpdbListEntryResponse{
		{
			RPDBListEntry: model.RPDBListEntry{WorkID: 202},
			Work:          rpdbWorkCard{RPDBWork: model.RPDBWork{Title: "缺失作品"}},
		},
		{
			RPDBListEntry: model.RPDBListEntry{WorkID: 101},
			Work:          rpdbWorkCard{RPDBWork: model.RPDBWork{Title: "路线作品"}},
		},
		{
			RPDBListEntry: model.RPDBListEntry{WorkID: 101},
			Work:          rpdbWorkCard{RPDBWork: model.RPDBWork{Title: "路线作品"}},
		},
	}

	queryCount := countRPDBGuideStepQueries(t, db)
	steps, err := loadRPDBTomTomGuideSteps(entries)
	if err != nil {
		t.Fatalf("load guide steps: %v", err)
	}
	content, missing := buildRPDBTomTomExport(entries, steps)
	const wantContent = "/way #84 42.10 65.30 [1/2] 路线作品 · 旧城区\n" +
		"/way 暮色森林 48.60 72.40 [2/2] 路线作品 · 大教堂广场\n" +
		"/way #84 42.10 65.30 [1/2] 路线作品 · 旧城区\n" +
		"/way 暮色森林 48.60 72.40 [2/2] 路线作品 · 大教堂广场"
	if content != wantContent {
		t.Fatalf("unexpected TomTom content:\n%s", content)
	}
	if *queryCount != 1 {
		t.Fatalf("expected exactly one guide-step SELECT, got %d", *queryCount)
	}
	if len(missing) != 1 || missing[0]["work_id"] != uint(202) || missing[0]["title"] != "缺失作品" {
		t.Fatalf("unexpected missing_coordinates: %#v", missing)
	}
}

func TestRPDBListTomTomExportReturns500OnGuideStepQueryError(t *testing.T) {
	server, _, work, token := newRPDBInteractionTestServer(t)
	if resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/list",
		map[string]interface{}{},
		token,
	); resp.Code != http.StatusCreated {
		t.Fatalf("add work to list: status=%d body=%s", resp.Code, resp.Body.String())
	}
	var list model.RPDBList
	if err := database.DB.Where("is_default = ?", true).First(&list).Error; err != nil {
		t.Fatalf("load default list: %v", err)
	}

	callbackName := "test:fail_rpdb_guide_step_query:" + t.Name()
	if err := database.DB.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if rpdbGuideStepQuery(tx) {
			tx.AddError(errors.New("forced guide-step query failure"))
		}
	}); err != nil {
		t.Fatalf("register failing query callback: %v", err)
	}
	t.Cleanup(func() {
		_ = database.DB.Callback().Query().Remove(callbackName)
	})

	resp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/lists/"+strconv.FormatUint(uint64(list.ID), 10)+"/export?format=tomtom",
		nil,
		token,
	)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected export 500, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func countRPDBGuideStepQueries(t *testing.T, db *gorm.DB) *int {
	t.Helper()
	count := 0
	callbackName := "test:count_rpdb_guide_step_queries:" + t.Name()
	if err := db.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if rpdbGuideStepQuery(tx) {
			count++
		}
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})
	return &count
}

func rpdbGuideStepQuery(tx *gorm.DB) bool {
	if tx.Statement.Table == (model.RPDBGuideStep{}).TableName() {
		return true
	}
	return tx.Statement.Schema != nil && tx.Statement.Schema.Table == (model.RPDBGuideStep{}).TableName()
}
