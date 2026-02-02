mod lua_parser;
mod wow_path;
mod scanner;
mod sync_meta;
mod writer;
mod addon_installer;
mod chat_log;

use std::path::{Path, PathBuf};
use serde_json::Value;
use tauri::Manager;
use crate::writer::replace_trp3_profiles;

#[tauri::command]
async fn parse_trp3_file(path: String, variable: String) -> Result<serde_json::Value, String> {
    lua_parser::parse_variable(Path::new(&path), &variable)
        .map_err(|e| e.to_string())
}

#[tauri::command]
async fn detect_wow_paths() -> Vec<wow_path::WowInstallation> {
    wow_path::detect_wow_paths()
}

#[tauri::command]
async fn validate_wow_path(path: String) -> bool {
    wow_path::validate_wow_path(&path)
}

#[tauri::command]
async fn normalize_wow_path(path: String) -> Option<String> {
    wow_path::normalize_wow_path(&path).map(|p| p.to_string_lossy().to_string())
}

#[tauri::command]
async fn scan_profiles(wow_path: String) -> Result<scanner::ScanResult, String> {
    scanner::scan_profiles(&wow_path)
}

#[tauri::command]
async fn get_profile_detail(
    wow_path: String,
    profile_id: String,
) -> Result<scanner::ProfileDetail, String> {
    scanner::get_profile_detail(&wow_path, &profile_id)
}

#[tauri::command]
async fn is_wow_running() -> bool {
    writer::is_wow_running()
}

#[tauri::command]
async fn write_profile(path: String, raw_lua: String) -> Result<(), String> {
    let path = std::path::PathBuf::from(path);
    writer::write_profile_to_local(&path, &raw_lua)
        .map_err(|e| e.to_string())
}

#[tauri::command]
async fn update_profile(
    wow_path: String,
    profile_id: String,
    updates: Value,
) -> Result<(), String> {
    let (lua_path, mut profiles) = find_profiles_file(&wow_path, &profile_id)?;
    let obj = profiles
        .as_object_mut()
        .ok_or_else(|| "TRP3_Profiles 数据格式错误".to_string())?;
    let profile = obj
        .get_mut(&profile_id)
        .ok_or_else(|| "未找到指定人物卡".to_string())?;

    apply_updates(profile, &updates)?;

    replace_trp3_profiles(&lua_path, &profiles)
        .map_err(|e| e.to_string())
}

#[tauri::command]
async fn clear_sync_cache(app: tauri::AppHandle) -> Result<(), String> {
    let app_dir = app
        .path()
        .app_data_dir()
        .map_err(|_| "无法定位应用数据目录".to_string())?;
    let db_path = app_dir.join("sync_meta.db");
    if db_path.exists() {
        std::fs::remove_file(&db_path)
            .map_err(|e| format!("清除缓存失败: {}", e))?;
    }
    Ok(())
}

fn find_profiles_file(wow_path: &str, profile_id: &str) -> Result<(PathBuf, Value), String> {
    let normalized = wow_path::normalize_wow_path(wow_path)
        .ok_or_else(|| "未找到有效的WoW路径，请选择包含 WTF/Account 的目录".to_string())?;
    let account_root = normalized.join("Account");
    if !account_root.exists() {
        return Err("WTF/Account 目录不存在".to_string());
    }

    let entries = std::fs::read_dir(&account_root)
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

        let data = lua_parser::parse_variable(&lua_path, "TRP3_Profiles")
            .map_err(|e| e.to_string())?;
        if data
            .as_object()
            .and_then(|obj| obj.get(profile_id))
            .is_some()
        {
            return Ok((lua_path, data));
        }
    }

    Err("未找到指定人物卡".to_string())
}

