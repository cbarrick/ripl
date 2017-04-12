//! A parser for logic programs.
//!
//! A parser lifts a buffered reader into an interator over term [`Structure`]s
//! by way of a [`NameSpace`] and [`OpTable`]. The `NameSpace` will be used to
//! assign names to the symbols of the `Structure`s, and the `OpTable` will be
//! used to parse operators. The references to the `NameSpace` and `OpTable`
//! are treated with a single lifetime, `'ctx`, because they are assumed to be
//! owned by roughly the same calling context.
//!
//! Errors at both the I/O and syntax levels are saved into a buffer and may be
//! accessed through the [`errs`] method. If there are any errors, then the
//! structures emitted by the parser cannot be assumed to accurately represent
//! the (possibly invalid) source program.
//!
//! For more information on the syntax of logic programs, see the Wikipedia
//! article on the [syntax and semantics of Prolog][1].
//!
//! [`Structure`]: ../repr/struct.Structure.html
//! [`NameSpace`]: ../namespace/struct.NameSpace.html
//! [`OpTable`]: ../operators/struct.OpTable.html
//! [`errs`]: #method.errs
//!
//! [1]: https://en.wikipedia.org/wiki/Prolog_syntax_and_semantics

use std::io::BufRead;
use std::mem;
use std::vec::Drain;

use syntax::error::{SyntaxError, Result};
use syntax::lexer::{Lexer, Token};
use syntax::namespace::{NameSpace, Name};
use syntax::operators::{OpTable, Op};
use syntax::repr::{Structure, Symbol};

/// An iterator over [`Structure`]s in UTF-8 text.
///
/// The parser requires a reference to a [`NameSpace`] to assign names to
/// constants and a reference to an [`OpTable`] to specify the operators and
/// their precedence. The lifetime `'ctx` refers to both references.
///
/// The parser is implemented using the [precedence climbing method][1] and is
/// independent of the set of operators. Further, the operator table is allowed
/// to be modified at runtime.
///
/// [`Structure`]: ../repr/struct.Structure.html
/// [`NameSpace`]: ../namespace/struct.NameSpace.html
/// [`OpTable`]: ../operators/struct.OpTable.html
///
/// [1]: https://en.wikipedia.org/wiki/Operator-precedence_parser#Precedence_climbing_method
pub struct Parser<'ctx, B: BufRead> {
    ops: &'ctx OpTable<'ctx>,
    lexer: Lexer<'ctx, B>,
    peeked: Option<Token<'ctx>>,
    errs: Vec<SyntaxError>,
    vars: Vec<Name<'ctx>>,
    buf: Vec<Symbol<'ctx>>,
}

// Public API
// --------------------------------------------------

impl<'ctx, B: BufRead> Parser<'ctx, B> {
    /// Constructs a new `Parser` from the given reader, namespace, and operator table.
    pub fn new(reader: B, ns: &'ctx NameSpace, ops: &'ctx OpTable<'ctx>) -> Parser<'ctx, B> {
        Parser {
            ops: ops,
            lexer: Lexer::new(reader, ns),
            peeked: None,
            errs: Vec::new(),
            vars: Vec::with_capacity(32),
            buf: Vec::with_capacity(256),
        }
    }

    /// Returns a draining iterator over the set of errors.
    pub fn errs(&mut self) -> Drain<SyntaxError> {
        self.errs.drain(0..)
    }
}

impl<'ctx, B: BufRead> Iterator for Parser<'ctx, B> {
    type Item = Box<Structure<'ctx>>;

