use std::path::PathBuf;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum LuaParseError {
    #[error("文件不存在: {0}")]
    FileNotFound(PathBuf),

    #[error("文件读取失败: {0}")]
    IoError(#[from] std::io::Error),

    #[error("语法错误 (行{line}): {message}")]
    SyntaxError { line: usize, message: String },

    #[error("变量不存在: {0}")]
    VariableNotFound(String),

    #[error("意外的token: 期望 {expected}, 实际 {actual}")]
    UnexpectedToken { expected: String, actual: String },

    #[error("意外的文件结束")]
    UnexpectedEof,
}
