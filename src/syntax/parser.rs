//! A parser for logic programs.
//!
//! The syntax of Ripl closely resembles [ISO Prolog][1]. Prolog is a simple
//! operator precedence language with function symbols, lists, variables,
//! numbers, and infix, prefix, and postfix operators.
//!
//! The parsing logic for Prolog is independent of the set of operators,
//! allowing the operator to be modified at runtime. The parser is implemented
//! using the [precedence climbing method][2], however the notion of precedence
//! is inverted from the textbook definition. In Prolog, greater precedence is
//! given to the outermost operators (`+` is said to be of greater precedence
//! than `*`). Thus the parsing algorithm is a precedence *descending* method.
//!
//! [1]: https://en.wikipedia.org/wiki/Prolog_syntax_and_semantics
//! [2]: https://en.wikipedia.org/wiki/Operator-precedence_parser

use std::fmt;
use std::io::BufRead;
use std::iter::Peekable;
use std::mem;
use std::vec::Drain;

use syntax::lexer::{Lexer, Token};
use syntax::namespace::{NameSpace, Name};
use syntax::operators::{OpTable, Op};
use repr::{Structure, Symbol};

/// An iterator over `Structure`s in UTF-8 text.
///
/// The parser requires a reference to a `NameSpace` to assign names to
/// constants and a reference to an `OpTable` to specify the operators and
/// their precedence. The lifetime `'a` refers to both references.
///
/// A `Parser` maintains a list of encountered syntax errors. If this list is
/// non-empty, then the structures emitted cannot be assumed to be valid.
pub struct Parser<'a, B: BufRead> {
    ops: &'a OpTable<'a>,
    lexer: Peekable<Lexer<'a, B>>,
    errs: Vec<SyntaxError>,
    vars: Vec<Name<'a>>,
    buf: Vec<Symbol<'a>>,
}

/// The location and description of syntax errors.
#[derive(Debug)]
#[derive(Clone)]
#[derive(PartialEq, Eq)]
#[derive(PartialOrd, Ord)]
pub struct SyntaxError {
    pub line: usize,
    pub col: usize,
    pub msg: String,
}

/// A type alias for results with possible `SyntaxError`s.
pub type Result<T> = ::std::result::Result<T, SyntaxError>;

// Public API
// --------------------------------------------------

impl<'a, B: BufRead> Parser<'a, B> {
    pub fn new(reader: B, ns: &'a NameSpace, ops: &'a OpTable<'a>) -> Parser<'a, B> {
        Parser {
            ops: ops,
            lexer: Lexer::new(reader, ns).peekable(),
            errs: Vec::new(),
            vars: Vec::with_capacity(32),
            buf: Vec::with_capacity(256),
        }
    }

    pub fn errs(&mut self) -> Drain<SyntaxError> {
        self.errs.drain(0..)
    }
}

impl<'a, B: BufRead> Iterator for Parser<'a, B> {
    type Item = Box<Structure<'a>>;

    fn next(&mut self) -> Option<Box<Structure<'a>>> {
        self.vars.clear();
        self.buf.clear();
        match self.read(1200) {
            Ok(_) => {
                if self.buf.len() == 0 {
                    None
                } else if let Some(Token::Dot(..)) = self.lexer.next() {
                    let structure = unsafe { struct_from_vec(self.buf.clone()) };
                    Some(structure)
                } else {
                    // TODO: get line and col numbers
                    self.errs.push(SyntaxError {
                        line: 0,
                        col: 0,
                        msg: "operator priority clash".to_string(),
                    });
                    self.next()
                }
            }
            Err(err) => {
                self.errs.push(err);
                return self.next();
            }
        }
    }
}

// Parsing Logic
// --------------------------------------------------

/// Converts a vector of symbols into a structure.
///
/// This is unsafe because an arbitrary vector of symbols in not necessarily a
/// valid structure. Assuming the correctness of the parsing algorithm, it is
/// safe to call this function on the completed buffer.
unsafe fn struct_from_vec<'a>(vec: Vec<Symbol<'a>>) -> Box<Structure<'a>> {
    mem::transmute(vec.into_boxed_slice())
}

