use lang::namespace::{NameSpace, Name};

/// A lexer for Prolog source code.
///
/// Implemented as an iterator over `Token`s.
pub struct Lexer<'ns, I> {
    inner: I,
    ns: &'ns NameSpace,
    buf: String,
    line: u32,
    col: u32,
}

/// A lexical item of Prolog.
///
/// Every `Token` includes its line and column as the first two members. When
/// relevant, the third member gives an interpreted value of the token.
///
/// Lexical errors are given as a `Token::Err` whose value is the error message.
#[derive(Debug)]
#[derive(Clone, Copy)]
#[derive(PartialEq)]
pub enum Token<'ns> {
    Err(u32, u32, &'static str), // TODO: Change error from str to an error code
    Funct(u32, u32, Name<'ns>),
    Str(u32, u32, Name<'ns>),
    Var(u32, u32, Name<'ns>),
    Int(u32, u32, i64),
    Float(u32, u32, f64),
    ParenOpen(u32, u32),
    ParenClose(u32, u32),
    BracketOpen(u32, u32),
    BracketClose(u32, u32),
    BraceOpen(u32, u32),
    BraceClose(u32, u32),
    Bar(u32, u32),
    Comma(u32, u32),
    Dot(u32, u32),
}

// Public API
// --------------------------------------------------

impl<'ns, I> Lexer<'ns, I>
    where I: Iterator<Item = char>
{
    /// Constructs a new lexer from a stream of chars.
    ///
    /// The line and column counts both start at 1.
    pub fn new(chars: I, ns: &'ns NameSpace) -> Lexer<'ns, I> {
        Lexer {
            inner: chars,
            ns: ns,
            buf: String::with_capacity(32),
            line: 1,
            col: 1,
        }
    }
}

impl<'ns, I> Iterator for Lexer<'ns, I>
    where I: Iterator<Item = char>
{
    type Item = Token<'ns>;

    /// Generates the next token in the underlying stream.
    fn next(&mut self) -> Option<Token<'ns>> {
        let next = match self.buf.pop() {
            Some(ch) => Some(ch),
            None => self.inner.next(),
        };
        match next {
            Some('(') => self.lex_simple('('),
            Some(')') => self.lex_simple(')'),
            Some('[') => self.lex_simple('['),
            Some(']') => self.lex_simple(']'),
            Some('{') => self.lex_simple('{'),
            Some('}') => self.lex_simple('}'),
            Some(',') => self.lex_simple(','),
            Some('|') => self.lex_simple('|'),
            Some('.') => self.lex_simple('.'),
            Some('%') => self.lex_comment(),
            Some('_') => self.lex_var('_'),
            Some('\'') => self.lex_quote('\''),
            Some('\"') => self.lex_quote('\"'),
            Some('-') => self.lex_minus(),
            Some('0') => self.lex_zero(),
            Some(ch) if ch.is_digit(10) => self.lex_decimal(ch),
            Some(ch) if ch.is_whitespace() => self.lex_space(ch),
            Some(ch) if ch.is_control() => self.lex_space(ch),
            Some(ch) if ch.is_uppercase() => self.lex_var(ch),
            Some(ch) => self.lex_functor(ch),
            None => None,
        }
    }
}

// Lexing Logic
// --------------------------------------------------
// This impl gives the various private lexing routines.
//
// When these functions are called, the first character of the token has
// already been read from the underlying stream. When appropriate, the first
// character is passed as an argument.
//
// The functions may use the buffer as scratch space to build a token string.
// The buffer is guarenteed to be empty when the functions are called. The
// functions MUST clear the buffer before returning. If a function reads past
// its token, it may write a single character to the buffer after it is
// cleared. In this case, that character is treated as the first character of
// the next token.

impl<'ns, I> Lexer<'ns, I>
    where I: Iterator<Item = char>
{
    /// Returns the token for a simple function symbol.
    fn lex_functor(&mut self, first: char) -> Option<Token<'ns>> {
        if is_symbolic(first) {
            return self.lex_symbolic(first);
        }

        let line = self.line;
        let col = self.col;
        self.buf.push(first); // assume first char is valid
        loop {
            match self.inner.next() {
                Some('_') => {
                    self.buf.push('_');
                }
                Some(ch) if ch.is_alphanumeric() => {
                    self.buf.push(ch);
                }
                Some(ch) => {
                    let tok = Token::Funct(line, col, self.get_name());
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    self.buf.push(ch);
                    return Some(tok);
                }
                None => {
                    let tok = Token::Funct(line, col, self.get_name());
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    return Some(tok);
                }
            }
        }
    }

    /// Returns the token for a function symbol starting with a symbolic char.
    fn lex_symbolic(&mut self, first: char) -> Option<Token<'ns>> {
        let line = self.line;
        let col = self.col;
        self.buf.push(first); // assume first char is valid
        loop {
            match self.inner.next() {
                Some('_') => {
                    self.buf.push('_');
                }
                Some(ch) if is_symbolic(ch) => {
                    self.buf.push(ch);
                }
                Some(ch) => {
                    let tok = Token::Funct(line, col, self.get_name());
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    self.buf.push(ch);
                    return Some(tok);
                }
                None => {
                    let tok = Token::Funct(line, col, self.get_name());
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    return Some(tok);
                }
            }
        }
    }

    /// Returns the token for a variable term.
    fn lex_var(&mut self, first: char) -> Option<Token<'ns>> {
        let line = self.line;
        let col = self.col;
        self.buf.push(first); // assume first char is valid
        loop {
            match self.inner.next() {
                Some('_') => {
                    self.buf.push('_');
                }
                Some(ch) if ch.is_alphanumeric() => {
                    self.buf.push(ch);
                }
                Some(ch) => {
                    let tok = Token::Var(line, col, self.get_name());
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    self.buf.push(ch);
                    return Some(tok);
                }
                None => {
                    let tok = Token::Var(line, col, self.get_name());
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    return Some(tok);
                }
            }
        }
    }

    /// Returns the token for a symbol starting with a minus.
    fn lex_minus(&mut self) -> Option<Token<'ns>> {
        let line = self.line;
        let col = self.col;
        self.buf.push('-');
        match self.inner.next() {
            Some('0') => self.lex_zero(),
            Some(ch) if ch.is_digit(10) => self.lex_decimal(ch),
            Some(ch) if is_symbolic(ch) => self.lex_functor(ch),
            Some(ch) => {
                let tok = Token::Funct(line, col, self.get_name());
                self.col += self.buf.len() as u32;
                self.buf.clear();
                self.buf.push(ch);
                Some(tok)
            }
            None => {
                let tok = Token::Funct(line, col, self.get_name());
                self.col += self.buf.len() as u32;
                self.buf.clear();
                Some(tok)
            }
        }
    }

    /// Returns the token for a binary, octal, hexidecimal, or decimal number.
    fn lex_zero(&mut self) -> Option<Token<'ns>> {
        let line = self.line;
        let col = self.col;
        let radix: u32;
        self.buf.push('0');
        match self.inner.next() {
            Some('x') => radix = 16,
            Some('o') => radix = 8,
            Some('b') => radix = 2,
            Some('.') => return self.lex_decimal('.'),
            Some(ch) if ch.is_digit(10) => return self.lex_decimal(ch),
            Some(ch) => {
                self.col += 1;
                self.buf.push(ch);
                return Some(Token::Int(line, col, 0));
            }
            None => {
                self.col += 1;
                return Some(Token::Int(line, col, 0));
            }
        }

        // we don't add the radix char ('x', 'o', or 'b') to the buffer,
        // but we still need to adjust the column count.
        self.col += 1;

        loop {
            match self.inner.next() {
                Some(ch) if ch.is_digit(radix) => self.buf.push(ch),
                Some(ch) => {
                    let tok = match i64::from_str_radix(self.buf.as_str(), radix) {
                        Ok(x) => Token::Int(line, col, x),
                        Err(_) => Token::Err(line, col, "cannot parse number"),
                    };
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    self.buf.push(ch);
                    return Some(tok);
                }
                None => {
                    let tok = match i64::from_str_radix(self.buf.as_str(), radix) {
                        Ok(x) => Token::Int(line, col, x),
                        Err(_) => Token::Err(line, col, "cannot parse number"),
                    };
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    return Some(tok);
                }
            }
        }
    }

    /// Returns the token for a decimal number.
    fn lex_decimal(&mut self, first: char) -> Option<Token<'ns>> {
        let line = self.line;
        let col = self.col;
        let mut seen_dot = first == '.';
        let mut seen_e = false;
        self.buf.push(first);
        loop {
            match self.inner.next() {
                Some(ch) if ch.is_digit(10) => self.buf.push(ch),
                Some('_') => self.buf.push('_'),
                Some('.') => {
                    if seen_dot {
                        let tok = match self.buf.parse::<f64>() {
                            Ok(x) => Token::Float(line, col, x),
                            Err(_) => Token::Err(line, col, "cannot parse number"),
                        };
                        self.col += self.buf.len() as u32;
                        self.buf.clear();
                        self.buf.push('.');
                        return Some(tok);
                    }
                    self.buf.push('.');
                    seen_dot = true;
                }
                Some('e') => {
                    if seen_e {
                        let tok = match self.buf.parse::<f64>() {
                            Ok(x) => Token::Float(line, col, x),
                            Err(_) => Token::Err(line, col, "cannot parse number"),
                        };
                        self.col += self.buf.len() as u32;
                        self.buf.clear();
                        self.buf.push('e');
                        return Some(tok);
                    }
                    self.buf.push('e');
                    seen_dot = true;
                    seen_e = true;
                    match self.inner.next() {
                        Some('-') => self.buf.push('-'),
                        Some(ch) if ch.is_digit(10) => self.buf.push(ch),
                        Some(ch) => {
                            let tok = match self.buf.parse::<f64>() {
                                Ok(x) => Token::Float(line, col, x),
                                Err(_) => Token::Err(line, col, "cannot parse number"),
                            };
                            self.col += self.buf.len() as u32;
                            self.buf.clear();
                            self.buf.push(ch);
                            return Some(tok);
                        }
                        None => {
                            let tok = match self.buf.parse::<f64>() {
                                Ok(x) => Token::Float(line, col, x),
                                Err(_) => Token::Err(line, col, "cannot parse number"),
                            };
                            self.col += self.buf.len() as u32;
                            self.buf.clear();
                            return Some(tok);
                        }
                    }
                }
                Some(ch) => {
                    let tok = if seen_dot {
                        match self.buf.parse::<f64>() {
                            Ok(x) => Token::Float(line, col, x),
                            Err(_) => Token::Err(line, col, "cannot parse number"),
                        }
                    } else {
                        match self.buf.parse::<i64>() {
                            Ok(x) => Token::Int(line, col, x),
                            Err(_) => Token::Err(line, col, "cannot parse number"),
                        }
                    };
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    self.buf.push(ch);
                    return Some(tok);
                }
                None => {
                    let tok = if seen_dot {
                        match self.buf.parse::<f64>() {
                            Ok(x) => Token::Float(line, col, x),
                            Err(_) => Token::Err(line, col, "cannot parse number"),
                        }
                    } else {
                        match self.buf.parse::<i64>() {
                            Ok(x) => Token::Int(line, col, x),
                            Err(_) => Token::Err(line, col, "cannot parse number"),
                        }
                    };
                    self.col += self.buf.len() as u32;
                    self.buf.clear();
                    return Some(tok);
                }
            }
        }
    }

    /// Retuns a token giving the text of a comment.
    fn lex_comment(&mut self) -> Option<Token<'ns>> {
        while let Some(ch) = self.inner.next() {
            if ch == '\n' {
                break;
            }
        }
        self.line += 1;
        self.col = 1;
        self.next()
    }

    /// Returns a Functor or String for a token enclosed in quotes.
    ///
    /// Escape sequences are replaced and the token will not include the
    /// surrounding quotes. An Err token is returned if the quote is unclosed.
    fn lex_quote(&mut self, quote: char) -> Option<Token<'ns>> {
        let line = self.line;
        let col = self.col;
        self.col += 1;
        loop {
            match self.inner.next() {
                Some('\\') => {
                    self.col += 2;
                    match self.inner.next() {
                        Some('n') => self.buf.push('\n'),
                        Some('r') => self.buf.push('\r'),
                        Some('t') => self.buf.push('\t'),
                        Some('\\') => self.buf.push('\\'),
                        Some(ch) => self.buf.push(ch),
                        None => {
                            self.buf.clear();
                            return Some(Token::Err(line, col, "unclosed quote"));
                        }
                    };
                }
                Some('\n') => {
                    self.col = 1;
                    self.line += 1;
                    self.buf.push('\n');
                }
                Some(ch) if ch == quote => {
                    self.col += 1;
                    let tok = match quote {
                        '\"' => Token::Str(line, col, self.get_name()),
                        '\'' => Token::Funct(line, col, self.get_name()),
                        _ => panic!("unsupported quote"),
                    };
                    self.buf.clear();
                    return Some(tok);
                }
                Some(ch) => {
                    self.col += 1;
                    self.buf.push(ch);
                }
                None => {
                    self.buf.clear();
                    return Some(Token::Err(self.line, self.col, "unclosed quote"));
                }
            }
        }
    }

    /// Returns the token for a single char symbol.
    fn lex_simple(&mut self, ch: char) -> Option<Token<'ns>> {
        let line = self.line;
        let col = self.col;
        self.col += 1;
        match ch {
            '(' => Some(Token::ParenOpen(line, col)),
            ')' => Some(Token::ParenClose(line, col)),
            '[' => Some(Token::BracketOpen(line, col)),
            ']' => Some(Token::BracketClose(line, col)),
            '{' => Some(Token::BraceOpen(line, col)),
            '}' => Some(Token::BraceClose(line, col)),
            ',' => Some(Token::Comma(line, col)),
            '|' => Some(Token::Bar(line, col)),
            '.' => Some(Token::Dot(line, col)),
            _ => panic!("lex_simple called without a grouping symbol"),
        }
    }

    /// Skips whitespace/control characters and returns the next token.
    fn lex_space(&mut self, first: char) -> Option<Token<'ns>> {
        let mut ch = Some(first);
        loop {
            match ch {
                Some('\n') => {
                    self.line += 1;
                    self.col = 1;
                }
                Some(ch) if ch.is_whitespace() || ch.is_control() => {
                    self.col += 1;
                }
                Some(ch) => {
                    self.buf.push(ch);
                    return self.next();
                }
                None => return None,
            };
            ch = self.inner.next();
        }
    }
}

