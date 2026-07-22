use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::path::PathBuf;

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

/// 收听者信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Listener {
    #[serde(rename = "gameID")]
    pub game_id: String,
    #[serde(rename = "profileID", skip_serializing_if = "Option::is_none")]
    pub profile_id: Option<String>,
    /// RPBox v2 immutable profile snapshot key.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub snapshot_id: Option<String>,
    /// Resolved listener snapshot, when the v2 log captured one.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub snapshot: Option<ProfileSnapshot>,
}

/// Immutable, compact TRP3 identity captured at record time.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ProfileSnapshot {
    #[serde(rename = "ref", skip_serializing_if = "Option::is_none")]
    pub ref_id: Option<String>,
    #[serde(rename = "gameID", skip_serializing_if = "Option::is_none")]
    pub game_id: Option<String>,
    #[serde(rename = "n", skip_serializing_if = "Option::is_none")]
    pub display_name: Option<String>,
    #[serde(rename = "pn", skip_serializing_if = "Option::is_none")]
    pub profile_name: Option<String>,
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
    #[serde(skip_serializing_if = "Option::is_none")]
    pub at: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub rev: Option<i64>,
}

/// One side of an identity transition.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct IdentityEndpoint {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ref_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub snapshot_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub display_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub profile_name: Option<String>,
}

/// A local exact or remote observed TRP3 profile transition.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct IdentityEvent {
    pub kind: String,
    pub certainty: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub from: Option<IdentityEndpoint>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub to: Option<IdentityEndpoint>,
}

/// 聊天记录（返回给前端的统一格式）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatRecord {
    /// Stable account-scoped key used by selection, deduplication and archiving.
    pub record_key: String,
    pub account_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub record_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub schema_version: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sequence: Option<i64>,
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
    /// TRP3 profile ref ID
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ref_id: Option<String>,
    /// Immutable profile snapshot reference and resolved historical identity.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub profile_snapshot_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub profile_snapshot: Option<ProfileSnapshot>,
    /// How reliable the displayed historical identity is.
    pub identity_source: String,
    /// Profile switch/update timeline event (`mk = S`).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub event: Option<IdentityEvent>,
    /// 完整的TRP3 profile JSON（用于服务端存储）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub raw_profile: Option<String>,
    /// 收听者列表（新增字段，向前兼容）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub listeners: Option<Vec<Listener>>,
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

fn sanitize_npc_whisper_content(content: &str) -> String {
    let mut start = 0;
    for (idx, ch) in content.char_indices() {
        if ch == '\u{FFFD}' {
            start = idx + ch.len_utf8();
            continue;
        }
        break;
    }

    content[start..].trim_start().to_string()
}

fn value_as_i64(value: Option<&Value>) -> Option<i64> {
    value.and_then(|v| v.as_i64().or_else(|| v.as_f64().map(|n| n as i64)))
}

fn value_as_string(value: Option<&Value>) -> Option<String> {
    value.and_then(|v| {
        v.as_str().map(ToOwned::to_owned).or_else(|| {
            v.as_i64()
                .map(|n| n.to_string())
                .or_else(|| v.as_f64().map(|n| format!("{}", n as i64)))
        })
    })
}

fn non_empty(value: Option<String>) -> Option<String> {
    value.filter(|s| !s.trim().is_empty())
}

fn profile_snapshot_from_value(value: &Value) -> Option<ProfileSnapshot> {
    let obj = value.as_object()?;
    let first_name = non_empty(value_as_string(obj.get("FN")));
    let last_name = non_empty(value_as_string(obj.get("LN")));
    let explicit_name = non_empty(value_as_string(obj.get("n")));
    let display_name = explicit_name.or_else(|| match (&first_name, &last_name) {
        (Some(first), Some(last)) => Some(format!("{} {}", first, last)),
        (Some(first), None) => Some(first.clone()),
        _ => None,
    });

    Some(ProfileSnapshot {
        ref_id: non_empty(value_as_string(obj.get("ref"))),
        game_id: non_empty(value_as_string(obj.get("gameID"))),
        display_name,
        profile_name: non_empty(value_as_string(obj.get("pn"))),
        first_name,
        last_name,
        title: non_empty(value_as_string(obj.get("TI"))),
        icon: non_empty(value_as_string(obj.get("IC"))),
        color: non_empty(value_as_string(obj.get("CH"))),
        at: value_as_i64(obj.get("at")),
        rev: value_as_i64(obj.get("rev")),
    })
}

