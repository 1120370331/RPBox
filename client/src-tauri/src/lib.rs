mod lua_parser;
mod wow_path;
mod scanner;
mod sync_meta;
mod writer;

use std::path::Path;

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
async fn is_wow_running() -> bool {
    writer::is_wow_running()
}

#[tauri::command]
async fn write_profile(path: String, raw_lua: String) -> Result<(), String> {
    let path = std::path::PathBuf::from(path);
    writer::write_profile_to_local(&path, &raw_lua)
        .map_err(|e| e.to_string())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![
            parse_trp3_file,
            detect_wow_paths,
            validate_wow_path,
            normalize_wow_path,
            scan_profiles,
            is_wow_running,
            write_profile
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