// Helpers
// --------------------------------------------------

impl<'ns, I> Lexer<'ns, I>
    where I: Iterator<Item = char>
{
    /// Converts the buffer to a `Name`.
    fn get_name(&mut self) -> Name<'ns> {
        self.ns.name(self.buf.clone())
    }
}

fn is_special(ch: char) -> bool {
    "\'\",.|%{[()]}".contains(ch)
}

fn is_symbolic(ch: char) -> bool {
    !ch.is_alphanumeric() && !ch.is_whitespace() && !ch.is_control() && !is_special(ch)
}

// Tests
// --------------------------------------------------

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    #[rustfmt_skip]
    fn basic() {
        let pl = "_abcd ABCD foobar 'hello world' +++\n\
                  % this is a comment\n\
                  123 456.789 8.765e43 1e-1\n\
                  0xDEADBEEF 0o644 0b11001100 0987654321 0.123\n\
                  -> -0xff -1.23 (-)\n\
                  \t\t   \t\n";
        let ns = NameSpace::new();
        let mut toks = Lexer::new(pl.chars(), &ns);
        assert_eq!(toks.next().unwrap(), Token::Var(1, 1, ns.name("_abcd")));
        assert_eq!(toks.next().unwrap(), Token::Var(1, 7, ns.name("ABCD")));
        assert_eq!(toks.next().unwrap(), Token::Funct(1, 12, ns.name("foobar")));
        assert_eq!(toks.next().unwrap(), Token::Funct(1, 19, ns.name("hello world")));
        assert_eq!(toks.next().unwrap(), Token::Funct(1, 33, ns.name("+++")));
        assert_eq!(toks.next().unwrap(), Token::Int(3, 1, 123));
        assert_eq!(toks.next().unwrap(), Token::Float(3, 5, 456.789));
        assert_eq!(toks.next().unwrap(), Token::Float(3, 13, 8.765e43));
        assert_eq!(toks.next().unwrap(), Token::Float(3, 22, 1e-1));
        assert_eq!(toks.next().unwrap(), Token::Int(4, 1, 0xDEADBEEF));
        assert_eq!(toks.next().unwrap(), Token::Int(4, 12, 0o644));
        assert_eq!(toks.next().unwrap(), Token::Int(4, 18, 0b11001100));
        assert_eq!(toks.next().unwrap(), Token::Int(4, 29, 0987654321));
        assert_eq!(toks.next().unwrap(), Token::Float(4, 40, 0.123));
        assert_eq!(toks.next().unwrap(), Token::Funct(5, 1, ns.name("->")));
        assert_eq!(toks.next().unwrap(), Token::Int(5, 4, -0xff));
        assert_eq!(toks.next().unwrap(), Token::Float(5, 10, -1.23));
        assert_eq!(toks.next().unwrap(), Token::ParenOpen(5, 16));
        assert_eq!(toks.next().unwrap(), Token::Funct(5, 17, ns.name("-")));
        assert_eq!(toks.next().unwrap(), Token::ParenClose(5, 18));
        assert_eq!(toks.next(), None);
    }

    #[test]
    #[rustfmt_skip]
    fn realistic() {
        let pl = "member(H, [H|T]).\n\
                  member(X, [_|T]) :- member(X, T).\n";
        let ns = NameSpace::new();
        let mut toks = Lexer::new(pl.chars(), &ns);

        // member(H, [H|T]).
        assert_eq!(toks.next().unwrap(), Token::Funct(1, 1, ns.name("member")));
        assert_eq!(toks.next().unwrap(), Token::ParenOpen(1, 7));
        assert_eq!(toks.next().unwrap(), Token::Var(1, 8, ns.name("H")));
        assert_eq!(toks.next().unwrap(), Token::Comma(1, 9));
        assert_eq!(toks.next().unwrap(), Token::BracketOpen(1, 11));
        assert_eq!(toks.next().unwrap(), Token::Var(1, 12, ns.name("H")));
        assert_eq!(toks.next().unwrap(), Token::Bar(1, 13));
        assert_eq!(toks.next().unwrap(), Token::Var(1, 14, ns.name("T")));
        assert_eq!(toks.next().unwrap(), Token::BracketClose(1, 15));
        assert_eq!(toks.next().unwrap(), Token::ParenClose(1, 16));
        assert_eq!(toks.next().unwrap(), Token::Dot(1, 17));

        // member(X, [_|T]) :- member(X, T).
        assert_eq!(toks.next().unwrap(), Token::Funct(2, 1, ns.name("member")));
        assert_eq!(toks.next().unwrap(), Token::ParenOpen(2, 7));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 8, ns.name("X")));
        assert_eq!(toks.next().unwrap(), Token::Comma(2, 9));
        assert_eq!(toks.next().unwrap(), Token::BracketOpen(2, 11));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 12, ns.name("_")));
        assert_eq!(toks.next().unwrap(), Token::Bar(2, 13));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 14, ns.name("T")));
        assert_eq!(toks.next().unwrap(), Token::BracketClose(2, 15));
        assert_eq!(toks.next().unwrap(), Token::ParenClose(2, 16));
        assert_eq!(toks.next().unwrap(), Token::Funct(2, 18, ns.name(":-")));
        assert_eq!(toks.next().unwrap(), Token::Funct(2, 21, ns.name("member")));
        assert_eq!(toks.next().unwrap(), Token::ParenOpen(2, 27));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 28, ns.name("X")));
        assert_eq!(toks.next().unwrap(), Token::Comma(2, 29));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 31, ns.name("T")));
        assert_eq!(toks.next().unwrap(), Token::ParenClose(2, 32));
        assert_eq!(toks.next().unwrap(), Token::Dot(2, 33));

        assert_eq!(toks.next(), None);
    }
}
