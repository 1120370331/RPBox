use crate::wow_path;
use serde::{Deserialize, Serialize};
use std::fs;
use std::io::{self, Read};
use std::path::{Path, PathBuf};
use std::time::Duration;

#[derive(Debug, Serialize, Deserialize)]
pub struct InstalledAddonInfo {
    pub installed: bool,
    pub version: Option<String>,
    pub path: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Trp3AddonInfo {
    pub id: String,
    pub name: String,
    pub installed: bool,
    pub latest_version: String,
    pub requires_update: bool,
    pub version: Option<String>,
    pub path: Option<String>,
    pub curseforge_url: String,
    pub source_url: String,
    pub download_url: String,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Trp3AddonCheckResult {
    pub wow_path: String,
    pub addons_dir: String,
    pub latest_check_available: bool,
    pub latest_check_note: String,
    pub addons: Vec<Trp3AddonInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Trp3AddonInstallRequest {
    pub addon_id: String,
    pub download_url: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AddonInstallProgress {
    pub plugin_id: String,
    pub label: String,
    pub detail: String,
    pub percent: Option<u8>,
    pub downloaded_bytes: u64,
    pub total_bytes: Option<u64>,
}

struct Trp3AddonDefinition {
    id: &'static str,
    name: &'static str,
    primary_folder: &'static str,
    folders: &'static [&'static str],
    toc: &'static str,
    curseforge_url: &'static str,
    source_url: &'static str,
}

const TRP3_ADDONS: &[Trp3AddonDefinition] = &[
    Trp3AddonDefinition {
        id: "total-rp-3",
        name: "Total RP 3",
        primary_folder: "totalRP3",
        folders: &["totalRP3", "totalRP3_Data"],
        toc: "totalRP3.toc",
        curseforge_url: "https://www.curseforge.com/wow/addons/total-rp-3/files",
        source_url: "https://github.com/Total-RP/Total-RP-3",
    },
    Trp3AddonDefinition {
        id: "total-rp-3-extended",
        name: "Total RP 3: Extended",
        primary_folder: "totalRP3_Extended",
        folders: &[
            "totalRP3_Extended",
            "totalRP3_Extended_ImpExport",
            "totalRP3_Extended_Tools",
        ],
        toc: "totalRP3_Extended.toc",
        curseforge_url: "https://www.curseforge.com/wow/addons/total-rp-3-extended/files",
        source_url: "https://github.com/Total-RP/Total-RP-3-Extended",
    },
];

/// 获取插件安装路径
/// wow_path 可能是:
/// - WTF 目录: D:\World of Warcraft\_retail_\WTF
/// - 版本目录: D:\World of Warcraft\_retail_
pub fn get_addon_path(wow_path: &str, _flavor: &str) -> PathBuf {
    get_version_dir(wow_path)
        .join("Interface")
        .join("AddOns")
        .join("RPBox_Addon")
}

fn get_version_dir(wow_path: &str) -> PathBuf {
    if let Some(wtf_path) = wow_path::normalize_wow_path(wow_path) {
        return wtf_path
            .parent()
            .map(|p| p.to_path_buf())
            .unwrap_or(wtf_path);
    }

    let raw_path = PathBuf::from(wow_path);
    if raw_path.ends_with("WTF") {
        return raw_path
            .parent()
            .map(|p| p.to_path_buf())
            .unwrap_or(raw_path);
    }
    if raw_path.ends_with("Account") {
        return raw_path
            .parent()
            .and_then(|wtf| wtf.parent())
            .map(|p| p.to_path_buf())
            .unwrap_or(raw_path);
    }
    raw_path
}

fn find_toc_path(addon_path: &Path, toc_name: &str) -> Option<PathBuf> {
    let direct_path = addon_path.join(toc_name);
    if direct_path.exists() {
        return Some(direct_path);
    }

    let expected = toc_name.to_lowercase();
    let entries = fs::read_dir(addon_path).ok()?;
    for entry in entries.flatten() {
        let path = entry.path();
        if !path.is_file() {
            continue;
        }

        let Some(file_name) = path.file_name().and_then(|name| name.to_str()) else {
            continue;
        };

        if file_name.to_lowercase() == expected {
            return Some(path);
        }
    }

    None
}

/// 检查 TRP3 相关插件是否已安装并读取本地版本。
pub fn check_trp3_addons(wow_path: &str) -> Trp3AddonCheckResult {
    let version_dir = get_version_dir(wow_path);
    let addons_dir = version_dir.join("Interface").join("AddOns");
    let addons = TRP3_ADDONS
        .iter()
        .map(|addon| {
            let addon_path = addons_dir.join(addon.primary_folder);

            if !addon_path.exists() {
                return Trp3AddonInfo {
                    id: addon.id.to_string(),
                    name: addon.name.to_string(),
                    installed: false,
                    latest_version: String::new(),
                    requires_update: true,
                    version: None,
                    path: None,
                    curseforge_url: addon.curseforge_url.to_string(),
                    source_url: addon.source_url.to_string(),
                    download_url: String::new(),
                };
            }

            let toc_path = find_toc_path(&addon_path, addon.toc);
            let version = toc_path.as_deref().and_then(read_toc_version);
            let installed = toc_path.is_some();

            Trp3AddonInfo {
                id: addon.id.to_string(),
                name: addon.name.to_string(),
                installed,
                latest_version: String::new(),
                requires_update: !installed,
                version,
                path: Some(addon_path.to_string_lossy().to_string()),
                curseforge_url: addon.curseforge_url.to_string(),
                source_url: addon.source_url.to_string(),
                download_url: String::new(),
            }
        })
        .collect();

    Trp3AddonCheckResult {
        wow_path: version_dir.to_string_lossy().to_string(),
        addons_dir: addons_dir.to_string_lossy().to_string(),
        latest_check_available: false,
        latest_check_note:
            "本地插件检测完成；最新版本与下载地址由 RPBox 后端同步 Total RP GitHub Releases。"
                .to_string(),
        addons,
    }
}

/// 下载并安装指定 TRP3 插件包。
pub fn install_trp3_addon(
    wow_path: &str,
    addon_id: &str,
    download_url: &str,
) -> Result<Trp3AddonCheckResult, String> {
    install_trp3_addon_with_progress(wow_path, addon_id, download_url, |_| {})
}

/// 下载并安装指定 TRP3 插件包，并回传下载/写入阶段进度。
pub fn install_trp3_addon_with_progress<F>(
    wow_path: &str,
    addon_id: &str,
    download_url: &str,
    mut progress: F,
) -> Result<Trp3AddonCheckResult, String>
where
    F: FnMut(AddonInstallProgress),
{
    let addon = TRP3_ADDONS
        .iter()
        .find(|addon| addon.id == addon_id)
        .ok_or_else(|| format!("未知的 TRP3 插件: {}", addon_id))?;

    let url = download_url.trim();
    if url.is_empty() {
        return Err("缺少 TRP3 插件下载地址，请先从 RPBox 后端获取 GitHub 元数据".to_string());
    }
    validate_trp3_download_url(url, addon)?;

    let zip_data = download_zip_with_progress(url, addon.id, &mut progress)?;
    emit_progress(
        &mut progress,
        addon.id,
        "写入中",
        "正在解压到 AddOns 目录",
        Some(100),
        zip_data.len() as u64,
        Some(zip_data.len() as u64),
    );

    let version_dir = get_version_dir(wow_path);
    let addons_dir = version_dir.join("Interface").join("AddOns");
    fs::create_dir_all(&addons_dir).map_err(|e| format!("创建 AddOns 目录失败: {}", e))?;

    extract_trp3_zip(&zip_data, &addons_dir, addon)?;
    verify_extracted_addon(&addons_dir, addon)?;
    emit_progress(
        &mut progress,
        addon.id,
        "检测中",
        "正在确认本地插件版本",
        Some(100),
        zip_data.len() as u64,
        Some(zip_data.len() as u64),
    );

    Ok(check_trp3_addons(wow_path))
}

/// 安装前端已经下载好的 TRP3 插件包。
pub fn install_trp3_addon_zip_data(
    wow_path: &str,
    addon_id: &str,
    zip_data: &[u8],
) -> Result<Trp3AddonCheckResult, String> {
    let addon = TRP3_ADDONS
        .iter()
        .find(|addon| addon.id == addon_id)
        .ok_or_else(|| format!("未知的 TRP3 插件: {}", addon_id))?;

    let version_dir = get_version_dir(wow_path);
    let addons_dir = version_dir.join("Interface").join("AddOns");
    fs::create_dir_all(&addons_dir).map_err(|e| format!("创建 AddOns 目录失败: {}", e))?;

    extract_trp3_zip(&zip_data, &addons_dir, addon)?;
    verify_extracted_addon(&addons_dir, addon)?;

    Ok(check_trp3_addons(wow_path))
}

/// 下载并安装所有 TRP3 必需插件包。
pub fn install_all_trp3_addons(
    wow_path: &str,
    addons: &[Trp3AddonInstallRequest],
) -> Result<Trp3AddonCheckResult, String> {
    for addon in TRP3_ADDONS {
        let download_url = addons
            .iter()
            .find(|request| request.addon_id == addon.id)
            .map(|request| request.download_url.as_str())
            .ok_or_else(|| format!("缺少 {} 的下载地址", addon.name))?;
        install_trp3_addon(wow_path, addon.id, download_url)?;
    }
    Ok(check_trp3_addons(wow_path))
}

/// 卸载指定 TRP3 插件目录；只删除 AddOns 下的插件本体，不触碰 WTF/SavedVariables。
pub fn uninstall_trp3_addon(
    wow_path: &str,
    addon_id: &str,
) -> Result<Trp3AddonCheckResult, String> {
    let addon = TRP3_ADDONS
        .iter()
        .find(|addon| addon.id == addon_id)
        .ok_or_else(|| format!("未知的 TRP3 插件: {}", addon_id))?;

    let version_dir = get_version_dir(wow_path);
    let addons_dir = version_dir.join("Interface").join("AddOns");

    for folder in addon.folders {
        let addon_path = addons_dir.join(folder);
        if !addon_path.exists() {
            continue;
        }
        if addon_path.is_dir() {
            fs::remove_dir_all(&addon_path)
                .map_err(|e| format!("删除 {} 失败: {}", folder, e))?;
        } else {
            fs::remove_file(&addon_path)
                .map_err(|e| format!("删除 {} 失败: {}", folder, e))?;
        }
    }

    Ok(check_trp3_addons(wow_path))
}

fn validate_trp3_download_url(url: &str, addon: &Trp3AddonDefinition) -> Result<(), String> {
    let parsed = reqwest::Url::parse(url).map_err(|_| "下载地址格式无效".to_string())?;

    if looks_like_rpbox_trp3_download(&parsed, addon) {
        return Ok(());
    }

    if parsed.scheme() != "https" {
        return Err("TRP3 插件下载必须使用 HTTPS，开发环境仅允许本机 RPBox 后端地址".to_string());
    }

    if !looks_like_total_rp_github_download(url, addon) {
        return Err("仅允许安装 Total RP 官方 GitHub Release 或其镜像下载地址".to_string());
    }

    let lowered = decode_url_repeated(url);
    if !lowered.contains(".zip")
        && !lowered.contains("/zipball/")
        && !lowered.contains("/repos/total-rp/")
    {
        return Err("TRP3 插件下载地址必须指向 zip 包或 GitHub zipball".to_string());
    }

    Ok(())
}

fn looks_like_rpbox_trp3_download(parsed: &reqwest::Url, addon: &Trp3AddonDefinition) -> bool {
    let host = parsed.host_str().unwrap_or("").to_ascii_lowercase();
    let is_local = matches!(host.as_str(), "localhost" | "127.0.0.1" | "::1");
    let is_trusted_api = is_local
        || host == "api.rpbox.app"
        || host == "ksxvodevhonx.sealosbja.site"
        || host == "api.totalrpbox.com"
        || host == "totalrpbox.com"
        || host == "www.totalrpbox.com";
    if !is_trusted_api {
        return false;
    }

    let scheme_ok = parsed.scheme() == "https" || (parsed.scheme() == "http" && is_local);
    if !scheme_ok {
        return false;
    }

    let expected_path = format!("/api/v1/addon/trp3/download/{}", addon.id);
    parsed.path().eq_ignore_ascii_case(&expected_path)
}

fn looks_like_total_rp_github_download(url: &str, addon: &Trp3AddonDefinition) -> bool {
    let lowered = decode_url_repeated(url);
    match addon.id {
        "total-rp-3" => {
            lowered.contains("github.com/total-rp/total-rp-3/releases/download/")
                || lowered.contains("api.github.com/repos/total-rp/total-rp-3/zipball/")
                || lowered.contains("codeload.github.com/total-rp/total-rp-3/zip/")
        }
        "total-rp-3-extended" => {
            lowered.contains("github.com/total-rp/total-rp-3-extended/releases/download/")
                || lowered.contains("api.github.com/repos/total-rp/total-rp-3-extended/zipball/")
                || lowered.contains("codeload.github.com/total-rp/total-rp-3-extended/zip/")
        }
        _ => false,
    }
}

fn decode_url_repeated(value: &str) -> String {
    let mut decoded = value.to_string();
    for _ in 0..4 {
        let next = percent_decode_ascii(&decoded);
        if next == decoded {
            break;
        }
        decoded = next;
    }
    decoded.to_ascii_lowercase()
}

fn percent_decode_ascii(value: &str) -> String {
    let bytes = value.as_bytes();
    let mut output = String::with_capacity(value.len());
    let mut index = 0;
    while index < bytes.len() {
        if bytes[index] == b'%' && index + 2 < bytes.len() {
            if let Ok(hex) = std::str::from_utf8(&bytes[index + 1..index + 3]) {
                if let Ok(decoded) = u8::from_str_radix(hex, 16) {
                    output.push(decoded as char);
                    index += 3;
                    continue;
                }
            }
        }
        output.push(bytes[index] as char);
        index += 1;
    }
    output
}

fn verify_extracted_addon(addons_dir: &Path, addon: &Trp3AddonDefinition) -> Result<(), String> {
    let addon_path = addons_dir.join(addon.primary_folder);
    if !addon_path.exists() {
        return Err(format!("插件包中未找到 {}", addon.primary_folder));
    }
    if find_toc_path(&addon_path, addon.toc).is_none() {
        return Err(format!("插件包中未找到 {}", addon.toc));
    }
    Ok(())
}

fn extract_trp3_zip(data: &[u8], dest: &Path, addon: &Trp3AddonDefinition) -> Result<(), String> {
    let cursor = std::io::Cursor::new(data);
    let mut archive = zip::ZipArchive::new(cursor).map_err(|e| format!("打开 zip 失败: {}", e))?;
    let mut extracted = 0usize;

    for i in 0..archive.len() {
        let mut file = archive
            .by_index(i)
            .map_err(|e| format!("读取 zip 条目失败: {}", e))?;

        let Some(relative_path) = trp3_addon_relative_path(file.enclosed_name(), addon) else {
            continue;
        };
        let outpath = dest.join(relative_path);

        if file.name().ends_with('/') {
            fs::create_dir_all(&outpath).map_err(|e| format!("创建目录失败: {}", e))?;
        } else {
            if let Some(p) = outpath.parent() {
                if !p.exists() {
                    fs::create_dir_all(p).map_err(|e| format!("创建目录失败: {}", e))?;
                }
            }
            let mut outfile =
                fs::File::create(&outpath).map_err(|e| format!("创建文件失败: {}", e))?;
            io::copy(&mut file, &mut outfile).map_err(|e| format!("写入文件失败: {}", e))?;
            extracted += 1;
        }
    }

    if extracted == 0 {
        return Err("插件包中没有可安装的 Total RP 插件文件".to_string());
    }

    Ok(())
}

fn trp3_addon_relative_path(
    enclosed_name: Option<&Path>,
    addon: &Trp3AddonDefinition,
) -> Option<PathBuf> {
    let path = enclosed_name?;
    let components = path
        .components()
        .map(|component| component.as_os_str().to_string_lossy().to_string())
        .collect::<Vec<_>>();

    if components.is_empty() {
        return None;
    }

    let addon_index = components
        .iter()
        .position(|component| addon.folders.iter().any(|folder| component == folder))?;

    let mut relative = PathBuf::new();
    for component in components.iter().skip(addon_index) {
        relative.push(component);
    }
    Some(relative)
}

fn download_zip_with_progress<F>(
    url: &str,
    plugin_id: &str,
    progress: &mut F,
) -> Result<Vec<u8>, String>
where
    F: FnMut(AddonInstallProgress),
{
    emit_progress(
        progress,
        plugin_id,
        "连接中",
        "正在连接下载源",
        None,
        0,
        None,
    );

    let client = reqwest::blocking::Client::builder()
        .user_agent("RPBox/0.2 TRP3 GitHub addon installer")
        .connect_timeout(Duration::from_secs(15))
        .timeout(Duration::from_secs(300))
        .build()
        .map_err(|e| format!("初始化下载客户端失败: {}", e))?;

    let mut response = client
        .get(url)
        .send()
        .map_err(|e| format!("下载插件失败: {}", e))?;

    if !response.status().is_success() {
        return Err(format!("下载插件失败，HTTP 状态: {}", response.status()));
    }

    let total_bytes: Option<u64> = response.content_length();
    let mut downloaded_bytes = 0u64;
    let mut data = Vec::with_capacity(total_bytes.unwrap_or(0).min(64 * 1024 * 1024) as usize);
    let mut buffer = [0u8; 64 * 1024];
    let mut last_percent: Option<u8> = None;
    let mut last_reported_bytes = 0u64;

    loop {
        let read = response
            .read(&mut buffer)
            .map_err(|e| format!("读取插件包失败: {}", e))?;
        if read == 0 {
            break;
        }

        data.extend_from_slice(&buffer[..read]);
        downloaded_bytes += read as u64;

        let percent = total_bytes.map(|total: u64| {
            ((downloaded_bytes.saturating_mul(100) / total.max(1)).min(99)) as u8
        });
        let should_report = percent != last_percent || downloaded_bytes - last_reported_bytes >= 256 * 1024;

        if should_report {
            last_percent = percent;
            last_reported_bytes = downloaded_bytes;
            let label = match percent {
                Some(value) => format!("下载中 {}%", value),
                None => "下载中".to_string(),
            };
            let detail = match total_bytes {
                Some(total) => format!("{} / {}", format_bytes(downloaded_bytes), format_bytes(total)),
                None => format!("已下载 {}", format_bytes(downloaded_bytes)),
            };
            emit_progress(
                progress,
                plugin_id,
                &label,
                &detail,
                percent,
                downloaded_bytes,
                total_bytes,
            );
        }
    }

    emit_progress(
        progress,
        plugin_id,
        "下载完成",
        &match total_bytes {
            Some(total) => format!("{} / {}", format_bytes(downloaded_bytes), format_bytes(total)),
            None => format!("已下载 {}", format_bytes(downloaded_bytes)),
        },
        Some(100),
        downloaded_bytes,
        total_bytes,
    );

    Ok(data)
}

fn emit_progress<F>(
    progress: &mut F,
    plugin_id: &str,
    label: &str,
    detail: &str,
    percent: Option<u8>,
    downloaded_bytes: u64,
    total_bytes: Option<u64>,
) where
    F: FnMut(AddonInstallProgress),
{
    progress(AddonInstallProgress {
        plugin_id: plugin_id.to_string(),
        label: label.to_string(),
        detail: detail.to_string(),
        percent,
        downloaded_bytes,
        total_bytes,
    });
}

fn format_bytes(bytes: u64) -> String {
    const UNITS: [&str; 4] = ["B", "KB", "MB", "GB"];
    if bytes == 0 {
        return "0 B".to_string();
    }

    let mut value = bytes as f64;
    let mut unit_index = 0usize;
    while value >= 1024.0 && unit_index < UNITS.len() - 1 {
        value /= 1024.0;
        unit_index += 1;
    }

    if unit_index == 0 {
        format!("{} {}", bytes, UNITS[unit_index])
    } else {
        format!("{:.1} {}", value, UNITS[unit_index])
    }
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
    let version_dir = get_version_dir(wow_path);
    let addons_dir = version_dir.join("Interface").join("AddOns");

    // 确保 AddOns 目录存在
    fs::create_dir_all(&addons_dir).map_err(|e| format!("创建 AddOns 目录失败: {}", e))?;

    let addon_path = addons_dir.join("RPBox_Addon");

    // 直接解压覆盖（不删除旧文件，避免文件锁定问题）
    extract_zip(zip_data, &addons_dir)?;

    Ok(addon_path.to_string_lossy().to_string())
}

/// 从 URL 下载并安装 RPBox 插件，并回传下载/写入阶段进度。
pub fn install_addon_from_url_with_progress<F>(
    wow_path: &str,
    flavor: &str,
    download_url: &str,
    plugin_id: &str,
    mut progress: F,
) -> Result<String, String>
where
    F: FnMut(AddonInstallProgress),
{
    let url = download_url.trim();
    if url.is_empty() {
        return Err("缺少 RPBox 插件下载地址".to_string());
    }

    let zip_data = download_zip_with_progress(url, plugin_id, &mut progress)?;
    emit_progress(
        &mut progress,
        plugin_id,
        "写入中",
        "正在解压到 AddOns 目录",
        Some(100),
        zip_data.len() as u64,
        Some(zip_data.len() as u64),
    );
    install_addon(wow_path, flavor, &zip_data)
}

/// 解压 zip 文件
fn extract_zip(data: &[u8], dest: &Path) -> Result<(), String> {
    let cursor = std::io::Cursor::new(data);
    let mut archive = zip::ZipArchive::new(cursor).map_err(|e| format!("打开 zip 失败: {}", e))?;

    for i in 0..archive.len() {
        let mut file = archive
            .by_index(i)
            .map_err(|e| format!("读取 zip 条目失败: {}", e))?;

        let outpath = match file.enclosed_name() {
            Some(path) => dest.join(path),
            None => continue,
        };

        if file.name().ends_with('/') {
            fs::create_dir_all(&outpath).map_err(|e| format!("创建目录失败: {}", e))?;
        } else {
            if let Some(p) = outpath.parent() {
                if !p.exists() {
                    fs::create_dir_all(p).map_err(|e| format!("创建目录失败: {}", e))?;
                }
            }
            let mut outfile =
                fs::File::create(&outpath).map_err(|e| format!("创建文件失败: {}", e))?;
            io::copy(&mut file, &mut outfile).map_err(|e| format!("写入文件失败: {}", e))?;
        }
    }

    Ok(())
}

/// 卸载插件
pub fn uninstall_addon(wow_path: &str, flavor: &str) -> Result<(), String> {
    let addon_path = get_addon_path(wow_path, flavor);
    if addon_path.exists() {
        fs::remove_dir_all(&addon_path).map_err(|e| format!("删除插件失败: {}", e))?;
    }
    Ok(())
}
