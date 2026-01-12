use crate::lua_parser::error::LuaParseError;

#[derive(Debug, Clone, PartialEq)]
pub enum Token {
    // Literals
    String(String),
    Number(f64),
    True,
    False,
    Nil,

    // Identifiers
    Identifier(String),

    // Punctuation
    LeftBrace,    // {
    RightBrace,   // }
    LeftBracket,  // [
    RightBracket, // ]
    Equals,       // =
    Comma,        // ,

    // End of file
    Eof,
}

pub struct Lexer<'a> {
    input: &'a str,
    chars: std::iter::Peekable<std::str::CharIndices<'a>>,
    current_line: usize,
}

impl<'a> Lexer<'a> {
    pub fn new(input: &'a str) -> Self {
        Self {
            input,
            chars: input.char_indices().peekable(),
            current_line: 1,
        }
    }

    pub fn current_line(&self) -> usize {
        self.current_line
    }

    fn skip_whitespace_and_comments(&mut self) {
        loop {
            match self.chars.peek() {
                Some(&(_, ' ')) | Some(&(_, '\t')) | Some(&(_, '\r')) => {
                    self.chars.next();
                }
                Some(&(_, '\n')) => {
                    self.chars.next();
                    self.current_line += 1;
                }
                Some(&(_, '-')) => {
                    let pos = self.chars.peek().map(|&(i, _)| i).unwrap_or(0);
                    if self.input[pos..].starts_with("--") {
                        self.skip_comment();
                    } else {
                        break;
                    }
                }
                _ => break,
            }
        }
    }

    fn skip_comment(&mut self) {
        // Skip "--"
        self.chars.next();
        self.chars.next();

        // Check for long comment --[[ ]]
        if let Some(&(pos, '[')) = self.chars.peek() {
            let rest = &self.input[pos..];
            if rest.starts_with("[[") || rest.starts_with("[=") {
                self.skip_long_comment();
                return;
            }
        }

        // Single line comment
        while let Some(&(_, c)) = self.chars.peek() {
            if c == '\n' {
                break;
            }
            self.chars.next();
        }
    }

    fn skip_long_comment(&mut self) {
        // Skip opening [[ or [=...=[
        self.chars.next(); // [
        let mut eq_count = 0;
        while let Some(&(_, '=')) = self.chars.peek() {
            self.chars.next();
            eq_count += 1;
        }
        self.chars.next(); // [

        // Find closing ]=...=]
        let closing = format!("]{}]", "=".repeat(eq_count));
        while let Some(&(pos, c)) = self.chars.peek() {
            if c == '\n' {
                self.current_line += 1;
            }
            if self.input[pos..].starts_with(&closing) {
                for _ in 0..closing.len() {
                    self.chars.next();
                }
                return;
            }
            self.chars.next();
        }
    }

    pub fn next_token(&mut self) -> Result<Token, LuaParseError> {
        self.skip_whitespace_and_comments();

        match self.chars.peek() {
            None => Ok(Token::Eof),
            Some(&(_, c)) => match c {
                '{' => { self.chars.next(); Ok(Token::LeftBrace) }
                '}' => { self.chars.next(); Ok(Token::RightBrace) }
                '[' => { self.chars.next(); Ok(Token::LeftBracket) }
                ']' => { self.chars.next(); Ok(Token::RightBracket) }
                '=' => { self.chars.next(); Ok(Token::Equals) }
                ',' => { self.chars.next(); Ok(Token::Comma) }
                '"' | '\'' => self.read_string(),
                '0'..='9' | '-' | '.' => self.read_number(),
                _ if c.is_alphabetic() || c == '_' => self.read_identifier(),
                _ => Err(LuaParseError::SyntaxError {
                    line: self.current_line,
                    message: format!("意外的字符: '{}'", c),
                }),
            },
        }
    }

    fn read_string(&mut self) -> Result<Token, LuaParseError> {
        let quote = self.chars.next().unwrap().1;
        let mut result = String::new();
        let start_line = self.current_line;

        loop {
            match self.chars.next() {
                None => {
                    return Err(LuaParseError::SyntaxError {
                        line: start_line,
                        message: "未闭合的字符串".to_string(),
                    });
                }
                Some((_, c)) if c == quote => break,
                Some((_, '\\')) => {
                    match self.chars.next() {
                        Some((_, 'n')) => result.push('\n'),
                        Some((_, 't')) => result.push('\t'),
                        Some((_, 'r')) => result.push('\r'),
                        Some((_, '\\')) => result.push('\\'),
                        Some((_, '"')) => result.push('"'),
                        Some((_, '\'')) => result.push('\''),
                        Some((_, '\n')) => {
                            self.current_line += 1;
                            result.push('\n');
                        }
                        Some((_, c)) => result.push(c),
                        None => {
                            return Err(LuaParseError::SyntaxError {
                                line: self.current_line,
                                message: "字符串中的转义序列不完整".to_string(),
                            });
                        }
                    }
                }
                Some((_, '\n')) => {
                    self.current_line += 1;
                    result.push('\n');
                }
                Some((_, c)) => result.push(c),
            }
        }

        Ok(Token::String(result))
    }

    fn read_number(&mut self) -> Result<Token, LuaParseError> {
        let start_pos = self.chars.peek().map(|&(i, _)| i).unwrap_or(0);
        let mut end_pos = start_pos;
        let mut has_dot = false;
        let mut has_exp = false;

        // Handle negative sign
        if let Some(&(_, '-')) = self.chars.peek() {
            self.chars.next();
            end_pos += 1;
        }

        while let Some(&(pos, c)) = self.chars.peek() {
            match c {
                '0'..='9' => {
                    self.chars.next();
                    end_pos = pos + 1;
                }
                '.' if !has_dot && !has_exp => {
                    has_dot = true;
                    self.chars.next();
                    end_pos = pos + 1;
                }
                'e' | 'E' if !has_exp => {
                    has_exp = true;
                    self.chars.next();
                    end_pos = pos + 1;
                    if let Some(&(_, '+')) | Some(&(_, '-')) = self.chars.peek() {
                        self.chars.next();
                        end_pos += 1;
                    }
                }
                _ => break,
            }
        }

        let num_str = &self.input[start_pos..end_pos];
        num_str.parse::<f64>()
            .map(Token::Number)
            .map_err(|_| LuaParseError::SyntaxError {
                line: self.current_line,
                message: format!("无效的数字: {}", num_str),
            })
    }

    fn read_identifier(&mut self) -> Result<Token, LuaParseError> {
        let start_pos = self.chars.peek().map(|&(i, _)| i).unwrap_or(0);
        let mut end_pos = start_pos;

        while let Some(&(pos, c)) = self.chars.peek() {
            if c.is_alphanumeric() || c == '_' {
                self.chars.next();
                end_pos = pos + c.len_utf8();
            } else {
                break;
            }
        }

        let ident = &self.input[start_pos..end_pos];
        let token = match ident {
            "true" => Token::True,
            "false" => Token::False,
            "nil" => Token::Nil,
            _ => Token::Identifier(ident.to_string()),
        };

        Ok(token)
    }
}