fn apply_updates(profile: &mut Value, updates: &Value) -> Result<(), String> {
    let player = profile
        .as_object_mut()
        .ok_or_else(|| "人物卡结构错误".to_string())?
        .entry("player")
        .or_insert_with(|| Value::Object(Default::default()));

    let characteristics = player
        .as_object_mut()
        .ok_or_else(|| "人物卡结构错误".to_string())?
        .entry("characteristics")
        .or_insert_with(|| Value::Object(Default::default()));

    if let Some(chars) = updates.get("characteristics") {
        set_str_field(characteristics, &["FN"], chars.get("firstName"));
        set_str_field(characteristics, &["LN"], chars.get("lastName"));
        set_str_field(characteristics, &["TI"], chars.get("title"));
        set_str_field(characteristics, &["RA"], chars.get("race"));
        set_str_field(characteristics, &["CL"], chars.get("class"));
        set_str_field(characteristics, &["AG"], chars.get("age"));
        set_str_field(characteristics, &["EC"], chars.get("eyeColor"));
        set_str_field(characteristics, &["HE"], chars.get("height"));
        set_str_field(characteristics, &["WE"], chars.get("weight"));
    }

    if let Some(about_updates) = updates.get("about") {
        let about = player
            .as_object_mut()
            .ok_or_else(|| "人物卡结构错误".to_string())?
            .entry("about")
            .or_insert_with(|| Value::Object(Default::default()));

        // 使用模板1写入自由文本，保持兼容
        about
            .as_object_mut()
            .ok_or_else(|| "人物卡结构错误".to_string())?
            .insert("TE".to_string(), Value::from(1));

        let t1 = about
            .as_object_mut()
            .ok_or_else(|| "人物卡结构错误".to_string())?
            .entry("T1")
            .or_insert_with(|| Value::Object(Default::default()));

        if let Some(text) = about_updates.get("text").and_then(|v| v.as_str()) {
            t1.as_object_mut()
                .ok_or_else(|| "人物卡结构错误".to_string())?
                .insert("TX".to_string(), Value::from(text));
        }
        if let Some(title) = about_updates.get("title").and_then(|v| v.as_str()) {
            profile
                .as_object_mut()
                .ok_or_else(|| "人物卡结构错误".to_string())?
                .insert("profileName".to_string(), Value::from(title));
        }
    }

    Ok(())
}

