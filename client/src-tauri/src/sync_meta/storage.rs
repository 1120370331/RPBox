use chrono::{DateTime, Utc};
use rusqlite::{Connection, params};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyncMetadata {
    pub profile_id: String,
    pub last_synced_at: DateTime<Utc>,
    pub last_synced_checksum: String,
    pub cloud_version: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum SyncStatus {
    Synced,
    LocalChanged,
    CloudChanged,
    Conflict,
    NotSynced,
}

pub struct SyncStore {
    conn: Connection,
}

impl SyncStore {
    pub fn new(app_data_dir: &PathBuf) -> Result<Self, rusqlite::Error> {
        let db_path = app_data_dir.join("sync_meta.db");
        let conn = Connection::open(db_path)?;

        conn.execute(
            "CREATE TABLE IF NOT EXISTS sync_metadata (
                profile_id TEXT PRIMARY KEY,
                last_synced_at TEXT NOT NULL,
                last_synced_checksum TEXT NOT NULL,
                cloud_version INTEGER NOT NULL
            )",
            [],
        )?;

        Ok(Self { conn })
    }

    pub fn get_metadata(&self, profile_id: &str) -> Option<SyncMetadata> {
        let mut stmt = self.conn
            .prepare("SELECT profile_id, last_synced_at, last_synced_checksum, cloud_version FROM sync_metadata WHERE profile_id = ?")
            .ok()?;

        stmt.query_row(params![profile_id], |row| {
            let synced_at: String = row.get(1)?;
            Ok(SyncMetadata {
                profile_id: row.get(0)?,
                last_synced_at: synced_at.parse().unwrap_or_else(|_| Utc::now()),
                last_synced_checksum: row.get(2)?,
                cloud_version: row.get(3)?,
            })
        }).ok()
    }

    pub fn update_metadata(&self, meta: &SyncMetadata) -> Result<(), rusqlite::Error> {
        self.conn.execute(
            "INSERT OR REPLACE INTO sync_metadata
             (profile_id, last_synced_at, last_synced_checksum, cloud_version)
             VALUES (?1, ?2, ?3, ?4)",
            params![
                meta.profile_id,
                meta.last_synced_at.to_rfc3339(),
                meta.last_synced_checksum,
                meta.cloud_version
            ],
        )?;
        Ok(())
    }

    pub fn get_sync_status(
        &self,
        profile_id: &str,
        local_checksum: &str,
        cloud_version: Option<i32>,
    ) -> SyncStatus {
        let meta = match self.get_metadata(profile_id) {
            Some(m) => m,
            None => return SyncStatus::NotSynced,
        };

        let local_changed = local_checksum != meta.last_synced_checksum;
        let cloud_changed = cloud_version
            .map(|v| v > meta.cloud_version)
            .unwrap_or(false);

        match (local_changed, cloud_changed) {
            (false, false) => SyncStatus::Synced,
            (true, false) => SyncStatus::LocalChanged,
            (false, true) => SyncStatus::CloudChanged,
            (true, true) => SyncStatus::Conflict,
        }
    }
}
