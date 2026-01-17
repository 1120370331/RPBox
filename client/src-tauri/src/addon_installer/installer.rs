use std::fs;
use std::io::{self, Read, Write};
use std::path::{Path, PathBuf};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct InstalledAddonInfo {
    pub installed: bool,
    pub version: Option<String>,
    pub path: Option<String>,
}

/// 获取插件安装路径
/// wow_path 可能是:
/// - WTF 目录: D:\World of Warcraft\_retail_\WTF
/// - 版本目录: D:\World of Warcraft\_retail_
pub fn get_addon_path(wow_path: &str, _flavor: &str) -> PathBuf {
    let path = PathBuf::from(wow_path);

    // 判断是 WTF 目录还是版本目录
    let version_dir = if path.ends_with("WTF") {
        // WTF 目录，向上一级
        path.parent().map(|p| p.to_path_buf()).unwrap_or(path)
    } else {
        // 已经是版本目录 (_retail_ 等)
        path
    };

    version_dir.join("Interface").join("AddOns").join("RPBox_Addon")
}

/// 检查插件是否已安装并获取版本
pub fn check_addon_installed(wow_path: &str, flavor: &str) -> InstalledAddonInfo {
    let addon_path = get_addon_path(wow_path, flavor);
    let toc_path = addon_path.join("RPBox_Addon.toc");

    if !toc_path.exists() {
        return InstalledAddonInfo {
            installed: false,
            version: None,
            path: None,
        };
    }

    let version = read_toc_version(&toc_path);
    InstalledAddonInfo {
        installed: true,
        version,
        path: Some(addon_path.to_string_lossy().to_string()),
    }
}

/// 从 .toc 文件读取版本号
fn read_toc_version(toc_path: &Path) -> Option<String> {
    let content = fs::read_to_string(toc_path).ok()?;
    for line in content.lines() {
        let line = line.trim();
        if line.starts_with("## Version:") {
            return Some(line.replace("## Version:", "").trim().to_string());
        }
    }
    None
}

/// 安装插件（从zip数据）
/// wow_path 可能是 WTF 目录或版本目录
pub fn install_addon(wow_path: &str, _flavor: &str, zip_data: &[u8]) -> Result<String, String> {
    let path = PathBuf::from(wow_path);

    // 判断是 WTF 目录还是版本目录
    let version_dir = if path.ends_with("WTF") {
        path.parent()
            .ok_or_else(|| "无效的WoW路径".to_string())?
            .to_path_buf()
    } else {
        path
    };

    let addons_dir = version_dir.join("Interface").join("AddOns");

    // 确保 AddOns 目录存在
    fs::create_dir_all(&addons_dir)
        .map_err(|e| format!("创建 AddOns 目录失败: {}", e))?;

    let addon_path = addons_dir.join("RPBox_Addon");

    // 直接解压覆盖（不删除旧文件，避免文件锁定问题）
    extract_zip(zip_data, &addons_dir)?;

    Ok(addon_path.to_string_lossy().to_string())
}

/// 解压 zip 文件
fn extract_zip(data: &[u8], dest: &Path) -> Result<(), String> {
    let cursor = std::io::Cursor::new(data);
    let mut archive = zip::ZipArchive::new(cursor)
        .map_err(|e| format!("打开 zip 失败: {}", e))?;

    for i in 0..archive.len() {
        let mut file = archive.by_index(i)
            .map_err(|e| format!("读取 zip 条目失败: {}", e))?;

        let outpath = match file.enclosed_name() {
            Some(path) => dest.join(path),
            None => continue,
        };

        if file.name().ends_with('/') {
            fs::create_dir_all(&outpath)
                .map_err(|e| format!("创建目录失败: {}", e))?;
        } else {
            if let Some(p) = outpath.parent() {
                if !p.exists() {
                    fs::create_dir_all(p)
                        .map_err(|e| format!("创建目录失败: {}", e))?;
                }
            }
            let mut outfile = fs::File::create(&outpath)
                .map_err(|e| format!("创建文件失败: {}", e))?;
            io::copy(&mut file, &mut outfile)
                .map_err(|e| format!("写入文件失败: {}", e))?;
        }
    }

    Ok(())
}

/// 卸载插件
pub fn uninstall_addon(wow_path: &str, flavor: &str) -> Result<(), String> {
    let addon_path = get_addon_path(wow_path, flavor);
    if addon_path.exists() {
        fs::remove_dir_all(&addon_path)
            .map_err(|e| format!("删除插件失败: {}", e))?;
    }
    Ok(())
}
