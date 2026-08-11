use std::fs::{self, OpenOptions};
use std::io::Write;
use std::ops::Range;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

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

fn escape_lua_string(input: &str) -> String {
    let mut out = String::with_capacity(input.len());
    for ch in input.chars() {
        match ch {
            '\\' => out.push_str("\\\\"),
            '"' => out.push_str("\\\""),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{0008}' => out.push_str("\\b"),
            '\u{000C}' => out.push_str("\\f"),
            _ => {
                let code = ch as u32;
                if code <= 0x1F || code == 0x7F {
                    out.push('\\');
                    out.push_str(&format!("{:03}", code));
                } else {
                    out.push(ch);
                }
            }
        }
    }
    out
}

fn is_lua_identifier(key: &str) -> bool {
    let mut chars = key.chars();
    let first = match chars.next() {
        Some(c) => c,
        None => return false,
    };
    if !first.is_ascii_alphabetic() && first != '_' {
        return false;
    }
    chars.all(|c| c.is_ascii_alphanumeric() || c == '_')
}

fn is_lua_keyword(key: &str) -> bool {
    matches!(
        key,
        "and"
            | "break"
            | "do"
            | "else"
            | "elseif"
            | "end"
            | "false"
            | "for"
            | "function"
            | "goto"
            | "if"
            | "in"
            | "local"
            | "nil"
            | "not"
            | "or"
            | "repeat"
            | "return"
            | "then"
            | "true"
            | "until"
            | "while"
    )
}

fn backup_file(path: &PathBuf) -> Result<(), WriteError> {
    if path.exists() {
        let backup_path = path.with_extension("lua.rpbox_backup");
        fs::copy(path, &backup_path).map_err(|e| WriteError::BackupFailed(e.to_string()))?;
    }
    Ok(())
}

pub fn write_profile_to_local(lua_path: &PathBuf, raw_lua: &str) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    backup_file(lua_path)?;

    fs::write(lua_path, raw_lua).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    Ok(())
}

