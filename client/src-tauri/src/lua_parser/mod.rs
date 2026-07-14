mod error;
mod lexer;
mod parser;

pub use error::LuaParseError;
pub use parser::parse_variable;

#[cfg(test)]
mod tests;