impl<'a, B: BufRead> Parser<'a, B> {
    /// Reads the next term up to, but not including, the trailing period.
    ///
    /// The return value is the precedence of the term upon success
    /// or otherwise a syntax error.
    ///
    /// Upon returning, the parse tree exists in the buffer.
    fn read(&mut self, max_prec: u32) -> Result<u32> {
        self.read_primary(max_prec)?;
        let mut prec = 0;
        loop {
            match self.lexer.peek() {
                Some(&Token::Bar(.., name)) |
                Some(&Token::Comma(.., name)) |
                Some(&Token::Funct(.., name)) => {
                    match self.ops.get_compatible(name, prec, max_prec) {
                        None => break,
                        Some(op) => {
                            self.lexer.next();
                            match op {
                                Op::XFY(..) => {
                                    prec = self.read(op.prec())?;
                                    self.buf.push(Symbol::Funct(2, name));
                                }
                                Op::YFX(..) | Op::XFX(..) => {
                                    prec = self.read(op.prec() - 1)?;
                                    self.buf.push(Symbol::Funct(2, name));
                                }
                                _ => {
                                    self.buf.push(Symbol::Funct(1, name));
                                }
                            }
                        }
                    }
                }
                _ => break,
            }
        }
        Ok(prec)
    }

    /// Reads the left side of an infix operator at a given precedence.
    fn read_primary(&mut self, prec: u32) -> Result<u32> {
        match self.lexer.next() {
            Some(Token::Space(..)) |
            Some(Token::Comment(..)) => {
                return self.read_primary(prec);
            }

            Some(Token::Bar(.., name)) |
            Some(Token::Comma(.., name)) |
            Some(Token::Funct(.., name)) => {
                match self.lexer.peek() {
                    // Compound term
                    Some(&Token::ParenOpen(..)) => {
                        let arity = self.read_args()?;
                        self.buf.push(Symbol::Funct(arity, name));
                        Ok(prec)
                    }

                    // Definitly an atom
                    Some(&Token::ParenClose(..)) |
                    Some(&Token::BracketClose(..)) |
                    Some(&Token::BraceClose(..)) => {
                        self.buf.push(Symbol::Funct(0, name));
                        Ok(prec)
                    }

                    // Possibly prefix operator
                    _ => {
                        match self.ops.get_prefix(name, prec) {
                            Some(Op::FX(p, _)) => {
                                self.read(p - 1)?;
                                self.buf.push(Symbol::Funct(1, name));
                                Ok(p)
                            }
                            Some(Op::FY(p, _)) => {
                                self.read(p)?;
                                self.buf.push(Symbol::Funct(1, name));
                                Ok(p)
                            }
                            _ => {
                                self.buf.push(Symbol::Funct(0, name));
                                Ok(prec)
                            }
                        }
                    }
                }
            }

            Some(Token::Str(.., val)) => {
                self.buf.push(Symbol::Str(val.as_str()));
                Ok(prec)
            }

            Some(Token::Var(.., val)) => {
                match self.vars.iter().position(|name| *name == val) {
                    Some(n) => {
                        self.buf.push(Symbol::Var(n));
                        Ok(prec)
                    }
                    None => {
                        let n = self.vars.len();
                        self.vars.push(val);
                        self.buf.push(Symbol::Var(n));
                        Ok(prec)
                    }
                }
            }

            Some(Token::Int(.., val)) => {
                self.buf.push(Symbol::Int(val));
                Ok(prec)
            }

            Some(Token::Float(.., val)) => {
                self.buf.push(Symbol::Float(val));
                Ok(prec)
            }

            Some(Token::ParenOpen(line, col)) => {
                self.read(1200)?;
                match self.lexer.next() {
                    Some(Token::ParenClose(..)) => Ok(prec),
                    Some(Token::Err(line, col, err)) => syntax_error(line, col, err),
                    _ => syntax_error(line, col, "unclosed paren"),
                }
            }

            // TODO
            Some(Token::BracketOpen(line, col)) => {
                syntax_error(line, col, "lists are not yet supported")
            }

            // TODO
            Some(Token::BraceOpen(line, col)) => {
                syntax_error(line, col, "braces are not yet supported")
            }

            Some(Token::ParenClose(line, col)) => syntax_error(line, col, "unbalanced ')'"),
            Some(Token::BracketClose(line, col)) => syntax_error(line, col, "unbalanced ']'"),
            Some(Token::BraceClose(line, col)) => syntax_error(line, col, "unbalanced '}'"),
            Some(Token::Dot(line, col)) => syntax_error(line, col, "unexpected period"),
            Some(Token::Err(line, col, msg)) => syntax_error(line, col, msg),
            None => syntax_error(0, 0, "unexpected eof"),
        }
    }

