mod addon_installer;
mod chat_log;
mod local_versions;
mod lua_parser;
mod scanner;
mod sync_meta;
mod wow_path;
mod writer;

use crate::writer::replace_trp3_profiles;
use serde_json::Value;
use std::path::{Path, PathBuf};
use std::process::Command;
use tauri::{Emitter, Manager};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_deep_link::DeepLinkExt;

#[tauri::command]
async fn parse_trp3_file(path: String, variable: String) -> Result<serde_json::Value, String> {
    lua_parser::parse_variable(Path::new(&path), &variable).map_err(|e| e.to_string())
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
    writer::write_profile_to_local(&path, &raw_lua).map_err(|e| e.to_string())
}

#[tauri::command]
async fn update_profile(
    wow_path: String,
    account_id: Option<String>,
    profile_id: String,
    updates: Value,
) -> Result<(), String> {
    let (lua_path, mut profiles) =
        find_profiles_file(&wow_path, account_id.as_deref(), &profile_id)?;
    let obj = profiles
        .as_object_mut()
        .ok_or_else(|| "TRP3_Profiles 数据格式错误".to_string())?;
    let profile = obj
        .get_mut(&profile_id)
        .ok_or_else(|| "未找到指定人物卡".to_string())?;

    apply_updates(profile, &updates)?;

    replace_trp3_profiles(&lua_path, &profiles).map_err(|e| e.to_string())
}

#[derive(Debug, serde::Serialize)]
struct CharacterCardProfileWriteResult {
    snapshot: Option<local_versions::LocalTRP3Snapshot>,
}

const CHARACTER_CARD_SHARED_CHARACTERISTICS: &[&str] = &[
    "FN", "LN", "TI", "FT", "RA", "CL", "EC", "EH", "AG", "HE", "WE", "BP", "RE", "RS", "IC", "CH",
];

fn merge_character_card_profile(
    existing: Option<&Value>,
    exported: &Value,
) -> Result<Value, String> {
    let exported_profile = exported
        .as_object()
        .ok_or_else(|| "导出的人物卡结构错误".to_string())?;
    let exported_characteristics = exported_profile
        .get("player")
        .and_then(Value::as_object)
        .and_then(|player| player.get("characteristics"))
        .and_then(Value::as_object)
        .ok_or_else(|| "导出的人物卡缺少 player.characteristics".to_string())?;

    let mut target_profile = match existing {
        Some(Value::Object(profile)) => profile.clone(),
        Some(_) => return Err("本地目标 profile 结构错误".to_string()),
        None => Default::default(),
    };

    match exported_profile
        .get("profileName")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|name| !name.is_empty())
    {
        Some(name) => {
            target_profile.insert("profileName".to_string(), Value::String(name.to_string()));
        }
        None => {
            target_profile.remove("profileName");
        }
    }

    let mut target_player = match target_profile.remove("player") {
        Some(Value::Object(player)) => player,
        Some(_) => return Err("本地目标 profile.player 结构错误".to_string()),
        None => Default::default(),
    };
    let mut target_characteristics = match target_player.remove("characteristics") {
        Some(Value::Object(characteristics)) => characteristics,
        Some(_) => return Err("本地目标 profile.player.characteristics 结构错误".to_string()),
        None => Default::default(),
    };

    for key in CHARACTER_CARD_SHARED_CHARACTERISTICS {
        match exported_characteristics.get(*key) {
            Some(Value::String(value)) if !value.trim().is_empty() => {
                target_characteristics
                    .insert((*key).to_string(), Value::String(value.trim().to_string()));
            }
            Some(value) if !value.is_null() && !value.is_string() => {
                target_characteristics.insert((*key).to_string(), value.clone());
            }
            _ => {
                target_characteristics.remove(*key);
            }
        }
    }

    target_player.insert(
        "characteristics".to_string(),
        Value::Object(target_characteristics),
    );
    target_profile.insert("player".to_string(), Value::Object(target_player));
    Ok(Value::Object(target_profile))
}

