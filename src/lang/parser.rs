use std::iter::Peekable;
use std::mem;

use lang::lexer::{Lexer, Token};
use lang::namespace::{NameSpace, Name};
use lang::repr::{Symbol, Structure};

pub struct Parser<'ns, I>
    where I: Iterator<Item = char>
{
    inner: Peekable<Lexer<'ns, I>>,
    vars: Vec<Name<'ns>>,
    buf: Vec<Symbol<'ns>>,
}

// Public API
// --------------------------------------------------

impl<'ns, I> Parser<'ns, I>
    where I: Iterator<Item = char>
{
    pub fn new(chars: I, ns: &'ns NameSpace) -> Parser<'ns, I> {
        Parser {
            inner: Lexer::new(chars, ns).peekable(),
            vars: Vec::with_capacity(32),
            buf: Vec::with_capacity(256),
        }
    }
}

impl<'ns, I> Iterator for Parser<'ns, I>
    where I: Iterator<Item = char>
{
    type Item = Box<Structure<'ns>>;
    fn next(&mut self) -> Option<Box<Structure<'ns>>> {
        // parse the next term
        self.buf.clear();
        self.term(1200);

        match self.inner.next() {
            Some(Token::Dot(..)) => {
                if self.buf.len() == 0 {
                    None
                } else {
                    let structure = unsafe { struct_from_vec(self.buf.clone()) };
                    Some(structure)
                }
            }

            _ => panic!("expected operator"),
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
unsafe fn struct_from_vec<'ns>(vec: Vec<Symbol<'ns>>) -> Box<Structure<'ns>> {
    mem::transmute(vec.into_boxed_slice())
}

impl<'ns, I> Parser<'ns, I>
    where I: Iterator<Item = char>
{
    fn term(&mut self, prec: usize) {
        match self.inner.next() {
            Some(Token::Str(_, _, val)) => {
                self.buf.push(Symbol::Str(val.as_str()));
            }

            Some(Token::Var(_, _, val)) => {
                match self.vars.iter().position(|name| *name == val) {
                    Some(n) => self.buf.push(Symbol::Var(n)),
                    None => {
                        let n = self.vars.len();
                        self.vars.push(val);
                        self.buf.push(Symbol::Var(n));
                    }
                }
            }

            Some(Token::Int(_, _, val)) => {
                self.buf.push(Symbol::Int(val));
            }

            Some(Token::Float(_, _, val)) => {
                self.buf.push(Symbol::Float(val));
            }

            Some(Token::ParenOpen(_, _)) => {
                self.term(1200);
                if let Some(Token::ParenClose(_, _)) = self.inner.next() {
                    // It worked, do nothing
                } else {
                    panic!("expected close paren")
                }
            }

            Some(Token::BracketOpen(_, _)) => {
                // TODO
                panic!("lists not yet supported")
            }

            Some(Token::BraceOpen(_, _)) => {
                // TODO
                panic!("braces not yet supported")
            }

            Some(Token::Funct(_, _, val)) => {
                match self.inner.peek() {
                    // Compound term
                    Some(&Token::ParenOpen(_, _)) => {
                        // We reserve space for the functor by pushing a 0-ary
                        // function symbol before parsing the args. Once we
                        // know the true arity, we update the symbol.
                        let i = self.buf.len();
                        self.buf.push(Symbol::Funct(0, val));
                        let arity = self.args();
                        self.buf[i] = Symbol::Funct(arity, val);
                    }

                    // Definitly an Atom
                    Some(&Token::ParenClose(_, _)) |
                    Some(&Token::BracketClose(_, _)) |
                    Some(&Token::BraceClose(_, _)) => {
                        self.buf.push(Symbol::Funct(0, val));
                        return;
                    }

                    // Possibly prefix operator
                    _ => {
                        // TODO: Handle operator case
                        self.buf.push(Symbol::Funct(0, val));
                        return;
                    }
                }
            }

            None => return,
            _ => panic!("parser does not support all tokens"), // TODO: cover all cases
        }
        return; // TODO: implement operator parsing
    }

    fn args(&mut self) -> u32 {
        if let Some(Token::ParenOpen(_, _)) = self.inner.next() {
            let mut arity = 1;
            loop {
                self.term(999);
                match self.inner.next() {
                    Some(Token::ParenClose(_, _)) => return arity,
                    Some(Token::Comma(_, _)) => arity += 1,
                    _ => panic!("expected comma in argument list"),
                }
            }
        } else {
            // `args` MUST NOT be called if the next token is not a paren.
            unreachable!();
        }
    }
}

// Tests
// --------------------------------------------------

#[cfg(test)]
mod test {
    use super::*;
    use lang::repr::Symbol::*;

    #[test]
    fn basic() {
        let ns = NameSpace::new();
        let pl = "foo(bar, baz(123, 456.789), \"hello world\", X).\n";
        let st = vec![Funct(4, ns.name("foo")),
                      Funct(0, ns.name("bar")),
                      Funct(2, ns.name("baz")),
                      Int(123),
                      Float(456.789),
                      Str("hello world"),
                      Var(0)];
        let st = unsafe { struct_from_vec(st) };
        st.validate();
        let mut parser = Parser::new(pl.chars(), &ns);
        assert_eq!(parser.next(), Some(st));
    }
}
