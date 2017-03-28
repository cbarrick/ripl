use std::ops::Deref;

use syntax::namespace::Name;

#[derive(Debug)]
#[derive(Clone, Copy)]
#[derive(PartialEq)]
pub enum Symbol<'ns> {
    Funct(u32, Name<'ns>),
    Str(&'ns str),
    Var(usize),
    Int(i64),
    Float(f64),
}

#[derive(Debug)]
#[derive(PartialEq)]
pub struct Structure<'ns>([Symbol<'ns>]);

// Structure
// --------------------------------------------------

impl<'ns> Structure<'ns> {
    pub fn as_slice(&self) -> &[Symbol<'ns>] {
        &self.0
    }

    pub fn functor(&self) -> Symbol<'ns> {
        self.as_slice().last().unwrap().clone()
    }

    pub fn arity(&self) -> usize {
        self.functor().arity()
    }

    pub fn validate(&self) {
        let mut n: i32 = 1;
        for sym in self.as_slice() {
            n -= 1;
            n += sym.arity() as i32;
        }
        if n != 0 {
            panic!("invalid structure");
        }
    }
}

impl<'ns> Deref for Structure<'ns> {
    type Target = [Symbol<'ns>];
    fn deref(&self) -> &[Symbol<'ns>] {
        self.as_slice()
    }
}

// Symbol
// --------------------------------------------------

impl<'ns> Symbol<'ns> {
    pub fn arity(&self) -> usize {
        match *self {
            Symbol::Funct(n, _) => n as usize,
            _ => 0,
        }
    }
}
