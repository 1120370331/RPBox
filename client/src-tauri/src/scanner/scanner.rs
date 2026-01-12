use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

use crate::lua_parser;

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
    pub modified_at: DateTime<Utc>,
}

pub fn scan_profiles(wow_path: &str) -> Result<ScanResult, String> {
    let base_path = PathBuf::from(wow_path);
    let account_path = base_path.join("WTF").join("Account");

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
    let sv_path = account_path.join("SavedVariables").join("totalRP3.lua");
    if sv_path.exists() {
        if let Ok(profile_list) = extract_profiles(&sv_path) {
            profiles = profile_list;
        }
    }

    Ok(AccountInfo {
        account_id: account_id.to_string(),
        realms,
        profiles,
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

fn extract_profiles(lua_path: &PathBuf) -> Result<Vec<ProfileSummary>, String> {
    let content = std::fs::read(lua_path)
        .map_err(|e| format!("读取文件失败: {}", e))?;

    let checksum = format!("{:x}", md5::compute(&content));
    let content_str = String::from_utf8_lossy(&content);

    let modified_at = std::fs::metadata(lua_path)
        .and_then(|m| m.modified())
        .map(|t| DateTime::<Utc>::from(t))
        .unwrap_or_else(|_| Utc::now());

    let data = lua_parser::parse_variable(lua_path, "TRP3_Profiles")
        .map_err(|e| e.to_string())?;

    let mut profiles = Vec::new();

    if let Some(obj) = data.as_object() {
        for (id, profile) in obj {
            let name = profile["profileName"]
                .as_str()
                .unwrap_or("未命名")
                .to_string();

            let icon = profile["player"]["characteristics"]["IC"]
                .as_str()
                .map(|s| s.to_string());

            profiles.push(ProfileSummary {
                id: id.clone(),
                name,
                icon,
                checksum: checksum.clone(),
                modified_at,
            });
        }
    }

    Ok(profiles)
}