    /// Reads an argument list for a compound term or list.
    /// TODO: support lists
    fn read_args(&mut self) -> Result<u32> {
        let front = self.lexer.next();
        match front {
            Some(Token::ParenOpen(..)) => (),
            None => return syntax_error(0, 0, "unexpected eof"),
            _ => panic!("must not call read_args in this context"),
        }

        let mut arity = 1;
        loop {
            self.read(999)?;
            match self.lexer.next() {
                Some(Token::ParenClose(..)) => return Ok(arity),
                Some(Token::Comma(..)) => arity += 1,

                Some(Token::Err(line, col, msg)) => return syntax_error(line, col, msg),
                Some(tok) => {
                    let msg = format!("expected comma between arguments, found '{}'", tok);
                    return syntax_error(tok.line(), tok.col(), msg);
                }
                None => return syntax_error(0, 0, "unexpected eof"),
            }
        }
    }
}

// SyntaxError
// --------------------------------------------------

fn syntax_error<T>(line: usize, col: usize, msg: T) -> Result<u32>
    where T: Into<String>
{
    Err(SyntaxError {
        line: line,
        col: col,
        msg: msg.into(),
    })
}

impl<'a> fmt::Display for SyntaxError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{}:{}: {}", self.line, self.col, self.msg)
    }
}

// Tests
// --------------------------------------------------

#[cfg(test)]
mod test {
    use super::*;
    use repr::Symbol::*;

    #[test]
    fn basic() {
        let ns = NameSpace::new();
        let ops = OpTable::default(&ns);

        let pl = "+foo(bar, baz(123, 456.789), \"hello world\", X).\n";
        let st = vec![Funct(0, ns.name("bar")),
                      Int(123),
                      Float(456.789),
                      Funct(2, ns.name("baz")),
                      Str("hello world"),
                      Var(0),
                      Funct(4, ns.name("foo")),
                      Funct(1, ns.name("+"))];
        let st = unsafe { struct_from_vec(st) };

        let mut parser = Parser::new(pl.as_bytes(), &ns, &ops);
        assert_eq!(parser.errs().count(), 0);
        assert_eq!(parser.next(), Some(st));
    }

    #[test]
    fn basic_operators() {
        let ns = NameSpace::new();
        let ops = OpTable::default(&ns);

        let pl = "a * b + c * d.\n";
        let st = vec![Funct(0, ns.name("a")),
                      Funct(0, ns.name("b")),
                      Funct(2, ns.name("*")),
                      Funct(0, ns.name("c")),
                      Funct(0, ns.name("d")),
                      Funct(2, ns.name("*")),
                      Funct(2, ns.name("+"))];
        let st = unsafe { struct_from_vec(st) };

        let mut parser = Parser::new(pl.as_bytes(), &ns, &ops);
        assert_eq!(parser.next(), Some(st));
        assert_eq!(parser.errs().count(), 0);
    }

    #[test]
    fn realistic() {
        let ns = NameSpace::new();
        let ops = OpTable::default(&ns);

        // TODO: update to list syntax
        let pl = "member(H, list(H,T)).\n\
                  member(X, list(_,T)) :- member(X, T).\n";

        let first =
            &[Var(0), Var(0), Var(1), Funct(2, ns.name("list")), Funct(2, ns.name("member"))];
        let second = &[Var(0),
                       Var(1),
                       Var(2),
                       Funct(2, ns.name("list")),
                       Funct(2, ns.name("member")),
                       Var(0),
                       Var(2),
                       Funct(2, ns.name("member")),
                       Funct(2, ns.name(":-"))];

        let mut parser = Parser::new(pl.as_bytes(), &ns, &ops);

        assert_eq!(parser.next().unwrap().as_slice(), first);
        assert_eq!(parser.errs().count(), 0);

        assert_eq!(parser.next().unwrap().as_slice(), second);
        assert_eq!(parser.errs().count(), 0);
    }
}
