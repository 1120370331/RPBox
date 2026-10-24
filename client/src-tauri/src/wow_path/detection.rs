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
    pub flavor: String,
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

const FLAVORS: &[&str] = &[
    "_retail_",
    "_classic_",
    "_classic_era_",
    "_classic_anniversary_",
    "_ptr_",
    "_beta_",
    "_classic_ptr_",
    "_classic_beta_",
];

fn client_flavor(path: &std::path::Path) -> Option<String> {
    let metadata = std::fs::read_to_string(path.join(".build.info")).ok()?;
    let mut lines = metadata.lines();
    let product_column = lines
        .next()?
        .split('|')
        .position(|name| name.starts_with("Product!"))?;
    let mut flavors = Vec::new();
    for line in lines {
        let product = line.split('|').nth(product_column)?.trim();
        let flavor = match product {
            "wow" => "_retail_",
            "wow_classic" => "_classic_",
            "wow_classic_era" => "_classic_era_",
            "wow_classic_anniversary" => "_classic_anniversary_",
            "wowt" => "_ptr_",
            "wow_beta" => "_beta_",
            "wow_classic_ptr" => "_classic_ptr_",
            "wow_classic_beta" => "_classic_beta_",
            _ => continue,
        };
        if !flavors.contains(&flavor) {
            flavors.push(flavor);
        }
    }
    if flavors.len() == 1 {
        Some(flavors[0].to_string())
    } else {
        None
    }
}

// Inspect a client directory or all installed editions under a Battle.net root.
// Folder edition names are preferred; client executables identify renamed installations.
pub fn inspect_wow_installations(input: &str) -> Vec<WowInstallation> {
    let mut base = PathBuf::from(input);
    if base.ends_with("Account") {
        base.pop();
    }
    if base.ends_with("WTF") {
        base.pop();
    }
    let mut candidates = vec![base.clone()];
    candidates.extend(FLAVORS.iter().map(|flavor| base.join(flavor)));
    candidates
        .into_iter()
        .filter_map(|path| {
            if !path.is_dir() {
                return None;
            }
            let name = path.file_name()?.to_str()?;
            let flavor = if FLAVORS.contains(&name) {
                name.to_string()
            } else if let Some(flavor) = client_flavor(&path) {
                flavor
            } else if path.join("Wow.exe").exists() || path.join("World of Warcraft.app").exists() {
                "_retail_".into()
            } else {
                return None;
            };
            if !path.join("WTF").join("Account").is_dir()
                && !path.join("Wow.exe").is_file()
                && !path.join("WowClassic.exe").is_file()
                && !path.join("World of Warcraft.app").exists()
            {
                return None;
            }
            Some(WowInstallation {
                version: if flavor.contains("classic") {
                    WowVersion::Classic
                } else {
                    WowVersion::Retail
                },
                accounts: get_accounts(&path).unwrap_or_default(),
                path,
                flavor,
            })
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn detects_each_installed_edition_and_requires_a_specific_root() {
        let temp = tempfile::tempdir().unwrap();
        for flavor in [
            "_retail_",
            "_classic_",
            "_classic_era_",
            "_classic_anniversary_",
        ] {
            std::fs::create_dir_all(temp.path().join(flavor).join("WTF/Account/WOW1")).unwrap();
        }
        let detected = inspect_wow_installations(&temp.path().to_string_lossy());
        assert_eq!(detected.len(), 4);
        assert!(detected.iter().all(|item| item.accounts == vec!["WOW1"]));
        assert!(normalize_wow_path(&temp.path().to_string_lossy()).is_none());
        assert_eq!(
            normalize_wow_path(&temp.path().join("_classic_era_").to_string_lossy()).unwrap(),
            temp.path().join("_classic_era_/WTF")
        );
    }

    #[test]
    fn detects_renamed_client_from_product_metadata_without_guessing_classic_edition() {
        let temp = tempfile::tempdir().unwrap();
        std::fs::write(temp.path().join("WowClassic.exe"), "").unwrap();
        assert!(inspect_wow_installations(&temp.path().to_string_lossy()).is_empty());
        std::fs::write(
            temp.path().join(".build.info"),
            "Active!DEC:1|Product!STRING:0\n1|wow_classic_era\n",
        )
        .unwrap();
        let detected = inspect_wow_installations(&temp.path().to_string_lossy());
        assert_eq!(detected[0].flavor, "_classic_era_");
    }
}

fn check_wow_installation(base_path: &PathBuf) -> Option<Vec<WowInstallation>> {
    let results = inspect_wow_installations(&base_path.to_string_lossy());
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

    // Resolve roots with exactly one installed client. Multiple clients must be selected explicitly.
    let installs = inspect_wow_installations(&path.to_string_lossy());
    if installs.len() == 1 {
        let wtf = installs[0].path.join("WTF");
        if wtf.join("Account").exists() {
            return Some(wtf);
        }
    }
    None
}
