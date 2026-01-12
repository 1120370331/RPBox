mod lexer;
mod parser;
mod error;

pub use error::LuaParseError;
pub use parser::parse_variable;

#[cfg(test)]
mod tests;
