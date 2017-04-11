use std::error::Error;
use std::fs::File;
use std::io::{BufRead, BufReader};
use std::path::Path;

pub mod error;
pub mod lexer;
pub mod namespace;
pub mod operators;
pub mod parser;
pub mod repr;

use self::error::*;
use self::namespace::*;
use self::operators::*;
use self::parser::Parser;
use self::repr::*;

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

    // pub fn parse_file<P: AsRef<Path>>(&self, path: P) -> Result<Parser<BufReader<File>>> {
    //     let path = path.as_ref();
    //     let f = match File::open(path) {
    //         Ok(f) => f,
    //         Err(err) => return syntax_error(0, 0, err.description()),
    //     };
    //     let bf = BufReader::new(f);
    //     Ok(self.parse(bf))
    // }
}
