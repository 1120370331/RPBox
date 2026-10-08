package api

import "strings"

const commentImageContentFallback = "【图片评论】请使用新版 RPBox 查看配图。"

// compatibleCommentContent keeps image-only comments readable by clients that
// only render content. Apply it to response copies after visibility filtering;
// stored content and the image fields remain unchanged.
func compatibleCommentContent(content, imageURL, reviewStatus string) string {
	if strings.TrimSpace(content) == "" && strings.TrimSpace(imageURL) != "" && reviewStatus == commentImageReviewApproved {
		return commentImageContentFallback
	}
	return content
}
