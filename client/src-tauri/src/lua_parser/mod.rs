mod error;
mod lexer;
mod parser;

pub use error::LuaParseError;
pub use parser::{parse_variable, parse_variable_from_str};

#[cfg(test)]
mod tests;
