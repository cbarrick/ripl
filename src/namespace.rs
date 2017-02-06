use std::cell::RefCell;
use std::mem;

pub struct NameSpace {
    strings: RefCell<Vec<Box<str>>>,
}

impl NameSpace {
    pub fn new() -> NameSpace {
        NameSpace { strings: RefCell::new(Vec::with_capacity(256)) }
    }

    pub fn intern(&self, tok: &str) -> usize {
        let mut strings = self.strings.borrow_mut();
        for (i, s) in strings.iter().enumerate() {
            if **s == *tok {
                return i;
            }
        }
        strings.push(tok.to_string().into_boxed_str());
        strings.len() - 1
    }

    pub fn lookup(&self, idx: usize) -> Option<&str> {
        let strings = self.strings.borrow();
        match strings.get(idx) {
            None => None,
            Some(boxed_str) => {
                // Saftey:
                // Extend the lifetime of the string to the lifetime of self.
                // This is safe because we never free our owned strings until we are freed.
                let s = unsafe { mem::transmute::<&str, &str>(&*boxed_str) };
                Some(s)
            }
        }
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn basic() {
        let ns = NameSpace::new();
        let a = ns.intern("foo");
        let b = ns.intern("bar");
        assert_ne!(a, b);
        assert_eq!(ns.strings.borrow().len(), 2);
    }

    #[test]
    fn dedupe() {
        let ns = NameSpace::new();
        let a = ns.intern("foo");
        let b = ns.intern("foo");
        assert_eq!(a, b);
        assert_eq!(ns.strings.borrow().len(), 1);
    }

    #[test]
    fn index() {
        let ns = NameSpace::new();
        let a = ns.intern("foo");
        let b = ns.intern("bar");
        assert_eq!(ns.lookup(a), Some("foo"));
        assert_eq!(ns.lookup(b), Some("bar"));
        assert_eq!(ns.lookup(1337), None);
    }
}