fn trp3_from_snapshot(snapshot: &ProfileSnapshot) -> TRP3Info {
    TRP3Info {
        first_name: snapshot
            .first_name
            .clone()
            .or_else(|| snapshot.display_name.clone()),
        last_name: snapshot.last_name.clone(),
        title: snapshot.title.clone(),
        icon: snapshot.icon.clone(),
        color: snapshot.color.clone(),
    }
}

fn profile_from_cache(profile_cache: &Value, ref_id: &str) -> (Option<TRP3Info>, Option<String>) {
    let Some(profile) = profile_cache.get(ref_id) else {
        return (None, None);
    };

    let trp3 = TRP3Info {
        first_name: profile
            .get("FN")
            .and_then(|v| v.as_str())
            .map(ToOwned::to_owned),
        last_name: profile
            .get("LN")
            .and_then(|v| v.as_str())
            .map(ToOwned::to_owned),
        title: profile
            .get("TI")
            .and_then(|v| v.as_str())
            .map(ToOwned::to_owned),
        icon: profile
            .get("IC")
            .and_then(|v| v.as_str())
            .map(ToOwned::to_owned),
        color: profile
            .get("CH")
            .and_then(|v| v.as_str())
            .map(ToOwned::to_owned),
    };

    (Some(trp3), serde_json::to_string(profile).ok())
}

fn resolve_snapshot(profile_snapshots: &Value, snapshot_id: &str) -> Option<ProfileSnapshot> {
    profile_snapshots
        .get(snapshot_id)
        .and_then(profile_snapshot_from_value)
}

fn parse_identity_endpoint(
    value: Option<&Value>,
    profile_snapshots: &Value,
) -> Option<IdentityEndpoint> {
    let obj = value?.as_object()?;
    let snapshot_id = non_empty(value_as_string(obj.get("ps")));
    let snapshot = snapshot_id
        .as_deref()
        .and_then(|id| resolve_snapshot(profile_snapshots, id));

    Some(IdentityEndpoint {
        ref_id: non_empty(value_as_string(obj.get("ref")))
            .or_else(|| snapshot.as_ref().and_then(|s| s.ref_id.clone())),
        snapshot_id,
        display_name: non_empty(value_as_string(obj.get("n")))
            .or_else(|| snapshot.as_ref().and_then(|s| s.display_name.clone())),
        profile_name: non_empty(value_as_string(obj.get("pn")))
            .or_else(|| snapshot.as_ref().and_then(|s| s.profile_name.clone())),
    })
}

fn parse_identity_event(value: Option<&Value>, profile_snapshots: &Value) -> Option<IdentityEvent> {
    let obj = value?.as_object()?;
    let kind = non_empty(value_as_string(obj.get("kind")))?;
    Some(IdentityEvent {
        kind,
        certainty: non_empty(value_as_string(obj.get("certainty")))
            .unwrap_or_else(|| "observed".to_string()),
        from: parse_identity_endpoint(obj.get("from"), profile_snapshots),
        to: parse_identity_endpoint(obj.get("to"), profile_snapshots),
    })
}

fn stable_record_key(account_id: &str, identity: &str) -> String {
    let digest = md5::compute(format!("{}|{}", account_id, identity));
    format!("rpbox-{:x}", digest)
}

