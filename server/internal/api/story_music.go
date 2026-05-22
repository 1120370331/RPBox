package api

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm"
)

const maxStoryMusicBytes int64 = 30 << 20

var storyMusicExtContentTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".flac": "audio/flac",
	".webm": "audio/webm",
}

type storyMusicUploadResult struct {
	URL         string
	StorageKey  string
	ContentType string
	Size        int64
}

type updateStoryMusicTrackRequest struct {
	Name   *string  `json:"name"`
	Color  *string  `json:"color"`
	Volume *float64 `json:"volume"`
	Tags   []string `json:"tags"`
}

type createStoryMusicSegmentRequest struct {
	TrackID        uint     `json:"trackId" binding:"required"`
	StartEntryID   uint     `json:"startEntryId" binding:"required"`
	EndEntryID     *uint    `json:"endEntryId"`
	Loop           *bool    `json:"loop"`
	AutoPlay       *bool    `json:"autoPlay"`
	FadeInSeconds  *float64 `json:"fadeInSeconds"`
	FadeOutSeconds *float64 `json:"fadeOutSeconds"`
	Volume         *float64 `json:"volume"`
}

type updateStoryMusicSegmentRequest struct {
	TrackID        *uint    `json:"trackId"`
	StartEntryID   *uint    `json:"startEntryId"`
	EndEntryID     *uint    `json:"endEntryId"`
	Loop           *bool    `json:"loop"`
	AutoPlay       *bool    `json:"autoPlay"`
	FadeInSeconds  *float64 `json:"fadeInSeconds"`
	FadeOutSeconds *float64 `json:"fadeOutSeconds"`
	Volume         *float64 `json:"volume"`
}

func (s *Server) getStoryMusic(c *gin.Context) {
	story, ok := s.requireEditableStory(c)
	if !ok {
		return
	}

	query := database.DB.Where("user_id = ?", story.UserID).Order("updated_at DESC")
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR file_name LIKE ? OR tags LIKE ?", like, like, like)
	}

	var tracks []model.StoryMusicTrack
	if err := query.Find(&tracks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载音乐失败"})
		return
	}
	s.hydrateStoryMusicTracks(tracks)

	var segments []model.StoryMusicSegment
	if err := database.DB.Where("story_id = ?", story.ID).Order("created_at ASC").Find(&segments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载音乐定位失败"})
		return
	}

	var playlists []model.StoryMusicPlaylist
	if err := database.DB.Where("user_id = ?", story.UserID).Order("updated_at DESC").Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载歌单失败"})
		return
	}
	s.hydrateStoryMusicPlaylists(playlists, false)

	c.JSON(http.StatusOK, gin.H{
		"tracks":    tracks,
		"segments":  segments,
		"playlists": playlists,
	})
}

func (s *Server) uploadStoryMusicTrack(c *gin.Context) {
	story, ok := s.requireEditableStory(c)
	if !ok {
		return
	}

	header, err := c.FormFile("audio")
	if err != nil {
		header, err = c.FormFile("file")
	}
	if err != nil || header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择音频文件"})
		return
	}

	uploaded, err := s.saveUploadedStoryMusic(c, header, fmt.Sprintf("story-music/%d", story.UserID))
	if err != nil {
		if errors.Is(err, errAttachmentTooLarge) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "音频不能超过30MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		name = defaultStoryMusicName(header.Filename)
	}

	track := model.StoryMusicTrack{
		UserID:     story.UserID,
		Name:       truncateString(name, 128),
		FileName:   filepath.Base(header.Filename),
		MimeType:   uploaded.ContentType,
		Size:       uploaded.Size,
		URL:        uploaded.URL,
		StorageKey: uploaded.StorageKey,
		Color:      normalizeStoryMusicColor(c.PostForm("color")),
		Volume:     clampStoryMusicVolume(parseFloatForm(c.PostForm("volume"), 0.75)),
		Tags:       encodeStoryMusicTags(parseStoryMusicTagsInput(c.PostForm("tags"))),
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&track).Error; err != nil {
			return err
		}
		return tx.FirstOrCreate(&model.StoryMusicTrackStory{}, model.StoryMusicTrackStory{
			StoryID: story.ID,
			TrackID: track.ID,
		}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存音乐失败"})
		return
	}

	tracks := []model.StoryMusicTrack{track}
	s.hydrateStoryMusicTracks(tracks)
	c.JSON(http.StatusCreated, tracks[0])
}

