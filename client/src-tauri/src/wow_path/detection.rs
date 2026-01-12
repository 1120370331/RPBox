use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum WowVersion {
    Retail,
    Classic,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WowInstallation {
    pub path: PathBuf,
    pub version: WowVersion,
    pub accounts: Vec<String>,
}

const COMMON_PATHS: &[&str] = &[
    "C:\\Program Files (x86)\\World of Warcraft",
    "C:\\Program Files\\World of Warcraft",
    "D:\\World of Warcraft",
    "D:\\Games\\World of Warcraft",
    "E:\\World of Warcraft",
    "E:\\Games\\World of Warcraft",
];

pub fn detect_wow_paths() -> Vec<WowInstallation> {
    let mut installations = Vec::new();

    for path_str in COMMON_PATHS {
        let path = PathBuf::from(path_str);
        if let Some(install) = check_wow_installation(&path) {
            installations.extend(install);
        }
    }

    installations
}

fn check_wow_installation(base_path: &PathBuf) -> Option<Vec<WowInstallation>> {
    if !base_path.exists() {
        return None;
    }

    let mut results = Vec::new();

    // Check _retail_
    let retail_path = base_path.join("_retail_");
    if let Some(accounts) = get_accounts(&retail_path) {
        results.push(WowInstallation {
            path: retail_path,
            version: WowVersion::Retail,
            accounts,
        });
    }

    // Check _classic_
    let classic_path = base_path.join("_classic_");
    if let Some(accounts) = get_accounts(&classic_path) {
        results.push(WowInstallation {
            path: classic_path,
            version: WowVersion::Classic,
            accounts,
        });
    }

    if results.is_empty() {
        None
    } else {
        Some(results)
    }
}

fn get_accounts(version_path: &PathBuf) -> Option<Vec<String>> {
    let account_path = version_path.join("WTF").join("Account");
    if !account_path.exists() {
        return None;
    }

    let mut accounts = Vec::new();
    if let Ok(entries) = std::fs::read_dir(&account_path) {
        for entry in entries.flatten() {
            if entry.path().is_dir() {
                if let Some(name) = entry.file_name().to_str() {
                    // Skip SavedVariables directory
                    if name != "SavedVariables" {
                        accounts.push(name.to_string());
                    }
                }
            }
        }
    }

    if accounts.is_empty() {
        None
    } else {
        Some(accounts)
    }
}

pub fn validate_wow_path(path: &str) -> bool {
    normalize_wow_path(path).is_some()
}

/// 智能识别WoW路径，无论用户选择哪个级别的文件夹
/// 支持: WoW根目录, _retail_目录, WTF目录, Account目录
pub fn normalize_wow_path(path: &str) -> Option<PathBuf> {
    let path = PathBuf::from(path);
    if !path.exists() {
        return None;
    }

    // 检查是否是 Account 目录
    if path.ends_with("Account") {
        let wtf = path.parent()?;
        if wtf.ends_with("WTF") {
            return Some(wtf.to_path_buf());
        }
    }

    // 检查是否是 WTF 目录
    if path.ends_with("WTF") {
        let account = path.join("Account");
        if account.exists() {
            return Some(path);
        }
    }

    // 检查是否是 _retail_ 或 _classic_ 目录
    let name = path.file_name()?.to_str()?;
    if name.starts_with("_") && name.ends_with("_") {
        let wtf = path.join("WTF");
        if wtf.join("Account").exists() {
            return Some(wtf);
        }
    }

    // 检查是否是 WoW 根目录
    let retail_wtf = path.join("_retail_").join("WTF");
    if retail_wtf.join("Account").exists() {
        return Some(retail_wtf);
    }

    let classic_wtf = path.join("_classic_").join("WTF");
    if classic_wtf.join("Account").exists() {
        return Some(classic_wtf);
    }

    None
}
