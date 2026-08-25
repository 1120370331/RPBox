package api

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxRPDBMusicianMIDIBytes int64 = 10 << 20

var musicianMIDIHeader = []byte{'M', 'T', 'h', 'd', 0, 0, 0, 6}

// uploadRPDBMusicianMIDI validates and stores a Standard MIDI file for an RPDB work.
func (s *Server) uploadRPDBMusicianMIDI(c *gin.Context) {
	header, err := c.FormFile("file")
	if err != nil || header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择 MIDI 文件"})
		return
	}

	extension := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	if extension != ".mid" && extension != ".midi" && extension != ".kar" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .mid、.midi 或 .kar 文件"})
		return
	}
	if header.Size <= 0 || header.Size > maxRPDBMusicianMIDIBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MIDI 文件不能为空且不能超过 10MB"})
		return
	}

	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取 MIDI 文件"})
		return
	}
	prefix := make([]byte, 14)
	_, readErr := io.ReadFull(file, prefix)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || !isStandardMIDIHeader(prefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不是有效的 Standard MIDI 格式"})
		return
	}

	fileURL, err := s.saveUploadedAttachment(c, header, "rpdb/musician-midi")
	if err != nil {
		if errors.Is(err, errAttachmentTooLarge) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MIDI 文件不能超过 10MB"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 MIDI 文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":  fileURL,
		"name": filepath.Base(header.Filename),
		"size": header.Size,
	})
}

func isStandardMIDIHeader(header []byte) bool {
	if len(header) < 14 || !bytes.Equal(header[:len(musicianMIDIHeader)], musicianMIDIHeader) {
		return false
	}
	format := binary.BigEndian.Uint16(header[8:10])
	tracks := binary.BigEndian.Uint16(header[10:12])
	division := binary.BigEndian.Uint16(header[12:14])
	if format > 2 || tracks == 0 || division == 0 {
		return false
	}
	return format != 0 || tracks == 1
}
