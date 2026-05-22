package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm"
)

type storyMusicPlaylistRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Color       string `json:"color"`
	TrackIDs    []uint `json:"trackIds"`
}

type updateStoryMusicPlaylistRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
	IsPublic    *bool   `json:"isPublic"`
}

type shareStoryMusicPlaylistRequest struct {
	IsPublic *bool `json:"isPublic"`
}

func (s *Server) listStoryMusicPlaylists(c *gin.Context) {
	userID := c.GetUint("userID")

	query := database.DB.Where("user_id = ?", userID).Order("updated_at DESC")
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	var playlists []model.StoryMusicPlaylist
	if err := query.Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载歌单失败"})
		return
	}
	s.hydrateStoryMusicPlaylists(playlists, false)

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}

func (s *Server) createStoryMusicPlaylist(c *gin.Context) {
	userID := c.GetUint("userID")

	var req storyMusicPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "歌单名称不能为空"})
		return
	}

	playlist := model.StoryMusicPlaylist{
		UserID:      userID,
		Name:        truncateString(name, 128),
		Description: truncateString(strings.TrimSpace(req.Description), 512),
		Color:       normalizeStoryMusicColor(req.Color),
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&playlist).Error; err != nil {
			return err
		}
		return s.replaceStoryMusicPlaylistTracks(tx, playlist.ID, userID, req.TrackIDs)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建歌单失败"})
		return
	}

	playlists := []model.StoryMusicPlaylist{playlist}
	s.hydrateStoryMusicPlaylists(playlists, false)
	c.JSON(http.StatusCreated, playlists[0])
}

func (s *Server) updateStoryMusicPlaylist(c *gin.Context) {
	playlist, ok := s.requireOwnedStoryMusicPlaylist(c)
	if !ok {
		return
	}

	var req updateStoryMusicPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "歌单名称不能为空"})
			return
		}
		playlist.Name = truncateString(name, 128)
	}
	if req.Description != nil {
		playlist.Description = truncateString(strings.TrimSpace(*req.Description), 512)
	}
	if req.Color != nil {
		playlist.Color = normalizeStoryMusicColor(*req.Color)
	}
	if req.IsPublic != nil {
		playlist.IsPublic = *req.IsPublic
		if playlist.IsPublic && playlist.ShareCode == "" {
			code, err := s.generateUniqueStoryMusicPlaylistShareCode()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "生成分享链接失败"})
				return
			}
			playlist.ShareCode = code
		}
	}

	if err := database.DB.Save(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存歌单失败"})
		return
	}

	playlists := []model.StoryMusicPlaylist{playlist}
	s.hydrateStoryMusicPlaylists(playlists, false)
	c.JSON(http.StatusOK, playlists[0])
}

func (s *Server) deleteStoryMusicPlaylist(c *gin.Context) {
	playlist, ok := s.requireOwnedStoryMusicPlaylist(c)
	if !ok {
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("playlist_id = ?", playlist.ID).Delete(&model.StoryMusicPlaylistTrack{}).Error; err != nil {
			return err
		}
		return tx.Delete(&playlist).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除歌单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "歌单已删除"})
}

func (s *Server) shareStoryMusicPlaylist(c *gin.Context) {
	playlist, ok := s.requireOwnedStoryMusicPlaylist(c)
	if !ok {
		return
	}

	var req shareStoryMusicPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if playlist.ShareCode == "" {
		code, err := s.generateUniqueStoryMusicPlaylistShareCode()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成分享链接失败"})
			return
		}
		playlist.ShareCode = code
	}
	if req.IsPublic != nil {
		playlist.IsPublic = *req.IsPublic
	}

	if err := database.DB.Save(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存歌单失败"})
		return
	}

	playlists := []model.StoryMusicPlaylist{playlist}
	s.hydrateStoryMusicPlaylists(playlists, false)
	c.JSON(http.StatusOK, playlists[0])
}

