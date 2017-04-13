pub mod lexer;
pub mod namespace;
pub mod operators;
pub mod parser;
mod error;
mod repr;

pub use self::error::{Result, SyntaxError};
pub use self::repr::{Structure, Symbol};
use self::namespace::*;
use self::operators::*;
use self::parser::*;

use std::fs::File;
use std::io::{BufRead, BufReader};
use std::path::Path;
use std::mem;

/// Everything you need to parse a Prolog file.
///
/// A `Context` abstracts all of the ugly details around constructing parsers
/// for files and other buffered readers. Each context has a unique
/// `NameSpace`, meaning the terms produced by one context will not unify
/// with structures produced by another. The underlying namespace can be
/// accessed using the `.ns()` method.
///
/// Each `Context` wraps an `OpTable` that controls the operator parsing. The
/// operator table can be manipulated with the `.ops()` method.
pub struct Context<'a> {
    ns: NameSpace,
    ops: OpTable<'a>,
}

impl<'a> Context<'a> {
    /// Constructs a new `Context` with the default operators.
    pub fn new() -> Context<'a> {
        // SAFTEY: The operator table must not outlive the namespace.
        let ns = NameSpace::new();
        let ops = unsafe { mem::transmute(OpTable::default(&ns)) };
        Context { ns: ns, ops: ops }
    }

    /// Access the underlying `NameSpace`.
    pub fn ns(&self) -> &NameSpace {
        &self.ns
    }

    /// Manipulate the underlying `OpTable`.
    pub fn ops(&mut self) -> &mut OpTable<'a> {
        &mut self.ops
    }

    /// Parse some buffered reader.
    ///
    /// A `Parser` is an iterator over `Result<Box<Structure>, SyntaxError>`.
    pub fn parse<B: BufRead>(&self, reader: B) -> Parser<B> {
        Parser::new(reader, &self.ns, &self.ops)
    }

    /// Parse a file at the given path.
    ///
    /// See the `parse` method for more details.
    pub fn parse_file<P: AsRef<Path>>(&self, path: P) -> Parser<BufReader<File>> {
        let path = path.as_ref();
        let f = File::open(path).unwrap();
        let bf = BufReader::new(f);
        self.parse(bf)
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use super::repr::Symbol::*;

    #[test]
    fn parse_file() {
        let ctx = Context::new();

        let first = &[
            Var(0),
            Var(0),
            Var(1),
            List(false, 2),
            Funct(2, ctx.ns.name("member")),
        ];

        let second = &[
            Var(0),
            Var(1),
            Var(2),
            List(false, 2),
            Funct(2, ctx.ns.name("member")),
            Var(0),
            Var(2),
            Funct(2, ctx.ns.name("member")),
            Funct(2, ctx.ns.name(":-")),
        ];

        let mut parser = ctx.parse_file("./src/syntax/test/parse_test.pl");
        assert_eq!(parser.next().unwrap().unwrap().as_slice(), first);
        assert_eq!(parser.next().unwrap().unwrap().as_slice(), second);
        assert_eq!(parser.next(), None);
    }
}
