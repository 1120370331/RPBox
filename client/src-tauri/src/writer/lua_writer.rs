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
        Value::String(s) => {
            let escaped = s
                .replace('\\', "\\\\")
                .replace('"', "\\\"")
                .replace('\n', "\\n")
                .replace('\r', "\\r")
                .replace('\t', "\\t");
            format!("\"{}\"", escaped)
        }
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
                // Lua identifiers must start with letter or underscore, not digit
                let is_valid_identifier = k.chars().next().map_or(false, |c| c.is_ascii_alphabetic() || c == '_')
                    && k.chars().all(|c| c.is_ascii_alphanumeric() || c == '_');
                let key = if is_valid_identifier {
                    k.clone()
                } else {
                    // Escape special characters in key
                    let escaped_key = k
                        .replace('\\', "\\\\")
                        .replace('"', "\\\"")
                        .replace('\n', "\\n")
                        .replace('\r', "\\r")
                        .replace('\t', "\\t");
                    format!("[\"{}\"]", escaped_key)
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

    let new_table = to_lua_table(profiles, 0);

    // 如果文件不存在，创建父目录并写入完整的 Lua 文件
    if !lua_path.exists() {
        if let Some(parent) = lua_path.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| WriteError::WriteFailed(format!("创建目录失败: {}", e)))?;
        }

        // 找到一个 profile ID 作为默认 profile
        let default_profile_id = profiles
            .as_object()
            .and_then(|obj| {
                // 优先找名为"默认人物卡"的 profile
                for (id, profile) in obj {
                    if let Some(name) = profile.get("profileName").and_then(|v| v.as_str()) {
                        if name == "默认人物卡" || name == "Default profile" {
                            return Some(id.clone());
                        }
                    }
                }
                // 否则用第一个 profile
                obj.keys().next().cloned()
            })
            .unwrap_or_default();

        // 创建完整的 TRP3 SavedVariables 文件
        let config_table = if !default_profile_id.is_empty() {
            format!("{{\n  [\"default_profile_id\"] = \"{}\"\n}}", default_profile_id)
        } else {
            "{}".to_string()
        };

        let full_content = format!(
            "TRP3_Profiles = {}\nTRP3_Characters = {{}}\nTRP3_Configuration = {}\nTRP3_Flyway = {{}}\n",
            new_table, config_table
        );
        fs::write(lua_path, full_content)
            .map_err(|e| WriteError::WriteFailed(e.to_string()))?;
        return Ok(());
    }

    let original = fs::read_to_string(lua_path)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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

/// 写入 TRP3 Extended 道具数据库
pub fn write_tools_db(sv_dir: &PathBuf, tools_data: &Value) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    let tools_path = sv_dir.join("totalRP3_Extended.lua");
    let new_table = to_lua_table(tools_data, 0);

    // 如果文件不存在，创建新文件
    if !tools_path.exists() {
        fs::create_dir_all(sv_dir)
            .map_err(|e| WriteError::WriteFailed(format!("创建目录失败: {}", e)))?;

        let content = format!("TRP3_Tools_DB = {}\n", new_table);
        fs::write(&tools_path, content)
            .map_err(|e| WriteError::WriteFailed(e.to_string()))?;
        return Ok(());
    }

    // 文件存在，替换 TRP3_Tools_DB 变量
    let original = fs::read_to_string(&tools_path)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    let replacement = format!("TRP3_Tools_DB = {}\n", new_table);

    // 查找 TRP3_Tools_DB 赋值块并替换
    if let Some(start) = original.find("TRP3_Tools_DB") {
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
                    new_content.push_str(&replacement["TRP3_Tools_DB = ".len()..]);
                    new_content.push_str(&original[end_pos..]);

                    backup_file(&tools_path)?;
                    fs::write(&tools_path, new_content)
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

    backup_file(&tools_path)?;
    fs::write(&tools_path, new_content)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    Ok(())
}