#[tauri::command]
async fn write_character_card_profile(
    wow_path: String,
    account_id: String,
    profile_id: String,
    profile: Value,
    snapshot_name: Option<String>,
) -> Result<CharacterCardProfileWriteResult, String> {
    if writer::is_wow_running() {
        return Err("检测到魔兽世界正在运行，请关闭游戏后再写入".to_string());
    }
    let trimmed_profile_id = profile_id.trim();
    if trimmed_profile_id.is_empty()
        || trimmed_profile_id.len() > 128
        || trimmed_profile_id
            .chars()
            .any(|character| character.is_control())
    {
        return Err("TRP3 profile ID 无效".to_string());
    }

    let lua_path = local_versions::resolve_lua_path(&wow_path, &account_id)?;
    if let Some(parent) = lua_path.parent() {
        std::fs::create_dir_all(parent)
            .map_err(|error| format!("创建 SavedVariables 目录失败: {}", error))?;
    }
    let (source, existing_profile) =
        writer::read_trp3_profile_source(&lua_path, trimmed_profile_id)
            .map_err(|error| error.to_string())?;
    let merged_profile =
        merge_character_card_profile(existing_profile.as_ref(), &profile)?;

    let had_local_file = source.is_some();
    let snapshot = local_versions::create_snapshot_with_reason(
        &wow_path,
        &account_id,
        snapshot_name.as_deref(),
        "before_character_card_writeback",
    )?;
    if had_local_file && snapshot.is_none() {
        return Err("写入前无法建立强制本地快照".to_string());
    }
    writer::write_trp3_profile_precisely(
        &lua_path,
        source.as_deref(),
        trimmed_profile_id,
        &merged_profile,
    )
    .map_err(|error| error.to_string())?;
    Ok(CharacterCardProfileWriteResult { snapshot })
}

#[tauri::command]
async fn list_local_trp3_snapshots(
    wow_path: String,
    account_id: String,
) -> Result<Vec<local_versions::LocalTRP3Snapshot>, String> {
    local_versions::list_snapshots(&wow_path, &account_id)
}

#[tauri::command]
async fn read_local_trp3_snapshot(
    wow_path: String,
    account_id: String,
    snapshot_id: String,
) -> Result<local_versions::LocalTRP3SnapshotDetail, String> {
    local_versions::read_snapshot(&wow_path, &account_id, &snapshot_id)
}

#[tauri::command]
async fn rename_local_trp3_snapshot(
    wow_path: String,
    account_id: String,
    snapshot_id: String,
    name: String,
) -> Result<local_versions::LocalTRP3Snapshot, String> {
    local_versions::rename_snapshot(&wow_path, &account_id, &snapshot_id, &name)
}

#[tauri::command]
async fn restore_local_trp3_snapshot(
    wow_path: String,
    account_id: String,
    snapshot_id: String,
    safety_snapshot_name: Option<String>,
) -> Result<local_versions::LocalTRP3RestoreResult, String> {
    local_versions::restore_snapshot(
        &wow_path,
        &account_id,
        &snapshot_id,
        safety_snapshot_name.as_deref(),
    )
}

#[tauri::command]
async fn delete_local_trp3_snapshot(
    wow_path: String,
    account_id: String,
    snapshot_id: String,
) -> Result<(), String> {
    local_versions::delete_snapshot(&wow_path, &account_id, &snapshot_id)
}

#[tauri::command]
async fn clear_sync_cache(app: tauri::AppHandle) -> Result<(), String> {
    let app_dir = app
        .path()
        .app_data_dir()
        .map_err(|_| "无法定位应用数据目录".to_string())?;
    let db_path = app_dir.join("sync_meta.db");
    if db_path.exists() {
        std::fs::remove_file(&db_path).map_err(|e| format!("清除缓存失败: {}", e))?;
    }
    Ok(())
}

