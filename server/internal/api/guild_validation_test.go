package api

import (
	"strings"
	"testing"
)

func TestNormalizeCreateGuildRequestRejectsOversizedName(t *testing.T) {
	req := CreateGuildRequest{
		Name: strings.Repeat("测", 129),
	}

	err := normalizeCreateGuildRequest(&req)
	if err == nil {
		t.Fatal("expected oversized guild name to be rejected")
	}
}

func TestNormalizeCreateGuildRequestRejectsInvalidFaction(t *testing.T) {
	req := CreateGuildRequest{
		Name:    "测试公会",
		Faction: "drop table",
	}

	err := normalizeCreateGuildRequest(&req)
	if err == nil {
		t.Fatal("expected invalid faction to be rejected")
	}
}

func TestNormalizeCreateGuildRequestTrimsAndDefaultsLayout(t *testing.T) {
	req := CreateGuildRequest{
		Name:        "  测试公会  ",
		Description: "  描述  ",
		Slogan:      "  标语  ",
		Layout:      99,
	}

	err := normalizeCreateGuildRequest(&req)
	if err != nil {
		t.Fatalf("expected request to be accepted, got error: %v", err)
	}
	if req.Name != "测试公会" {
		t.Fatalf("expected trimmed name, got %q", req.Name)
	}
	if req.Description != "描述" {
		t.Fatalf("expected trimmed description, got %q", req.Description)
	}
	if req.Slogan != "标语" {
		t.Fatalf("expected trimmed slogan, got %q", req.Slogan)
	}
	if req.Layout != 3 {
		t.Fatalf("expected invalid layout to default to 3, got %d", req.Layout)
	}
}

func TestNormalizeUpdateGuildRequestRejectsInvalidLayout(t *testing.T) {
	req := UpdateGuildRequest{
		Layout: 9,
	}

	err := normalizeUpdateGuildRequest(&req)
	if err == nil {
		t.Fatal("expected invalid update layout to be rejected")
	}
}

func TestNormalizeUpdateGuildRequestCountsVisibleLoreTextInsteadOfHTMLLength(t *testing.T) {
	req := UpdateGuildRequest{
		Lore: strings.Repeat(`<p><span data-node-view-wrapper="" data-node-view-content="" data-jump-title="很长的元数据">设定</span></p>`, 1500),
	}

	err := normalizeUpdateGuildRequest(&req)
	if err != nil {
		t.Fatalf("expected lore with short visible text to be accepted, got error: %v", err)
	}
}

func TestNormalizeUpdateGuildRequestRejectsOversizedVisibleLoreText(t *testing.T) {
	req := UpdateGuildRequest{
		Lore: "<p>" + strings.Repeat("设", 20001) + "</p>",
	}

	err := normalizeUpdateGuildRequest(&req)
	if err == nil {
		t.Fatal("expected oversized visible lore text to be rejected")
	}
}