func (s *Server) updateStoryMusicTrack(c *gin.Context) {
	story, track, ok := s.requireStoryMusicTrack(c)
	if !ok {
		return
	}

	var req updateStoryMusicTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "音乐名称不能为空"})
			return
		}
		track.Name = truncateString(name, 128)
	}
	if req.Color != nil {
		track.Color = normalizeStoryMusicColor(*req.Color)
	}
	if req.Volume != nil {
		track.Volume = clampStoryMusicVolume(*req.Volume)
	}
	if req.Tags != nil {
		track.Tags = encodeStoryMusicTags(req.Tags)
	}

	if err := database.DB.Save(&track).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存音乐失败"})
		return
	}

	_ = story
	tracks := []model.StoryMusicTrack{track}
	s.hydrateStoryMusicTracks(tracks)
	c.JSON(http.StatusOK, tracks[0])
}

func (s *Server) attachStoryMusicTrack(c *gin.Context) {
	story, track, ok := s.requireStoryMusicTrack(c)
	if !ok {
		return
	}

	if err := database.DB.FirstOrCreate(&model.StoryMusicTrackStory{}, model.StoryMusicTrackStory{
		StoryID: story.ID,
		TrackID: track.ID,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加入本剧情失败"})
		return
	}

	tracks := []model.StoryMusicTrack{track}
	s.hydrateStoryMusicTracks(tracks)
	c.JSON(http.StatusOK, tracks[0])
}

func (s *Server) detachStoryMusicTrack(c *gin.Context) {
	story, track, ok := s.requireStoryMusicTrack(c)
	if !ok {
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("story_id = ? AND track_id = ?", story.ID, track.ID).Delete(&model.StoryMusicSegment{}).Error; err != nil {
			return err
		}
		return tx.Where("story_id = ? AND track_id = ?", story.ID, track.ID).Delete(&model.StoryMusicTrackStory{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移除音乐失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已移除音乐"})
}

func (s *Server) deleteStoryMusicTrack(c *gin.Context) {
	_, track, ok := s.requireStoryMusicTrack(c)
	if !ok {
		return
	}

	var sameURLCount int64
	if track.URL != "" {
		database.DB.Model(&model.StoryMusicTrack{}).
			Where("url = ? AND id <> ?", track.URL, track.ID).
			Count(&sameURLCount)
	}
	uploadKey := extractUploadKey(c, track.URL)

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("track_id = ?", track.ID).Delete(&model.StoryMusicSegment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("track_id = ?", track.ID).Delete(&model.StoryMusicPlaylistTrack{}).Error; err != nil {
			return err
		}
		if err := tx.Where("track_id = ?", track.ID).Delete(&model.StoryMusicTrackStory{}).Error; err != nil {
			return err
		}
		return tx.Delete(&track).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除音乐失败"})
		return
	}

	if sameURLCount == 0 && uploadKey != "" {
		s.deleteUploadKey(uploadKey)
	}

	c.JSON(http.StatusOK, gin.H{"message": "音乐已删除"})
}

func (s *Server) createStoryMusicSegment(c *gin.Context) {
	story, ok := s.requireEditableStory(c)
	if !ok {
		return
	}

	var req createStoryMusicSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	track, ok := s.ensureStoryMusicTrackAttached(c, story, req.TrackID)
	if !ok {
		return
	}
	if !s.ensureStoryEntry(c, story.ID, req.StartEntryID, "开始条目不存在") {
		return
	}
	if req.EndEntryID != nil && !s.ensureStoryEntry(c, story.ID, *req.EndEntryID, "结束条目不存在") {
		return
	}

	loop := true
	if req.Loop != nil {
		loop = *req.Loop
	}
	autoPlay := true
	if req.AutoPlay != nil {
		autoPlay = *req.AutoPlay
	}

	segment := model.StoryMusicSegment{
		StoryID:        story.ID,
		TrackID:        track.ID,
		StartEntryID:   req.StartEntryID,
		EndEntryID:     req.EndEntryID,
		Loop:           loop,
		AutoPlay:       autoPlay,
		FadeInSeconds:  clampStoryMusicFade(valueOrDefault(req.FadeInSeconds, 1)),
		FadeOutSeconds: clampStoryMusicFade(valueOrDefault(req.FadeOutSeconds, 1)),
		Volume:         clampStoryMusicVolume(valueOrDefault(req.Volume, track.Volume)),
	}

	if err := database.DB.Create(&segment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存音乐定位失败"})
		return
	}

	c.JSON(http.StatusCreated, segment)
}

func (s *Server) updateStoryMusicSegment(c *gin.Context) {
	story, ok := s.requireEditableStory(c)
	if !ok {
		return
	}

	segmentID, err := strconvUintParam(c, "segmentId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的音乐定位"})
		return
	}

	var segment model.StoryMusicSegment
	if err := database.DB.Where("id = ? AND story_id = ?", segmentID, story.ID).First(&segment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐定位不存在"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求失败"})
		return
	}

	var req updateStoryMusicSegmentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	_, endEntryIDProvided := raw["endEntryId"]

	updates := map[string]interface{}{}

	if req.TrackID != nil {
		track, ok := s.ensureStoryMusicTrackAttached(c, story, *req.TrackID)
		if !ok {
			return
		}
		updates["track_id"] = track.ID
	}
	if req.StartEntryID != nil {
		if !s.ensureStoryEntry(c, story.ID, *req.StartEntryID, "开始条目不存在") {
			return
		}
		updates["start_entry_id"] = *req.StartEntryID
	}
	if req.EndEntryID != nil {
		if !s.ensureStoryEntry(c, story.ID, *req.EndEntryID, "结束条目不存在") {
			return
		}
		updates["end_entry_id"] = *req.EndEntryID
	} else if endEntryIDProvided {
		updates["end_entry_id"] = nil
	}
	if req.Loop != nil {
		updates["loop"] = *req.Loop
	}
	if req.AutoPlay != nil {
		updates["auto_play"] = *req.AutoPlay
	}
	if req.FadeInSeconds != nil {
		updates["fade_in_seconds"] = clampStoryMusicFade(*req.FadeInSeconds)
	}
	if req.FadeOutSeconds != nil {
		updates["fade_out_seconds"] = clampStoryMusicFade(*req.FadeOutSeconds)
	}
	if req.Volume != nil {
		updates["volume"] = clampStoryMusicVolume(*req.Volume)
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&segment).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存音乐定位失败"})
			return
		}
		if err := database.DB.First(&segment, segment.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载音乐定位失败"})
			return
		}
	}

	c.JSON(http.StatusOK, segment)
}

