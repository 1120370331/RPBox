package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/model"
)

const rpdbPublicCacheControl = "public, max-age=60, stale-while-revalidate=300"

// prepareRPDBHTTPResponse makes every RPDB response non-cacheable by default.
// A successful, verified anonymous public response may override this policy in
// writeRPDBConditionalJSON.
func prepareRPDBHTTPResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	vary := c.Writer.Header().Values("Vary")
	for _, value := range vary {
		for _, field := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(field), "Authorization") {
				return
			}
		}
	}
	vary = append(vary, "Authorization")
	c.Header("Vary", strings.Join(vary, ", "))
}

// writeRPDBConditionalJSON writes the exact JSON bytes used to derive the ETag.
// Requests carrying Authorization never receive public caching or a 304 based
// on an anonymous representation, even when the token is invalid.
func writeRPDBConditionalJSON(c *gin.Context, viewerID uint, public bool, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "编码响应失败"})
		return
	}

	authorizedRequest := len(c.Request.Header.Values("Authorization")) != 0
	if !public || viewerID != 0 || authorizedRequest {
		c.Data(http.StatusOK, "application/json; charset=utf-8", body)
		return
	}

	sum := sha256.Sum256(body)
	etag := fmt.Sprintf("\"%x\"", sum)
	c.Header("Cache-Control", rpdbPublicCacheControl)
	c.Header("ETag", etag)
	if rpdbIfNoneMatch(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}

func rpdbIfNoneMatch(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		candidate = strings.TrimSpace(strings.TrimPrefix(candidate, "W/"))
		if candidate == etag {
			return true
		}
	}
	return false
}

func rpdbWorkIsPublic(work model.RPDBWork) bool {
	return work.Status == model.RPDBStatusPublished &&
		work.ReviewStatus == model.RPDBReviewApproved &&
		work.IsPublic &&
		normalizeRPDBVisibility(work.Visibility, work.IsPublic) == model.RPDBVisibilityPublic
}

func rpdbWorkCardsArePublic(cards []rpdbWorkCard) bool {
	for _, card := range cards {
		if !rpdbWorkIsPublic(card.RPDBWork) {
			return false
		}
	}
	return true
}