/// 扫描所有账号的聊天记录
pub fn scan_chat_logs(wow_path: &str) -> Result<Vec<AccountChatLogs>, String> {
    // eprintln!("[RPBox] scan_chat_logs 输入路径: {}", wow_path);

    let normalized =
        wow_path::normalize_wow_path(wow_path).ok_or_else(|| "无效的WoW路径".to_string())?;

    // eprintln!("[RPBox] 规范化后路径: {:?}", normalized);

    let account_root = normalized.join("Account");
    // eprintln!("[RPBox] Account目录: {:?}, 存在: ", account_root, account_root.exists());

    if !account_root.exists() {
        return Err("WTF/Account 目录不存在".to_string());
    }

    let mut results = Vec::new();
    let entries = std::fs::read_dir(&account_root).map_err(|e| format!("读取目录失败: {}", e))?;

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
fn parse_account_chat_logs(
    account_path: &PathBuf,
    account_id: &str,
) -> Result<AccountChatLogs, String> {
    // WoW 把所有 SavedVariables 合并到一个以插件名命名的文件中
    let addon_file_path = account_path.join("SavedVariables").join("RPBox_Addon.lua");
    // 兼容旧的单独文件格式
    let chat_log_path = account_path
        .join("SavedVariables")
        .join("RPBox_ChatLog.lua");
    let sync_path = account_path.join("SavedVariables").join("RPBox_Sync.lua");
    let profile_cache_path = account_path
        .join("SavedVariables")
        .join("RPBox_ProfileCache.lua");
    let profile_snapshots_path = account_path
        .join("SavedVariables")
        .join("RPBox_ProfileSnapshots.lua");

    // 优先使用合并文件
    let use_addon_file = addon_file_path.exists();
    // eprintln!("[RPBox] 账号 {}: addon文件存在={}, 路径={:?}", account_id, use_addon_file, addon_file_path);

    let mut result = AccountChatLogs {
        account_id: account_id.to_string(),
        last_update: None,
        record_count: 0,
        records: Vec::new(),
    };

    // 读取同步状态
    let sync_file = if use_addon_file {
        &addon_file_path
    } else {
        &sync_path
    };
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
    let chat_file = if use_addon_file {
        &addon_file_path
    } else {
        &chat_log_path
    };
    if !chat_file.exists() {
        // eprintln!("[RPBox] 账号 {}: 聊天文件不存在", account_id);
        return Ok(result);
    }

    // eprintln!("[RPBox] 账号 {}: 开始解析聊天记录...", account_id);
    let chat_data = match lua_parser::parse_variable(chat_file, "RPBox_ChatLog") {
        Ok(data) => data,
        Err(e) => {
            eprintln!("[RPBox] 账号 {}: 解析失败: {}", account_id, e);
            return Err(e.to_string());
        }
    };

    // 读取角色卡缓存（新格式需要）
    let cache_file = if use_addon_file {
        &addon_file_path
    } else {
        &profile_cache_path
    };
    let profile_cache = if cache_file.exists() {
        lua_parser::parse_variable(cache_file, "RPBox_ProfileCache")
            .unwrap_or(Value::Object(Default::default()))
    } else {
        Value::Object(Default::default())
    };

    // v2 stores immutable profile history alongside the chat log. Keep the
    // standalone path as a compatibility fallback for early development builds.
    let snapshots_file = if use_addon_file {
        &addon_file_path
    } else {
        &profile_snapshots_path
    };
    let profile_snapshots = if snapshots_file.exists() {
        lua_parser::parse_variable(snapshots_file, "RPBox_ProfileSnapshots")
            .unwrap_or(Value::Object(Default::default()))
    } else {
        Value::Object(Default::default())
    };

    result.records = parse_chat_records(&chat_data, &profile_cache, &profile_snapshots, account_id);
    result.record_count = result.records.len() as i32;

    Ok(result)
}

/// 解析聊天记录数据结构
fn parse_chat_records(
    data: &Value,
    profile_cache: &Value,
    profile_snapshots: &Value,
    account_id: &str,
) -> Vec<ChatRecord> {
    let mut records = Vec::new();

    let obj = match data.as_object() {
        Some(o) => o,
        None => {
            // eprintln!("[RPBox] parse_chat_records: 数据不是object");
            return records;
        }
    };

    // eprintln!("[RPBox] parse_chat_records: 日期数量={}", obj.len());

    // 遍历日期
    for (date, hours) in obj {
        let hours_obj = match hours.as_object() {
            Some(o) => o,
            None => {
                // eprintln!("[RPBox]   hours不是object, 跳过");
                continue;
            }
        };

        // eprintln!("[RPBox]   小时数量={}", hours_obj.len());
        // 遍历小时
        for (hour, entries) in hours_obj {
            // eprintln!("[RPBox]   小时: ", hour);
            let entries_arr = match entries.as_array() {
                Some(a) => a,
                None => {
                    // eprintln!("[RPBox]     entries不是array, 类型={:?}", entries);
                    continue;
                }
            };

            // eprintln!("[RPBox]     记录数量={}", entries_arr.len());
            // 遍历记录
            for (ordinal, entry) in entries_arr.iter().enumerate() {
                if let Some(record) = parse_single_record(
                    entry,
                    profile_cache,
                    profile_snapshots,
                    account_id,
                    date,
                    hour,
                    ordinal,
                ) {
                    records.push(record);
                } else {
                    // eprintln!("[RPBox]     解析记录失败: {:?}", entry);
                }
            }
        }
    }

    // 按时间戳排序
    records.sort_by(|a, b| {
        a.timestamp
            .cmp(&b.timestamp)
            .then_with(|| a.sequence.cmp(&b.sequence))
            .then_with(|| a.record_key.cmp(&b.record_key))
    });
    records
}

/// 解析单条聊天记录（支持新旧两种格式）
fn parse_single_record(
    entry: &Value,
    profile_cache: &Value,
    profile_snapshots: &Value,
    account_id: &str,
    date: &str,
    hour: &str,
    ordinal: usize,
) -> Option<ChatRecord> {
    let obj = entry.as_object()?;

    // Compact addon format (v1 and v2). Lua numbers arrive as JSON floats.
    let t = value_as_i64(obj.get("t"));
    if let Some(t) = t {
        let channel = value_as_string(obj.get("c")).unwrap_or_default();
        let mut content = value_as_string(obj.get("m")).unwrap_or_default();
        let sender = value_as_string(obj.get("s")).unwrap_or_default();
        let mark = non_empty(value_as_string(obj.get("mk")));
        let npc = non_empty(value_as_string(obj.get("npc")));
        let nt = non_empty(value_as_string(obj.get("nt")));
        let record_id = non_empty(value_as_string(obj.get("id")));
        let schema_version = value_as_i64(obj.get("sv")).map(|n| n as i32);
        let session_id = non_empty(value_as_string(obj.get("sid")));
        let sequence = value_as_i64(obj.get("seq"));
        let profile_snapshot_id = non_empty(value_as_string(obj.get("ps")));
        let profile_snapshot = profile_snapshot_id
            .as_deref()
            .and_then(|id| resolve_snapshot(profile_snapshots, id));
        let explicit_ref = non_empty(value_as_string(obj.get("ref")));
        let profile_ref = explicit_ref
            .clone()
            .or_else(|| profile_snapshot.as_ref().and_then(|s| s.ref_id.clone()));
        let event = parse_identity_event(obj.get("ev"), profile_snapshots);

        if mark.as_deref() == Some("N") && nt.as_deref() == Some("whisper") {
            content = sanitize_npc_whisper_content(&content);
        }

        // Historical snapshot always wins. The mutable cache is only an
        // explicitly labelled display fallback for v1 logs that never captured
        // history. raw_profile remains the complete cache payload because the
        // compact snapshot intentionally omits about/misc and other TRP3 fields.
        let (trp3, raw_profile, identity_source) = if let Some(snapshot) = &profile_snapshot {
            let raw = profile_ref
                .as_deref()
                .and_then(|ref_id| profile_from_cache(profile_cache, ref_id).1);
            (
                Some(trp3_from_snapshot(snapshot)),
                raw,
                "snapshot".to_string(),
            )
        } else if let Some(ref_id) = profile_ref.as_deref() {
            let (trp3, raw) = profile_from_cache(profile_cache, ref_id);
            let source = if trp3.is_some() {
                "legacy_cache"
            } else {
                "game_id"
            };
            (trp3, raw, source.to_string())
        } else {
            (None, None, "game_id".to_string())
        };

        // 解析收听者列表
        let listeners = obj
            .get("listeners")
            .and_then(|v| v.as_array())
            .map(|arr| {
                arr.iter()
                    .filter_map(|item| {
                        let obj = item.as_object()?;
                        let snapshot_id = non_empty(value_as_string(obj.get("ps")));
                        let snapshot = snapshot_id
                            .as_deref()
                            .and_then(|id| resolve_snapshot(profile_snapshots, id));
                        let game_id = non_empty(value_as_string(obj.get("gameID")))
                            .or_else(|| snapshot.as_ref().and_then(|s| s.game_id.clone()))?;
                        let profile_id = non_empty(value_as_string(obj.get("profileID")))
                            .or_else(|| non_empty(value_as_string(obj.get("ref"))))
                            .or_else(|| snapshot.as_ref().and_then(|s| s.ref_id.clone()));
                        Some(Listener {
                            game_id,
                            profile_id,
                            snapshot_id,
                            snapshot,
                        })
                    })
                    .collect::<Vec<_>>()
            })
            .filter(|v| !v.is_empty());

        let key_identity = if let Some(id) = &record_id {
            format!("v2|{}|{}", session_id.as_deref().unwrap_or(""), id)
        } else if let (Some(sid), Some(seq)) = (&session_id, sequence) {
            format!("v2-seq|{}|{}", sid, seq)
        } else {
            format!(
                "compact|{}|{}|{}|{}|{}|{}|{}|{}",
                date,
                hour,
                ordinal,
                t,
                channel,
                sender,
                content,
                mark.as_deref().unwrap_or("")
            )
        };

        return Some(ChatRecord {
            record_key: stable_record_key(account_id, &key_identity),
            account_id: account_id.to_string(),
            record_id,
            schema_version,
            session_id,
            sequence,
            timestamp: t,
            channel,
            content,
            sender: ChatSender {
                game_id: sender,
                trp3,
            },
            mark,
            npc,
            nt,
            ref_id: profile_ref,
            profile_snapshot_id,
            profile_snapshot,
            identity_source,
            event,
            raw_profile,
            listeners,
        });
    }

    // 旧格式 (timestamp, channel, content, sender)
    let timestamp = value_as_i64(obj.get("timestamp"))?;
    let channel = obj.get("channel")?.as_str()?.to_string();
    let content = obj.get("content")?.as_str()?.to_string();

    let sender_obj = obj.get("sender")?.as_object()?;
    let game_id = sender_obj.get("gameID")?.as_str()?.to_string();

    let trp3 = sender_obj
        .get("trp3")
        .and_then(|t| serde_json::from_value::<TRP3Info>(t.clone()).ok());
    let identity_source = if trp3.is_some() {
        "embedded_legacy"
    } else {
        "game_id"
    }
    .to_string();

    Some(ChatRecord {
        record_key: stable_record_key(
            account_id,
            &format!(
                "legacy|{}|{}|{}|{}|{}|{}|{}",
                date, hour, ordinal, timestamp, channel, game_id, content
            ),
        ),
        account_id: account_id.to_string(),
        record_id: None,
        schema_version: None,
        session_id: None,
        sequence: None,
        timestamp,
        channel,
        content,
        sender: ChatSender { game_id, trp3 },
        mark: None,
        npc: None,
        nt: None,
        ref_id: None,
        profile_snapshot_id: None,
        profile_snapshot: None,
        identity_source,
        event: None,
        raw_profile: None,
        listeners: None,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn v2_record_prefers_immutable_snapshot_and_resolves_listener() {
        let data = json!({
            "2026-07-22": {
                "20": [{
                    "sv": 2,
                    "id": "session-a:7",
                    "sid": "session-a",
                    "seq": 7,
                    "t": 1_753_200_000,
                    "c": "SAY",
                    "m": "Still the old name",
                    "s": "Player-Realm",
                    "mk": "P",
                    "ref": "profile-a",
                    "ps": "snap-old",
                    "listeners": [{
                        "gameID": "Viewer-Realm",
                        "profileID": "viewer-profile",
                        "ps": "snap-viewer"
                    }]
                }]
            }
        });
        let cache = json!({
            "profile-a": {
                "FN": "New",
                "LN": "Name",
                "RA": "Human",
                "CL": "Mage",
                "about": { "T1": { "TX": "Full biography" } },
                "misc": { "PE": [{ "NA": "Full characteristic" }] }
            }
        });
        let snapshots = json!({
            "snap-old": {
                "ref": "profile-a",
                "gameID": "Player-Realm",
                "n": "Old Name",
                "pn": "Old card",
                "FN": "Old",
                "LN": "Name",
                "rev": 1
            },
            "snap-viewer": {
                "ref": "viewer-profile",
                "gameID": "Viewer-Realm",
                "n": "The Viewer",
                "pn": "Viewer card"
            }
        });

        let records = parse_chat_records(&data, &cache, &snapshots, "ACCOUNT-A");

        assert_eq!(records.len(), 1);
        let record = &records[0];
        assert_eq!(record.identity_source, "snapshot");
        assert_eq!(record.record_id.as_deref(), Some("session-a:7"));
        assert_eq!(record.profile_snapshot_id.as_deref(), Some("snap-old"));
        assert_eq!(
            record
                .profile_snapshot
                .as_ref()
                .and_then(|snapshot| snapshot.display_name.as_deref()),
            Some("Old Name")
        );
        assert_eq!(
            record
                .sender
                .trp3
                .as_ref()
                .and_then(|profile| profile.first_name.as_deref()),
            Some("Old")
        );
        let raw_profile: Value =
            serde_json::from_str(record.raw_profile.as_deref().expect("full cached profile"))
                .expect("valid raw profile JSON");
        assert_eq!(raw_profile["FN"], "New");
        assert_eq!(raw_profile["RA"], "Human");
        assert_eq!(raw_profile["CL"], "Mage");
        assert_eq!(raw_profile["about"]["T1"]["TX"], "Full biography");
        assert_eq!(raw_profile["misc"]["PE"][0]["NA"], "Full characteristic");
        assert_eq!(
            record
                .listeners
                .as_ref()
                .and_then(|listeners| listeners.first())
                .and_then(|listener| listener.snapshot.as_ref())
                .and_then(|snapshot| snapshot.display_name.as_deref()),
            Some("The Viewer")
        );
    }

    #[test]
    fn v2_identity_event_keeps_both_historical_names() {
        let data = json!({
            "2026-07-22": {
                "20": [{
                    "sv": 2,
                    "id": "session-a:8",
                    "sid": "session-a",
                    "seq": 8,
                    "t": 1_753_200_001,
                    "c": "SYSTEM",
                    "m": "",
                    "s": "Player-Realm",
                    "mk": "S",
                    "ev": {
                        "kind": "profile_switch",
                        "certainty": "exact",
                        "from": { "ref": "profile-a", "ps": "snap-a", "n": "Before" },
                        "to": { "ref": "profile-b", "ps": "snap-b", "n": "After" }
                    }
                }]
            }
        });

        let records = parse_chat_records(&data, &json!({}), &json!({}), "ACCOUNT-A");
        let event = records[0].event.as_ref().expect("identity event");

        assert_eq!(event.kind, "profile_switch");
        assert_eq!(event.certainty, "exact");
        assert_eq!(
            event
                .from
                .as_ref()
                .and_then(|value| value.display_name.as_deref()),
            Some("Before")
        );
        assert_eq!(
            event
                .to
                .as_ref()
                .and_then(|value| value.display_name.as_deref()),
            Some("After")
        );
    }

    #[test]
    fn stable_keys_do_not_collide_for_same_second_or_other_account() {
        let data = json!({
            "2026-07-22": {
                "20": [
                    {
                        "timestamp": 1_753_200_000,
                        "channel": "SAY",
                        "content": "first",
                        "sender": { "gameID": "Player-Realm" }
                    },
                    {
                        "timestamp": 1_753_200_000,
                        "channel": "SAY",
                        "content": "second",
                        "sender": { "gameID": "Player-Realm" }
                    }
                ]
            }
        });

        let first_account = parse_chat_records(&data, &json!({}), &json!({}), "ACCOUNT-A");
        let second_account = parse_chat_records(&data, &json!({}), &json!({}), "ACCOUNT-B");

        assert_ne!(first_account[0].record_key, first_account[1].record_key);
        assert_ne!(first_account[0].record_key, second_account[0].record_key);
        assert_eq!(first_account[0].account_id, "ACCOUNT-A");
    }
}