func (s *Server) deleteStoryMusicSegment(c *gin.Context) {
	story, ok := s.requireEditableStory(c)
	if !ok {
		return
	}

	segmentID, err := strconvUintParam(c, "segmentId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的音乐定位"})
		return
	}

	result := database.DB.Where("id = ? AND story_id = ?", segmentID, story.ID).Delete(&model.StoryMusicSegment{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除音乐定位失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐定位不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "音乐定位已删除"})
}

func (s *Server) loadPublicStoryMusic(storyID uint) ([]model.StoryMusicTrack, []model.StoryMusicSegment) {
	var tracks []model.StoryMusicTrack
	database.DB.
		Joins("JOIN story_music_track_stories ON story_music_track_stories.track_id = story_music_tracks.id").
		Where("story_music_track_stories.story_id = ?", storyID).
		Order("story_music_tracks.updated_at DESC").
		Find(&tracks)
	s.hydrateStoryMusicTracks(tracks)

	var segments []model.StoryMusicSegment
	database.DB.Where("story_id = ?", storyID).Order("created_at ASC").Find(&segments)
	return tracks, segments
}

func (s *Server) requireEditableStory(c *gin.Context) (model.Story, bool) {
	id, err := strconvUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的剧情ID"})
		return model.Story{}, false
	}

	var story model.Story
	if err := database.DB.Where("id = ?", id).First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return model.Story{}, false
	}

	if !s.canEditStory(c, story) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权编辑剧情"})
		return model.Story{}, false
	}

	return story, true
}