fn find_profiles_file(
    wow_path: &str,
    account_id: Option<&str>,
    profile_id: &str,
) -> Result<(PathBuf, Value), String> {
    let normalized = wow_path::normalize_wow_path(wow_path)
        .ok_or_else(|| "未找到有效的WoW路径，请选择包含 WTF/Account 的目录".to_string())?;
    let account_root = normalized.join("Account");
    if !account_root.exists() {
        return Err("WTF/Account 目录不存在".to_string());
    }
    let canonical_account_root = std::fs::canonicalize(&account_root)
        .map_err(|e| format!("无法解析 WTF/Account 目录: {}", e))?;

    if let Some(account_id) = account_id {
        validate_account_id(account_id)?;
        let account_dir = account_root.join(account_id);
        return load_profile_from_account(&canonical_account_root, &account_dir, profile_id)?
            .ok_or_else(|| "指定来源账号中未找到人物卡".to_string());
    }

    let entries = std::fs::read_dir(&account_root).map_err(|e| format!("读取目录失败: {}", e))?;

    for entry in entries.flatten() {
        if !entry.path().is_dir() {
            continue;
        }
        let name = entry.file_name().to_string_lossy().to_string();
        if name == "SavedVariables" {
            continue;
        }

        if let Some(result) =
            load_profile_from_account(&canonical_account_root, &entry.path(), profile_id)?
        {
            return Ok(result);
        }
    }

    Err("未找到指定人物卡".to_string())
}

fn validate_account_id(account_id: &str) -> Result<(), String> {
    let mut components = Path::new(account_id).components();
    let is_single_normal_component =
        matches!(components.next(), Some(std::path::Component::Normal(_)))
            && components.next().is_none();
    let is_unsafe = account_id.trim().is_empty()
        || account_id == "."
        || account_id == ".."
        || account_id.contains('/')
        || account_id.contains('\\')
        || account_id.contains('\0')
        || !is_single_normal_component;

    if is_unsafe {
        return Err("来源账号标识无效".to_string());
    }

    Ok(())
}

fn load_profile_from_account(
    canonical_account_root: &Path,
    account_dir: &Path,
    profile_id: &str,
) -> Result<Option<(PathBuf, Value)>, String> {
    if !account_dir.is_dir() {
        return Ok(None);
    }

    let lua_path = account_dir.join("SavedVariables").join("totalRP3.lua");
    if !lua_path.is_file() {
        return Ok(None);
    }

    let canonical_lua_path =
        std::fs::canonicalize(&lua_path).map_err(|e| format!("无法解析 TRP3 人物卡文件: {}", e))?;
    if !canonical_lua_path.starts_with(canonical_account_root) {
        return Err("TRP3 人物卡文件不在 WTF/Account 目录内".to_string());
    }

    let data = lua_parser::parse_variable(&canonical_lua_path, "TRP3_Profiles")
        .map_err(|e| e.to_string())?;
    if data
        .as_object()
        .and_then(|profiles| profiles.get(profile_id))
        .is_none()
    {
        return Ok(None);
    }

    Ok(Some((canonical_lua_path, data)))
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
        set_str_field(characteristics, &["FT"], chars.get("fullTitle"));
        set_str_field(characteristics, &["RA"], chars.get("race"));
        set_str_field(characteristics, &["CL"], chars.get("class"));
        set_str_field(characteristics, &["AG"], chars.get("age"));
        set_str_field(characteristics, &["EC"], chars.get("eyeColor"));
        set_str_field(characteristics, &["EH"], chars.get("eyeColorHex"));
        set_str_field(characteristics, &["HE"], chars.get("height"));
        set_str_field(characteristics, &["WE"], chars.get("weight"));
        set_str_field(characteristics, &["BP"], chars.get("birthplace"));
        set_str_field(characteristics, &["RE"], chars.get("residence"));
        set_integer_field(
            characteristics,
            &["RS"],
            chars.get("relationshipStatus"),
            "感情状态",
        )?;
        set_str_field(characteristics, &["IC"], chars.get("icon"));
        set_str_field(
            characteristics,
            &["CH"],
            chars
                .get("nameColor")
                .or_else(|| chars.get("classColorHex")),
        );
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
    let Some(value) = value else {
        return;
    };
    if value.is_null() {
        remove_field(target, path);
        return;
    }
    let Some(text) = value.as_str() else {
        return;
    };
    if text.is_empty() {
        remove_field(target, path);
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
            .insert(last.to_string(), Value::from(text.to_string()));
    }
}

