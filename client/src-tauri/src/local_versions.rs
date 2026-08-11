use chrono::{SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Component, Path, PathBuf};

use crate::{wow_path, writer};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocalTRP3Snapshot {
    pub id: String,
    pub name: String,
    pub account_id: String,
    #[serde(default)]
    pub reason: String,
    pub checksum: String,
    pub created_at: String,
    pub size_bytes: u64,
}

#[derive(Debug, Clone, Serialize)]
pub struct LocalTRP3SnapshotDetail {
    #[serde(flatten)]
    pub snapshot: LocalTRP3Snapshot,
    pub content: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct LocalTRP3RestoreResult {
    pub restored: LocalTRP3Snapshot,
    pub safety_snapshot: Option<LocalTRP3Snapshot>,
}

pub fn validate_account_id(account_id: &str) -> Result<(), String> {
    let mut components = Path::new(account_id).components();
    let normal =
        matches!(components.next(), Some(Component::Normal(_))) && components.next().is_none();
    if account_id.trim().is_empty()
        || account_id == "."
        || account_id == ".."
        || account_id.contains('/')
        || account_id.contains('\\')
        || account_id.contains('\0')
        || !normal
    {
        return Err("本地账号标识无效".to_string());
    }
    Ok(())
}

fn validate_snapshot_id(snapshot_id: &str) -> Result<(), String> {
    if snapshot_id.is_empty()
        || snapshot_id.len() > 96
        || !snapshot_id.chars().all(|character| {
            character.is_ascii_alphanumeric() || character == '-' || character == '_'
        })
    {
        return Err("本地版本标识无效".to_string());
    }
    Ok(())
}

pub fn resolve_lua_path(wow_path_value: &str, account_id: &str) -> Result<PathBuf, String> {
    validate_account_id(account_id)?;
    let normalized = wow_path::normalize_wow_path(wow_path_value)
        .ok_or_else(|| "未找到有效的WoW路径，请选择包含 WTF/Account 的目录".to_string())?;
    let account_root = normalized.join("Account");
    let canonical_root = fs::canonicalize(&account_root)
        .map_err(|error| format!("无法解析 WTF/Account 目录: {}", error))?;
    let account_dir = account_root.join(account_id);
    if !account_dir.is_dir() {
        return Err("本地账号目录不存在".to_string());
    }
    let canonical_account = fs::canonicalize(&account_dir)
        .map_err(|error| format!("无法解析本地账号目录: {}", error))?;
    if !canonical_account.starts_with(&canonical_root) {
        return Err("本地账号目录不在 WTF/Account 内".to_string());
    }
    Ok(canonical_account
        .join("SavedVariables")
        .join("totalRP3.lua"))
}

fn resolve_snapshot_dir(wow_path_value: &str, account_id: &str) -> Result<PathBuf, String> {
    let lua_path = resolve_lua_path(wow_path_value, account_id)?;
    let saved_variables = lua_path
        .parent()
        .ok_or_else(|| "无法定位 SavedVariables 目录".to_string())?;
    let directory = saved_variables.join("RPBox_Backups").join("totalRP3");
    fs::create_dir_all(&directory).map_err(|error| format!("创建本地版本目录失败: {}", error))?;
    let canonical_saved_variables = fs::canonicalize(saved_variables)
        .map_err(|error| format!("无法解析 SavedVariables 目录: {}", error))?;
    let canonical_directory =
        fs::canonicalize(&directory).map_err(|error| format!("无法解析本地版本目录: {}", error))?;
    if !canonical_directory.starts_with(&canonical_saved_variables) {
        return Err("本地版本目录不在 SavedVariables 内".to_string());
    }
    Ok(canonical_directory)
}

fn snapshot_paths(directory: &Path, snapshot_id: &str) -> Result<(PathBuf, PathBuf), String> {
    validate_snapshot_id(snapshot_id)?;
    Ok((
        directory.join(format!("{}.lua", snapshot_id)),
        directory.join(format!("{}.json", snapshot_id)),
    ))
}

#[cfg(windows)]
fn is_reparse_point(metadata: &fs::Metadata) -> bool {
    use std::os::windows::fs::MetadataExt;
    const FILE_ATTRIBUTE_REPARSE_POINT: u32 = 0x0400;
    metadata.file_attributes() & FILE_ATTRIBUTE_REPARSE_POINT != 0
}

#[cfg(not(windows))]
fn is_reparse_point(_: &fs::Metadata) -> bool {
    false
}

fn validate_regular_snapshot_file(
    directory: &Path,
    path: &Path,
    label: &str,
) -> Result<PathBuf, String> {
    let metadata = fs::symlink_metadata(path)
        .map_err(|error| format!("读取{}文件属性失败: {}", label, error))?;
    if metadata.file_type().is_symlink() || is_reparse_point(&metadata) {
        return Err(format!("{}不能是符号链接或 Windows reparse point", label));
    }
    if !metadata.file_type().is_file() {
        return Err(format!("{}不是普通文件", label));
    }
    let canonical =
        fs::canonicalize(path).map_err(|error| format!("解析{}路径失败: {}", label, error))?;
    if canonical.parent() != Some(directory) {
        return Err(format!("{}不在本地版本目录内", label));
    }
    Ok(canonical)
}

fn read_validated_snapshot_pair(
    directory: &Path,
    account_id: &str,
    snapshot_id: &str,
) -> Result<(LocalTRP3Snapshot, Vec<u8>), String> {
    let (lua_path, metadata_path) = snapshot_paths(directory, snapshot_id)?;
    let canonical_lua = validate_regular_snapshot_file(directory, &lua_path, "Lua 快照")?;
    let canonical_metadata =
        validate_regular_snapshot_file(directory, &metadata_path, "版本元数据")?;
    let metadata =
        fs::read(canonical_metadata).map_err(|error| format!("读取版本元数据失败: {}", error))?;
    let snapshot: LocalTRP3Snapshot = serde_json::from_slice(&metadata)
        .map_err(|error| format!("解析版本元数据失败: {}", error))?;
    if snapshot.id != snapshot_id {
        return Err("版本元数据 ID 与文件名不一致".to_string());
    }
    if snapshot.account_id != account_id {
        return Err("版本元数据账号与当前账号不一致".to_string());
    }
    let content =
        fs::read(canonical_lua).map_err(|error| format!("读取 Lua 快照失败: {}", error))?;
    let checksum = format!("{:x}", md5::compute(&content));
    if snapshot.checksum != checksum {
        return Err("Lua 快照内容校验失败".to_string());
    }
    if snapshot.size_bytes != content.len() as u64 {
        return Err("Lua 快照大小与元数据不一致".to_string());
    }
    Ok((snapshot, content))
}

fn path_entry_exists(path: &Path) -> bool {
    fs::symlink_metadata(path).is_ok()
}

fn write_new_snapshot_file(path: &Path, content: &[u8], label: &str) -> Result<(), String> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .map_err(|error| format!("创建{}失败: {}", label, error))?;
    file.write_all(content)
        .map_err(|error| format!("写入{}失败: {}", label, error))?;
    file.sync_all()
        .map_err(|error| format!("同步{}失败: {}", label, error))
}