func (s *Server) requireStoryMusicTrack(c *gin.Context) (model.Story, model.StoryMusicTrack, bool) {
	story, ok := s.requireEditableStory(c)
	if !ok {
		return model.Story{}, model.StoryMusicTrack{}, false
	}

	trackID, err := strconvUintParam(c, "trackId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的音乐ID"})
		return model.Story{}, model.StoryMusicTrack{}, false
	}

	var track model.StoryMusicTrack
	if err := database.DB.Where("id = ? AND user_id = ?", trackID, story.UserID).First(&track).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐不存在"})
		return model.Story{}, model.StoryMusicTrack{}, false
	}

	return story, track, true
}

func (s *Server) canEditStory(c *gin.Context, story model.Story) bool {
	userID := c.GetUint("userID")
	if userID == story.UserID {
		return true
	}

	var user model.User
	if err := database.DB.Select("id", "role").First(&user, userID).Error; err != nil {
		return false
	}
	return user.Role == "admin" || user.Role == "moderator"
}

func (s *Server) ensureStoryMusicTrackAttached(c *gin.Context, story model.Story, trackID uint) (model.StoryMusicTrack, bool) {
	var track model.StoryMusicTrack
	if err := database.DB.Where("id = ? AND user_id = ?", trackID, story.UserID).First(&track).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐不存在"})
		return model.StoryMusicTrack{}, false
	}

	var count int64
	database.DB.Model(&model.StoryMusicTrackStory{}).
		Where("story_id = ? AND track_id = ?", story.ID, track.ID).
		Count(&count)
	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先将音乐加入本剧情"})
		return model.StoryMusicTrack{}, false
	}

	return track, true
}

func (s *Server) ensureStoryEntry(c *gin.Context, storyID, entryID uint, message string) bool {
	var count int64
	database.DB.Model(&model.StoryEntry{}).Where("id = ? AND story_id = ?", entryID, storyID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return false
	}
	return true
}

func (s *Server) hydrateStoryMusicTracks(tracks []model.StoryMusicTrack) {
	if len(tracks) == 0 {
		return
	}

	ids := make([]uint, 0, len(tracks))
	for i := range tracks {
		ids = append(ids, tracks[i].ID)
		tracks[i].TagsList = decodeStoryMusicTags(tracks[i].Tags)
		if tracks[i].Color == "" {
			tracks[i].Color = "#B87333"
		}
		if tracks[i].Volume <= 0 {
			tracks[i].Volume = 0.75
		}
	}

	var links []model.StoryMusicTrackStory
	database.DB.Where("track_id IN ?", ids).Find(&links)
	storyIDsByTrack := make(map[uint][]uint)
	for _, link := range links {
		storyIDsByTrack[link.TrackID] = append(storyIDsByTrack[link.TrackID], link.StoryID)
	}
	for i := range tracks {
		tracks[i].StoryIDs = storyIDsByTrack[tracks[i].ID]
	}
}