fn set_str_field(target: &mut Value, path: &[&str], value: Option<&Value>) {
    let text = value.and_then(|v| v.as_str()).unwrap_or("").to_string();
    if text.is_empty() {
        return;
    }

    let mut current = target;
    for key in path.iter().take(path.len().saturating_sub(1)) {
        if !current.is_object() {
            *current = Value::Object(Default::default());
        }
        current = current
            .as_object_mut()
            .unwrap()
            .entry(key.to_string())
            .or_insert_with(|| Value::Object(Default::default()));
    }

    if let Some(last) = path.last() {
        if !current.is_object() {
            *current = Value::Object(Default::default());
        }
        current
            .as_object_mut()
            .unwrap()
            .insert(last.to_string(), Value::from(text));
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            parse_trp3_file,
            detect_wow_paths,
            validate_wow_path,
            normalize_wow_path,
            scan_profiles,
            get_profile_detail,
            is_wow_running,
            write_profile,
            update_profile,
            clear_sync_cache,
            apply_cloud_profile,
            apply_account_backup,
            check_addon_installed,
            install_addon,
            uninstall_addon,
            scan_chat_logs,
            save_text_file,
            save_binary_file
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

#[tauri::command]
async fn apply_cloud_profile(
    wow_path: String,
    account_id: String,
    profile_id: String,
    profile_json: String,
) -> Result<(), String> {
    let normalized = wow_path::normalize_wow_path(&wow_path)
        .ok_or_else(|| "未找到有效的WoW路径，请选择包含 WTF/Account 的目录".to_string())?;
    let sv_path = normalized
        .join("Account")
        .join(account_id)
        .join("SavedVariables")
        .join("totalRP3.lua");

    let mut profiles = lua_parser::parse_variable(&sv_path, "TRP3_Profiles")
        .map_err(|e| e.to_string())?;

    let profile_value: Value = serde_json::from_str(&profile_json)
        .map_err(|e| format!("云端数据解析失败: {}", e))?;

    profiles
        .as_object_mut()
        .ok_or_else(|| "TRP3_Profiles 数据格式错误".to_string())?
        .insert(profile_id, profile_value);

    replace_trp3_profiles(&sv_path, &profiles)
        .map_err(|e| e.to_string())
}

#[tauri::command]
async fn apply_account_backup(
    wow_path: String,
    account_id: String,
    profiles_json: String,
    tools_json: Option<String>,
    runtime_json: Option<String>,
    config_json: Option<String>,
    extra_json: Option<String>,
    raw_trp3_lua: Option<String>,
    raw_trp3_data_lua: Option<String>,
    raw_trp3_extended_lua: Option<String>,
) -> Result<(), String> {
    let normalized = wow_path::normalize_wow_path(&wow_path)
        .ok_or_else(|| "未找到有效的WoW路径，请选择包含 WTF/Account 的目录".to_string())?;
    let sv_dir = normalized
        .join("Account")
        .join(&account_id)
        .join("SavedVariables");
    let sv_path = sv_dir.join("totalRP3.lua");
    let data_path = sv_dir.join("totalRP3_Data.lua");
    let extended_path = sv_dir.join("totalRP3_Extended.lua");

    let use_raw_trp3 = raw_trp3_lua.as_ref().map(|s| !s.is_empty()).unwrap_or(false);
    let use_raw_data = raw_trp3_data_lua.as_ref().map(|s| !s.is_empty()).unwrap_or(false);
    let use_raw_extended = raw_trp3_extended_lua.as_ref().map(|s| !s.is_empty()).unwrap_or(false);

    if use_raw_trp3 {
        if let Some(raw) = raw_trp3_lua {
            writer::write_profile_to_local(&sv_path, &raw)
                .map_err(|e| e.to_string())?;
        }
    } else {
        // 解析云端备份的所有 profiles
        let cloud_profiles: serde_json::Map<String, Value> = serde_json::from_str(&profiles_json)
            .map_err(|e| format!("云端数据解析失败: {}", e))?;

        // 如果本地文件存在，读取并合并；否则直接使用云端数据
        let final_profiles = if sv_path.exists() {
            let mut local_profiles = lua_parser::parse_variable(&sv_path, "TRP3_Profiles")
                .map_err(|e| e.to_string())?;

            let local_map = local_profiles
                .as_object_mut()
                .ok_or_else(|| "TRP3_Profiles 数据格式错误".to_string())?;

            // 将云端的所有 profiles 合并到本地（覆盖同名的）
            for (profile_id, profile_data) in cloud_profiles {
                local_map.insert(profile_id, profile_data);
            }
            local_profiles
        } else {
            // 文件不存在，直接使用云端数据
            Value::Object(cloud_profiles)
        };

        replace_trp3_profiles(&sv_path, &final_profiles)
            .map_err(|e| e.to_string())?;
    }

    if use_raw_extended {
        if let Some(raw) = raw_trp3_extended_lua {
            writer::write_profile_to_local(&extended_path, &raw)
                .map_err(|e| e.to_string())?;
        }
    } else {
        // 写回道具数据库（如果有）
        if let Some(tools_data) = tools_json {
            if !tools_data.is_empty() {
                let tools_value: Value = serde_json::from_str(&tools_data)
                    .map_err(|e| format!("道具数据解析失败: {}", e))?;
                writer::write_tools_db(&sv_dir, &tools_value)
                    .map_err(|e| e.to_string())?;
            }
        }
    }

    if use_raw_data {
        if let Some(raw) = raw_trp3_data_lua {
            writer::write_profile_to_local(&data_path, &raw)
                .map_err(|e| e.to_string())?;
        }
    } else {
        // 写回运行时数据（如果有）
        if let Some(runtime_data) = runtime_json {
            if !runtime_data.is_empty() {
                let runtime_value: Value = serde_json::from_str(&runtime_data)
                    .map_err(|e| format!("运行时数据解析失败: {}", e))?;
                writer::write_runtime_data(&sv_dir, &runtime_value)
                    .map_err(|e| e.to_string())?;
            }
        }
    }

    if !use_raw_trp3 {
        // 写回配置数据（如果有）
        if let Some(config_data) = config_json {
            if !config_data.is_empty() {
                let config_value: Value = serde_json::from_str(&config_data)
                    .map_err(|e| format!("配置数据解析失败: {}", e))?;
                writer::write_config(&sv_path, &config_value)
                    .map_err(|e| e.to_string())?;
            }
        }
    }

    if let Some(extra_data) = extra_json {
        if !extra_data.is_empty() && (!use_raw_trp3 || !use_raw_extended) {
            let extra_value: Value = serde_json::from_str(&extra_data)
                .map_err(|e| format!("额外数据解析失败: {}", e))?;
            writer::write_extra_data(&sv_dir, &extra_value, !use_raw_trp3, !use_raw_extended)
                .map_err(|e| e.to_string())?;
        }
    }

    Ok(())
}

#[tauri::command]
async fn check_addon_installed(wow_path: String, flavor: String) -> addon_installer::InstalledAddonInfo {
    addon_installer::check_addon_installed(&wow_path, &flavor)
}

#[tauri::command]
async fn install_addon(wow_path: String, flavor: String, zip_data: Vec<u8>) -> Result<String, String> {
    addon_installer::install_addon(&wow_path, &flavor, &zip_data)
}

#[tauri::command]
async fn uninstall_addon(wow_path: String, flavor: String) -> Result<(), String> {
    addon_installer::uninstall_addon(&wow_path, &flavor)
}

#[tauri::command]
async fn scan_chat_logs(wow_path: String) -> Result<Vec<chat_log::AccountChatLogs>, String> {
    chat_log::scan_chat_logs(&wow_path)
}

#[tauri::command]
async fn save_text_file(path: String, content: String) -> Result<(), String> {
    std::fs::write(&path, content)
        .map_err(|e| format!("保存文件失败: {}", e))
}

#[tauri::command]
async fn save_binary_file(path: String, data: Vec<u8>) -> Result<(), String> {
    std::fs::write(&path, data)
        .map_err(|e| format!("保存文件失败: {}", e))
}
