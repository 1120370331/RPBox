use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{fs, path::PathBuf};

use crate::{lua_parser, wow_path};

/// 递归规范化 JSON，确保对象键按字母顺序排序
fn normalize_json(value: &Value) -> String {
    match value {
        Value::Object(map) => {
            let mut keys: Vec<_> = map.keys().collect();
            keys.sort();
            let pairs: Vec<String> = keys
                .iter()
                .map(|k| format!("\"{}\":{}", k, normalize_json(&map[*k])))
                .collect();
            format!("{{{}}}", pairs.join(","))
        }
        Value::Array(arr) => {
            let items: Vec<String> = arr.iter().map(normalize_json).collect();
            format!("[{}]", items.join(","))
        }
        _ => value.to_string(),
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScanResult {
    pub accounts: Vec<AccountInfo>,
    pub total_profiles: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountInfo {
    pub account_id: String,
    pub realms: Vec<RealmInfo>,
    pub profiles: Vec<ProfileSummary>,
    /// Raw totalRP3.lua content
    pub raw_trp3_lua: Option<String>,
    /// Raw totalRP3_Data.lua content
    pub raw_trp3_data_lua: Option<String>,
    /// Raw totalRP3_Extended.lua content
    pub raw_trp3_extended_lua: Option<String>,
    /// TRP3 Extended 道具数据库
    pub tools_db: Option<ToolsDbSummary>,
    /// TRP3 运行时数据 (他人人物卡等)
    pub runtime_data: Option<RuntimeDataSummary>,
    /// TRP3 配置数据
    pub config: Option<ConfigSummary>,
    /// TRP3 额外数据 (角色绑定、伙伴、预设等)
    pub extra_data: Option<ExtraDataSummary>,
}

/// TRP3 Extended 道具数据库摘要
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolsDbSummary {
    pub item_count: usize,
    pub checksum: String,
    pub raw_data: String,
}

/// TRP3 运行时数据摘要
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuntimeDataSummary {
    pub size_kb: usize,
    pub checksum: String,
    pub raw_data: String,
}

/// TRP3 配置数据摘要
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfigSummary {
    pub checksum: String,
    pub raw_data: String,
}

/// TRP3 额外数据摘要（包含所有其他重要变量）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExtraDataSummary {
    pub checksum: String,
    pub raw_data: String,  // JSON 对象，包含所有额外变量
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RealmInfo {
    pub name: String,
    pub characters: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProfileSummary {
    pub id: String,
    pub name: String,
    pub icon: Option<String>,
    pub checksum: String,
    pub raw_lua: String,
    pub account_id: String,
    pub saved_variables_path: String,
    pub modified_at: DateTime<Utc>,
}

fn read_lua_file(path: &PathBuf) -> Option<String> {
    let content = fs::read(path).ok()?;
    Some(String::from_utf8_lossy(&content).to_string())
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProfileDetail {
    pub id: String,
    pub name: String,
    pub icon: Option<String>,
    pub checksum: String,
    pub raw_lua: String,
    pub account_id: String,
    pub saved_variables_path: String,
    /// 完整的 characteristics 数据 (原始 TRP3 格式)
    pub characteristics: Option<Value>,
    /// 完整的 about 数据 (原始 TRP3 格式)
    pub about: Option<Value>,
    /// 完整的 character 数据 (原始 TRP3 格式)
    pub character: Option<Value>,
}

pub fn scan_profiles(wow_path: &str) -> Result<ScanResult, String> {
    let normalized = wow_path::normalize_wow_path(wow_path)
        .ok_or_else(|| "未找到有效的WoW路径，请选择包含 WTF/Account 的目录".to_string())?;

    let account_path = normalized.join("Account");

    if !account_path.exists() {
        return Err("WTF/Account 目录不存在".to_string());
    }

    let mut accounts = Vec::new();
    let mut total_profiles = 0;

    let entries = std::fs::read_dir(&account_path)
        .map_err(|e| format!("读取目录失败: {}", e))?;

    for entry in entries.flatten() {
        if !entry.path().is_dir() {
            continue;
        }

        let account_id = entry.file_name().to_string_lossy().to_string();
        if account_id == "SavedVariables" {
            continue;
        }

        if let Ok(account_info) = scan_account(&entry.path(), &account_id) {
            total_profiles += account_info.profiles.len();
            accounts.push(account_info);
        }
    }

    Ok(ScanResult {
        accounts,
        total_profiles,
    })
}

fn scan_account(account_path: &PathBuf, account_id: &str) -> Result<AccountInfo, String> {
    let mut realms = Vec::new();
    let mut profiles = Vec::new();
    let sv_dir = account_path.join("SavedVariables");

    // Scan realms (server directories)
    if let Ok(entries) = std::fs::read_dir(account_path) {
        for entry in entries.flatten() {
            let name = entry.file_name().to_string_lossy().to_string();
            if name == "SavedVariables" || !entry.path().is_dir() {
                continue;
            }
            if let Ok(realm_info) = scan_realm(&entry.path(), &name) {
                realms.push(realm_info);
            }
        }
    }

    // Scan profiles from SavedVariables
    let sv_path = sv_dir.join("totalRP3.lua");
    if sv_path.exists() {
        if let Ok(profile_list) = extract_profiles(&sv_path, account_id) {
            profiles = profile_list;
        }
    }

    let raw_trp3_lua = read_lua_file(&sv_dir.join("totalRP3.lua"));
    let raw_trp3_data_lua = read_lua_file(&sv_dir.join("totalRP3_Data.lua"));
    let raw_trp3_extended_lua = read_lua_file(&sv_dir.join("totalRP3_Extended.lua"));

    // Scan TRP3 Extended tools database
    let tools_db = scan_tools_db(account_path);

    // Scan TRP3 runtime data (other players' profiles)
    let runtime_data = scan_runtime_data(account_path);

    // Scan TRP3 configuration
    let config = scan_config(account_path);

    // Scan TRP3 extra data (characters, companions, presets, etc.)
    let extra_data = scan_extra_data(account_path);

    Ok(AccountInfo {
        account_id: account_id.to_string(),
        realms,
        profiles,
        raw_trp3_lua,
        raw_trp3_data_lua,
        raw_trp3_extended_lua,
        tools_db,
        runtime_data,
        config,
        extra_data,
    })
}

fn scan_realm(realm_path: &PathBuf, realm_name: &str) -> Result<RealmInfo, String> {
    let mut characters = Vec::new();

    if let Ok(entries) = std::fs::read_dir(realm_path) {
        for entry in entries.flatten() {
            if entry.path().is_dir() {
                let char_name = entry.file_name().to_string_lossy().to_string();
                characters.push(char_name);
            }
        }
    }

    Ok(RealmInfo {
        name: realm_name.to_string(),
        characters,
    })
}

fn extract_profiles(lua_path: &PathBuf, account_id: &str) -> Result<Vec<ProfileSummary>, String> {
    let content = fs::read(lua_path)
        .map_err(|e| format!("读取文件失败: {}", e))?;
    let content_str = String::from_utf8_lossy(&content);

    let modified_at = fs::metadata(lua_path)
        .and_then(|m| m.modified())
        .map(|t| DateTime::<Utc>::from(t))
        .unwrap_or_else(|_| Utc::now());

    let data = lua_parser::parse_variable(lua_path, "TRP3_Profiles")
        .map_err(|e| e.to_string())?;

    let mut profiles = Vec::new();

    if let Some(obj) = data.as_object() {
        for (id, profile) in obj {
            let name = profile
                .get("profileName")
                .and_then(|v| v.as_str())
                .unwrap_or("未命名")
                .to_string();

            let icon = profile
                .get("player")
                .and_then(|p| p.get("characteristics"))
                .and_then(|c| c.get("IC"))
                .and_then(|v| v.as_str())
                .map(|s| s.to_string());

            let raw_profile = serde_json::to_string(profile)
                .unwrap_or_else(|_| content_str.to_string());
            // 使用规范化 JSON 计算 checksum，确保键顺序稳定
            let normalized = normalize_json(profile);
            let checksum = format!("{:x}", md5::compute(normalized.as_bytes()));
            let sv_path = lua_path.display().to_string();

            profiles.push(ProfileSummary {
                id: id.clone(),
                name,
                icon,
                checksum,
                raw_lua: raw_profile,
                account_id: account_id.to_string(),
                saved_variables_path: sv_path,
                modified_at,
            });
        }
    }

    Ok(profiles)
}

pub fn get_profile_detail(wow_path: &str, profile_id: &str) -> Result<ProfileDetail, String> {
    let normalized = wow_path::normalize_wow_path(wow_path)
        .ok_or_else(|| "未找到有效的WoW路径，请选择包含 WTF/Account 的目录".to_string())?;
    let account_root = normalized.join("Account");
    if !account_root.exists() {
        return Err("WTF/Account 目录不存在".to_string());
    }

    let entries = fs::read_dir(&account_root)
        .map_err(|e| format!("读取目录失败: {}", e))?;

    for entry in entries.flatten() {
        if !entry.path().is_dir() {
            continue;
        }
        let name = entry.file_name().to_string_lossy().to_string();
        if name == "SavedVariables" {
            continue;
        }

        let lua_path = entry.path().join("SavedVariables").join("totalRP3.lua");
        if !lua_path.exists() {
            continue;
        }

        if let Ok(detail) = load_profile_detail(&lua_path, profile_id, &name) {
            return Ok(detail);
        }
    }

    Err("未找到指定人物卡".to_string())
}

fn load_profile_detail(lua_path: &PathBuf, profile_id: &str, account_id: &str) -> Result<ProfileDetail, String> {
    let data = lua_parser::parse_variable(lua_path, "TRP3_Profiles")
        .map_err(|e| e.to_string())?;
    let obj = data
        .as_object()
        .ok_or_else(|| "TRP3_Profiles 数据格式错误".to_string())?;

    let profile = obj
        .get(profile_id)
        .ok_or_else(|| "未找到指定人物卡".to_string())?;

    let raw_profile = serde_json::to_string(profile).unwrap_or_default();
    let normalized = normalize_json(profile);
    let checksum = format!("{:x}", md5::compute(normalized.as_bytes()));

    let name = profile
        .get("profileName")
        .and_then(|v| v.as_str())
        .unwrap_or("未命名")
        .to_string();

    let player = profile.get("player");

    let icon = player
        .and_then(|p| p.get("characteristics"))
        .and_then(|c| c.get("IC"))
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());

    // 直接提取完整的原始数据
    let characteristics = player
        .and_then(|p| p.get("characteristics"))
        .cloned();

    let about = player
        .and_then(|p| p.get("about"))
        .cloned();

    let character = player
        .and_then(|p| p.get("character"))
        .cloned();

    let sv_path = lua_path.display().to_string();

    Ok(ProfileDetail {
        id: profile_id.to_string(),
        name,
        icon,
        checksum,
        raw_lua: raw_profile,
        account_id: account_id.to_string(),
        saved_variables_path: sv_path,
        characteristics,
        about,
        character,
    })
}

/// 扫描 TRP3 Extended 道具数据库
fn scan_tools_db(account_path: &PathBuf) -> Option<ToolsDbSummary> {
    let tools_path = account_path
        .join("SavedVariables")
        .join("totalRP3_Extended.lua");

    if !tools_path.exists() {
        return None;
    }

    let mut data = lua_parser::parse_variable(&tools_path, "TRP3_Tools_DB").ok()?;

    // 合并 TRP3_Exchange_DB 的数据（来自其他玩家的道具）
    if let Ok(exchange_data) = lua_parser::parse_variable(&tools_path, "TRP3_Exchange_DB") {
        if let (Some(tools_map), Some(exchange_map)) = (data.as_object_mut(), exchange_data.as_object()) {
            for (key, value) in exchange_map {
                tools_map.insert(key.clone(), value.clone());
            }
        }
    }

    // 计算道具数量
    let item_count = data.as_object().map(|obj| obj.len()).unwrap_or(0);

    // 计算 checksum
    let normalized = normalize_json(&data);
    let checksum = format!("{:x}", md5::compute(normalized.as_bytes()));

    // 序列化为 JSON
    let raw_data = serde_json::to_string(&data).unwrap_or_default();

    Some(ToolsDbSummary {
        item_count,
        checksum,
        raw_data,
    })
}

/// 扫描 TRP3 运行时数据 (他人人物卡等)
fn scan_runtime_data(account_path: &PathBuf) -> Option<RuntimeDataSummary> {
    let data_path = account_path
        .join("SavedVariables")
        .join("totalRP3_Data.lua");

    if !data_path.exists() {
        return None;
    }

    // 读取文件内容计算大小和 checksum
    let content = fs::read(&data_path).ok()?;
    let size_kb = content.len() / 1024;
    let checksum = format!("{:x}", md5::compute(&content));

    // 解析为 JSON (可能很大，但我们需要完整数据用于备份)
    let data = lua_parser::parse_variable(&data_path, "TRP3_Register").ok()?;
    let raw_data = serde_json::to_string(&data).unwrap_or_default();

    Some(RuntimeDataSummary {
        size_kb,
        checksum,
        raw_data,
    })
}

/// 扫描 TRP3 配置数据
fn scan_config(account_path: &PathBuf) -> Option<ConfigSummary> {
    let config_path = account_path
        .join("SavedVariables")
        .join("totalRP3.lua");

    if !config_path.exists() {
        return None;
    }

    let data = lua_parser::parse_variable(&config_path, "TRP3_Configuration").ok()?;

    // 计算 checksum
    let normalized = normalize_json(&data);
    let checksum = format!("{:x}", md5::compute(normalized.as_bytes()));

    // 序列化为 JSON
    let raw_data = serde_json::to_string(&data).unwrap_or_default();

    Some(ConfigSummary {
        checksum,
        raw_data,
    })
}

/// 扫描 TRP3 额外数据（角色绑定、伙伴、预设等）
fn scan_extra_data(account_path: &PathBuf) -> Option<ExtraDataSummary> {
    let sv_dir = account_path.join("SavedVariables");
    let trp3_path = sv_dir.join("totalRP3.lua");
    let extended_path = sv_dir.join("totalRP3_Extended.lua");

    let mut extra = serde_json::Map::new();

    // 从 totalRP3.lua 提取额外变量
    if trp3_path.exists() {
        // TRP3_Characters - 角色绑定（最重要）
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_Characters") {
            extra.insert("TRP3_Characters".to_string(), data);
        }
        // TRP3_Companions - 伙伴数据
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_Companions") {
            extra.insert("TRP3_Companions".to_string(), data);
        }
        // TRP3_Presets - 预设
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_Presets") {
            extra.insert("TRP3_Presets".to_string(), data);
        }
        // TRP3_Notes - 笔记
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_Notes") {
            extra.insert("TRP3_Notes".to_string(), data);
        }
        // TRP3_Flyway - 迁移记录
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_Flyway") {
            extra.insert("TRP3_Flyway".to_string(), data);
        }
        // TRP3_MatureFilter - 成人过滤
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_MatureFilter") {
            extra.insert("TRP3_MatureFilter".to_string(), data);
        }
        // TRP3_Colors - 颜色
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_Colors") {
            extra.insert("TRP3_Colors".to_string(), data);
        }
        // TRP3_SavedAutomation - 自动化
        if let Ok(data) = lua_parser::parse_variable(&trp3_path, "TRP3_SavedAutomation") {
            extra.insert("TRP3_SavedAutomation".to_string(), data);
        }
    }

    // 从 totalRP3_Extended.lua 提取额外变量
    if extended_path.exists() {
        // TRP3_Exchange_DB - 交易所（来自其他玩家的数据）
        if let Ok(data) = lua_parser::parse_variable(&extended_path, "TRP3_Exchange_DB") {
            extra.insert("TRP3_Exchange_DB".to_string(), data);
        }
        // TRP3_Stashes - 隐藏物品
        if let Ok(data) = lua_parser::parse_variable(&extended_path, "TRP3_Stashes") {
            extra.insert("TRP3_Stashes".to_string(), data);
        }
        // TRP3_Drop - 掉落
        if let Ok(data) = lua_parser::parse_variable(&extended_path, "TRP3_Drop") {
            extra.insert("TRP3_Drop".to_string(), data);
        }
        // TRP3_Security - 安全
        if let Ok(data) = lua_parser::parse_variable(&extended_path, "TRP3_Security") {
            extra.insert("TRP3_Security".to_string(), data);
        }
        // TRP3_Extended_Flyway - Extended迁移
        if let Ok(data) = lua_parser::parse_variable(&extended_path, "TRP3_Extended_Flyway") {
            extra.insert("TRP3_Extended_Flyway".to_string(), data);
        }
    }

    if extra.is_empty() {
        return None;
    }

    let value = Value::Object(extra);
    let normalized = normalize_json(&value);
    let checksum = format!("{:x}", md5::compute(normalized.as_bytes()));
    let raw_data = serde_json::to_string(&value).unwrap_or_default();

    Some(ExtraDataSummary { checksum, raw_data })
}