func (s *Server) saveUploadedStoryMusic(c *gin.Context, header *multipart.FileHeader, subdir string) (storyMusicUploadResult, error) {
	if header == nil {
		return storyMusicUploadResult{}, errors.New("empty audio")
	}
	if header.Size > maxStoryMusicBytes {
		return storyMusicUploadResult{}, errAttachmentTooLarge
	}

	file, err := header.Open()
	if err != nil {
		return storyMusicUploadResult{}, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxStoryMusicBytes+1))
	if err != nil {
		return storyMusicUploadResult{}, err
	}
	if int64(len(data)) > maxStoryMusicBytes {
		return storyMusicUploadResult{}, errAttachmentTooLarge
	}
	if len(data) == 0 {
		return storyMusicUploadResult{}, errors.New("音频文件为空")
	}

	ext := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	contentType := strings.TrimSpace(strings.Split(header.Header.Get("Content-Type"), ";")[0])
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	if byExt := storyMusicExtContentTypes[ext]; byExt != "" {
		contentType = byExt
	}
	if !strings.HasPrefix(contentType, "audio/") {
		return storyMusicUploadResult{}, errors.New("仅支持音频文件")
	}
	if _, ok := storyMusicExtContentTypes[ext]; !ok {
		ext = storyMusicExtension(contentType)
	}
	if ext == "" {
		return storyMusicUploadResult{}, errors.New("不支持的音频格式")
	}

	cleanSubdir := cleanUploadSubdir(subdir)
	hash := md5.Sum(data)
	filename := fmt.Sprintf("%x%s", hash, ext)
	relativePath := path.Join(cleanSubdir, filename)

	if s.ossEnabled() {
		objectKey := s.buildOSSKey(cleanSubdir, filename)
		if err := s.uploadToOSS(objectKey, data, contentType); err != nil {
			return storyMusicUploadResult{}, err
		}
		urlPath := path.Join("/", uploadDirName, relativePath)
		return storyMusicUploadResult{
			URL:         buildPublicURL(c, urlPath),
			StorageKey:  relativePath,
			ContentType: contentType,
			Size:        int64(len(data)),
		}, nil
	}

	baseDir := filepath.Join(s.cfg.Storage.Path, uploadDirName)
	targetDir := baseDir
	if cleanSubdir != "" {
		targetDir = filepath.Join(baseDir, filepath.FromSlash(cleanSubdir))
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return storyMusicUploadResult{}, err
	}
	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return storyMusicUploadResult{}, err
	}

	urlPath := path.Join("/", uploadDirName, relativePath)
	return storyMusicUploadResult{
		URL:         buildPublicURL(c, urlPath),
		StorageKey:  relativePath,
		ContentType: contentType,
		Size:        int64(len(data)),
	}, nil
}

func storyMusicExtension(contentType string) string {
	switch strings.TrimSpace(strings.Split(contentType, ";")[0]) {
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "audio/mp4", "audio/x-m4a":
		return ".m4a"
	case "audio/aac":
		return ".aac"
	case "audio/flac", "audio/x-flac":
		return ".flac"
	case "audio/webm":
		return ".webm"
	default:
		return ""
	}
}

func defaultStoryMusicName(fileName string) string {
	base := filepath.Base(fileName)
	name := strings.TrimSpace(strings.TrimSuffix(base, filepath.Ext(base)))
	if name == "" {
		return "未命名音乐"
	}
	return truncateString(name, 128)
}

func normalizeStoryMusicColor(value string) string {
	color := strings.TrimSpace(value)
	if color == "" {
		return "#B87333"
	}
	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}
	if len(color) != 7 {
		return "#B87333"
	}
	return strings.ToUpper(color[:1]) + strings.ToUpper(color[1:])
}

func clampStoryMusicVolume(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampStoryMusicFade(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 20 {
		return 20
	}
	return value
}

func valueOrDefault(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

func parseFloatForm(value string, fallback float64) float64 {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseStoryMusicTagsInput(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(value), &tags); err == nil {
		return cleanStoryMusicTags(tags)
	}
	return cleanStoryMusicTags(strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，'
	}))
}

func encodeStoryMusicTags(tags []string) string {
	cleaned := cleanStoryMusicTags(tags)
	data, _ := json.Marshal(cleaned)
	return string(data)
}

func decodeStoryMusicTags(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(value), &tags); err == nil {
		return cleanStoryMusicTags(tags)
	}
	return cleanStoryMusicTags(strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，'
	}))
}

func cleanStoryMusicTags(tags []string) []string {
	seen := make(map[string]struct{})
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if len(tag) > 32 {
			tag = tag[:32]
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, tag)
		if len(cleaned) >= 12 {
			break
		}
	}
	return cleaned
}

func truncateString(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func strconvUintParam(c *gin.Context, name string) (uint, error) {
	value, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(value), nil
}
