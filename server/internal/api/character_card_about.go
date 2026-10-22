package api

import (
	"bytes"
	"encoding/json"
	"errors"
)

// normalizeCharacterCardTRP3About retains native TRP3 markup and all template
// options, including inactive templates. It is data, never trusted HTML.
func normalizeCharacterCardTRP3About(raw json.RawMessage) (string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", nil
	}
	if len(raw) > characterCardRichTextMaxBytes {
		return "", errors.New("TRP3 关于模板内容过大")
	}
	var about struct {
		Template int `json:"TE"`
		T1       struct {
			Text string `json:"TX"`
		} `json:"T1"`
		T2 []struct {
			Text string `json:"TX"`
			Icon string `json:"IC"`
		} `json:"T2"`
		T3 map[string]struct {
			Text string `json:"TX"`
			Icon string `json:"IC"`
		} `json:"T3"`
	}
	if err := json.Unmarshal(raw, &about); err != nil || about.Template < 1 || about.Template > 3 || len(about.T2) > 50 {
		return "", errors.New("TRP3 关于模板格式无效")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return "", errors.New("TRP3 关于模板格式无效")
	}
	return compact.String(), nil
}