func (s *Server) addStoryMusicPlaylistTrack(c *gin.Context) {
	playlist, ok := s.requireOwnedStoryMusicPlaylist(c)
	if !ok {
		return
	}

	trackID, err := strconvUintParam(c, "trackId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的音乐ID"})
		return
	}
	if !s.ensureUserStoryMusicTrack(c, playlist.UserID, trackID) {
		return
	}

	var maxOrder int
	database.DB.Model(&model.StoryMusicPlaylistTrack{}).
		Where("playlist_id = ?", playlist.ID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	if err := database.DB.FirstOrCreate(&model.StoryMusicPlaylistTrack{}, model.StoryMusicPlaylistTrack{
		PlaylistID: playlist.ID,
		TrackID:    trackID,
		SortOrder:  maxOrder + 1,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加入歌单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已加入歌单"})
}

func (s *Server) removeStoryMusicPlaylistTrack(c *gin.Context) {
	playlist, ok := s.requireOwnedStoryMusicPlaylist(c)
	if !ok {
		return
	}

	trackID, err := strconvUintParam(c, "trackId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的音乐ID"})
		return
	}

	result := database.DB.Where("playlist_id = ? AND track_id = ?", playlist.ID, trackID).
		Delete(&model.StoryMusicPlaylistTrack{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移出歌单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已移出歌单"})
}

func (s *Server) listPublicStoryMusicPlaylists(c *gin.Context) {
	query := database.DB.Where("is_public = ?", true).Order("updated_at DESC")
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	limit := 30
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 60 {
		limit = 60
	}

	var playlists []model.StoryMusicPlaylist
	if err := query.Limit(limit).Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载素材市场失败"})
		return
	}
	s.hydrateStoryMusicPlaylists(playlists, true)

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}

func (s *Server) getPublicStoryMusicPlaylist(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享链接"})
		return
	}

	var playlist model.StoryMusicPlaylist
	if err := database.DB.Where("share_code = ?", code).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "歌单不存在"})
		return
	}

	database.DB.Model(&playlist).Update("view_count", playlist.ViewCount+1)
	playlist.ViewCount++

	playlists := []model.StoryMusicPlaylist{playlist}
	s.hydrateStoryMusicPlaylists(playlists, true)
	playlist = playlists[0]

	tracks := s.listStoryMusicPlaylistTracks(playlist.ID)
	c.JSON(http.StatusOK, gin.H{
		"playlist": playlist,
		"tracks":   tracks,
	})
}

func (s *Server) importPublicStoryMusicPlaylist(c *gin.Context) {
	story, ok := s.requireEditableStory(c)
	if !ok {
		return
	}

	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享链接"})
		return
	}

	var playlist model.StoryMusicPlaylist
	if err := database.DB.Where("share_code = ?", code).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "歌单不存在"})
		return
	}

	sourceTracks := s.listStoryMusicPlaylistTracks(playlist.ID)
	if len(sourceTracks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "歌单没有可导入的音乐"})
		return
	}

	imported := make([]model.StoryMusicTrack, 0, len(sourceTracks))
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, source := range sourceTracks {
			track := model.StoryMusicTrack{
				UserID:     story.UserID,
				Name:       source.Name,
				FileName:   source.FileName,
				MimeType:   source.MimeType,
				Size:       source.Size,
				URL:        source.URL,
				StorageKey: source.StorageKey,
				Color:      source.Color,
				Volume:     source.Volume,
				Tags:       source.Tags,
			}
			if err := tx.Create(&track).Error; err != nil {
				return err
			}
			if err := tx.FirstOrCreate(&model.StoryMusicTrackStory{}, model.StoryMusicTrackStory{
				StoryID: story.ID,
				TrackID: track.ID,
			}).Error; err != nil {
				return err
			}
			imported = append(imported, track)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导入歌单失败"})
		return
	}

	s.hydrateStoryMusicTracks(imported)
	c.JSON(http.StatusOK, gin.H{"tracks": imported})
}