/// 将 serde_json::Value 转成 Lua table 字符串
pub fn to_lua_table(value: &Value, indent: usize) -> String {
    match value {
        Value::Null => "nil".to_string(),
        Value::Bool(b) => b.to_string(),
        Value::Number(n) => n.to_string(),
        Value::String(s) => {
            let escaped = escape_lua_string(s);
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
                // Lua identifiers must be valid and not keywords
                let is_valid_identifier = is_lua_identifier(k) && !is_lua_keyword(k);
                let key = if is_valid_identifier {
                    k.clone()
                } else {
                    // Escape special characters in key
                    let escaped_key = escape_lua_string(k);
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

#[derive(Debug, Clone, Copy)]
struct LuaTableRange {
    open_brace: usize,
    close_brace: usize,
}

#[derive(Debug, Clone)]
struct LuaProfileEntry {
    entry_start: usize,
    value_range: Range<usize>,
}

#[derive(Debug)]
struct LuaProfilesAnalysis {
    target: Option<LuaProfileEntry>,
    entry_count: usize,
    last_value_end: Option<usize>,
    trailing_comma: bool,
}

/// Reads only the selected profile from a TRP3 SavedVariables file.
///
/// The outer profile table is scanned without deserializing unrelated profiles, so comments,
/// long strings, and formatting in every non-target profile remain outside the write path.
pub fn read_trp3_profile_source(
    lua_path: &Path,
    profile_id: &str,
) -> Result<(Option<String>, Option<Value>), WriteError> {
    if !lua_path.exists() {
        return Ok((None, None));
    }

    let source = fs::read_to_string(lua_path).map_err(|error| {
        WriteError::WriteFailed(format!("读取 totalRP3.lua 失败: {}", error))
    })?;
    let table = locate_trp3_profiles_table(&source)?;
    let analysis = analyze_profiles_table(&source, table, profile_id)?;
    let profile = analysis
        .target
        .as_ref()
        .map(|entry| {
            let wrapped = format!(
                "RPBOX_Selected_Profile = {}",
                &source[entry.value_range.clone()]
            );
            crate::lua_parser::parse_variable_from_str(&wrapped, "RPBOX_Selected_Profile")
                .map_err(|error| {
                    WriteError::ParseFailed(format!("本地目标 profile 解析失败: {}", error))
                })
        })
        .transpose()?;

    Ok((Some(source), profile))
}

/// Replaces or inserts exactly one profile entry and atomically swaps the resulting file.
///
/// `expected_source` is compared again immediately before the atomic rename. If another process
/// changes the SavedVariables file after it was read, the write is aborted instead of overwriting
/// the newer file.
pub fn write_trp3_profile_precisely(
    lua_path: &Path,
    expected_source: Option<&str>,
    profile_id: &str,
    profile: &Value,
) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    let updated = match expected_source {
        Some(source) => upsert_profile_in_source(source, profile_id, profile)?,
        None => build_new_trp3_file(profile_id, profile),
    };

    atomic_replace_checked(lua_path, updated.as_bytes(), expected_source)
}

fn upsert_profile_in_source(
    source: &str,
    profile_id: &str,
    profile: &Value,
) -> Result<String, WriteError> {
    let table = locate_trp3_profiles_table(source)?;
    let analysis = analyze_profiles_table(source, table, profile_id)?;

    if let Some(entry) = analysis.target {
        let indent = line_indent_width(source, entry.entry_start);
        let serialized = to_lua_table(profile, indent);
        let mut updated = String::with_capacity(
            source.len() + serialized.len().saturating_sub(entry.value_range.len()),
        );
        updated.push_str(&source[..entry.value_range.start]);
        updated.push_str(&serialized);
        updated.push_str(&source[entry.value_range.end..]);
        return Ok(updated);
    }

    let newline = if source.contains("\r\n") { "\r\n" } else { "\n" };
    let escaped_profile_id = escape_lua_string(profile_id);
    let serialized = to_lua_table(profile, 2);
    let entry = format!("  [\"{}\"] = {},", escaped_profile_id, serialized);

    let mut base = source.to_string();
    let mut close_brace = table.close_brace;
    if analysis.entry_count > 0 && !analysis.trailing_comma {
        let last_value_end = analysis.last_value_end.ok_or_else(|| {
            WriteError::ParseFailed("TRP3_Profiles 缺少可验证的末尾条目".to_string())
        })?;
        base.insert(last_value_end, ',');
        close_brace += 1;
    }

    let insertion_point = start_of_closing_brace_line(&base, close_brace);
    let needs_leading_newline = !base[..insertion_point].ends_with('\n');
    let mut insertion = String::new();
    if needs_leading_newline {
        insertion.push_str(newline);
    }
    insertion.push_str(&entry);
    insertion.push_str(newline);

    let mut updated = String::with_capacity(base.len() + insertion.len());
    updated.push_str(&base[..insertion_point]);
    updated.push_str(&insertion);
    updated.push_str(&base[insertion_point..]);
    Ok(updated)
}

fn build_new_trp3_file(profile_id: &str, profile: &Value) -> String {
    let escaped_profile_id = escape_lua_string(profile_id);
    let serialized = to_lua_table(profile, 2);
    format!(
        "TRP3_Profiles = {{\n  [\"{0}\"] = {1},\n}}\nTRP3_Characters = {{}}\nTRP3_Configuration = {{\n  [\"default_profile_id\"] = \"{0}\"\n}}\nTRP3_Flyway = {{}}\n",
        escaped_profile_id, serialized
    )
}

fn locate_trp3_profiles_table(source: &str) -> Result<LuaTableRange, WriteError> {
    let bytes = source.as_bytes();
    let mut index = 0usize;
    let mut brace_depth = 0usize;
    let mut matches = Vec::new();

    while index < bytes.len() {
        if starts_line_comment(bytes, index) {
            index = skip_comment(source, index)?;
            continue;
        }
        if matches!(bytes[index], b'\'' | b'"') {
            index = skip_quoted_string(source, index)?;
            continue;
        }
        if long_bracket_open(bytes, index).is_some() {
            index = skip_long_bracket(source, index)?;
            continue;
        }

        match bytes[index] {
            b'{' => {
                brace_depth += 1;
                index += 1;
            }
            b'}' => {
                if brace_depth == 0 {
                    return Err(WriteError::ParseFailed(
                        "totalRP3.lua 存在未配对的右花括号".to_string(),
                    ));
                }
                brace_depth -= 1;
                index += 1;
            }
            byte if brace_depth == 0 && is_identifier_start(byte) => {
                let end = scan_identifier_end(bytes, index);
                if &source[index..end] != "TRP3_Profiles" {
                    index = end;
                    continue;
                }

                let mut cursor = end;
                skip_trivia(source, &mut cursor, bytes.len())?;
                if cursor >= bytes.len() || bytes[cursor] != b'=' {
                    index = end;
                    continue;
                }
                cursor += 1;
                skip_trivia(source, &mut cursor, bytes.len())?;
                if cursor >= bytes.len() || bytes[cursor] != b'{' {
                    return Err(WriteError::ParseFailed(
                        "TRP3_Profiles 必须直接赋值为 Lua table".to_string(),
                    ));
                }
                let close_brace = find_matching_brace(source, cursor)?;
                matches.push(LuaTableRange {
                    open_brace: cursor,
                    close_brace,
                });
                index = close_brace + 1;
            }
            _ => index += 1,
        }
    }

    match matches.as_slice() {
        [table] => Ok(*table),
        [] => Err(WriteError::ParseFailed(
            "totalRP3.lua 中未找到唯一的 TRP3_Profiles table，已中止写入".to_string(),
        )),
        _ => Err(WriteError::ParseFailed(
            "totalRP3.lua 中存在多个 TRP3_Profiles table，已中止写入".to_string(),
        )),
    }
}

fn analyze_profiles_table(
    source: &str,
    table: LuaTableRange,
    target_profile_id: &str,
) -> Result<LuaProfilesAnalysis, WriteError> {
    let bytes = source.as_bytes();
    let mut index = table.open_brace + 1;
    let mut target = None;
    let mut entry_count = 0usize;
    let mut last_value_end = None;
    let mut trailing_comma = false;

    loop {
        skip_trivia(source, &mut index, table.close_brace)?;
        if index >= table.close_brace {
            break;
        }

        let entry_start = index;
        let (key, after_key) = parse_profile_key(source, index, table.close_brace)?;
        index = after_key;
        skip_trivia(source, &mut index, table.close_brace)?;
        if index >= table.close_brace || bytes[index] != b'=' {
            return Err(WriteError::ParseFailed(
                "TRP3_Profiles 中存在无法识别的 profile 条目".to_string(),
            ));
        }
        index += 1;
        skip_trivia(source, &mut index, table.close_brace)?;
        if index >= table.close_brace || bytes[index] != b'{' {
            return Err(WriteError::ParseFailed(format!(
                "profile \"{}\" 不是 Lua table，已中止写入",
                key
            )));
        }

        let value_start = index;
        let value_end = find_matching_brace(source, value_start)? + 1;
        if value_end > table.close_brace {
            return Err(WriteError::ParseFailed(
                "profile 条目越过 TRP3_Profiles table 边界".to_string(),
            ));
        }
        entry_count += 1;
        last_value_end = Some(value_end);

        if key == target_profile_id {
            if target.is_some() {
                return Err(WriteError::ParseFailed(format!(
                    "profile \"{}\" 在本地文件中重复出现，已中止写入",
                    target_profile_id
                )));
            }
            target = Some(LuaProfileEntry {
                entry_start,
                value_range: value_start..value_end,
            });
        }

        index = value_end;
        skip_trivia(source, &mut index, table.close_brace)?;
        if index < table.close_brace && bytes[index] == b',' {
            trailing_comma = true;
            index += 1;
            continue;
        }

        trailing_comma = false;
        if index < table.close_brace {
            return Err(WriteError::ParseFailed(
                "TRP3_Profiles 的 profile 条目之间缺少逗号".to_string(),
            ));
        }
    }

    Ok(LuaProfilesAnalysis {
        target,
        entry_count,
        last_value_end,
        trailing_comma,
    })
}

fn parse_profile_key(
    source: &str,
    start: usize,
    limit: usize,
) -> Result<(String, usize), WriteError> {
    let bytes = source.as_bytes();
    if bytes[start] == b'[' && long_bracket_open(bytes, start).is_none() {
        let mut cursor = start + 1;
        skip_trivia(source, &mut cursor, limit)?;
        if cursor >= limit || !matches!(bytes[cursor], b'\'' | b'"') {
            return Err(WriteError::ParseFailed(
                "TRP3 profile ID 必须是字符串键".to_string(),
            ));
        }
        let literal_start = cursor;
        let literal_end = skip_quoted_string(source, cursor)?;
        let literal = &source[literal_start..literal_end];
        let wrapped = format!("RPBOX_Profile_Key = {}", literal);
        let key = crate::lua_parser::parse_variable_from_str(&wrapped, "RPBOX_Profile_Key")
            .map_err(|error| {
                WriteError::ParseFailed(format!("TRP3 profile ID 解析失败: {}", error))
            })?
            .as_str()
            .ok_or_else(|| {
                WriteError::ParseFailed("TRP3 profile ID 不是字符串".to_string())
            })?
            .to_string();
        cursor = literal_end;
        skip_trivia(source, &mut cursor, limit)?;
        if cursor >= limit || bytes[cursor] != b']' {
            return Err(WriteError::ParseFailed(
                "TRP3 profile ID 的方括号未闭合".to_string(),
            ));
        }
        return Ok((key, cursor + 1));
    }

    if is_identifier_start(bytes[start]) {
        let end = scan_identifier_end(bytes, start);
        return Ok((source[start..end].to_string(), end));
    }

    Err(WriteError::ParseFailed(
        "TRP3_Profiles 中存在不受支持的 profile 键".to_string(),
    ))
}

fn find_matching_brace(source: &str, open_brace: usize) -> Result<usize, WriteError> {
    let bytes = source.as_bytes();
    let mut index = open_brace;
    let mut depth = 0usize;

    while index < bytes.len() {
        if starts_line_comment(bytes, index) {
            index = skip_comment(source, index)?;
            continue;
        }
        if matches!(bytes[index], b'\'' | b'"') {
            index = skip_quoted_string(source, index)?;
            continue;
        }
        if long_bracket_open(bytes, index).is_some() {
            index = skip_long_bracket(source, index)?;
            continue;
        }

        match bytes[index] {
            b'{' => depth += 1,
            b'}' => {
                if depth == 0 {
                    return Err(WriteError::ParseFailed(
                        "Lua table 存在未配对的右花括号".to_string(),
                    ));
                }
                depth -= 1;
                if depth == 0 {
                    return Ok(index);
                }
            }
            _ => {}
        }
        index += 1;
    }

    Err(WriteError::ParseFailed(
        "Lua table 花括号未闭合，已中止写入".to_string(),
    ))
}

fn skip_trivia(source: &str, index: &mut usize, limit: usize) -> Result<(), WriteError> {
    let bytes = source.as_bytes();
    while *index < limit {
        if bytes[*index].is_ascii_whitespace() {
            *index += 1;
            continue;
        }
        if starts_line_comment(bytes, *index) {
            *index = skip_comment(source, *index)?;
            continue;
        }
        break;
    }
    Ok(())
}

fn starts_line_comment(bytes: &[u8], index: usize) -> bool {
    bytes.get(index) == Some(&b'-') && bytes.get(index + 1) == Some(&b'-')
}

fn skip_comment(source: &str, start: usize) -> Result<usize, WriteError> {
    let bytes = source.as_bytes();
    let content_start = start + 2;
    if long_bracket_open(bytes, content_start).is_some() {
        return skip_long_bracket(source, content_start);
    }
    Ok(bytes[content_start..]
        .iter()
        .position(|byte| *byte == b'\n')
        .map(|offset| content_start + offset + 1)
        .unwrap_or(bytes.len()))
}

fn skip_quoted_string(source: &str, start: usize) -> Result<usize, WriteError> {
    let bytes = source.as_bytes();
    let quote = bytes[start];
    let mut index = start + 1;
    while index < bytes.len() {
        match bytes[index] {
            b'\\' => {
                index += 1;
                if index >= bytes.len() {
                    break;
                }
                if bytes[index] == b'\r' && bytes.get(index + 1) == Some(&b'\n') {
                    index += 2;
                } else {
                    index += 1;
                }
            }
            byte if byte == quote => return Ok(index + 1),
            _ => index += 1,
        }
    }
    Err(WriteError::ParseFailed(
        "Lua 字符串未闭合，已中止写入".to_string(),
    ))
}

fn long_bracket_open(bytes: &[u8], start: usize) -> Option<(usize, usize)> {
    if bytes.get(start) != Some(&b'[') {
        return None;
    }
    let mut cursor = start + 1;
    let mut equals = 0usize;
    while bytes.get(cursor) == Some(&b'=') {
        equals += 1;
        cursor += 1;
    }
    if bytes.get(cursor) == Some(&b'[') {
        Some((equals, cursor + 1))
    } else {
        None
    }
}

fn skip_long_bracket(source: &str, start: usize) -> Result<usize, WriteError> {
    let bytes = source.as_bytes();
    let (equals, mut cursor) = long_bracket_open(bytes, start).ok_or_else(|| {
        WriteError::ParseFailed("Lua 长字符串起始标记无效".to_string())
    })?;
    while cursor < bytes.len() {
        if bytes[cursor] == b']' {
            let equals_end = cursor + 1 + equals;
            if equals_end < bytes.len()
                && bytes[cursor + 1..equals_end].iter().all(|byte| *byte == b'=')
                && bytes[equals_end] == b']'
            {
                return Ok(equals_end + 1);
            }
        }
        cursor += 1;
    }
    Err(WriteError::ParseFailed(
        "Lua 长字符串或长注释未闭合，已中止写入".to_string(),
    ))
}

fn is_identifier_start(byte: u8) -> bool {
    byte.is_ascii_alphabetic() || byte == b'_'
}

fn scan_identifier_end(bytes: &[u8], start: usize) -> usize {
    let mut end = start + 1;
    while end < bytes.len() && (bytes[end].is_ascii_alphanumeric() || bytes[end] == b'_') {
        end += 1;
    }
    end
}

fn line_indent_width(source: &str, position: usize) -> usize {
    let line_start = source[..position]
        .rfind('\n')
        .map(|index| index + 1)
        .unwrap_or(0);
    source.as_bytes()[line_start..position]
        .iter()
        .take_while(|byte| matches!(byte, b' ' | b'\t'))
        .map(|byte| if *byte == b'\t' { 2 } else { 1 })
        .sum()
}

fn start_of_closing_brace_line(source: &str, close_brace: usize) -> usize {
    let line_start = source[..close_brace]
        .rfind('\n')
        .map(|index| index + 1)
        .unwrap_or(close_brace);
    if source.as_bytes()[line_start..close_brace]
        .iter()
        .all(|byte| matches!(byte, b' ' | b'\t' | b'\r'))
    {
        line_start
    } else {
        close_brace
    }
}

fn atomic_replace_checked(
    path: &Path,
    content: &[u8],
    expected_source: Option<&str>,
) -> Result<(), WriteError> {
    let parent = path.parent().ok_or_else(|| {
        WriteError::WriteFailed("totalRP3.lua 路径缺少父目录".to_string())
    })?;
    fs::create_dir_all(parent)
        .map_err(|error| WriteError::WriteFailed(format!("创建目录失败: {}", error)))?;

    let file_name = path
        .file_name()
        .map(|name| name.to_string_lossy())
        .unwrap_or_else(|| "totalRP3.lua".into());
    let nonce = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos();
    let mut temporary_path = None;
    let mut temporary_file = None;
    for attempt in 0..16u8 {
        let candidate = parent.join(format!(
            ".{}.rpbox-tmp-{}-{}-{}",
            file_name,
            std::process::id(),
            nonce,
            attempt
        ));
        match OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&candidate)
        {
            Ok(file) => {
                temporary_path = Some(candidate);
                temporary_file = Some(file);
                break;
            }
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(error) => {
                return Err(WriteError::WriteFailed(format!(
                    "创建原子写入临时文件失败: {}",
                    error
                )))
            }
        }
    }

    let temporary_path = temporary_path.ok_or_else(|| {
        WriteError::WriteFailed("无法分配原子写入临时文件".to_string())
    })?;
    let mut temporary_file = temporary_file.expect("temporary file accompanies its path");
    if let Err(error) = temporary_file
        .write_all(content)
        .and_then(|_| temporary_file.flush())
        .and_then(|_| temporary_file.sync_all())
    {
        drop(temporary_file);
        let _ = fs::remove_file(&temporary_path);
        return Err(WriteError::WriteFailed(format!(
            "写入原子临时文件失败: {}",
            error
        )));
    }
    drop(temporary_file);

    let source_is_unchanged = match expected_source {
        Some(expected) => fs::read_to_string(path)
            .map(|current| current == expected)
            .unwrap_or(false),
        None => !path.exists(),
    };
    if !source_is_unchanged {
        let _ = fs::remove_file(&temporary_path);
        return Err(WriteError::WriteFailed(
            "写入前检测到 totalRP3.lua 已被其他程序修改；为保护其他人物卡，本次写入已取消"
                .to_string(),
        ));
    }

    if let Err(error) = replace_file_atomically(&temporary_path, path) {
        let _ = fs::remove_file(&temporary_path);
        return Err(WriteError::WriteFailed(format!(
            "原子替换 totalRP3.lua 失败: {}",
            error
        )));
    }

    let written = fs::read(path)
        .map_err(|error| WriteError::WriteFailed(format!("校验写入结果失败: {}", error)))?;
    if written != content {
        return Err(WriteError::WriteFailed(
            "totalRP3.lua 写后校验不一致，请使用写前快照回退".to_string(),
        ));
    }
    Ok(())
}

#[cfg(not(target_os = "windows"))]
fn replace_file_atomically(source: &Path, destination: &Path) -> std::io::Result<()> {
    fs::rename(source, destination)
}

#[cfg(target_os = "windows")]
fn replace_file_atomically(source: &Path, destination: &Path) -> std::io::Result<()> {
    use std::os::windows::ffi::OsStrExt;
    use std::ptr;

    #[link(name = "Kernel32")]
    extern "system" {
        fn ReplaceFileW(
            replaced_file_name: *const u16,
            replacement_file_name: *const u16,
            backup_file_name: *const u16,
            replace_flags: u32,
            exclude: *mut std::ffi::c_void,
            reserved: *mut std::ffi::c_void,
        ) -> i32;
    }

    // ReplaceFileW is the Windows primitive that atomically swaps an existing
    // destination. std::fs::rename intentionally refuses to overwrite files on
    // Windows, so it is only suitable when creating a brand-new save file.
    if !destination.exists() {
        return fs::rename(source, destination);
    }

    let destination_wide: Vec<u16> = destination
        .as_os_str()
        .encode_wide()
        .chain(std::iter::once(0))
        .collect();
    let source_wide: Vec<u16> = source
        .as_os_str()
        .encode_wide()
        .chain(std::iter::once(0))
        .collect();

    const REPLACEFILE_WRITE_THROUGH: u32 = 0x0000_0001;
    let replaced = unsafe {
        ReplaceFileW(
            destination_wide.as_ptr(),
            source_wide.as_ptr(),
            ptr::null(),
            REPLACEFILE_WRITE_THROUGH,
            ptr::null_mut(),
            ptr::null_mut(),
        )
    };

    if replaced == 0 {
        Err(std::io::Error::last_os_error())
    } else {
        Ok(())
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
            format!(
                "{{\n  [\"default_profile_id\"] = \"{}\"\n}}",
                default_profile_id
            )
        } else {
            "{}".to_string()
        };

        let full_content = format!(
            "TRP3_Profiles = {}\nTRP3_Characters = {{}}\nTRP3_Configuration = {}\nTRP3_Flyway = {{}}\n",
            new_table, config_table
        );
        fs::write(lua_path, full_content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;
        return Ok(());
    }

    let original =
        fs::read_to_string(lua_path).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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
    fs::write(lua_path, new_content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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
        fs::write(&tools_path, content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;
        return Ok(());
    }

    // 文件存在，替换 TRP3_Tools_DB 变量
    let original =
        fs::read_to_string(&tools_path).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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
    fs::write(&tools_path, new_content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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
        fs::write(&data_path, content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;
        return Ok(());
    }

    // 文件存在，替换 TRP3_Register 变量
    let original =
        fs::read_to_string(&data_path).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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
    fs::write(&data_path, new_content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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
        return Err(WriteError::WriteFailed(
            "totalRP3.lua 文件不存在".to_string(),
        ));
    }

    let original =
        fs::read_to_string(lua_path).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

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
    fs::write(lua_path, new_content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    Ok(())
}

/// 写入单个变量到 Lua 文件
fn write_variable_to_file(
    lua_path: &PathBuf,
    var_name: &str,
    data: &Value,
) -> Result<(), WriteError> {
    let new_table = to_lua_table(data, 0);

    // 如果文件不存在，跳过
    if !lua_path.exists() {
        return Ok(());
    }

    let original =
        fs::read_to_string(lua_path).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    let replacement = format!("{} = {}\n", var_name, new_table);

    // 查找变量赋值块并替换
    if let Some(start) = original.find(var_name) {
        let is_var_start =
            start == 0 || !original[..start].ends_with(|c: char| c.is_alphanumeric() || c == '_');
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
    fs::write(lua_path, new_content).map_err(|e| WriteError::WriteFailed(e.to_string()))?;

    Ok(())
}

/// 写入 TRP3 额外数据（角色绑定、伙伴、预设等）
pub fn write_extra_data(
    sv_dir: &PathBuf,
    extra_data: &Value,
    write_trp3_vars: bool,
    write_extended_vars: bool,
) -> Result<(), WriteError> {
    if is_wow_running() {
        return Err(WriteError::WowRunning);
    }

    let obj = extra_data
        .as_object()
        .ok_or_else(|| WriteError::ParseFailed("额外数据格式错误".to_string()))?;

    let trp3_path = sv_dir.join("totalRP3.lua");
    let extended_path = sv_dir.join("totalRP3_Extended.lua");

    // totalRP3.lua 中的变量
    let trp3_vars = [
        "TRP3_Characters",
        "TRP3_Companions",
        "TRP3_Presets",
        "TRP3_Notes",
        "TRP3_Flyway",
        "TRP3_MatureFilter",
        "TRP3_Colors",
        "TRP3_SavedAutomation",
    ];

    // totalRP3_Extended.lua 中的变量
    let extended_vars = [
        "TRP3_Exchange_DB",
        "TRP3_Stashes",
        "TRP3_Drop",
        "TRP3_Security",
        "TRP3_Extended_Flyway",
    ];

    if write_trp3_vars {
        for var_name in &trp3_vars {
            if let Some(data) = obj.get(*var_name) {
                write_variable_to_file(&trp3_path, var_name, data)?;
            }
        }
    }

    if write_extended_vars {
        for var_name in &extended_vars {
            if let Some(data) = obj.get(*var_name) {
                write_variable_to_file(&extended_path, var_name, data)?;
            }
        }
    }

    Ok(())
}

#[cfg(test)]
mod precise_profile_write_tests {
    use super::*;
    use serde_json::json;

    const MULTI_PROFILE_SOURCE: &str = r#"-- RPBox must preserve this header exactly
TRP3_Profiles = {
  ["untouched-a"] = {
    ["profileName"] = "Untouched A",
    ["biography"] = [==[A long string containing } braces, "quotes", and -- comments]==],
    -- Unknown local section { must not affect table matching }
    ["unknown"] = { ["nested"] = true, },
  },
  ["target-profile"] = {
    ["profileName"] = "Old target",
    ["unknownTopLevel"] = { ["keep"] = true, },
    ["player"] = {
      ["characteristics"] = { ["FN"] = "Old name", },
    },
  },
  ["untouched-b"] = {
    ["profileName"] = "Untouched B",
    ["payload"] = "literal { and } inside a string",
  },
}
TRP3_Characters = { ["untouched"] = true }
"#;

    #[test]
    fn replacing_one_profile_preserves_all_bytes_outside_its_value() {
        let table = locate_trp3_profiles_table(MULTI_PROFILE_SOURCE)
            .expect("the fixture should contain one profiles table");
        let before = analyze_profiles_table(MULTI_PROFILE_SOURCE, table, "target-profile")
            .expect("the target entry should be located")
            .target
            .expect("the target profile should exist");
        let replacement = json!({
            "profileName": "Updated target",
            "unknownTopLevel": { "keep": true },
            "player": { "characteristics": { "FN": "New name", "CH": "#69CCF0" } }
        });

        let updated = upsert_profile_in_source(
            MULTI_PROFILE_SOURCE,
            "target-profile",
            &replacement,
        )
        .expect("the target profile should be replaced surgically");
        let updated_table = locate_trp3_profiles_table(&updated)
            .expect("the updated profiles table should remain valid");
        let after = analyze_profiles_table(&updated, updated_table, "target-profile")
            .expect("the updated target should be located")
            .target
            .expect("the updated target should exist");

        assert_eq!(
            &updated[..after.value_range.start],
            &MULTI_PROFILE_SOURCE[..before.value_range.start]
        );
        assert_eq!(
            &updated[after.value_range.end..],
            &MULTI_PROFILE_SOURCE[before.value_range.end..]
        );
        assert!(updated.contains("[==[A long string containing } braces"));
        assert!(updated.contains("[\"payload\"] = \"literal { and } inside a string\""));
    }

    #[test]
    fn reading_selected_profile_does_not_parse_unrelated_long_strings() {
        let directory = tempfile::tempdir().expect("a temporary directory should be created");
        let path = directory.path().join("totalRP3.lua");
        fs::write(&path, MULTI_PROFILE_SOURCE).expect("the fixture should be written");

        let (source, profile) = read_trp3_profile_source(&path, "target-profile")
            .expect("only the target profile should need deserialization");

        assert_eq!(source.as_deref(), Some(MULTI_PROFILE_SOURCE));
        assert_eq!(profile.as_ref().unwrap()["profileName"], "Old target");
        assert_eq!(
            profile.as_ref().unwrap()["unknownTopLevel"]["keep"],
            true
        );
    }

    #[test]
    fn inserting_a_profile_keeps_existing_profile_content_intact() {
        let source = r#"TRP3_Profiles = {
  ["keep-me"] = {
    ["profileName"] = "Keep me",
    ["note"] = [=[unchanged } long text]=],
  }
}
TRP3_Configuration = {}
"#;
        let preserved_entry = r#"  ["keep-me"] = {
    ["profileName"] = "Keep me",
    ["note"] = [=[unchanged } long text]=],
  }"#;

        let updated = upsert_profile_in_source(
            source,
            "new-profile",
            &json!({ "profileName": "New profile", "player": { "characteristics": {} } }),
        )
        .expect("a new profile should be inserted safely");

        assert!(updated.contains(preserved_entry));
        assert!(updated.contains("[\"new-profile\"]"));
        let table = locate_trp3_profiles_table(&updated).expect("the table should remain valid");
        let inserted = analyze_profiles_table(&updated, table, "new-profile")
            .expect("the inserted entry should parse");
        assert!(inserted.target.is_some());
        assert_eq!(inserted.entry_count, 2);
    }

    #[test]
    fn ambiguous_tables_or_duplicate_target_profiles_abort() {
        let duplicate_table = "TRP3_Profiles = {}\nTRP3_Profiles = {}\n";
        let error = locate_trp3_profiles_table(duplicate_table)
            .expect_err("duplicate top-level tables must be rejected");
        assert!(error.to_string().contains("多个 TRP3_Profiles"));

        let duplicate_profile = r#"TRP3_Profiles = {
  ["same"] = {},
  ["same"] = {},
}
"#;
        let table = locate_trp3_profiles_table(duplicate_profile)
            .expect("the outer table itself is unique");
        let error = analyze_profiles_table(duplicate_profile, table, "same")
            .expect_err("duplicate target keys must be rejected");
        assert!(error.to_string().contains("重复出现"));
    }

    #[test]
    fn atomic_commit_refuses_to_overwrite_a_file_changed_after_read() {
        let directory = tempfile::tempdir().expect("a temporary directory should be created");
        let path = directory.path().join("totalRP3.lua");
        fs::write(&path, MULTI_PROFILE_SOURCE).expect("the fixture should be written");
        let externally_changed = "TRP3_Profiles = { [\"external\"] = {}, }\n";
        fs::write(&path, externally_changed).expect("the external edit should be simulated");

        let error = atomic_replace_checked(
            &path,
            b"TRP3_Profiles = { [\"target\"] = {}, }\n",
            Some(MULTI_PROFILE_SOURCE),
        )
        .expect_err("a stale source must never overwrite a newer file");

        assert!(error.to_string().contains("其他程序修改"));
        assert_eq!(
            fs::read_to_string(&path).expect("the changed file should remain readable"),
            externally_changed
        );
    }

    #[test]
    fn atomic_commit_replaces_an_existing_file() {
        let directory = tempfile::tempdir().expect("a temporary directory should be created");
        let path = directory.path().join("totalRP3.lua");
        fs::write(&path, MULTI_PROFILE_SOURCE).expect("the fixture should be written");
        let replacement = b"TRP3_Profiles = { [\"target\"] = {}, }\n";

        atomic_replace_checked(&path, replacement, Some(MULTI_PROFILE_SOURCE))
            .expect("an unchanged existing save file should be replaced atomically");

        assert_eq!(
            fs::read(&path).expect("the replaced file should remain readable"),
            replacement
        );
        assert_eq!(
            fs::read_dir(directory.path())
                .expect("the temporary directory should remain readable")
                .filter_map(Result::ok)
                .count(),
            1,
            "the atomic commit must not leave a temporary file behind"
        );
    }
}
