//! A lexer for logic programs.
//!
//! A [`Lexer`] lifts a buffered reader into an iterator over [`Token`]s.
//! Errors may occur at both the I/O and lexing levels. These are handled
//! in-band, meaning that a special token type, `Token::Err`, is reserved for
//! passing errors to the caller. This greatly simplifies error handling logic
//! when iterating over tokens.
//!
//! [`Lexer`]: ./struct.Lexer.html
//! [`Token`]: ./enum.Token.html

use std::fmt;
use std::io::BufRead;

use regex::Regex;
use unicode_normalization::UnicodeNormalization;

use syntax::namespace::{NameSpace, Name};
use syntax::error::SyntaxError;

/// A lexer for logic programs.
///
/// The lexer interface is an iterator over [`Token`]s.
///
/// [`Token`]: ./enum.Token.html
pub struct Lexer<'ns, B: BufRead> {
    reader: B,
    ns: &'ns NameSpace,
    line: usize,
    col: usize,
    skip_space: bool,

    // Two buffers: The first holds each line.
    // The second holds the normalized form of the line.
    buf_line: String,
    buf_norm: String,
}

/// A lexical item of a logic program.
///
/// Every `Token` includes its line and column as the first two members. When
/// relevant, the third member gives an interpreted value of the token.
///
/// Lexical errors are given as a `Token::Err` whose value is the error message.
#[derive(Debug)]
#[derive(PartialEq)]
pub enum Token<'ns> {
    Err(SyntaxError),
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

// Public API
// --------------------------------------------------

impl<'ns, B: BufRead> Lexer<'ns, B> {
    /// Constructs a new lexer from a stream of chars.
    ///
    /// By default, the lexer is configured to skip space and comment tokens.
    pub fn new(reader: B, ns: &'ns NameSpace) -> Self {
        Lexer {
            reader: reader,
            ns: ns,
            line: 0, // incremented on first line
            col: 1,
            skip_space: true,
            buf_line: String::with_capacity(128),
            buf_norm: String::with_capacity(128),
        }
    }

    /// Toggles whether space and comment tokens are reported.
    pub fn report_space(mut self, yes: bool) -> Self {
        self.skip_space = yes;
        self
    }

    /// Returns the line of the next token to be emitted by the lexer.
    pub fn line(&self) -> usize {
        self.line
    }

    /// Returns the column of the next token to be emitted by the lexer.
    pub fn col(&self) -> usize {
        self.col
    }
}

impl<'ns, B: BufRead> Iterator for Lexer<'ns, B> {
    type Item = Token<'ns>;

    /// Extracts the next token from the underlying reader.
    fn next(&mut self) -> Option<Token<'ns>> {
        // Refill the buffers.
        if self.buf_norm.len() <= self.col {
            self.line += 1;
            self.col = 1;
            self.buf_line.clear();
            match self.reader.read_line(&mut self.buf_line) {
                Ok(0) => return None, // Nothing more to read
                Ok(_) => (),          // The buffer is refilled successfully
                Err(e) => return Some(Token::Err(SyntaxError::wrap(self.line, self.col, e))),
            }

            // Perform Unicode normalization.
            // This has security, usability, and performance implications.
            self.buf_norm.clear();
            self.buf_norm.extend(self.buf_line.nfkc());
        }

        // Lex the next token.
        let (tok, len) = self.lex(&self.buf_norm[self.col - 1..]);
        self.col += len;

        // Skip space and comment tokens.
        match tok {
            Token::Space(..) if self.skip_space => self.next(),
            Token::Comment(..) if self.skip_space => self.next(),
            _ => Some(tok),
        }
    }
}