fn clean_snapshot_name(name: Option<&str>, created_at: &str, checksum: &str) -> String {
    let supplied = name.unwrap_or_default().trim();
    if supplied.is_empty() {
        return format!("{} · {}", created_at, &checksum[..8]);
    }
    supplied.chars().take(120).collect()
}

pub fn create_snapshot(
    wow_path_value: &str,
    account_id: &str,
    name: Option<&str>,
) -> Result<Option<LocalTRP3Snapshot>, String> {
    create_snapshot_with_reason(wow_path_value, account_id, name, "manual")
}

pub fn create_snapshot_with_reason(
    wow_path_value: &str,
    account_id: &str,
    name: Option<&str>,
    reason: &str,
) -> Result<Option<LocalTRP3Snapshot>, String> {
    let lua_path = resolve_lua_path(wow_path_value, account_id)?;
    if !lua_path.is_file() {
        return Ok(None);
    }
    let content =
        fs::read(&lua_path).map_err(|error| format!("读取 totalRP3.lua 失败: {}", error))?;
    let checksum = format!("{:x}", md5::compute(&content));
    let now = Utc::now();
    let created_at = now.to_rfc3339_opts(SecondsFormat::Millis, true);
    let base_id = format!("{}-{}", now.format("%Y%m%dT%H%M%S%3fZ"), &checksum[..10]);
    let directory = resolve_snapshot_dir(wow_path_value, account_id)?;
    let mut id = base_id.clone();
    let mut suffix = 1;
    while path_entry_exists(&snapshot_paths(&directory, &id)?.0)
        || path_entry_exists(&snapshot_paths(&directory, &id)?.1)
    {
        id = format!("{}-{}", base_id, suffix);
        suffix += 1;
    }
    let snapshot = LocalTRP3Snapshot {
        id: id.clone(),
        name: clean_snapshot_name(name, &created_at, &checksum),
        account_id: account_id.to_string(),
        reason: reason.trim().to_string(),
        checksum,
        created_at,
        size_bytes: content.len() as u64,
    };
    let (lua_snapshot_path, metadata_path) = snapshot_paths(&directory, &id)?;
    if let Err(error) = write_new_snapshot_file(&lua_snapshot_path, &content, "Lua 快照") {
        let _ = fs::remove_file(&lua_snapshot_path);
        return Err(error);
    }
    let metadata = serde_json::to_vec_pretty(&snapshot)
        .map_err(|error| format!("序列化本地版本失败: {}", error))?;
    if let Err(error) = write_new_snapshot_file(&metadata_path, &metadata, "版本元数据") {
        let _ = fs::remove_file(&lua_snapshot_path);
        return Err(error);
    }
    Ok(Some(snapshot))
}

