use std::cmp::{PartialOrd, Ordering};
use std::ops::Deref;

use syntax::namespace::{NameSpace, Name};

/// A specification for parsing operators.
///
/// An `Op` tells the parser how to handle operators and is comprised of three
/// components:
///
/// - The type of the operator, given by the discriminant of the enum,
///   specifies whether the operator is prefix, infix, or postfix. The `F`
///   indicates the position of the functor while `X` and `Y` indicate the
///   position of the arguments. `Y` means that the argument must be of equal
///   or lower precedence, and `X` means that the argument must be of strictly
///   lower precedence. A `Y` on the right means that the operator is right
///   associative, and likewise on the left means left associative. No `Y`
///   means non-associative.
/// - The precedence of the operator is given as a u32. A lower value equates
///   to a narrower scope. Thus multiplicative operators have *lower*
///   precedence than additive operators.
/// - The symbol to use for the operator is given by a `Name` which must be
///   assigned by the same `NameSpace` being used by the parser.
#[derive(Debug)]
#[derive(Clone, Copy)]
#[derive(PartialEq, Eq)]
pub enum Op<'ns> {
    XF(u32, Name<'ns>),
    YF(u32, Name<'ns>),
    XFX(u32, Name<'ns>),
    XFY(u32, Name<'ns>),
    YFX(u32, Name<'ns>),
    FY(u32, Name<'ns>),
    FX(u32, Name<'ns>),
}

/// The general categories of operators.
///
/// -`FX` and `FY` operators are `Prefix`.
/// -`XFX`, `XFY`, and `YFX`, operators are `Infix`.
/// -`XF` and `YF` operators are `Postfix`.
#[derive(Clone, Copy)]
#[derive(PartialEq, Eq)]
#[derive(PartialOrd, Ord)]
pub enum OpType {
    Prefix,
    Infix,
    Postfix,
}

/// A table of operators to be used by a `Parser`.
///
/// The table is implemented as a sorted list of `Op`s. Operators are sorted
/// first by name, then by type, and finally by precedence.
#[derive(Debug)]
pub struct OpTable<'ns>(Vec<Op<'ns>>);

// OpTable
// --------------------------------------------------

impl<'ns> OpTable<'ns> {
    /// Construct a new, empty operator table.
    pub fn new() -> OpTable<'ns> {
        OpTable(Vec::new())
    }

    /// View the table as a sorted slice of `Op`s.
    pub fn as_slice(&self) -> &[Op<'ns>] {
        &self.0
    }

    /// Insert a new operator into the table.
    ///
    /// TODO: remove any conflicting operators.
    pub fn insert(&mut self, op: Op<'ns>) {
        match self.binary_search(&op) {
            Ok(i) => self.0[i] = op,
            Err(i) => self.0.insert(i, op),
        }
    }

    /// Get a slice of all operators matching the given name.
    ///
    /// The resulting slice is in sorted order.
    pub fn get(&self, name: Name<'ns>) -> &[Op<'ns>] {
        let target = Op::FX(0, name);
        let i = match self.binary_search(&target) {
            Ok(i) => i,
            Err(i) => i,
        };
        let mut j = i;
        let n = self.len();
        while j < n && self[j].name() == name {
            j += 1;
        }
        &self[i..j]
    }

    /// Get the first prefix operator matching this name and equal or lower precedence.
    pub fn get_prefix(&self, name: Name<'ns>, prec: u32) -> Option<Op<'ns>> {
        self.get(name)
            .iter()
            .cloned()
            .find(|op| op.op_type() == OpType::Prefix && op.prec() <= prec)
    }

    /// Get the first infix operator matching this name and equal or lower precedence.
    pub fn get_infix(&self, name: Name<'ns>, prec: u32) -> Option<Op<'ns>> {
        self.get(name)
            .iter()
            .cloned()
            .find(|op| op.op_type() == OpType::Infix && op.prec() <= prec)
    }

    /// Get the first postfix operator matching this name and equal or lower precedence.
    pub fn get_postfix(&self, name: Name<'ns>, prec: u32) -> Option<Op<'ns>> {
        self.get(name)
            .iter()
            .cloned()
            .find(|op| op.op_type() == OpType::Postfix && op.prec() <= prec)
    }

    /// Get the first operator of the given name which is compatible with a
    /// left-hand argument of the given precedence and a given max precedence.
    ///
    /// Prefix operators are *never* compatible with a left-hand argument. For
    /// the other types, the associativity determines if the lhs precedence
    /// should be simply less than or strictly less than the precedence of the
    /// operator.
    pub fn get_compatible(&self, name: Name<'ns>, lhs_prec: u32, max_prec: u32) -> Option<Op<'ns>> {
        for op in self.get(name).iter().cloned() {
            let prec = op.prec();
            let y = lhs_prec <= prec && prec <= max_prec;
            let x = lhs_prec < prec && prec <= max_prec;
            match op {
                Op::YFX(..) | Op::YF(..) if y => return Some(op),
                Op::XFX(..) | Op::XFY(..) | Op::XF(..) if x => return Some(op),
                _ => (),
            }
        }
        return None;
    }
}

impl<'ns> From<Vec<Op<'ns>>> for OpTable<'ns> {
    fn from(mut vec: Vec<Op<'ns>>) -> OpTable<'ns> {
        vec.sort();
        let mut i = 0;
        while i < vec.len() - 1 {
            if vec[i].op_type() == vec[i + 1].op_type() && vec[i].name() == vec[i + 1].name() {
                vec.remove(i);
            } else {
                i += 1;
            }
        }
        OpTable(vec)
    }
}

impl<'a, 'ns> From<&'a [Op<'ns>]> for OpTable<'ns> {
    fn from(slice: &[Op<'ns>]) -> OpTable<'ns> {
        OpTable::from(slice.to_vec())
    }
}

impl<'ns> Deref for OpTable<'ns> {
    type Target = [Op<'ns>];
    fn deref(&self) -> &[Op<'ns>] {
        self.as_slice()
    }
}

impl<'ns> AsRef<[Op<'ns>]> for OpTable<'ns> {
    fn as_ref(&self) -> &[Op<'ns>] {
        self.as_slice()
    }
}

// Op
// --------------------------------------------------

impl<'ns> Op<'ns> {
    #[inline]
    pub fn op_type(&self) -> OpType {
        match *self {
            Op::FX(_, _) | Op::FY(_, _) => OpType::Prefix,
            Op::XFX(_, _) | Op::XFY(_, _) | Op::YFX(_, _) => OpType::Infix,
            Op::XF(_, _) | Op::YF(_, _) => OpType::Postfix,
        }
    }

    #[inline]
    pub fn name(&self) -> Name<'ns> {
        match *self {
            Op::XF(_, name) => name,
            Op::YF(_, name) => name,
            Op::XFX(_, name) => name,
            Op::XFY(_, name) => name,
            Op::YFX(_, name) => name,
            Op::FY(_, name) => name,
            Op::FX(_, name) => name,
        }
    }

    #[inline]
    pub fn prec(&self) -> u32 {
        match *self {
            Op::XF(prec, _) => prec,
            Op::YF(prec, _) => prec,
            Op::XFX(prec, _) => prec,
            Op::XFY(prec, _) => prec,
            Op::YFX(prec, _) => prec,
            Op::FY(prec, _) => prec,
            Op::FX(prec, _) => prec,
        }
    }
}

impl<'ns> PartialOrd for Op<'ns> {
    fn partial_cmp(&self, other: &Op<'ns>) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}

impl<'ns> Ord for Op<'ns> {
    fn cmp(&self, other: &Op<'ns>) -> Ordering {
        if self.name() != other.name() {
            self.name().cmp(&other.name())
        } else if self.op_type() != other.op_type() {
            self.op_type().cmp(&other.op_type())
        } else {
            self.prec().cmp(&other.prec())
        }
    }
}

// Default Operators
// --------------------------------------------------

#[cfg_attr(rustfmt, rustfmt_skip)]
impl<'ns> OpTable<'ns> {
    /// Returns the default set of Prolog operators, named in the given
    /// namespace.
    ///
    /// Because this module aims to be general for all logic programming
    /// languages, this function is likely to move somewhere more Prolog
    /// specific.
    pub fn default(ns: &'ns NameSpace) -> OpTable<'ns> {
        // TODO: This can be sorted by hand.
        OpTable::from(vec![
            Op::XFX(1200, ns.name("-->")),
            Op::XFX(1200, ns.name(":-")),
            Op::FX(1200, ns.name(":-")),
            Op::FX(1200, ns.name("?-")),
            Op::FX(1150, ns.name("dynamic")),
            Op::FX(1150, ns.name("discontiguous")),
            Op::FX(1150, ns.name("initialization")),
            Op::FX(1150, ns.name("meta_predicate")),
            Op::FX(1150, ns.name("module_transparent")),
            Op::FX(1150, ns.name("multifile")),
            Op::FX(1150, ns.name("public")),
            Op::FX(1150, ns.name("thread_local")),
            Op::FX(1150, ns.name("thread_initialization")),
            Op::FX(1150, ns.name("volatile")),
            Op::XFY(1100, ns.name(";")),
            Op::XFY(1100, ns.name("|")),
            Op::XFY(1050, ns.name("->")),
            Op::XFY(1050, ns.name("*->")),
            Op::XFY(1000, ns.name(",")),
            Op::XFX(990, ns.name(":=")),
            Op::FY(900, ns.name("\\+")),
            Op::XFX(700, ns.name("<")),
            Op::XFX(700, ns.name("=")),
            Op::XFX(700, ns.name("=..")),
            Op::XFX(700, ns.name("=@=")),
            Op::XFX(700, ns.name("\\=@=")),
            Op::XFX(700, ns.name("=:=")),
            Op::XFX(700, ns.name("=<")),
            Op::XFX(700, ns.name("==")),
            Op::XFX(700, ns.name("=\\=")),
            Op::XFX(700, ns.name(">")),
            Op::XFX(700, ns.name(">=")),
            Op::XFX(700, ns.name("@<")),
            Op::XFX(700, ns.name("@=<")),
            Op::XFX(700, ns.name("@>")),
            Op::XFX(700, ns.name("@>=")),
            Op::XFX(700, ns.name("\\=")),
            Op::XFX(700, ns.name("\\==")),
            Op::XFX(700, ns.name("as")),
            Op::XFX(700, ns.name("is")),
            Op::XFX(700, ns.name(">:<")),
            Op::XFX(700, ns.name(":<")),
            Op::XFY(600, ns.name(":")),
            Op::YFX(500, ns.name("+")),
            Op::YFX(500, ns.name("-")),
            Op::YFX(500, ns.name("/\\")),
            Op::YFX(500, ns.name("\\/")),
            Op::YFX(500, ns.name("xor")),
            Op::FX(500, ns.name("?")),
            Op::YFX(400, ns.name("*")),
            Op::YFX(400, ns.name("/")),
            Op::YFX(400, ns.name("//")),
            Op::YFX(400, ns.name("div")),
            Op::YFX(400, ns.name("rdiv")),
            Op::YFX(400, ns.name("<<")),
            Op::YFX(400, ns.name(">>")),
            Op::YFX(400, ns.name("mod")),
            Op::YFX(400, ns.name("rem")),
            Op::XFX(200, ns.name("**")),
            Op::XFY(200, ns.name("^")),
            Op::FY(200, ns.name("+")),
            Op::FY(200, ns.name("-")),
            Op::FY(200, ns.name("\\")),
            Op::YFX(100, ns.name(".")),
            Op::FX(1, ns.name("$")),
        ])
    }
}

// Tests
// --------------------------------------------------

#[cfg(test)]
mod test {
    use syntax::namespace::NameSpace;
    use super::*;

    #[test]
    #[cfg_attr(rustfmt, rustfmt_skip)]
    fn get() {
        let ns = NameSpace::new();
        let foo = ns.name("foo");
        let bar = ns.name("bar");
        let zap = ns.name("zap");
        let ops = OpTable::from(&[
            Op::FX(0, foo),
            Op::XFX(1, foo),
            Op::FX(2, bar),
            Op::FX(3, zap),
        ][..]);
        assert_eq!(ops.get_prefix(foo, 0), Some(Op::FX(0, foo)));
        assert_eq!(ops.get_prefix(foo, 1), Some(Op::FX(0, foo)));
        assert_eq!(ops.get_infix(foo, 0), None);
        assert_eq!(ops.get_infix(foo, 1), Some(Op::XFX(1, foo)));
        assert_eq!(ops.get_postfix(foo, 0), None);
    }

    #[test]
    #[cfg_attr(rustfmt, rustfmt_skip)]
    fn insert() {
        let ns = NameSpace::new();
        let foo = ns.name("foo");
        let bar = ns.name("bar");
        let zap = ns.name("zap");
        let mut ops = OpTable::new();
        ops.insert(Op::FX(0, foo));
        ops.insert(Op::XFX(1, foo));
        ops.insert(Op::FX(2, bar));
        ops.insert(Op::FX(3, zap));
        assert_eq!(ops.as_slice(), &[
            Op::FX(2, bar),
            Op::FX(0, foo),
            Op::XFX(1, foo),
            Op::FX(3, zap),
        ]);
    }
}
