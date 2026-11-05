package api

import (
	"context"
	"fmt"
)

// trp3LatestForFlavor selects packages by the client's interface family.
// Classic packages are pinned to releases whose TOCs declare the matching
// interface; an unavailable package is never replaced by a retail download.
func (s *Server) trp3LatestForFlavor(ctx context.Context, flavor string) (*TRP3LatestResponse, error) {
	if !validGameFlavor(flavor) {
		return nil, fmt.Errorf("unsupported game flavor: %s", flavor)
	}
	latest, err := s.loadTRP3Latest(ctx)
	if err != nil {
		return nil, err
	}
	selected := cloneTRP3Latest(latest)
	switch flavor {
	case "", "_retail_", "_forever_":
		return selected, nil
	case "_classic_", "_classic_era_", "_classic_anniversary_":
		selected.Source = "compatibility-catalog"
		selected.Note = "已按游戏客户端选择兼容插件包；TRP3 Extended 官方未提供怀旧服版本。"
		for index := range selected.Addons {
			addon := &selected.Addons[index]
			switch addon.ID {
			case "total-rp-3":
				s.setTRP3CompatibleRelease(addon, "3.4.1", "totalRP3-3.4.1.zip")
			case "musician":
				s.setTRP3CompatibleRelease(addon, "1.9.18.0", "Musician-1.9.18.0.zip")
			case "total-rp-3-extended":
				addon.LatestVersion = ""
				addon.DownloadURL = ""
				addon.FileName = ""
				addon.CompatibilityNote = "TRP3 Extended 官方未提供此客户端的兼容包"
			}
		}
		return selected, nil
	default:
		selected.Source = "compatibility-catalog"
		selected.Note = "测试客户端的接口版本可能变化，请在插件官方页面选择对应版本；自动安装暂不可用。"
		for index := range selected.Addons {
			addon := &selected.Addons[index]
			addon.LatestVersion = ""
			addon.DownloadURL = ""
			addon.FileName = ""
			addon.CompatibilityNote = "测试客户端暂无经过验证的自动安装包"
		}
		return selected, nil
	}
}

func (s *Server) setTRP3CompatibleRelease(addon *TRP3AddonLatestInfo, version, fileName string) {
	if addon == nil {
		return
	}
	addon.LatestVersion = version
	addon.FileName = fileName
	addon.FileDate = ""
	addon.SourceURL = fmt.Sprintf("https://github.com/%s/releases/tag/%s", addon.Repository, version)
	addon.DownloadURL = s.rewriteGitHubDownloadURL(
		fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", addon.Repository, version, fileName),
	)
	addon.CompatibilityNote = ""
}