pub fn list_snapshots(
    wow_path_value: &str,
    account_id: &str,
) -> Result<Vec<LocalTRP3Snapshot>, String> {
    let directory = resolve_snapshot_dir(wow_path_value, account_id)?;
    let mut snapshots = Vec::new();
    for entry in fs::read_dir(&directory).map_err(|error| format!("读取本地版本失败: {}", error))?
    {
        let entry = entry.map_err(|error| format!("读取本地版本失败: {}", error))?;
        if entry.path().extension().and_then(|value| value.to_str()) != Some("json") {
            continue;
        }
        let entry_path = entry.path();
        let Some(snapshot_id) = entry_path.file_stem().and_then(|value| value.to_str()) else {
            continue;
        };
        if let Ok((snapshot, _)) = read_validated_snapshot_pair(&directory, account_id, snapshot_id)
        {
            snapshots.push(snapshot);
        }
    }
    snapshots.sort_by(|left, right| right.created_at.cmp(&left.created_at));
    Ok(snapshots)
}

pub fn read_snapshot(
    wow_path_value: &str,
    account_id: &str,
    snapshot_id: &str,
) -> Result<LocalTRP3SnapshotDetail, String> {
    let directory = resolve_snapshot_dir(wow_path_value, account_id)?;
    let (snapshot, content) = read_validated_snapshot_pair(&directory, account_id, snapshot_id)?;
    let content =
        String::from_utf8(content).map_err(|error| format!("Lua 快照不是 UTF-8: {}", error))?;
    Ok(LocalTRP3SnapshotDetail { snapshot, content })
}

pub fn rename_snapshot(
    wow_path_value: &str,
    account_id: &str,
    snapshot_id: &str,
    name: &str,
) -> Result<LocalTRP3Snapshot, String> {
    let trimmed = name.trim();
    if trimmed.is_empty() {
        return Err("版本名称不能为空".to_string());
    }
    let directory = resolve_snapshot_dir(wow_path_value, account_id)?;
    let (mut snapshot, _) = read_validated_snapshot_pair(&directory, account_id, snapshot_id)?;
    let (_, metadata_path) = snapshot_paths(&directory, snapshot_id)?;
    validate_regular_snapshot_file(&directory, &metadata_path, "版本元数据")?;
    snapshot.name = trimmed.chars().take(120).collect();
    fs::write(
        metadata_path,
        serde_json::to_vec_pretty(&snapshot)
            .map_err(|error| format!("序列化本地版本失败: {}", error))?,
    )
    .map_err(|error| format!("保存版本名称失败: {}", error))?;
    Ok(snapshot)
}

pub fn restore_snapshot(
    wow_path_value: &str,
    account_id: &str,
    snapshot_id: &str,
    safety_snapshot_name: Option<&str>,
) -> Result<LocalTRP3RestoreResult, String> {
    if writer::is_wow_running() {
        return Err("检测到魔兽世界正在运行，请关闭游戏后再恢复".to_string());
    }
    let detail = read_snapshot(wow_path_value, account_id, snapshot_id)?;
    let safety_snapshot = create_snapshot_with_reason(
        wow_path_value,
        account_id,
        safety_snapshot_name,
        "before_restore",
    )?;
    let lua_path = resolve_lua_path(wow_path_value, account_id)?;
    writer::write_profile_to_local(&lua_path, &detail.content)
        .map_err(|error| error.to_string())?;
    Ok(LocalTRP3RestoreResult {
        restored: detail.snapshot,
        safety_snapshot,
    })
}

