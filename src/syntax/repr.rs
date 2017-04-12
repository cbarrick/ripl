//! The core representation of logical structures.
//!
//! Logic programming is a homoiconic programming paradigm, meaning the
//! syntactic structures which appear in the source code are equivalent to the
//! structures being manipulated by the program. This module houses two of the
//! core types in this paradigm: [`Symbol`] and [`Structure`]. Because of their
//! centrallity in both parsing and evaluation, it is crucial that these
//! data-structures be implemented as compactly and efficiently as possible.
//!
//! A `Symbol` represents a single string, variable, numeric or function symbol
//! of a logic program, and are small (2 words on a 64bit machine). A
//! `Structure` is an array of symbols forming a tree in postfix order.
//!
//! [`Symbol`]: ./enum.Symbol.html
//! [`Structure`]: ./struct.Structure.html

use std::ops::Deref;

use syntax::namespace::Name;

/// An atomic symbol of a logic program.
///
/// Symbols are guaranteed to fit within two words on 64bit architectures.
#[derive(Debug)]
#[derive(Clone, Copy)]
#[derive(PartialEq)]
pub enum Symbol<'ns> {
    Funct(u32, Name<'ns>),
    Str(&'ns str),
    Var(usize),
    Int(i64),
    Float(f64),
    List(bool, u32),
}

/// A tree of `Symbol`s.
#[derive(Debug)]
#[derive(PartialEq)]
pub struct Structure<'ns>([Symbol<'ns>]);

// Structure
// --------------------------------------------------

impl<'ns> Structure<'ns> {
    /// Views the `Structure` as a slice of symbols.
    pub fn as_slice(&self) -> &[Symbol<'ns>] {
        &self.0
    }

    /// Gets the root of the tree.
    pub fn functor(&self) -> Symbol<'ns> {
        self.as_slice().last().unwrap().clone()
    }

    /// Gets the arity of the root of the tree.
    pub fn arity(&self) -> usize {
        self.functor().arity()
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
    /// Gets the arity of the symbol.
    ///
    /// Function symbols can have any arity.
    /// Lists are binary.
    /// Everything else is 0-ary.
    pub fn arity(&self) -> usize {
        match *self {
            Symbol::Funct(n, _) => n as usize,
            Symbol::List(true, 0) => 0,
            Symbol::List(..) => 2,
            _ => 0,
        }
    }
}