func (s *Server) requireOwnedStoryMusicPlaylist(c *gin.Context) (model.StoryMusicPlaylist, bool) {
	userID := c.GetUint("userID")
	playlistID, err := strconvUintParam(c, "playlistId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的歌单ID"})
		return model.StoryMusicPlaylist{}, false
	}

	var playlist model.StoryMusicPlaylist
	if err := database.DB.Where("id = ? AND user_id = ?", playlistID, userID).First(&playlist).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "歌单不存在"})
		return model.StoryMusicPlaylist{}, false
	}
	return playlist, true
}

func (s *Server) ensureUserStoryMusicTrack(c *gin.Context, userID, trackID uint) bool {
	var count int64
	database.DB.Model(&model.StoryMusicTrack{}).Where("id = ? AND user_id = ?", trackID, userID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "音乐不存在"})
		return false
	}
	return true
}

func (s *Server) replaceStoryMusicPlaylistTracks(tx *gorm.DB, playlistID, userID uint, trackIDs []uint) error {
	if err := tx.Where("playlist_id = ?", playlistID).Delete(&model.StoryMusicPlaylistTrack{}).Error; err != nil {
		return err
	}

	seen := make(map[uint]struct{})
	sortOrder := 1
	for _, trackID := range trackIDs {
		if _, ok := seen[trackID]; ok {
			continue
		}
		seen[trackID] = struct{}{}

		var count int64
		if err := tx.Model(&model.StoryMusicTrack{}).Where("id = ? AND user_id = ?", trackID, userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			continue
		}
		if err := tx.Create(&model.StoryMusicPlaylistTrack{
			PlaylistID: playlistID,
			TrackID:    trackID,
			SortOrder:  sortOrder,
		}).Error; err != nil {
			return err
		}
		sortOrder++
	}
	return nil
}

func (s *Server) hydrateStoryMusicPlaylists(playlists []model.StoryMusicPlaylist, includeAuthors bool) {
	if len(playlists) == 0 {
		return
	}

	playlistIDs := make([]uint, 0, len(playlists))
	userIDs := make([]uint, 0, len(playlists))
	for i := range playlists {
		playlistIDs = append(playlistIDs, playlists[i].ID)
		userIDs = append(userIDs, playlists[i].UserID)
		if playlists[i].Color == "" {
			playlists[i].Color = "#B87333"
		}
	}

	var links []model.StoryMusicPlaylistTrack
	database.DB.Where("playlist_id IN ?", playlistIDs).
		Order("sort_order ASC, created_at ASC").
		Find(&links)

	trackIDsByPlaylist := make(map[uint][]uint)
	for _, link := range links {
		trackIDsByPlaylist[link.PlaylistID] = append(trackIDsByPlaylist[link.PlaylistID], link.TrackID)
	}

	authorNames := map[uint]string{}
	if includeAuthors {
		var users []model.User
		database.DB.Select("id", "username").Where("id IN ?", userIDs).Find(&users)
		for _, user := range users {
			authorNames[user.ID] = user.Username
		}
	}

	for i := range playlists {
		playlists[i].TrackIDs = trackIDsByPlaylist[playlists[i].ID]
		playlists[i].TrackCount = len(playlists[i].TrackIDs)
		if includeAuthors {
			playlists[i].AuthorName = authorNames[playlists[i].UserID]
		}
	}
}

func (s *Server) listStoryMusicPlaylistTracks(playlistID uint) []model.StoryMusicTrack {
	var tracks []model.StoryMusicTrack
	database.DB.
		Joins("JOIN story_music_playlist_tracks ON story_music_playlist_tracks.track_id = story_music_tracks.id").
		Where("story_music_playlist_tracks.playlist_id = ?", playlistID).
		Order("story_music_playlist_tracks.sort_order ASC, story_music_playlist_tracks.created_at ASC").
		Find(&tracks)
	s.hydrateStoryMusicTracks(tracks)
	return tracks
}

func (s *Server) generateUniqueStoryMusicPlaylistShareCode() (string, error) {
	for i := 0; i < 10; i++ {
		code := generateShareCode()
		var count int64
		if err := database.DB.Model(&model.StoryMusicPlaylist{}).Where("share_code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", gorm.ErrDuplicatedKey
}