pub fn delete_snapshot(
    wow_path_value: &str,
    account_id: &str,
    snapshot_id: &str,
) -> Result<(), String> {
    let directory = resolve_snapshot_dir(wow_path_value, account_id)?;
    let (lua_path, metadata_path) = snapshot_paths(&directory, snapshot_id)?;
    read_validated_snapshot_pair(&directory, account_id, snapshot_id)?;
    validate_regular_snapshot_file(&directory, &lua_path, "Lua 快照")?;
    validate_regular_snapshot_file(&directory, &metadata_path, "版本元数据")?;
    fs::remove_file(&lua_path).map_err(|error| format!("删除 Lua 快照失败: {}", error))?;
    if let Err(error) = fs::remove_file(&metadata_path) {
        return Err(format!("Lua 快照已删除，但元数据删除失败: {}", error));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    const ACCOUNT_ID: &str = "ACCOUNT-ONE";

    struct SnapshotFixture {
        _temp: TempDir,
        wtf_path: PathBuf,
        lua_path: PathBuf,
    }

    impl SnapshotFixture {
        fn new(content: &str) -> Self {
            let temp = tempfile::tempdir().expect("temporary fixture should be created");
            let wtf_path = temp.path().join("WTF");
            let saved_variables = wtf_path
                .join("Account")
                .join(ACCOUNT_ID)
                .join("SavedVariables");
            fs::create_dir_all(&saved_variables).expect("SavedVariables fixture should be created");
            let lua_path = saved_variables.join("totalRP3.lua");
            fs::write(&lua_path, content).expect("TRP3 fixture should be written");
            Self {
                _temp: temp,
                wtf_path,
                lua_path,
            }
        }

        fn wow_path(&self) -> &str {
            self.wtf_path
                .to_str()
                .expect("temporary fixture path should be UTF-8")
        }

        fn snapshot_dir(&self) -> PathBuf {
            self.lua_path
                .parent()
                .expect("fixture Lua should have a parent")
                .join("RPBox_Backups")
                .join("totalRP3")
        }
    }

    #[cfg(unix)]
    fn create_file_symlink(source: &Path, target: &Path) -> std::io::Result<()> {
        std::os::unix::fs::symlink(source, target)
    }

    #[cfg(windows)]
    fn create_file_symlink(source: &Path, target: &Path) -> std::io::Result<()> {
        std::os::windows::fs::symlink_file(source, target)
    }

    fn assert_snapshot_operations_reject(fixture: &SnapshotFixture, snapshot_id: &str) {
        assert!(read_snapshot(fixture.wow_path(), ACCOUNT_ID, snapshot_id).is_err());
        assert!(rename_snapshot(fixture.wow_path(), ACCOUNT_ID, snapshot_id, "不应成功",).is_err());
        assert!(restore_snapshot(
            fixture.wow_path(),
            ACCOUNT_ID,
            snapshot_id,
            Some("不应创建"),
        )
        .is_err());
        assert!(delete_snapshot(fixture.wow_path(), ACCOUNT_ID, snapshot_id).is_err());
    }

    #[test]
    fn rejects_path_like_snapshot_ids() {
        assert!(validate_snapshot_id("../escape").is_err());
        assert!(validate_snapshot_id("safe_2026-abc").is_ok());
    }

    #[test]
    fn creates_default_name_from_time_and_hash() {
        assert_eq!(
            clean_snapshot_name(None, "2026-08-11T00:00:00Z", "1234567890abcdef"),
            "2026-08-11T00:00:00Z · 12345678",
        );
    }

    #[test]
    fn snapshot_fixture_supports_create_read_rename_safe_restore_and_delete() {
        let original = "TRP3_Profiles = { original = true }\n";
        let changed = "TRP3_Profiles = { changed = true }\n";
        let fixture = SnapshotFixture::new(original);

        let snapshot = create_snapshot(fixture.wow_path(), ACCOUNT_ID, Some("初始版本"))
            .expect("snapshot creation should succeed")
            .expect("an existing Lua file must create a snapshot");
        let detail = read_snapshot(fixture.wow_path(), ACCOUNT_ID, &snapshot.id)
            .expect("snapshot should be readable");
        assert_eq!(detail.content, original);
        assert_eq!(
            detail.snapshot.checksum,
            format!("{:x}", md5::compute(original))
        );

        let renamed = rename_snapshot(fixture.wow_path(), ACCOUNT_ID, &snapshot.id, "可读名称")
            .expect("snapshot should be renameable");
        assert_eq!(renamed.name, "可读名称");

        fs::write(&fixture.lua_path, changed).expect("current Lua should be changed");
        let restored = restore_snapshot(
            fixture.wow_path(),
            ACCOUNT_ID,
            &snapshot.id,
            Some("回退前保护"),
        )
        .expect("restore should succeed after creating a safety snapshot");
        let safety = restored
            .safety_snapshot
            .expect("restore must preserve the current file first");
        assert_eq!(fs::read_to_string(&fixture.lua_path).unwrap(), original);
        assert_eq!(safety.checksum, format!("{:x}", md5::compute(changed)));
        assert_eq!(
            list_snapshots(fixture.wow_path(), ACCOUNT_ID)
                .unwrap()
                .len(),
            2
        );

        delete_snapshot(fixture.wow_path(), ACCOUNT_ID, &snapshot.id)
            .expect("a complete regular snapshot pair should be deletable");
        assert!(read_snapshot(fixture.wow_path(), ACCOUNT_ID, &snapshot.id).is_err());
        assert_eq!(
            list_snapshots(fixture.wow_path(), ACCOUNT_ID)
                .unwrap()
                .len(),
            1
        );
    }

    #[test]
    fn rejects_tampered_snapshot_content_and_metadata_identity() {
        let fixture = SnapshotFixture::new("TRP3_Profiles = { safe = true }\n");
        let snapshot = create_snapshot(fixture.wow_path(), ACCOUNT_ID, None)
            .unwrap()
            .expect("fixture should create a snapshot");
        let (lua_path, metadata_path) =
            snapshot_paths(&fixture.snapshot_dir(), &snapshot.id).unwrap();

        fs::write(&lua_path, "tampered").expect("snapshot content should be tampered");
        assert_snapshot_operations_reject(&fixture, &snapshot.id);
        fs::write(&lua_path, "TRP3_Profiles = { safe = true }\n")
            .expect("snapshot content should be restored");

        let mut bad_metadata = snapshot.clone();
        bad_metadata.id = "different-id".to_string();
        fs::write(
            &metadata_path,
            serde_json::to_vec_pretty(&bad_metadata).unwrap(),
        )
        .expect("metadata identity should be tampered");
        assert_snapshot_operations_reject(&fixture, &snapshot.id);

        bad_metadata.id = snapshot.id.clone();
        bad_metadata.account_id = "OTHER-ACCOUNT".to_string();
        fs::write(
            &metadata_path,
            serde_json::to_vec_pretty(&bad_metadata).unwrap(),
        )
        .expect("metadata account should be tampered");
        assert_snapshot_operations_reject(&fixture, &snapshot.id);
    }

    #[test]
    fn rejects_symlinked_lua_snapshot_when_platform_supports_file_symlinks() {
        let fixture = SnapshotFixture::new("TRP3_Profiles = { safe = true }\n");
        let snapshot = create_snapshot(fixture.wow_path(), ACCOUNT_ID, None)
            .unwrap()
            .expect("fixture should create a snapshot");
        let (lua_path, _) = snapshot_paths(&fixture.snapshot_dir(), &snapshot.id).unwrap();
        let outside = fixture._temp.path().join("outside-snapshot.lua");
        fs::write(&outside, "outside").unwrap();
        fs::remove_file(&lua_path).unwrap();
        if create_file_symlink(&outside, &lua_path).is_err() {
            return;
        }

        assert_snapshot_operations_reject(&fixture, &snapshot.id);
    }

    #[test]
    fn rejects_symlinked_metadata_when_platform_supports_file_symlinks() {
        let fixture = SnapshotFixture::new("TRP3_Profiles = { safe = true }\n");
        let snapshot = create_snapshot(fixture.wow_path(), ACCOUNT_ID, None)
            .unwrap()
            .expect("fixture should create a snapshot");
        let (_, metadata_path) = snapshot_paths(&fixture.snapshot_dir(), &snapshot.id).unwrap();
        let outside = fixture._temp.path().join("outside-snapshot.json");
        fs::copy(&metadata_path, &outside).unwrap();
        fs::remove_file(&metadata_path).unwrap();
        if create_file_symlink(&outside, &metadata_path).is_err() {
            return;
        }

        assert_snapshot_operations_reject(&fixture, &snapshot.id);
    }
}
