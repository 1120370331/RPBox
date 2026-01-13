use std::path::PathBuf;
use serde::{Deserialize, Serialize};
use serde_json::Value;

use crate::lua_parser;
use crate::wow_path;

/// TRP3角色卡信息（用于返回给前端）
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TRP3Info {
    #[serde(rename = "FN", skip_serializing_if = "Option::is_none")]
    pub first_name: Option<String>,
    #[serde(rename = "LN", skip_serializing_if = "Option::is_none")]
    pub last_name: Option<String>,
    #[serde(rename = "TI", skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
    #[serde(rename = "IC", skip_serializing_if = "Option::is_none")]
    pub icon: Option<String>,
    #[serde(rename = "CH", skip_serializing_if = "Option::is_none")]
    pub color: Option<String>,
}

/// 聊天记录发送者
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatSender {
    #[serde(rename = "gameID")]
    pub game_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trp3: Option<TRP3Info>,
}

/// 聊天记录（返回给前端的统一格式）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatRecord {
    pub timestamp: i64,
    pub channel: String,
    pub sender: ChatSender,
    pub content: String,
    /// 消息标记: P(Player), N(NPC), B(Background)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mark: Option<String>,
    /// NPC名字（仅NPC消息）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub npc: Option<String>,
    /// NPC说话类型: say/yell/whisper（仅NPC消息）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nt: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AccountChatLogs {
    pub account_id: String,
    pub last_update: Option<i64>,
    pub record_count: i32,
    pub records: Vec<ChatRecord>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncState {
    pub addon: Option<AddonState>,
    pub client: Option<ClientState>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddonState {
    #[serde(rename = "lastUpdate")]
    pub last_update: Option<i64>,
    #[serde(rename = "recordCount")]
    pub record_count: Option<i32>,
    pub version: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClientState {
    #[serde(rename = "lastSync")]
    pub last_sync: Option<i64>,
    #[serde(rename = "syncedCount")]
    pub synced_count: Option<i32>,
    #[serde(rename = "clearedBefore")]
    pub cleared_before: Option<i64>,
}

/// 扫描所有账号的聊天记录
pub fn scan_chat_logs(wow_path: &str) -> Result<Vec<AccountChatLogs>, String> {
    eprintln!("[RPBox] scan_chat_logs 输入路径: {}", wow_path);

    let normalized = wow_path::normalize_wow_path(wow_path)
        .ok_or_else(|| "无效的WoW路径".to_string())?;

    eprintln!("[RPBox] 规范化后路径: {:?}", normalized);

    let account_root = normalized.join("Account");
    eprintln!("[RPBox] Account目录: {:?}, 存在: {}", account_root, account_root.exists());

    if !account_root.exists() {
        return Err("WTF/Account 目录不存在".to_string());
    }

    let mut results = Vec::new();
    let entries = std::fs::read_dir(&account_root)
        .map_err(|e| format!("读取目录失败: {}", e))?;

    for entry in entries.flatten() {
        if !entry.path().is_dir() {
            continue;
        }
        let account_id = entry.file_name().to_string_lossy().to_string();
        if account_id == "SavedVariables" {
            continue;
        }

        match parse_account_chat_logs(&entry.path(), &account_id) {
            Ok(logs) => results.push(logs),
            Err(e) => eprintln!("[RPBox] 解析账号 {} 失败: {}", account_id, e),
        }
    }

    Ok(results)
}

/// 解析单个账号的聊天记录
fn parse_account_chat_logs(account_path: &PathBuf, account_id: &str) -> Result<AccountChatLogs, String> {
    // WoW 把所有 SavedVariables 合并到一个以插件名命名的文件中
    let addon_file_path = account_path.join("SavedVariables").join("RPBox_Addon.lua");
    // 兼容旧的单独文件格式
    let chat_log_path = account_path.join("SavedVariables").join("RPBox_ChatLog.lua");
    let sync_path = account_path.join("SavedVariables").join("RPBox_Sync.lua");
    let profile_cache_path = account_path.join("SavedVariables").join("RPBox_ProfileCache.lua");

    // 优先使用合并文件
    let use_addon_file = addon_file_path.exists();
    eprintln!("[RPBox] 账号 {}: addon文件存在={}, 路径={:?}", account_id, use_addon_file, addon_file_path);

    let mut result = AccountChatLogs {
        account_id: account_id.to_string(),
        last_update: None,
        record_count: 0,
        records: Vec::new(),
    };

    // 读取同步状态
    let sync_file = if use_addon_file { &addon_file_path } else { &sync_path };
    if sync_file.exists() {
        if let Ok(sync_data) = lua_parser::parse_variable(sync_file, "RPBox_Sync") {
            if let Ok(state) = serde_json::from_value::<SyncState>(sync_data) {
                if let Some(addon) = state.addon {
                    result.last_update = addon.last_update;
                }
            }
        }
    }

    // 读取聊天记录
    let chat_file = if use_addon_file { &addon_file_path } else { &chat_log_path };
    if !chat_file.exists() {
        eprintln!("[RPBox] 账号 {}: 聊天文件不存在", account_id);
        return Ok(result);
    }

    eprintln!("[RPBox] 账号 {}: 开始解析聊天记录...", account_id);
    let chat_data = match lua_parser::parse_variable(chat_file, "RPBox_ChatLog") {
        Ok(data) => {
            let dtype = if data.is_object() { "object" } else if data.is_array() { "array" } else { "other" };
            eprintln!("[RPBox] 账号 {}: 解析成功, 数据类型={}", account_id, dtype);
            data
        }
        Err(e) => {
            eprintln!("[RPBox] 账号 {}: 解析失败: {}", account_id, e);
            return Err(e.to_string());
        }
    };

    // 读取角色卡缓存（新格式需要）
    let cache_file = if use_addon_file { &addon_file_path } else { &profile_cache_path };
    let profile_cache = if cache_file.exists() {
        lua_parser::parse_variable(cache_file, "RPBox_ProfileCache")
            .unwrap_or(Value::Object(Default::default()))
    } else {
        Value::Object(Default::default())
    };

    result.records = parse_chat_records(&chat_data, &profile_cache);
    result.record_count = result.records.len() as i32;

    Ok(result)
}

/// 解析聊天记录数据结构
fn parse_chat_records(data: &Value, profile_cache: &Value) -> Vec<ChatRecord> {
    let mut records = Vec::new();

    let obj = match data.as_object() {
        Some(o) => o,
        None => {
            eprintln!("[RPBox] parse_chat_records: 数据不是object");
            return records;
        }
    };

    eprintln!("[RPBox] parse_chat_records: 日期数量={}", obj.len());

    // 遍历日期
    for (date, hours) in obj {
        let hours_type = if hours.is_object() { "object" }
            else if hours.is_array() { "array" }
            else if hours.is_string() { "string" }
            else if hours.is_number() { "number" }
            else { "other" };
        eprintln!("[RPBox] 日期: {}, hours类型: {}", date, hours_type);

        // 如果是array，打印内容看看
        if hours.is_array() {
            eprintln!("[RPBox]   hours内容(array): {:?}", hours);
        }

        let hours_obj = match hours.as_object() {
            Some(o) => o,
            None => {
                eprintln!("[RPBox]   hours不是object, 跳过");
                continue;
            }
        };

        eprintln!("[RPBox]   小时数量={}", hours_obj.len());
        // 遍历小时
        for (hour, entries) in hours_obj {
            eprintln!("[RPBox]   小时: {}", hour);
            let entries_arr = match entries.as_array() {
                Some(a) => a,
                None => {
                    eprintln!("[RPBox]     entries不是array, 类型={:?}", entries);
                    continue;
                }
            };

            eprintln!("[RPBox]     记录数量={}", entries_arr.len());
            // 遍历记录
            for entry in entries_arr {
                if let Some(record) = parse_single_record(entry, profile_cache) {
                    records.push(record);
                } else {
                    eprintln!("[RPBox]     解析记录失败: {:?}", entry);
                }
            }
        }
    }

    // 按时间戳排序
    records.sort_by_key(|r| r.timestamp);
    records
}

/// 解析单条聊天记录（支持新旧两种格式）
fn parse_single_record(entry: &Value, profile_cache: &Value) -> Option<ChatRecord> {
    let obj = entry.as_object()?;

    // 尝试新格式 (t, c, m, s, mk, ref, npc)
    // 时间戳可能是整数或浮点数
    let t = obj.get("t").and_then(|v| {
        v.as_i64().or_else(|| v.as_f64().map(|f| f as i64))
    });
    if let Some(t) = t {
        let channel = obj.get("c").and_then(|v| v.as_str()).unwrap_or("").to_string();
        let content = obj.get("m").and_then(|v| v.as_str()).unwrap_or("").to_string();
        let sender = obj.get("s").and_then(|v| v.as_str()).unwrap_or("").to_string();
        let mark = obj.get("mk").and_then(|v| v.as_str()).map(|s| s.to_string());
        let npc = obj.get("npc").and_then(|v| v.as_str()).map(|s| s.to_string());
        let nt = obj.get("nt").and_then(|v| v.as_str()).map(|s| s.to_string());
        let profile_ref = obj.get("ref").and_then(|v| v.as_str());

        // 从ProfileCache获取TRP3信息
        let trp3 = profile_ref.and_then(|ref_id| {
            eprintln!("[RPBox] 查找ProfileCache: ref={}", ref_id);
            let result = profile_cache.get(ref_id).and_then(|p| {
                eprintln!("[RPBox]   找到Profile: {:?}", p);
                Some(TRP3Info {
                    first_name: p.get("FN").and_then(|v| v.as_str()).map(|s| s.to_string()),
                    last_name: p.get("LN").and_then(|v| v.as_str()).map(|s| s.to_string()),
                    title: p.get("TI").and_then(|v| v.as_str()).map(|s| s.to_string()),
                    icon: p.get("IC").and_then(|v| v.as_str()).map(|s| s.to_string()),
                    color: p.get("CH").and_then(|v| v.as_str()).map(|s| s.to_string()),
                })
            });
            if result.is_none() {
                eprintln!("[RPBox]   未找到Profile");
            }
            result
        });

        return Some(ChatRecord {
            timestamp: t,
            channel,
            content,
            sender: ChatSender { game_id: sender, trp3 },
            mark,
            npc,
            nt,
        });
    }

    // 旧格式 (timestamp, channel, content, sender)
    let timestamp = obj.get("timestamp")?.as_i64()?;
    let channel = obj.get("channel")?.as_str()?.to_string();
    let content = obj.get("content")?.as_str()?.to_string();

    let sender_obj = obj.get("sender")?.as_object()?;
    let game_id = sender_obj.get("gameID")?.as_str()?.to_string();

    let trp3 = sender_obj.get("trp3").and_then(|t| {
        serde_json::from_value::<TRP3Info>(t.clone()).ok()
    });

    Some(ChatRecord {
        timestamp,
        channel,
        content,
        sender: ChatSender { game_id, trp3 },
        mark: None,
        npc: None,
        nt: None,
    })
}
