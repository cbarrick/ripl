//! Assigns names to tokens.
//!
//! Logic programming is a homoiconic programming paradigm, meaning the
//! syntactic structures which appear in the source code are equivalent to the
//! structures being manipulated by the program. To avoid costly string
//! operations, we must use a lightweight representation for atomic symbols.
//!
//! This lightweight representation is the [`Name`]. A `Name` is essentially
//! a `&str` string slice, except equality is optimized to a single pointer
//! comparison.
//!
//! To ensure that all equivalent strings are represented by the same `Name`,
//! we employ a [`NameSpace`]. A `NameSpace` is essentially a string interner.
//! It takes ownership of strings and issues corresponding `Name`s, which must
//! not outlive the `NameSpace` which issued them.
//!
//! [`NameSpace`]: ./struct.NameSpace.html
//! [`Name`]: ./struct.Name.html

use std::cell::RefCell;
use std::cmp::{Ordering, PartialOrd};
use std::collections::HashSet;
use std::fmt;
use std::marker::PhantomData;
use std::mem;
use std::ops::Deref;

/// Assigns `Name`s to strings.
///
/// Equivalent strings will be assigned the same `Name`.
///
/// A `NameSpace` is effectivly a string interner.
pub struct NameSpace {
    strings: RefCell<HashSet<Box<str>>>,
}

/// A lightweight representation of a string.
///
/// A `Name` is almost exactly like a `&'ns str` where `'ns` is the lifetime
/// of the `NameSpace` to which it belongs. The major difference is that names
/// are compared for equality only by the value of the pointer, not the
/// contents of the string. Thus `Name`s for the same string but from different
/// `NameSpace`s are not equal.
#[derive(Clone, Copy)]
#[derive(PartialEq, Eq)]
pub struct Name<'ns> {
    ptr: *const str,
    pha: PhantomData<&'ns str>,
}

// NameSpace
// --------------------------------------------------

impl NameSpace {
    /// Constructs a new `NameSpace`.
    pub fn new() -> NameSpace {
        NameSpace { strings: RefCell::new(HashSet::new()) }
    }

    /// Returns a `Name` for the token.
    pub fn name<'ns, S>(&'ns self, tok: S) -> Name<'ns>
    where
        S: Into<String> + AsRef<str>,
    {
        // If the token is already in the set,
        // fetch the old key and convert it into a Name
        {
            let strings = self.strings.borrow();
            if let Some(s) = strings.get(tok.as_ref()) {
                let s = unsafe { mem::transmute::<&str, &'ns str>(s) };
                return Name::from(s);
            }
        }

        // Otherwise, turn this token into a name and insert it into the set.
        let mut strings = self.strings.borrow_mut();
        let boxed = tok.into().into_boxed_str();
        let s = unsafe { mem::transmute::<&str, &'ns str>(boxed.as_ref()) };
        strings.insert(boxed);
        Name::from(s)
    }

    /// Returns the number of unique `Name`s issued.
    pub fn len(&self) -> usize {
        self.strings.borrow().len()
    }
}

// Name
// --------------------------------------------------

impl<'ns> Name<'ns> {
    pub fn as_str(&self) -> &'ns str {
        unsafe { mem::transmute(self.ptr) }
    }
}

impl<'ns> From<&'ns str> for Name<'ns> {
    fn from(string: &'ns str) -> Name {
        Name {
            ptr: string as *const str,
            pha: PhantomData,
        }
    }
}

impl<'ns> Into<&'ns str> for Name<'ns> {
    fn into(self) -> &'ns str {
        self.as_str()
    }
}

impl<'ns> AsRef<str> for Name<'ns> {
    fn as_ref(&self) -> &str {
        self.as_str()
    }
}

impl<'ns> Deref for Name<'ns> {
    type Target = str;
    fn deref(&self) -> &str {
        self.as_str()
    }
}

impl<'ns> PartialOrd for Name<'ns> {
    fn partial_cmp(&self, other: &Name<'ns>) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}

impl<'ns> Ord for Name<'ns> {
    fn cmp(&self, other: &Name<'ns>) -> Ordering {
        self.as_str().cmp(other.as_str())
    }
}

impl<'ns> fmt::Display for Name<'ns> {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl<'ns> fmt::Debug for Name<'ns> {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{:?}@{:?}", self.as_str(), self.ptr)
    }
}

// Names are both Send and Sync because they are immutable.
unsafe impl<'ns> Send for Name<'ns> {}
unsafe impl<'ns> Sync for Name<'ns> {}

// Tests
// --------------------------------------------------

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn basic() {
        let ns = NameSpace::new();
        let a = ns.name("foo");
        let b = ns.name("bar");
        assert_ne!(a, b);
        assert_eq!(ns.len(), 2);
    }

    #[test]
    fn dedupe() {
        let ns = NameSpace::new();
        let a = ns.name("foo");
        let b = ns.name("foo");
        assert_eq!(a, b);
        assert_eq!(ns.len(), 1);
    }

    #[test]
    fn order() {
        let ns = NameSpace::new();
        let a = ns.name("foo");
        let b = ns.name("bar");
        assert!(b < a);
    }

    #[test]
    fn eq() {
        let ns1 = NameSpace::new();
        let a = ns1.name("foo");
        let b = ns1.name("foo");
        let ns2 = NameSpace::new();
        let c = ns2.name("foo");
        assert_eq!(a, b);
        assert_ne!(b, c);
    }
}
