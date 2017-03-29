use std::fmt;
use std::io::BufRead;

use regex::Regex;

use syntax::namespace::{NameSpace, Name};

/// A lexer for logic programs.
///
/// Implemented as an iterator over `Token`s.
pub struct Lexer<'ns, B: BufRead> {
    reader: B,
    buf: String,
    ns: &'ns NameSpace,
    line: usize,
    col: usize,
}

/// A lexical item of a logic program.
///
/// Every `Token` includes its line and column as the first two members. When
/// relevant, the third member gives an interpreted value of the token.
///
/// Lexical errors are given as a `Token::Err` whose value is the error message.
#[derive(Debug)]
#[derive(Clone, Copy)]
#[derive(PartialEq)]
pub enum Token<'ns> {
    Err(usize, usize, &'static str),
    Funct(usize, usize, Name<'ns>),
    Str(usize, usize, Name<'ns>),
    Var(usize, usize, Name<'ns>),
    Int(usize, usize, i64),
    Float(usize, usize, f64),
    ParenOpen(usize, usize),
    ParenClose(usize, usize),
    BracketOpen(usize, usize),
    BracketClose(usize, usize),
    BraceOpen(usize, usize),
    BraceClose(usize, usize),
    Bar(usize, usize, Name<'ns>),
    Comma(usize, usize, Name<'ns>),
    Dot(usize, usize),
    Space(usize, usize),
    Comment(usize, usize),
}

#[derive(Debug)]
#[derive(Clone)]
#[derive(PartialEq, Eq)]
pub struct LexErr(usize, usize, String);

pub type Result<T> = ::std::io::Result<T>;

// Public API
// --------------------------------------------------

impl<'ns, B: BufRead> Lexer<'ns, B> {
    /// Constructs a new lexer from a stream of chars.
    pub fn new(reader: B, ns: &'ns NameSpace) -> Lexer<'ns, B> {
        Lexer {
            reader: reader,
            buf: String::with_capacity(128),
            ns: ns,
            line: 0,
            col: 0,
        }
    }
}

impl<'ns, B: BufRead> Iterator for Lexer<'ns, B> {
    type Item = Token<'ns>;

    /// Extracts the next token from the underlying stream.
    fn next(&mut self) -> Option<Token<'ns>> {
        if self.buf.len() == 0 {
            match self.reader.read_line(&mut self.buf) {
                Ok(0) => return None,
                Ok(_) => (),
                Err(e) => panic!(e),
            }
            self.line += 1;
            self.col = 1;
        }
        let (tok, len) = self.lex(self.buf.as_str());
        self.col += len;
        self.buf.drain(..len);

        // skip space and comments
        match tok {
            Token::Space(..) => self.next(),
            Token::Comment(..) => self.next(),
            _ => Some(tok),
        }
    }
}

impl<'ns> Token<'ns> {
    #[inline]
    pub fn line(&self) -> usize {
        match *self {
            Token::Err(line, ..) => line,
            Token::Funct(line, ..) => line,
            Token::Str(line, ..) => line,
            Token::Var(line, ..) => line,
            Token::Int(line, ..) => line,
            Token::Float(line, ..) => line,
            Token::ParenOpen(line, ..) => line,
            Token::ParenClose(line, ..) => line,
            Token::BracketOpen(line, ..) => line,
            Token::BracketClose(line, ..) => line,
            Token::BraceOpen(line, ..) => line,
            Token::BraceClose(line, ..) => line,
            Token::Bar(line, ..) => line,
            Token::Comma(line, ..) => line,
            Token::Dot(line, ..) => line,
            Token::Space(line, ..) => line,
            Token::Comment(line, ..) => line,
        }
    }

    #[inline]
    pub fn col(&self) -> usize {
        match *self {
            Token::Err(_, col, _) => col,
            Token::Funct(_, col, _) => col,
            Token::Str(_, col, _) => col,
            Token::Var(_, col, _) => col,
            Token::Int(_, col, _) => col,
            Token::Float(_, col, _) => col,
            Token::ParenOpen(_, col) => col,
            Token::ParenClose(_, col) => col,
            Token::BracketOpen(_, col) => col,
            Token::BracketClose(_, col) => col,
            Token::BraceOpen(_, col) => col,
            Token::BraceClose(_, col) => col,
            Token::Bar(_, col, _) => col,
            Token::Comma(_, col, _) => col,
            Token::Dot(_, col) => col,
            Token::Space(_, col) => col,
            Token::Comment(_, col) => col,
        }
    }
}

impl<'ns> fmt::Display for Token<'ns> {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match *self {
            Token::Err(.., msg) => write!(f, "{}", msg),
            Token::Funct(.., val) => write!(f, "{}", val),
            Token::Str(.., val) => write!(f, "{}", val),
            Token::Var(.., val) => write!(f, "{}", val),
            Token::Int(.., val) => write!(f, "{}", val),
            Token::Float(.., val) => write!(f, "{}", val),
            Token::ParenOpen(..) => f.write_str("("),
            Token::ParenClose(..) => f.write_str(")"),
            Token::BracketOpen(..) => f.write_str("["),
            Token::BracketClose(..) => f.write_str("]"),
            Token::BraceOpen(..) => f.write_str("{"),
            Token::BraceClose(..) => f.write_str("}"),
            Token::Bar(..) => f.write_str("|"),
            Token::Comma(..) => f.write_str(","),
            Token::Dot(..) => f.write_str("."),
            Token::Space(..) => f.write_str("SPACE"),
            Token::Comment(..) => f.write_str("COMMENT"),
        }
    }
}

// Lexing Logic
// --------------------------------------------------

impl<'ns, B: BufRead> Lexer<'ns, B> {
    fn lex(&self, line: &str) -> (Token<'ns>, usize) {
        match line.chars().nth(0) {
            Some('(') => self.lex_simple(line),
            Some(')') => self.lex_simple(line),
            Some('[') => self.lex_simple(line),
            Some(']') => self.lex_simple(line),
            Some('{') => self.lex_simple(line),
            Some('}') => self.lex_simple(line),
            Some(',') => self.lex_simple(line),
            Some('|') => self.lex_simple(line),
            Some('.') => self.lex_simple(line),
            Some('%') => self.lex_comment(line),
            Some('_') => self.lex_var(line),
            Some('\'') => self.lex_quote(line),
            Some('\"') => self.lex_quote(line),
            Some('-') => self.lex_minus(line),
            Some('0') => self.lex_zero(line),
            Some(ch) if ch.is_digit(10) => self.lex_decimal(line),
            Some(ch) if ch.is_whitespace() => self.lex_space(line),
            Some(ch) if ch.is_control() => self.lex_space(line),
            Some(ch) if ch.is_uppercase() => self.lex_var(line),
            Some(_) => self.lex_functor(line),
            None => panic!(),
        }
    }

    /// Returns the token for a simple function symbol.
    fn lex_functor(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = Regex::new(r"^(\w+|[\p{S}\p{Pc}\p{Pd}\p{Po}]+)").unwrap();
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let tok = Token::Funct(self.line, self.col, self.ns.name(s));
        (tok, s.len())
    }

    /// Returns the token for a variable term.
    fn lex_var(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = Regex::new(r"^[_\p{Lu}]\w*").unwrap();
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let tok = Token::Var(self.line, self.col, self.ns.name(s));
        (tok, s.len())
    }

    /// Returns the token for a symbol starting with a minus.
    fn lex_minus(&self, line: &str) -> (Token<'ns>, usize) {
        let mut len = 0;
        let tok = match line.chars().nth(1) {
            Some('0') => {
                let (subtok, sublen) = self.lex_zero(&line[1..]);
                len += 1 + sublen;
                match subtok {
                    Token::Int(_, _, val) => Token::Int(self.line, self.col, -val),
                    Token::Float(_, _, val) => Token::Float(self.line, self.col, -val),
                    _ => unreachable!("lex_zero must return a numeric token"),
                }
            }
            Some(ch) if ch.is_digit(10) => {
                let (subtok, sublen) = self.lex_decimal(&line[1..]);
                len += 1 + sublen;
                match subtok {
                    Token::Int(_, _, val) => Token::Int(self.line, self.col, -val),
                    Token::Float(_, _, val) => Token::Float(self.line, self.col, -val),
                    _ => unreachable!("lex_zero must return a numeric token"),
                }
            }
            Some(ch) if is_symbolic(ch) => {
                return self.lex_functor(line);
            }
            _ => {
                len += 1;
                Token::Funct(self.line, self.col, self.ns.name("-"))
            }
        };
        (tok, len)
    }

    /// Returns the token for a binary, octal, hexidecimal, or decimal number.
    fn lex_zero(&self, line: &str) -> (Token<'ns>, usize) {
        let mut len = 0;

        // We know the first char is '0'. The second char gives the radix.
        // If base 10, jump to `self.lex_decimal`.
        let radix: u32;
        match line.chars().nth(1) {
            Some('x') => radix = 16,
            Some('o') => radix = 8,
            Some('b') => radix = 2,
            Some('.') => return self.lex_decimal(line),
            Some(ch) if ch.is_digit(10) => return self.lex_decimal(line),
            _ => return (Token::Int(self.line, self.col, 0), 1),
        }
        len += 2;

        // Buffer up all chars in the given radix.
        let mut buf = String::with_capacity(32);
        for ch in line[2..].chars() {
            match ch {
                ch if ch.is_digit(radix) => {
                    len += ch.len_utf8();
                    buf.push(ch);
                }
                _ => break,
            }
        }

        // Parse the buffer into an integer.
        let tok = match i64::from_str_radix(buf.as_str(), radix) {
            Ok(x) => Token::Int(self.line, self.col, x),
            Err(_) => unreachable!("the buffer must be valid in the given radix"),
        };
        (tok, len)
    }

    /// Returns the token for a decimal number.
    fn lex_decimal(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = Regex::new(r"^\d+(\.\d+)?(e-?\d+)?").unwrap();
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let float = s.chars().any(|ch| ch == 'e' || ch == '.');
        let tok = match float {
            true => Token::Float(self.line, self.col, s.parse().unwrap()),
            false => Token::Int(self.line, self.col, s.parse().unwrap()),
        };
        (tok, s.len())
    }

    /// Returns a Functor or String for a token enclosed in quotes.
    ///
    /// Escape sequences are replaced and the token will not include the
    /// surrounding quotes. An error is returned if the quote is unclosed.
    fn lex_quote(&self, line: &str) -> (Token<'ns>, usize) {
        let quote = line.chars().nth(0).unwrap();
        let mut buf = String::with_capacity(32);
        let mut escape = false;
        let mut ok = false;
        for ch in line.chars().skip(1) {
            if escape {
                match ch {
                    'n' => buf.push('\n'),
                    'r' => buf.push('\r'),
                    't' => buf.push('\t'),
                    '\\' => buf.push('\\'),
                    ch => buf.push(ch),
                }
                escape = false;
            } else {
                match ch {
                    '\\' => escape = true,
                    ch if ch == quote => {
                        ok = true;
                        break;
                    }
                    ch => buf.push(ch),
                }
            }
        }

        let len = buf.len() + 2;
        let tok = match ok {
            true if quote == '\"' => Token::Str(self.line, self.col, self.ns.name(buf)),
            true => Token::Funct(self.line, self.col, self.ns.name(buf)),
            false => Token::Err(self.line, self.col, "unclosed quote"),
        };
        (tok, len)
    }

    /// Returns the token for a single char symbol.
    fn lex_simple(&self, line: &str) -> (Token<'ns>, usize) {
        let tok = match line.chars().nth(0).unwrap() {
            '(' => Token::ParenOpen(self.line, self.col),
            ')' => Token::ParenClose(self.line, self.col),
            '[' => Token::BracketOpen(self.line, self.col),
            ']' => Token::BracketClose(self.line, self.col),
            '{' => Token::BraceOpen(self.line, self.col),
            '}' => Token::BraceClose(self.line, self.col),
            ',' => Token::Comma(self.line, self.col, self.ns.name(",")),
            '|' => Token::Bar(self.line, self.col, self.ns.name("|")),
            '.' => Token::Dot(self.line, self.col),
            _ => unreachable!("lex_simple must be called with a simple character"),
        };
        (tok, 1)
    }

    /// Returns the next whitespace token.
    fn lex_space(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = Regex::new(r"^[\s\p{C}]+").unwrap();
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let tok = Token::Space(self.line, self.col);
        (tok, s.len())
    }

    /// Retuns a token giving the text of a comment.
    fn lex_comment(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = Regex::new(r"^%.*").unwrap();
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let tok = Token::Space(self.line, self.col);
        (tok, s.len())
    }
}

// Helpers
// --------------------------------------------------

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
    #[cfg_attr(rustfmt, rustfmt_skip)]
    fn basic() {
        let pl = "_abcd ABCD foobar 'hello world' +++\n\
                  % this is a comment\n\
                  123 456.789 8.765e43 1e-1\n\
                  0xDEADBEEF 0o644 0b11001100 0987654321 0.123\n\
                  -> -0xff -1.23 (-)\n\
                  \t\t   \t\n";
        let ns = NameSpace::new();
        let mut toks = Lexer::new(pl.as_bytes(), &ns);
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
    #[cfg_attr(rustfmt, rustfmt_skip)]
    fn realistic() {
        let pl = "member(H, [H|T]).\n\
                  member(X, [_|T]) :- member(X, T).\n";
        let ns = NameSpace::new();
        let mut toks = Lexer::new(pl.as_bytes(), &ns);

        // member(H, [H|T]).
        assert_eq!(toks.next().unwrap(), Token::Funct(1, 1, ns.name("member")));
        assert_eq!(toks.next().unwrap(), Token::ParenOpen(1, 7));
        assert_eq!(toks.next().unwrap(), Token::Var(1, 8, ns.name("H")));
        assert_eq!(toks.next().unwrap(), Token::Comma(1, 9, ns.name(",")));
        assert_eq!(toks.next().unwrap(), Token::BracketOpen(1, 11));
        assert_eq!(toks.next().unwrap(), Token::Var(1, 12, ns.name("H")));
        assert_eq!(toks.next().unwrap(), Token::Bar(1, 13, ns.name("|")));
        assert_eq!(toks.next().unwrap(), Token::Var(1, 14, ns.name("T")));
        assert_eq!(toks.next().unwrap(), Token::BracketClose(1, 15));
        assert_eq!(toks.next().unwrap(), Token::ParenClose(1, 16));
        assert_eq!(toks.next().unwrap(), Token::Dot(1, 17));

        // member(X, [_|T]) :- member(X, T).
        assert_eq!(toks.next().unwrap(), Token::Funct(2, 1, ns.name("member")));
        assert_eq!(toks.next().unwrap(), Token::ParenOpen(2, 7));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 8, ns.name("X")));
        assert_eq!(toks.next().unwrap(), Token::Comma(2, 9, ns.name(",")));
        assert_eq!(toks.next().unwrap(), Token::BracketOpen(2, 11));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 12, ns.name("_")));
        assert_eq!(toks.next().unwrap(), Token::Bar(2, 13, ns.name("|")));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 14, ns.name("T")));
        assert_eq!(toks.next().unwrap(), Token::BracketClose(2, 15));
        assert_eq!(toks.next().unwrap(), Token::ParenClose(2, 16));
        assert_eq!(toks.next().unwrap(), Token::Funct(2, 18, ns.name(":-")));
        assert_eq!(toks.next().unwrap(), Token::Funct(2, 21, ns.name("member")));
        assert_eq!(toks.next().unwrap(), Token::ParenOpen(2, 27));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 28, ns.name("X")));
        assert_eq!(toks.next().unwrap(), Token::Comma(2, 29, ns.name(",")));
        assert_eq!(toks.next().unwrap(), Token::Var(2, 31, ns.name("T")));
        assert_eq!(toks.next().unwrap(), Token::ParenClose(2, 32));
        assert_eq!(toks.next().unwrap(), Token::Dot(2, 33));

        assert_eq!(toks.next(), None);
    }
}