fn remove_field(target: &mut Value, path: &[&str]) {
    let Some((last, parents)) = path.split_last() else {
        return;
    };
    let mut current = target;
    for key in parents {
        let Some(next) = current
            .as_object_mut()
            .and_then(|object| object.get_mut(*key))
        else {
            return;
        };
        current = next;
    }
    if let Some(object) = current.as_object_mut() {
        object.remove(*last);
    }
}

fn set_integer_field(
    target: &mut Value,
    path: &[&str],
    value: Option<&Value>,
    field_name: &str,
) -> Result<(), String> {
    let Some(value) = value else {
        return Ok(());
    };

    if value.is_null() || matches!(value, Value::String(text) if text.trim().is_empty()) {
        remove_field(target, path);
        return Ok(());
    }

    let number = match value {
        Value::String(text) => text
            .trim()
            .parse::<i64>()
            .map_err(|_| format!("{}必须是整数枚举值", field_name))?,
        Value::Number(number) => number
            .as_i64()
            .ok_or_else(|| format!("{}必须是整数枚举值", field_name))?,
        _ => return Err(format!("{}必须是整数枚举值", field_name)),
    };

    let mut current = target;
    for key in path.iter().take(path.len().saturating_sub(1)) {
        if !current.is_object() {
            *current = Value::Object(Default::default());
        }
        current = current
            .as_object_mut()
            .expect("object was initialized above")
            .entry(key.to_string())
            .or_insert_with(|| Value::Object(Default::default()));
    }

    if let Some(last) = path.last() {
        if !current.is_object() {
            *current = Value::Object(Default::default());
        }
        current
            .as_object_mut()
            .expect("object was initialized above")
            .insert(last.to_string(), Value::from(number));
    }

    Ok(())
}

#[cfg(test)]
mod profile_lookup_tests {
    use super::*;
    use std::fs;
    use std::time::{SystemTime, UNIX_EPOCH};

    struct TestWowDir {
        root: PathBuf,
    }

    impl TestWowDir {
        fn new() -> Self {
            let unique = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .expect("system clock should be after Unix epoch")
                .as_nanos();
            let root = std::env::temp_dir().join(format!(
                "rpbox-profile-lookup-{}-{unique}",
                std::process::id()
            ));
            fs::create_dir_all(root.join("WTF").join("Account"))
                .expect("test account root should be created");
            Self { root }
        }

        fn wtf_path(&self) -> PathBuf {
            self.root.join("WTF")
        }

