use std::fs::File;
use std::io::{BufRead, BufReader};
use std::path::Path;

pub mod error;
pub mod lexer;
pub mod namespace;
pub mod operators;
pub mod parser;
pub mod repr;

pub use self::error::*;

use self::namespace::*;
use self::operators::*;
use self::parser::Parser;

pub struct Context<'a> {
    ns: NameSpace,
    ops: OpTable<'a>,
}

impl<'a> Context<'a> {
    pub fn new() -> Context<'a> {
        Context {
            ns: NameSpace::new(),
            ops: OpTable::new(),
        }
    }

    pub fn parse<B: BufRead>(&self, reader: B) -> Parser<B> {
        Parser::new(reader, &self.ns, &self.ops)
    }

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

    #[test]
    fn parse_file() {
        let ctx = Context::new();
        for r in ctx.parse_file("./src/syntax/test/parse_test.pl") {
            ()
        }
    }
}
