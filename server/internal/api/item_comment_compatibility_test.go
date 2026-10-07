package api

import "testing"

func TestItemCommentCompatibilityPreservesVisibilityAndStoredContent(t *testing.T) {
	testCommentCompatibilityEndpoint(t, true)
}