impl<'ns> Token<'ns> {
    /// Returns the line number of the start of the token.
    #[inline]
    pub fn line(&self) -> usize {
        match *self {
            Token::Err(ref err) => err.line(),
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

    /// Returns the column number of the start of the token.
    #[inline]
    pub fn col(&self) -> usize {
        match *self {
            Token::Err(ref err) => err.col(),
            Token::Funct(_, col, ..) => col,
            Token::Str(_, col, ..) => col,
            Token::Var(_, col, ..) => col,
            Token::Int(_, col, ..) => col,
            Token::Float(_, col, ..) => col,
            Token::ParenOpen(_, col, ..) => col,
            Token::ParenClose(_, col, ..) => col,
            Token::BracketOpen(_, col, ..) => col,
            Token::BracketClose(_, col, ..) => col,
            Token::BraceOpen(_, col, ..) => col,
            Token::BraceClose(_, col, ..) => col,
            Token::Bar(_, col, ..) => col,
            Token::Comma(_, col, ..) => col,
            Token::Dot(_, col, ..) => col,
            Token::Space(_, col, ..) => col,
            Token::Comment(_, col, ..) => col,
        }
    }
}

impl<'ns> fmt::Display for Token<'ns> {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match *self {
            Token::Err(ref err) => write!(f, "{}", err),
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

            // TODO: Space and Comment should report their content.
            Token::Space(..) => f.write_str("SPACE"),
            Token::Comment(..) => f.write_str("COMMENT"),
        }
    }
}

// Lexing Logic
// --------------------------------------------------

impl<'ns, B: BufRead> Lexer<'ns, B> {
    /// The main switch of the lexer.
    fn lex(&self, line: &str) -> (Token<'ns>, usize) {
        match line.chars().nth(0).unwrap() {
            '(' => self.lex_simple(line),
            ')' => self.lex_simple(line),
            '[' => self.lex_simple(line),
            ']' => self.lex_simple(line),
            '{' => self.lex_simple(line),
            '}' => self.lex_simple(line),
            ',' => self.lex_simple(line),
            '|' => self.lex_simple(line),
            '.' => self.lex_simple(line),
            '%' => self.lex_comment(line),
            '_' => self.lex_var(line),
            '\'' => self.lex_quote(line),
            '\"' => self.lex_quote(line),
            '-' => self.lex_minus(line),
            '0' => self.lex_zero(line),
            ch if ch.is_digit(10) => self.lex_decimal(line),
            ch if ch.is_whitespace() => self.lex_space(line),
            ch if ch.is_control() => self.lex_space(line),
            ch if ch.is_uppercase() => self.lex_var(line),
            _ => self.lex_functor(line),
        }
    }

    /// Returns the token for the next function symbol.
    ///
    /// Function symbols are composed of either only alphanumeric characters
    /// and underscores or only symbols and punctuation. Function symbols may
    /// not start with a capital or underscore (though this is not checked).
    ///
    /// Commas, periods, and pipes are not allowed within other function
    /// symbols.
    ///
    /// The token MUST be at the start of the line.
    fn lex_functor(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = {
                let pattern = r"^([\w\d]+|[\p{S}\p{Pc}\p{Pd}\p{Po}]+)";
                Regex::new(pattern).unwrap()
            };
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str().split(|ch| ch == ',' || ch == '.' || ch == '|').nth(0).unwrap();
        let tok = Token::Funct(self.line(), self.col(), self.ns.name(s));
        (tok, s.len())
    }

    /// Returns the token for a variable term.
    ///
    /// Variables start with a capital letter or underscore and are composed
    /// only of letters and underscores.
    ///
    /// The token MUST be at the start of the line.
    fn lex_var(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = {
                let pattern = r"^[\p{Lu}_][\w\d]*";
                Regex::new(pattern).unwrap()
            };
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let tok = Token::Var(self.line(), self.col(), self.ns.name(s));
        (tok, s.len())
    }

    /// Returns the token for a symbol starting with a minus.
    ///
    /// A minus can start both numeric and function symbol tokens.
    ///
    /// The token MUST be at the start of the line.
    fn lex_minus(&self, line: &str) -> (Token<'ns>, usize) {
        let mut len = 0;
        let tok = match line.chars().nth(1) {
            Some('0') => {
                let (subtok, sublen) = self.lex_zero(&line[1..]);
                len += 1 + sublen;
                match subtok {
                    Token::Int(_, _, val) => Token::Int(self.line(), self.col(), -val),
                    Token::Float(_, _, val) => Token::Float(self.line(), self.col(), -val),
                    _ => unreachable!("lex_zero must return a numeric token"),
                }
            }
            Some(ch) if ch.is_digit(10) => {
                let (subtok, sublen) = self.lex_decimal(&line[1..]);
                len += 1 + sublen;
                match subtok {
                    Token::Int(_, _, val) => Token::Int(self.line(), self.col(), -val),
                    Token::Float(_, _, val) => Token::Float(self.line(), self.col(), -val),
                    _ => unreachable!("lex_zero must return a numeric token"),
                }
            }
            _ => return self.lex_functor(line),
        };
        (tok, len)
    }

