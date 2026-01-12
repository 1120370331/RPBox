use std::path::PathBuf;
use std::fs;

#[derive(Debug)]
pub enum WriteError {
    WowRunning,
    BackupFailed(String),
    WriteFailed(String),
}

impl std::fmt::Display for WriteError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            WriteError::WowRunning => write!(f, "魔兽世界正在运行"),
            WriteError::BackupFailed(e) => write!(f, "备份失败: {}", e),
            WriteError::WriteFailed(e) => write!(f, "写入失败: {}", e),
        }
    }
}

pub fn is_wow_running() -> bool {
    #[cfg(target_os = "windows")]
    {
        use std::process::Command;
        let output = Command::new("tasklist")
            .args(["/FI", "IMAGENAME eq Wow.exe"])
            .output();

        if let Ok(out) = output {
            let result = String::from_utf8_lossy(&out.stdout);
            return result.contains("Wow.exe");
        }
    }
    false
}

fn backup_file(path: &PathBuf) -> Result<(), WriteError> {
    if path.exists() {
        let backup_path = path.with_extension("lua.rpbox_backup");
        fs::copy(path, &backup_path)
            .map_err(|e| WriteError::BackupFailed(e.to_string()))?;
    }
    Ok(())
}

pub fn write_profile_to_local(
    lua_path: &PathBuf,
    raw_lua: &str,
) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    backup_file(lua_path)?;

    fs::write(lua_path, raw_lua)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    Ok(())
}