        fn write_profile(&self, account_id: &str, marker: &str) -> PathBuf {
            let saved_variables = self
                .wtf_path()
                .join("Account")
                .join(account_id)
                .join("SavedVariables");
            fs::create_dir_all(&saved_variables)
                .expect("test SavedVariables directory should be created");
            let lua_path = saved_variables.join("totalRP3.lua");
            fs::write(
                &lua_path,
                format!(
                    r#"TRP3_Profiles = {{
  ["shared-profile"] = {{
    ["marker"] = "{marker}",
  }},
}}
"#
                ),
            )
            .expect("test profile fixture should be written");
            lua_path
        }
    }

    impl Drop for TestWowDir {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.root);
        }
    }

    #[test]
    fn exact_account_lookup_does_not_select_a_duplicate_profile_from_another_account() {
        let wow = TestWowDir::new();
        wow.write_profile("ACCOUNT-A", "ACCOUNT-A");
        let expected_path = wow.write_profile("ACCOUNT-B", "ACCOUNT-B");
        let wow_path = wow.wtf_path();

        let (lua_path, profiles) = find_profiles_file(
            wow_path.to_str().expect("temporary path should be UTF-8"),
            Some("ACCOUNT-B"),
            "shared-profile",
        )
        .expect("profile should be found in the requested account");

        assert_eq!(
            lua_path,
            fs::canonicalize(expected_path).expect("fixture path should be canonicalizable")
        );
        assert_eq!(
            profiles["shared-profile"]["marker"].as_str(),
            Some("ACCOUNT-B")
        );
    }

    #[test]
    fn exact_account_lookup_rejects_parent_directory_traversal() {
        let wow = TestWowDir::new();
        wow.write_profile("ACCOUNT-A", "ACCOUNT-A");
        let wow_path = wow.wtf_path();

        let error = find_profiles_file(
            wow_path.to_str().expect("temporary path should be UTF-8"),
            Some("../ACCOUNT-A"),
            "shared-profile",
        )
        .expect_err("unsafe account identifiers must be rejected");

        assert_eq!(error, "来源账号标识无效");
    }

    #[test]
    fn exact_account_lookup_rejects_an_empty_account_identifier() {
        let wow = TestWowDir::new();
        wow.write_profile("ACCOUNT-A", "ACCOUNT-A");
        let wow_path = wow.wtf_path();

        let error = find_profiles_file(
            wow_path.to_str().expect("temporary path should be UTF-8"),
            Some(""),
            "shared-profile",
        )
        .expect_err("an explicitly supplied account identifier cannot be empty");

        assert_eq!(error, "来源账号标识无效");
    }

    #[test]
    fn exact_account_lookup_does_not_fall_back_when_profile_is_missing() {
        let wow = TestWowDir::new();
        wow.write_profile("ACCOUNT-A", "ACCOUNT-A");
        let wow_path = wow.wtf_path();

        let error = find_profiles_file(
            wow_path.to_str().expect("temporary path should be UTF-8"),
            Some("ACCOUNT-B"),
            "shared-profile",
        )
        .expect_err("account-scoped lookup must not scan other accounts");

        assert_eq!(error, "指定来源账号中未找到人物卡");
    }

    #[test]
    fn relationship_status_is_written_as_a_lua_number() {
        let mut profile = serde_json::json!({
            "player": { "characteristics": { "RS": 1 } }
        });
        let updates = serde_json::json!({
            "characteristics": { "relationshipStatus": "4" }
        });

        apply_updates(&mut profile, &updates).expect("numeric relationship status should be valid");

        assert_eq!(profile["player"]["characteristics"]["RS"], 4);
        assert!(profile["player"]["characteristics"]["RS"].is_number());
    }

    #[test]
    fn relationship_status_rejects_non_numeric_values_without_overwriting() {
        let mut profile = serde_json::json!({
            "player": { "characteristics": { "RS": 2 } }
        });
        let updates = serde_json::json!({
            "characteristics": { "relationshipStatus": "married" }
        });

        let error = apply_updates(&mut profile, &updates)
            .expect_err("non-numeric relationship status must be rejected");

        assert_eq!(error, "感情状态必须是整数枚举值");
        assert_eq!(profile["player"]["characteristics"]["RS"], 2);
    }

    #[test]
    fn empty_relationship_status_removes_the_existing_value() {
        let mut profile = serde_json::json!({
            "player": { "characteristics": { "RS": 3 } }
        });
        let updates = serde_json::json!({
            "characteristics": { "relationshipStatus": "" }
        });

        apply_updates(&mut profile, &updates).expect("empty values should clear existing data");

        assert!(profile["player"]["characteristics"].get("RS").is_none());
    }

    #[test]
    fn empty_string_fields_are_removed_but_omitted_fields_are_preserved() {
        let mut profile = serde_json::json!({
            "player": {
                "characteristics": {
                    "FN": "旧名字",
                    "LN": "保留的姓氏",
                    "RS": 2
                }
            }
        });
        let updates = serde_json::json!({
            "characteristics": { "firstName": "" }
        });

        apply_updates(&mut profile, &updates).expect("an empty string should clear the field");

        let characteristics = &profile["player"]["characteristics"];
        assert!(characteristics.get("FN").is_none());
        assert_eq!(characteristics["LN"], "保留的姓氏");
        assert_eq!(characteristics["RS"], 2);
    }

    #[test]
    fn character_card_writeback_preserves_unknown_local_sections_and_ignores_rpbox_only_data() {
        let existing = serde_json::json!({
            "profileName": "旧档案名",
            "unknownTopLevel": { "keep": true },
            "player": {
                "characteristics": {
                    "FN": "旧名字",
                    "LN": "应被空值删除",
                    "customCharacteristic": "保留"
                },
                "about": { "T1": { "TX": "本地长传记" } },
                "misc": { "PE": { "1": { "TX": "本地第一印象" } }, "custom": 7 }
            }
        });
        let exported = serde_json::json!({
            "profileName": "新档案名",
            "rpboxOnly": { "summary": "不得写入" },
            "player": {
                "characteristics": { "FN": "新名字", "LN": "", "RS": 3 },
                "misc": { "PE": { "1": { "TX": "RPBox 第一印象" } } },
                "about": { "T1": { "TX": "RPBox 内容" } }
            }
        });

        let merged = merge_character_card_profile(Some(&existing), &exported)
            .expect("shared fields should merge into the existing profile");

        assert_eq!(merged["profileName"], "新档案名");
        assert_eq!(merged["unknownTopLevel"]["keep"], true);
        assert_eq!(merged["player"]["characteristics"]["FN"], "新名字");
        assert_eq!(merged["player"]["characteristics"]["RS"], 3);
        assert!(merged["player"]["characteristics"].get("LN").is_none());
        assert_eq!(
            merged["player"]["characteristics"]["customCharacteristic"],
            "保留"
        );
        assert_eq!(merged["player"]["about"]["T1"]["TX"], "本地长传记");
        assert_eq!(merged["player"]["misc"]["PE"]["1"]["TX"], "本地第一印象");
        assert_eq!(merged["player"]["misc"]["custom"], 7);
        assert!(merged.get("rpboxOnly").is_none());
    }

    #[test]
    fn character_card_writeback_creates_a_minimal_new_trp3_profile() {
        let exported = serde_json::json!({
            "profileName": "新人物",
            "player": {
                "characteristics": { "FN": "新", "LN": "人物", "CH": "AABBCC" },
                "misc": { "PE": { "1": { "TX": "不得写入" } } }
            }
        });

        let merged = merge_character_card_profile(None, &exported)
            .expect("a blank RPBox card should create a local profile");

        assert_eq!(merged["profileName"], "新人物");
        assert_eq!(merged["player"]["characteristics"]["FN"], "新");
        assert_eq!(merged["player"]["characteristics"]["CH"], "AABBCC");
        assert!(merged["player"].get("misc").is_none());
        assert_eq!(
            merged
                .as_object()
                .expect("profile should be an object")
                .len(),
            2
        );
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let builder = tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            None::<Vec<&str>>,
        ));

    let builder = builder.plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
        focus_main_window(app);
    }));

    builder
        .setup(|app| {
            focus_main_window(&app.handle());

            if cfg!(target_os = "linux") || cfg!(all(debug_assertions, target_os = "windows")) {
                if let Err(error) = app.deep_link().register_all() {
                    eprintln!("failed to register desktop deep links: {error}");
                }
            }

            Ok(())
        })
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
            write_character_card_profile,
            list_local_trp3_snapshots,
            read_local_trp3_snapshot,
            rename_local_trp3_snapshot,
            restore_local_trp3_snapshot,
            delete_local_trp3_snapshot,
            clear_sync_cache,
            apply_cloud_profile,
            apply_account_backup,
            check_addon_installed,
            check_trp3_addons,
            install_trp3_addon,
            install_trp3_addon_with_progress,
            install_trp3_addon_zip,
            install_all_trp3_addons,
            uninstall_trp3_addon,
            open_addons_folder,
            install_addon,
            install_addon_from_url,
            uninstall_addon,
            scan_chat_logs,
            save_text_file,
            save_binary_file
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn focus_main_window<R: tauri::Runtime>(app: &tauri::AppHandle<R>) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
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

    let mut profiles =
        lua_parser::parse_variable(&sv_path, "TRP3_Profiles").map_err(|e| e.to_string())?;

    let profile_value: Value =
        serde_json::from_str(&profile_json).map_err(|e| format!("云端数据解析失败: {}", e))?;

    profiles
        .as_object_mut()
        .ok_or_else(|| "TRP3_Profiles 数据格式错误".to_string())?
        .insert(profile_id, profile_value);

    replace_trp3_profiles(&sv_path, &profiles).map_err(|e| e.to_string())
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

    let use_raw_trp3 = raw_trp3_lua
        .as_ref()
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    let use_raw_data = raw_trp3_data_lua
        .as_ref()
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    let use_raw_extended = raw_trp3_extended_lua
        .as_ref()
        .map(|s| !s.is_empty())
        .unwrap_or(false);

    if use_raw_trp3 {
        if let Some(raw) = raw_trp3_lua {
            writer::write_profile_to_local(&sv_path, &raw).map_err(|e| e.to_string())?;
        }
    } else {
        // 解析云端备份的所有 profiles
        let cloud_profiles: serde_json::Map<String, Value> =
            serde_json::from_str(&profiles_json).map_err(|e| format!("云端数据解析失败: {}", e))?;

        // 如果本地文件存在，读取并合并；否则直接使用云端数据
        let final_profiles = if sv_path.exists() {
            let mut local_profiles =
                lua_parser::parse_variable(&sv_path, "TRP3_Profiles").map_err(|e| e.to_string())?;

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

        replace_trp3_profiles(&sv_path, &final_profiles).map_err(|e| e.to_string())?;
    }

    if use_raw_extended {
        if let Some(raw) = raw_trp3_extended_lua {
            writer::write_profile_to_local(&extended_path, &raw).map_err(|e| e.to_string())?;
        }
    } else {
        // 写回道具数据库（如果有）
        if let Some(tools_data) = tools_json {
            if !tools_data.is_empty() {
                let tools_value: Value = serde_json::from_str(&tools_data)
                    .map_err(|e| format!("道具数据解析失败: {}", e))?;
                writer::write_tools_db(&sv_dir, &tools_value).map_err(|e| e.to_string())?;
            }
        }
    }

    if use_raw_data {
        if let Some(raw) = raw_trp3_data_lua {
            writer::write_profile_to_local(&data_path, &raw).map_err(|e| e.to_string())?;
        }
    } else {
        // 写回运行时数据（如果有）
        if let Some(runtime_data) = runtime_json {
            if !runtime_data.is_empty() {
                let runtime_value: Value = serde_json::from_str(&runtime_data)
                    .map_err(|e| format!("运行时数据解析失败: {}", e))?;
                writer::write_runtime_data(&sv_dir, &runtime_value).map_err(|e| e.to_string())?;
            }
        }
    }

    if !use_raw_trp3 {
        // 写回配置数据（如果有）
        if let Some(config_data) = config_json {
            if !config_data.is_empty() {
                let config_value: Value = serde_json::from_str(&config_data)
                    .map_err(|e| format!("配置数据解析失败: {}", e))?;
                writer::write_config(&sv_path, &config_value).map_err(|e| e.to_string())?;
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
async fn check_addon_installed(
    wow_path: String,
    flavor: String,
) -> addon_installer::InstalledAddonInfo {
    addon_installer::check_addon_installed(&wow_path, &flavor)
}

#[tauri::command]
async fn check_trp3_addons(wow_path: String) -> addon_installer::Trp3AddonCheckResult {
    addon_installer::check_trp3_addons(&wow_path)
}

#[tauri::command]
async fn install_trp3_addon(
    wow_path: String,
    addon_id: String,
    download_url: String,
) -> Result<addon_installer::Trp3AddonCheckResult, String> {
    tauri::async_runtime::spawn_blocking(move || {
        addon_installer::install_trp3_addon(&wow_path, &addon_id, &download_url)
    })
    .await
    .map_err(|e| format!("TRP3 插件安装任务失败: {}", e))?
}

#[tauri::command]
async fn install_trp3_addon_with_progress(
    app: tauri::AppHandle,
    wow_path: String,
    addon_id: String,
    download_url: String,
) -> Result<addon_installer::Trp3AddonCheckResult, String> {
    tauri::async_runtime::spawn_blocking(move || {
        let progress_app = app.clone();
        addon_installer::install_trp3_addon_with_progress(
            &wow_path,
            &addon_id,
            &download_url,
            move |progress| {
                let _ = progress_app.emit("addon-install-progress", progress);
            },
        )
    })
    .await
    .map_err(|e| format!("TRP3 插件安装任务失败: {}", e))?
}

#[tauri::command]
async fn install_trp3_addon_zip(
    wow_path: String,
    addon_id: String,
    zip_data: Vec<u8>,
) -> Result<addon_installer::Trp3AddonCheckResult, String> {
    tauri::async_runtime::spawn_blocking(move || {
        addon_installer::install_trp3_addon_zip_data(&wow_path, &addon_id, &zip_data)
    })
    .await
    .map_err(|e| format!("TRP3 插件安装任务失败: {}", e))?
}

#[tauri::command]
async fn install_all_trp3_addons(
    wow_path: String,
    addons: Vec<addon_installer::Trp3AddonInstallRequest>,
) -> Result<addon_installer::Trp3AddonCheckResult, String> {
    tauri::async_runtime::spawn_blocking(move || {
        addon_installer::install_all_trp3_addons(&wow_path, &addons)
    })
    .await
    .map_err(|e| format!("TRP3 插件安装任务失败: {}", e))?
}

#[tauri::command]
async fn uninstall_trp3_addon(
    wow_path: String,
    addon_id: String,
) -> Result<addon_installer::Trp3AddonCheckResult, String> {
    tauri::async_runtime::spawn_blocking(move || {
        addon_installer::uninstall_trp3_addon(&wow_path, &addon_id)
    })
    .await
    .map_err(|e| format!("TRP3 插件卸载任务失败: {}", e))?
}

#[tauri::command]
async fn open_addons_folder(wow_path: String) -> Result<String, String> {
    tauri::async_runtime::spawn_blocking(move || {
        if wow_path.trim().is_empty() {
            return Err("未选择魔兽目录".to_string());
        }

        let addons_dir = addon_installer::get_addons_dir(&wow_path);
        std::fs::create_dir_all(&addons_dir)
            .map_err(|e| format!("创建 AddOns 目录失败: {}", e))?;

        open_folder_in_file_manager(&addons_dir)?;
        Ok(addons_dir.to_string_lossy().to_string())
    })
    .await
    .map_err(|e| format!("打开 AddOns 目录任务失败: {}", e))?
}

fn open_folder_in_file_manager(path: &Path) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    let mut command = {
        let mut command = Command::new("explorer.exe");
        command.arg(path);
        command
    };

    #[cfg(target_os = "macos")]
    let mut command = {
        let mut command = Command::new("open");
        command.arg(path);
        command
    };

    #[cfg(all(unix, not(target_os = "macos")))]
    let mut command = {
        let mut command = Command::new("xdg-open");
        command.arg(path);
        command
    };

    command
        .spawn()
        .map_err(|e| format!("启动系统文件管理器失败: {}", e))?;
    Ok(())
}

#[tauri::command]
async fn install_addon(
    wow_path: String,
    flavor: String,
    zip_data: Vec<u8>,
) -> Result<String, String> {
    tauri::async_runtime::spawn_blocking(move || {
        addon_installer::install_addon(&wow_path, &flavor, &zip_data)
    })
    .await
    .map_err(|e| format!("RPBox 插件安装任务失败: {}", e))?
}

#[tauri::command]
async fn install_addon_from_url(
    app: tauri::AppHandle,
    wow_path: String,
    flavor: String,
    download_url: String,
    plugin_id: String,
) -> Result<String, String> {
    tauri::async_runtime::spawn_blocking(move || {
        let progress_app = app.clone();
        addon_installer::install_addon_from_url_with_progress(
            &wow_path,
            &flavor,
            &download_url,
            &plugin_id,
            move |progress| {
                let _ = progress_app.emit("addon-install-progress", progress);
            },
        )
    })
    .await
    .map_err(|e| format!("RPBox 插件安装任务失败: {}", e))?
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
    std::fs::write(&path, content).map_err(|e| format!("保存文件失败: {}", e))
}

#[tauri::command]
async fn save_binary_file(path: String, data: Vec<u8>) -> Result<(), String> {
    std::fs::write(&path, data).map_err(|e| format!("保存文件失败: {}", e))
}