    /// Returns the token for a number with a leading zero.
    ///
    /// This routine uses the second character to dertermine the radix:
    /// - 'x' for hexadecimal
    /// - 'o' for octal
    /// - 'b' for binary
    /// - otherwise decimal is assumed
    ///
    /// The token MUST be at the start of the line.
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
            _ => return (Token::Int(self.line(), self.col(), 0), 1),
        }
        len += 2;

        // Buffer up all chars in the given radix.
        let mut buf = String::with_capacity(32);
        buf.push('0');
        for ch in line.chars().skip(2) {
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
            Ok(x) => Token::Int(self.line(), self.col(), x),
            Err(_) => unreachable!("the buffer must be valid in the given radix"),
        };
        (tok, len)
    }

    /// Returns the token for a decimal number.
    ///
    /// Numbers follow the standard scientific notation and are allowed to be
    /// broken up arbitrarily by underscores.
    ///
    /// This routine does not handle leading signs. See `lex_minus`.
    ///
    /// The token MUST be at the start of the line.
    fn lex_decimal(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = {
                let pattern = r"^\d[\d_]*(\.[\d_]+)?(e-?[\d_]+)?";
                Regex::new(pattern).unwrap()
            };
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let float = s.chars().any(|ch| ch == 'e' || ch == '.');
        let tok = match float {
            true => Token::Float(self.line(), self.col(), s.parse().unwrap()),
            false => Token::Int(self.line(), self.col(), s.parse().unwrap()),
        };
        (tok, s.len())
    }

    /// Returns a token for a function symbol or string enclosed in quotes.
    ///
    /// Escape sequences are replaced and the token will not include the
    /// surrounding quotes. An error is returned if the quote is unclosed.
    ///
    /// The token MUST be at the start of the line.
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
            true if quote == '\"' => Token::Str(self.line(), self.col(), self.ns.name(buf)),
            true => Token::Funct(self.line(), self.col(), self.ns.name(buf)),
            false => Token::Err(SyntaxError::unbalanced(self.line(), self.col(), quote)),
        };
        (tok, len)
    }

    /// Returns the token for a single char symbol.
    ///
    /// These include the various parens as well as the comma, bar, and period.
    ///
    /// The token MUST be at the start of the line.
    fn lex_simple(&self, line: &str) -> (Token<'ns>, usize) {
        let tok = match line.chars().nth(0).unwrap() {
            '(' => Token::ParenOpen(self.line(), self.col()),
            ')' => Token::ParenClose(self.line(), self.col()),
            '[' => Token::BracketOpen(self.line(), self.col()),
            ']' => Token::BracketClose(self.line(), self.col()),
            '{' => Token::BraceOpen(self.line(), self.col()),
            '}' => Token::BraceClose(self.line(), self.col()),
            ',' => Token::Comma(self.line(), self.col(), self.ns.name(",")),
            '|' => Token::Bar(self.line(), self.col(), self.ns.name("|")),
            '.' => Token::Dot(self.line(), self.col()),
            _ => unreachable!("lex_simple must be called with a simple character"),
        };
        (tok, 1)
    }

    /// Returns the next whitespace token.
    ///
    /// This includes characters in the unicode Whitespace and Other
    /// categories, including control characters.
    ///
    /// The token MUST be at the start of the line.
    fn lex_space(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = {
                let pattern = r"^[\s\p{C}]+";
                Regex::new(pattern).unwrap()
            };
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let tok = Token::Space(self.line(), self.col());
        (tok, s.len())
    }

    /// Retuns a token for a comment.
    ///
    /// Comments start with '%' and extend to the end of the line.
    ///
    /// The token MUST be at the start of the line.
    fn lex_comment(&self, line: &str) -> (Token<'ns>, usize) {
        lazy_static! {
            static ref RE: Regex = {
                let pattern = r"^%.*";
                Regex::new(pattern).unwrap()
            };
        }

        let m = RE.find(line).unwrap();
        let s = m.as_str();
        let tok = Token::Space(self.line(), self.col());
        (tok, s.len())
    }
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
        assert!(toks.next().is_none());
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

        assert!(toks.next().is_none());
    }
}