    fn next(&mut self) -> Option<Box<Structure<'ctx>>> {
        self.vars.clear();
        self.buf.clear();
        match self.read(1200) {
            Ok(_) => {
                if self.buf.len() == 0 {
                    // `read` produced no results.
                    // Must be at end of input.
                    None
                } else if let Some(Token::Dot(..)) = self.next_tok() {
                    let structure = unsafe { struct_from_vec(self.buf.clone()) };
                    Some(structure)
                } else {
                    let line = self.lexer.line();
                    let col = self.lexer.col();
                    self.errs.push(SyntaxError::priority_clash(line, col));
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
unsafe fn struct_from_vec<'ctx>(vec: Vec<Symbol<'ctx>>) -> Box<Structure<'ctx>> {
    mem::transmute(vec.into_boxed_slice())
}

impl<'ctx, B: BufRead> Parser<'ctx, B> {
    /// Reads the next term up to, but not including, the trailing period.
    ///
    /// The return value is the precedence of the term upon success or
    /// otherwise a syntax error. Upon successfully returning, the parse tree
    /// exists in the buffer.
    ///
    /// The algorithm implemented here is [Precedence climbing][1].
    ///
    /// [1]: https://en.wikipedia.org/wiki/Operator-precedence_parser#Precedence_climbing_method
    fn read(&mut self, max_prec: u32) -> Result<u32> {
        // Check that we're not at EOF.
        if self.peek_tok().is_none() {
            return Ok(0);
        }

        // Precedence "climbing" algorithm.
        // Lower precedence values equate to higher logical precedence.
        // Thus all comparisons are the opposite of the pseudo-code.
        let mut prec = self.read_primary(max_prec)?;
        loop {
            match self.peek_tok() {
                Some(&Token::Bar(.., name)) |
                Some(&Token::Comma(.., name)) |
                Some(&Token::Funct(.., name)) => {
                    match self.ops.get_compatible(name, max_prec, prec) {
                        None => break,
                        Some(op) => {
                            self.next_tok();
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

    /// Reads a primary at the given precedence.
    ///
    /// A primary is a terminal from the point-of-view of the operator-
    /// precedence parser. This includes atoms, compounds, variables, numbers,
    /// lists, strings. This step also recursively descends to parse terms
    /// grouped in parens.
    fn read_primary(&mut self, max_prec: u32) -> Result<u32> {
        match self.next_tok() {
            // Skip spaces and comments.
            Some(Token::Space(..)) |
            Some(Token::Comment(..)) => {
                return self.read_primary(max_prec);
            }

            // Atoms, compounds, and prefix operators.
            Some(Token::Bar(.., name)) |
            Some(Token::Comma(.., name)) |
            Some(Token::Funct(.., name)) => {
                match self.peek_tok() {
                    // Compound term
                    Some(&Token::ParenOpen(..)) => {
                        let arity = self.read_args()?;
                        self.buf.push(Symbol::Funct(arity, name));
                        Ok(0)
                    }

                    // Definitly an atom
                    Some(&Token::ParenClose(..)) |
                    Some(&Token::BracketClose(..)) |
                    Some(&Token::BraceClose(..)) => {
                        self.buf.push(Symbol::Funct(0, name));
                        Ok(0)
                    }

                    // Possibly prefix operator
                    _ => {
                        match self.ops.get_prefix(name, max_prec) {
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
                                Ok(0)
                            }
                        }
                    }
                }
            }

            // Strings.
            Some(Token::Str(.., val)) => {
                self.buf.push(Symbol::Str(val.as_str()));
                Ok(0)
            }

            // Variables.
            Some(Token::Var(.., val)) => {
                match self.vars.iter().position(|name| *name == val) {
                    Some(n) => {
                        self.buf.push(Symbol::Var(n));
                        Ok(0)
                    }
                    None => {
                        let n = self.vars.len();
                        self.vars.push(val);
                        self.buf.push(Symbol::Var(n));
                        Ok(0)
                    }
                }
            }

            // Numbers.
            Some(Token::Int(.., val)) => {
                self.buf.push(Symbol::Int(val));
                Ok(0)
            }
            Some(Token::Float(.., val)) => {
                self.buf.push(Symbol::Float(val));
                Ok(0)
            }

            // Parens.
            Some(Token::ParenOpen(line, col)) => {
                self.read(1200)?;
                match self.next_tok() {
                    Some(Token::ParenClose(..)) => Ok(0),
                    _ => Err(SyntaxError::unbalanced(line, col, ')')),
                }
            }

            // TODO: Lists and braces.
            Some(Token::BracketOpen(line, col)) => Err(SyntaxError::todo(line, col)),
            Some(Token::BraceOpen(line, col)) => Err(SyntaxError::todo(line, col)),

            // Syntax errors.
            Some(Token::ParenClose(line, col)) => Err(SyntaxError::unbalanced(line, col, ')')),
            Some(Token::BracketClose(line, col)) => Err(SyntaxError::unbalanced(line, col, ']')),
            Some(Token::BraceClose(line, col)) => Err(SyntaxError::unbalanced(line, col, '}')),
            Some(Token::Dot(line, col)) => Err(SyntaxError::unexpected(line, col, "period")),
            Some(Token::Err(e)) => Err(e),
            None => Err(SyntaxError::unexpected(self.lexer.line(), self.lexer.col(), "eof")),
        }
    }

    /// Reads a list of argument for a compound term or list.
    ///
    /// Because the precedence of the comma operator is 1000, the precedence of
    /// arguments must be less than 1000 to avoid conflicting. This can be
    /// ensured by wrapping arguments in parens.
    ///
    /// TODO: support lists
    fn read_args(&mut self) -> Result<u32> {
        let front = self.next_tok();
        match front {
            Some(Token::ParenOpen(..)) => (),
            None => {
                let line = self.lexer.line();
                let col = self.lexer.col();
                return Err(SyntaxError::unexpected(line, col, "eof"));
            }
            _ => panic!("must not call read_args in this context"),
        }

        let mut arity = 1;
        loop {
            self.read(999)?;
            match self.next_tok() {
                Some(Token::ParenClose(..)) => return Ok(arity),
                Some(Token::Comma(..)) => arity += 1,

                Some(tok) => return Err(SyntaxError::priority_clash(tok.line(), tok.col())),
                None => {
                    let line = self.lexer.line();
                    let col = self.lexer.col();
                    return Err(SyntaxError::unexpected(line, col, "eof"));
                }
            }
        }
    }

    /// Implement token peeking.
    ///
    /// We implement peeking manually instead of using `std::iter::Peekable`.
    /// This lets us keep direct ownership of the lexer and call its methods.
    fn peek_tok(&mut self) -> Option<&Token<'ctx>> {
        match self.peeked {
            Some(ref tok) => Some(tok),
            None => {
                self.peeked = self.next_tok();
                match self.peeked {
                    Some(ref tok) => Some(tok),
                    None => None,
                }
            }
        }
    }

    /// Get the next token from the lexer.
    ///
    /// Calling `self.lexer.next()` directly outside of this or `peek_tok`
    /// will poison the peek cache.
    fn next_tok(&mut self) -> Option<Token<'ctx>> {
        match self.peeked.take() {
            Some(tok) => Some(tok),
            None => {
                match self.lexer.next() {
                    None => None,
                    Some(tok) => Some(tok),
                }
            }
        }
    }
}

// Tests
// --------------------------------------------------

#[cfg(test)]
mod test {
    use super::*;
    use syntax::repr::Symbol::*;

    #[test]
    fn basic() {
        let ns = NameSpace::new();
        let ops = OpTable::default(&ns);

        let pl = "-foo(bar, baz(123, 456.789), \"hello world\", X).\n";
        let st = vec![Funct(0, ns.name("bar")),
                      Int(123),
                      Float(456.789),
                      Funct(2, ns.name("baz")),
                      Str("hello world"),
                      Var(0),
                      Funct(4, ns.name("foo")),
                      Funct(1, ns.name("-"))];
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
