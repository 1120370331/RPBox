use std::path::PathBuf;
use std::fs;

use serde_json::Value;

#[derive(Debug)]
pub enum WriteError {
    WowRunning,
    BackupFailed(String),
    WriteFailed(String),
    ParseFailed(String),
}

impl std::fmt::Display for WriteError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            WriteError::WowRunning => write!(f, "魔兽世界正在运行"),
            WriteError::BackupFailed(e) => write!(f, "备份失败: {}", e),
            WriteError::WriteFailed(e) => write!(f, "写入失败: {}", e),
            WriteError::ParseFailed(e) => write!(f, "解析失败: {}", e),
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

/// 将 serde_json::Value 转成 Lua table 字符串
pub fn to_lua_table(value: &Value, indent: usize) -> String {
    match value {
        Value::Null => "nil".to_string(),
        Value::Bool(b) => b.to_string(),
        Value::Number(n) => n.to_string(),
        Value::String(s) => format!("\"{}\"", s.replace('\\', "\\\\").replace('"', "\\\"")),
        Value::Array(arr) => {
            if arr.is_empty() {
                return "{}".to_string();
            }
            let mut parts = Vec::new();
            for v in arr {
                parts.push(format!(
                    "{}{}",
                    " ".repeat(indent + 2),
                    to_lua_table(v, indent + 2)
                ));
            }
            format!("{{\n{}\n{}}}", parts.join(",\n"), " ".repeat(indent))
        }
        Value::Object(map) => {
            if map.is_empty() {
                return "{}".to_string();
            }
            let mut parts = Vec::new();
            for (k, v) in map {
                let key = if k.chars().all(|c| c.is_ascii_alphanumeric() || c == '_') {
                    k.clone()
                } else {
                    format!("[\"{}\"]", k)
                };
                parts.push(format!(
                    "{}{} = {}",
                    " ".repeat(indent + 2),
                    key,
                    to_lua_table(v, indent + 2)
                ));
            }
            format!("{{\n{}\n{}}}", parts.join(",\n"), " ".repeat(indent))
        }
    }
}

/// 用新的 TRP3_Profiles 覆盖原文件中的 TRP3_Profiles 变量，保留其他内容。
pub fn replace_trp3_profiles(lua_path: &PathBuf, profiles: &Value) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    let original = fs::read_to_string(lua_path)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    let new_table = to_lua_table(profiles, 0);
    let replacement = format!("TRP3_Profiles = {}\n", new_table);

    // 查找 TRP3_Profiles 赋值块并替换
    if let Some(start) = original.find("TRP3_Profiles") {
        if let Some(eq_pos) = original[start..].find('=') {
            let eq_index = start + eq_pos;
            if let Some(brace_pos_rel) = original[eq_index..].find('{') {
                let idx = eq_index + brace_pos_rel;
                let mut depth = 0usize;
                let mut end = None;
                for (i, ch) in original[idx..].char_indices() {
                    match ch {
                        '{' => depth += 1,
                        '}' => {
                            if depth > 0 {
                                depth -= 1;
                                if depth == 0 {
                                    end = Some(idx + i + 1);
                                    break;
                                }
                            }
                        }
                        _ => {}
                    }
                }

                if let Some(end_pos) = end {
                    let mut new_content = String::new();
                    new_content.push_str(&original[..eq_index]);
                    new_content.push_str("= ");
                    new_content.push_str(&replacement["TRP3_Profiles = ".len()..]); // reuse table+newline
                    new_content.push_str(&original[end_pos..]);

                    backup_file(lua_path)?;
                    fs::write(lua_path, new_content)
                        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;
                    return Ok(());
                }
            }
        }
    }

    // 如果未找到，追加到文件末尾
    let mut new_content = original;
    if !new_content.ends_with('\n') {
        new_content.push('\n');
    }
    new_content.push_str(&replacement);

    backup_file(lua_path)?;
    fs::write(lua_path, new_content)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    Ok(())
}