/// 写入 TRP3 运行时数据 (他人人物卡等)
pub fn write_runtime_data(sv_dir: &PathBuf, runtime_data: &Value) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    let data_path = sv_dir.join("totalRP3_Data.lua");
    let new_table = to_lua_table(runtime_data, 0);

    // 如果文件不存在，创建新文件
    if !data_path.exists() {
        fs::create_dir_all(sv_dir)
            .map_err(|e| WriteError::WriteFailed(format!("创建目录失败: {}", e)))?;

        let content = format!("TRP3_Register = {}\n", new_table);
        fs::write(&data_path, content)
            .map_err(|e| WriteError::WriteFailed(e.to_string()))?;
        return Ok(());
    }

    // 文件存在，替换 TRP3_Register 变量
    let original = fs::read_to_string(&data_path)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    let replacement = format!("TRP3_Register = {}\n", new_table);

    // 查找 TRP3_Register 赋值块并替换
    if let Some(start) = original.find("TRP3_Register") {
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
                    new_content.push_str(&replacement["TRP3_Register = ".len()..]);
                    new_content.push_str(&original[end_pos..]);

                    backup_file(&data_path)?;
                    fs::write(&data_path, new_content)
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

    backup_file(&data_path)?;
    fs::write(&data_path, new_content)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    Ok(())
}

/// 写入 TRP3 配置数据
pub fn write_config(lua_path: &PathBuf, config_data: &Value) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    let new_table = to_lua_table(config_data, 0);

    // 文件必须存在（配置在 totalRP3.lua 中）
    if !lua_path.exists() {
        return Err(WriteError::WriteFailed("totalRP3.lua 文件不存在".to_string()));
    }

    let original = fs::read_to_string(lua_path)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    let replacement = format!("TRP3_Configuration = {}\n", new_table);

    // 查找 TRP3_Configuration 赋值块并替换
    if let Some(start) = original.find("TRP3_Configuration") {
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
                    new_content.push_str(&replacement["TRP3_Configuration = ".len()..]);
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

/// 写入单个变量到 Lua 文件
fn write_variable_to_file(lua_path: &PathBuf, var_name: &str, data: &Value) -> Result<(), WriteError> {
    let new_table = to_lua_table(data, 0);

    // 如果文件不存在，跳过
    if !lua_path.exists() {
        return Ok(());
    }

    let original = fs::read_to_string(lua_path)
        .map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    let replacement = format!("{} = {}\n", var_name, new_table);

    // 查找变量赋值块并替换
    if let Some(start) = original.find(var_name) {
        let is_var_start = start == 0 || !original[..start].ends_with(|c: char| c.is_alphanumeric() || c == '_');
        if is_var_start {
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
                        new_content.push_str(&original[..start]);
                        new_content.push_str(&replacement);
                        new_content.push_str(&original[end_pos..]);

                        backup_file(lua_path)?;
                        fs::write(lua_path, new_content)
                            .map_err(|e| WriteError::WriteFailed(e.to_string()))?;
                        return Ok(());
                    }
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

/// 写入 TRP3 额外数据（角色绑定、伙伴、预设等）
pub fn write_extra_data(sv_dir: &PathBuf, extra_data: &Value) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    let obj = extra_data.as_object().ok_or_else(|| {
        WriteError::ParseFailed("额外数据格式错误".to_string())
    })?;

    let trp3_path = sv_dir.join("totalRP3.lua");
    let extended_path = sv_dir.join("totalRP3_Extended.lua");

    // totalRP3.lua 中的变量
    let trp3_vars = [
        "TRP3_Characters", "TRP3_Companions", "TRP3_Presets",
        "TRP3_Notes", "TRP3_Flyway", "TRP3_MatureFilter",
        "TRP3_Colors", "TRP3_SavedAutomation",
    ];

    // totalRP3_Extended.lua 中的变量
    let extended_vars = [
        "TRP3_Exchange_DB", "TRP3_Stashes", "TRP3_Drop",
        "TRP3_Security", "TRP3_Extended_Flyway",
    ];

    for var_name in &trp3_vars {
        if let Some(data) = obj.get(*var_name) {
            write_variable_to_file(&trp3_path, var_name, data)?;
        }
    }

    for var_name in &extended_vars {
        if let Some(data) = obj.get(*var_name) {
            write_variable_to_file(&extended_path, var_name, data)?;
        }
    }

    Ok(())
}
