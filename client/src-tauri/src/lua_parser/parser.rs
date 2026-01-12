use std::path::Path;
use serde_json::{json, Value};

use crate::lua_parser::error::LuaParseError;
use crate::lua_parser::lexer::{Lexer, Token};

pub fn parse_variable(path: &Path, var_name: &str) -> Result<Value, LuaParseError> {
    if !path.exists() {
        return Err(LuaParseError::FileNotFound(path.to_path_buf()));
    }

    let content = std::fs::read(path)?;
    let content = String::from_utf8_lossy(&content);

    parse_variable_from_str(&content, var_name)
}

pub fn parse_variable_from_str(content: &str, var_name: &str) -> Result<Value, LuaParseError> {
    let mut parser = Parser::new(content);
    parser.parse_file(var_name)
}

struct Parser<'a> {
    lexer: Lexer<'a>,
    current_token: Token,
}

impl<'a> Parser<'a> {
    fn new(input: &'a str) -> Self {
        let mut lexer = Lexer::new(input);
        let current_token = lexer.next_token().unwrap_or(Token::Eof);
        Self { lexer, current_token }
    }

    fn advance(&mut self) -> Result<(), LuaParseError> {
        self.current_token = self.lexer.next_token()?;
        Ok(())
    }

    fn current_line(&self) -> usize {
        self.lexer.current_line()
    }

    fn parse_file(&mut self, var_name: &str) -> Result<Value, LuaParseError> {
        loop {
            match &self.current_token {
                Token::Eof => {
                    return Err(LuaParseError::VariableNotFound(var_name.to_string()));
                }
                Token::Identifier(name) if name == var_name => {
                    self.advance()?;
                    self.expect(Token::Equals)?;
                    return self.parse_value();
                }
                _ => {
                    self.skip_assignment()?;
                }
            }
        }
    }

    fn skip_assignment(&mut self) -> Result<(), LuaParseError> {
        // Skip identifier
        if matches!(self.current_token, Token::Identifier(_)) {
            self.advance()?;
        }
        // Skip = and value
        if self.current_token == Token::Equals {
            self.advance()?;
            self.skip_value()?;
        }
        Ok(())
    }

    fn skip_value(&mut self) -> Result<(), LuaParseError> {
        match &self.current_token {
            Token::LeftBrace => {
                self.advance()?;
                let mut depth = 1;
                while depth > 0 {
                    match &self.current_token {
                        Token::LeftBrace => depth += 1,
                        Token::RightBrace => depth -= 1,
                        Token::Eof => return Err(LuaParseError::UnexpectedEof),
                        _ => {}
                    }
                    self.advance()?;
                }
            }
            _ => {
                self.advance()?;
            }
        }
        Ok(())
    }
}

impl<'a> Parser<'a> {
    fn expect(&mut self, expected: Token) -> Result<(), LuaParseError> {
        if std::mem::discriminant(&self.current_token) == std::mem::discriminant(&expected) {
            self.advance()?;
            Ok(())
        } else {
            Err(LuaParseError::UnexpectedToken {
                expected: format!("{:?}", expected),
                actual: format!("{:?}", self.current_token),
            })
        }
    }

    fn parse_value(&mut self) -> Result<Value, LuaParseError> {
        match &self.current_token {
            Token::String(s) => {
                let val = json!(s.clone());
                self.advance()?;
                Ok(val)
            }
            Token::Number(n) => {
                let val = json!(*n);
                self.advance()?;
                Ok(val)
            }
            Token::True => {
                self.advance()?;
                Ok(json!(true))
            }
            Token::False => {
                self.advance()?;
                Ok(json!(false))
            }
            Token::Nil => {
                self.advance()?;
                Ok(Value::Null)
            }
            Token::LeftBrace => self.parse_table(),
            _ => Err(LuaParseError::SyntaxError {
                line: self.current_line(),
                message: format!("期望值, 实际: {:?}", self.current_token),
            }),
        }
    }

    fn parse_table(&mut self) -> Result<Value, LuaParseError> {
        self.advance()?; // consume '{'

        let mut map = serde_json::Map::new();
        let mut array: Vec<Value> = Vec::new();
        let mut array_index = 1u64;
        let mut is_array = true;

        while self.current_token != Token::RightBrace {
            let (key, value) = self.parse_table_entry(&mut array_index)?;

            if let Some(k) = key {
                is_array = false;
                map.insert(k, value);
            } else {
                array.push(value);
            }

            // Handle comma
            if self.current_token == Token::Comma {
                self.advance()?;
            }
        }

        self.advance()?; // consume '}'

        if is_array && !array.is_empty() {
            Ok(Value::Array(array))
        } else if !map.is_empty() {
            // Merge array into map with numeric keys
            for (i, v) in array.into_iter().enumerate() {
                map.insert((i + 1).to_string(), v);
            }
            Ok(Value::Object(map))
        } else {
            Ok(Value::Object(map))
        }
    }

    fn parse_table_entry(&mut self, array_index: &mut u64) -> Result<(Option<String>, Value), LuaParseError> {
        match &self.current_token {
            // ["key"] = value
            Token::LeftBracket => {
                self.advance()?;
                let key = match &self.current_token {
                    Token::String(s) => s.clone(),
                    Token::Number(n) => n.to_string(),
                    _ => {
                        return Err(LuaParseError::SyntaxError {
                            line: self.current_line(),
                            message: "表键必须是字符串或数字".to_string(),
                        });
                    }
                };
                self.advance()?;
                self.expect(Token::RightBracket)?;
                self.expect(Token::Equals)?;
                let value = self.parse_value()?;
                Ok((Some(key), value))
            }
            // key = value (identifier key)
            Token::Identifier(name) => {
                let key = name.clone();
                self.advance()?;
                if self.current_token == Token::Equals {
                    self.advance()?;
                    let value = self.parse_value()?;
                    Ok((Some(key), value))
                } else {
                    // It's actually a value, not a key
                    let value = match key.as_str() {
                        "true" => json!(true),
                        "false" => json!(false),
                        "nil" => Value::Null,
                        _ => json!(key),
                    };
                    *array_index += 1;
                    Ok((None, value))
                }
            }
            // Implicit array value
            _ => {
                let value = self.parse_value()?;
                *array_index += 1;
                Ok((None, value))
            }
        }
    }
}
