use std::borrow::Borrow;
use std::collections::hash_map::RandomState;
use std::hash::{BuildHasher, Hash, Hasher};
use std::mem;
use std::ops::Index;
use std::ptr;
use std::sync::Arc;

use rand;

/// An optionally persistent map implemented as a Hash Array Mapped Trie.
#[derive(Clone)]
pub struct CloneMap<K, V, S = RandomState>
where
    K: Hash + Eq + Clone,
    V: Clone,
    S: BuildHasher,
{
    branch_power: u32,
    seed: u32,
    hash_builder: S,
    root: Arc<CNode<K, V>>,
}


#[derive(Clone)]
enum Branch<K, V>
where
    K: Hash + Eq + Clone,
    V: Clone,
{
    C(CNode<K, V>),
    M(CloneMap<K, V>),
    S(Store<K, V>),
}


#[derive(Clone)]
struct CNode<K, V>
where
    K: Hash + Eq + Clone,
    V: Clone,
{
    bitmap: u64,
    branches: Vec<Arc<Branch<K, V>>>,
}


#[derive(Clone)]
struct Store<K, V>
where
    K: Hash + Eq + Clone,
    V: Clone,
{
    hash: u64,
    key: K,
    val: V,
}


// Pubic API
// --------------------------------------------------

impl<K, V> CloneMap<K, V, RandomState>
where
    K: Hash + Eq + Clone,
    V: Clone,
{
    /// Creates an empty `CloneMap` with a default branching factor.
    ///
    /// # Examples
    ///
    /// ```
    /// use ripl::collections::CloneMap;
    /// let mut map: CloneMap<&str, isize> = CloneMap::new();
    /// ```
    pub fn new() -> CloneMap<K, V> {
        CloneMap::with_branch_factor(32)
    }


    /// Creates an empty `CloneMap` with some branching factor.
    ///
    /// The branching factor is rounded up to the next power of two, which must
    /// be less than 64.
    ///
    /// # Examples
    ///
    /// ```
    /// use ripl::collections::CloneMap;
    /// let mut map: CloneMap<&str, isize> = CloneMap::with_branch_factor(16);
    /// ```
    pub fn with_branch_factor(n: u32) -> CloneMap<K, V> {
        if n > 64 {
            panic!("branch factor cannot exceed 64")
        }
        let w = n.next_power_of_two().trailing_zeros();
        CloneMap {
            branch_power: w,
            seed: rand::random(),
            hash_builder: RandomState::new(),
            root: Arc::new(CNode::new()),
        }
    }
}


impl<K, V, S> CloneMap<K, V, S>
where
    K: Hash + Eq + Clone,
    V: Clone,
    S: BuildHasher,
{
    fn hash<Q: ?Sized>(&self, q: &Q) -> u64
    where
        Q: Hash + Eq,
    {
        let mut hasher = self.hash_builder.build_hasher();
        hasher.write_u32(self.seed);
        q.hash(&mut hasher);
        hasher.finish()
    }


    // TODO: compact-on-remove must be implemented before this.
    // TODO: update examples for other methods once this is enabled.
    // /// Returns true if the map contains no elements.
    // ///
    // /// # Examples
    // ///
    // /// ```
    // /// use ripl::collections::CloneMap;
    // ///
    // /// let mut a = CloneMap::new();
    // /// assert!(a.is_empty());
    // /// a.insert(1, "a");
    // /// assert!(!a.is_empty());
    // /// ```
    // pub fn is_empty(&self) -> bool {
    //     self.root.bitmap == 0
    // }


    /// Clears the map, removing all key-value pairs. Keeps the allocated memory
    /// for reuse.
    ///
    /// # Examples
    ///
    /// ```
    /// use ripl::collections::CloneMap;
    ///
    /// let mut a = CloneMap::new();
    /// a.insert(1, "a");
    /// a.clear();
    /// // assert!(a.is_empty());
    /// ```
    pub fn clear(&mut self) {
        self.root = Arc::new(CNode::new());
    }


    /// Returns a reference to the value corresponding to the key.
    ///
    /// The key may be any borrowed form of the map's key type, but
    /// [`Hash`] and [`Eq`] on the borrowed form *must* match those for
    /// the key type.
    ///
    /// [`Eq`]: doc.rust-lang.org/std/cmp/trait.Eq.html
    /// [`Hash`]: doc.rust-lang.org/std/hash/trait.Hash.html
    ///
    /// # Examples
    ///
    /// ```
    /// use ripl::collections::CloneMap;
    ///
    /// let mut map = CloneMap::new();
    /// map.insert(1, "a");
    /// assert_eq!(map.get(&1), Some(&"a"));
    /// assert_eq!(map.get(&2), None);
    /// ```
    pub fn get<Q: ?Sized>(&self, key: &Q) -> Option<&V>
    where
        K: Borrow<Q>,
        Q: Hash + Eq,
    {
        let hash = self.hash(key);
        self.root.get(hash, key, 0, self.branch_power)
    }


    /// Removes a key from the map, returning the value at the key if the key
    /// was previously in the map.
    ///
    /// The key may be any borrowed form of the map's key type, but
    /// [`Hash`] and [`Eq`] on the borrowed form *must* match those for
    /// the key type.
    ///
    /// [`Eq`]: doc.rust-lang.org/std/cmp/trait.Eq.html
    /// [`Hash`]: doc.rust-lang.org/std/hash/trait.Hash.html
    ///
    /// # Examples
    ///
    /// ```
    /// use ripl::collections::CloneMap;
    ///
    /// let mut map = CloneMap::new();
    /// map.insert(1, "a");
    /// assert_eq!(map.remove(&1), Some("a"));
    /// assert_eq!(map.remove(&1), None);
    /// ```
    pub fn remove<Q: ?Sized>(&mut self, key: &Q) -> Option<V>
    where
        K: Borrow<Q>,
        Q: Hash + Eq,
    {
        let hash = self.hash(key);
        let mut root = Arc::make_mut(&mut self.root);
        root.remove(hash, key, 0, self.branch_power)
    }


    /// Inserts a key-value pair into the map.
    ///
    /// If the map did not have this key present, `None` is returned.
    ///
    /// If the map did have this key present, the value is updated, and the old
    /// value is returned. The key is not updated, though.
    ///
    /// # Examples
    ///
    /// ```
    /// use ripl::collections::CloneMap;
    ///
    /// let mut map = CloneMap::new();
    /// assert_eq!(map.insert(37, "a"), None);
    /// // assert_eq!(map.is_empty(), false);
    ///
    /// map.insert(37, "b");
    /// assert_eq!(map.insert(37, "c"), Some("b"));
    /// assert_eq!(map[&37], "c");
    /// ```
    pub fn insert(&mut self, key: K, val: V) -> Option<V> {
        let hash = self.hash(&key);
        let mut root = Arc::make_mut(&mut self.root);
        root.insert(hash, key, val, 0, self.branch_power)
    }
}


impl<'a, K, Q, V, S> Index<&'a Q> for CloneMap<K, V, S>
where
    K: Hash + Eq + Clone + Borrow<Q>,
    Q: Hash + Eq,
    V: Clone,
    S: BuildHasher,
{
    type Output = V;
    fn index(&self, index: &Q) -> &V {
        self.get(index).expect("no entry found for key")
    }
}

// CNode
// --------------------------------------------------

impl<K, V> CNode<K, V>
where
    K: Hash + Eq + Clone,
    V: Clone,
{
    /// Constructs an empty `CNode`.
    fn new() -> CNode<K, V> {
        CNode {
            bitmap: 0,
            branches: vec![],
        }
    }


    /// Computes the index of the branch matching the hash.
    ///
    /// The return values are called `flag` and `pos` where `pos` is the index
    /// of the branch, and `flag` is a mask for the bitmap.
    ///
    /// If `self.bitmap & flag == 0`, then no such branch exists. If a branch
    /// is inserted at `pos`, then the `flag` must become set on the bitmap.
    /// Likewise, if a branch is removed at `pos`,  then the `flag` must become
    /// unset on the bitmap.
    ///
    /// As long as the bitmap is propperly maintained, the `pos` will be valid
    /// if the flag is set on the bitmap. In this situation, it is safe to call
    /// `self.branches.get_unchecked(pos)`.
    fn flagpos(&self, hash: u64, level: u32, w: u32) -> (u64, usize) {
        let index = (hash >> level) & ((1 << w) - 1);
        let flag = 1 << index;
        let pos = ((flag - 1) & self.bitmap).count_ones() as usize;
        (flag, pos)
    }


    /// Searches the tree for the `key`.
    ///
    /// The argument `level` gives the current depth of the search and
    /// increases by `w` with each recursion. `w` is the branching power of the
    /// tree (the log of the branching factor).
    fn get<Q: ?Sized>(&self, hash: u64, key: &Q, level: u32, w: u32) -> Option<&V>
    where
        K: Borrow<Q>,
        Q: Hash + Eq,
    {
        let (flag, pos) = self.flagpos(hash, level, w);

        // Simple case: return None if there is no matching branch.
        if self.bitmap & flag == 0 {
            return None;
        }

        // Otherwise the data may live down an existing branch.
        // SAFTEY: pos is safe because we've checked the flag against bitmap.
        let branch = unsafe { self.branches.get_unchecked(pos) };

        match **branch {
            // Recurse on M and C branches.
            Branch::M(ref m) => m.get(key),
            Branch::C(ref c) => c.get(hash, key, level + w, w),

            // S branches are leaves and may constain the key.
            Branch::S(ref s) => {
                if s.key.borrow() == key {
                    Some(&s.val)
                } else {
                    None
                }
            },
        }
    }


    /// Removes a key-value pair from the map.
    ///
    /// This operation will trigger path-copying if we are not the exclusive
    /// owner of the next node in the search.
    ///
    /// The argument `level` gives the current depth of the search and
    /// increases by `w` with each recursion. `w` is the branching power of the
    /// tree (the log of the branching factor).
    fn remove<Q: ?Sized>(&mut self, hash: u64, key: &Q, level: u32, w: u32) -> Option<V>
    where
        K: Borrow<Q>,
        Q: Hash + Eq,
    {
        let (flag, pos) = self.flagpos(hash, level, w);

        // Simple case: return None if there is no matching branch.
        if self.bitmap & flag == 0 {
            return None;
        }

        {
            // We may need to mutate the existing branch.
            // This will clone the branch if we are not the exclusive owner.
            // SAFTEY: pos is safe because we've checked the flag against bitmap.
            let mut branch = unsafe { self.branches.get_unchecked_mut(pos) };
            let mut branch = Arc::make_mut(branch);

            match *branch {
                // Recurse on M and C branches.
                Branch::M(ref mut m) => return m.remove(key),
                Branch::C(ref mut c) => return c.remove(hash, key, level + w, w),

                // For S branches:
                // - If the key doesn't match, we return None.
                // - If the keys match, we break this scope to delete the branch.
                Branch::S(ref mut s) => {
                    if s.key.borrow() != key {
                        return None;
                    }
                },
            }
        }

        // Remove the S branch
        // TODO: compact the tree if we only have one child.
        let branch = self.branches.remove(pos);
        let branch = Arc::try_unwrap(branch);
        match branch {
            Ok(Branch::S(s)) => {
                self.bitmap ^= flag;
                return Some(s.val);
            },

            // UNREACHABLE: The call to `Arc::make_mut` ensures we are the
            // exclusive owner of the branch at `pos`. Thus the call to
            // `Arc::try_unwrap` will succeed. We also know that the branch
            // is an S branch because all other cases are handle by the
            // match statement. Thus the `if let` above will always trigger.
            _ => unreachable!(),
        }
    }


    /// Inserts a key-value pair into the map.
    ///
    /// Returns the previous value associated with the key.
    ///
    /// This operation will trigger path-copying if we are not the exclusive
    /// owner of the next node in the search.
    ///
    /// The argument `level` gives the current depth of the search and
    /// increases by `w` with each recursion. `w` is the branching power of the
    /// tree (the log of the branching factor).
    fn insert(&mut self, hash: u64, key: K, val: V, level: u32, w: u32) -> Option<V> {
        let (flag, pos) = self.flagpos(hash, level, w);

        // Simple case: insert if we have a vacancy.
        if self.bitmap & flag == 0 {
            self.branches
                .insert(pos, Arc::new(Branch::S(Store::new(hash, key, val))));
            self.bitmap |= flag;
            return None;
        }

        // Otherwise we need to mutate an existing branch.
        // This will clone the branch if we are not the exclusive owner.
        // SAFTEY: pos is safe because we've checked the flag against bitmap.
        let mut branch = unsafe { self.branches.get_unchecked_mut(pos) };
        let mut branch = Arc::make_mut(branch);
        let branch_ptr = branch as *mut Branch<K, V>;

        match *branch {
            // Recurse on M and C branches.
            Branch::M(ref mut m) => return m.insert(key, val),
            Branch::C(ref mut c) => return c.insert(hash, key, val, level + w, w),

            // For S branches:
            // - If the key is a match, replace the value.
            // - In the case of a hash collision, split into an M branch.
            // - In the case of a partial collision, split into a C branch.
            Branch::S(ref mut s) => {
                if key == s.key {
                    let old_val = mem::replace(&mut s.val, val);
                    return Some(old_val);
                }

                // To split the branch, we must take ownership of s.
                // SAFTEY: the branch MUST be replaced before returning.
                let s = unsafe { mem::replace(s, mem::uninitialized()) };
                let new_branch: Branch<K, V>;

                if hash == s.hash {
                    let mut m = CloneMap::with_branch_factor(1 << w);
                    m.insert(s.key, s.val);
                    m.insert(key, val);
                    new_branch = Branch::M(m);
                } else {
                    let mut c = CNode::new();
                    c.insert(s.hash, s.key, s.val, level + w, w);
                    c.insert(hash, key, val, level + w, w);
                    new_branch = Branch::C(c);
                }

                // SAFTEY: ensure that the branch is replaced.
                unsafe { ptr::replace(branch_ptr, new_branch) };
                None
            },
        }
    }
}


// Store
// --------------------------------------------------

impl<K, V> Store<K, V>
where
    K: Hash + Eq + Clone,
    V: Clone,
{
    fn new(hash: u64, key: K, val: V) -> Store<K, V> {
        Store {
            hash: hash,
            key: key,
            val: val,
        }
    }
}

// Unit Tests
// --------------------------------------------------

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn basic() {
        let mut m = CloneMap::new();

        // Large enough to cause collisions.
        let item_count: usize = 1 << 16;

        // Map every i to i+1.
        // Returns None because the key shouldn't yet exist.
        for i in 0..item_count {
            let val = m.insert(i, i + 1);
            assert_eq!(val, None);
        }

        // Remap even i to i*2
        // Should return the old values.
        for i in 0..item_count {
            if i % 2 == 0 {
                let val = m.insert(i, i * 2);
                assert_eq!(val, Some(i + 1));
            }
        }

        // Remove every third value.
        for i in 0..item_count {
            if i % 3 == 0 {
                let val = m.remove(&i);
                if i % 2 == 0 {
                    assert_eq!(val, Some(i * 2))
                } else {
                    assert_eq!(val, Some(i + 1))
                }
            }
        }

        // Check all.
        for i in 0..item_count {
            let val = m.get(&i);
            if i % 3 == 0 {
                assert_eq!(val, None);
            } else if i % 2 == 0 {
                assert_eq!(val, Some(&(i * 2)));
            } else {
                assert_eq!(val, Some(&(i + 1)));
            }
        }
    }

    #[test]
    fn clear() {
        let mut m = CloneMap::new();
        let item_count: usize = 1 << 16;

        for i in 0..item_count {
            m.insert(i, i + 1);
        }

        for i in 0..item_count {
            assert_eq!(m.get(&i), Some(&(i + 1)));
        }

        m.clear();

        for i in 0..item_count {
            assert_eq!(m.get(&i), None);
        }
    }
}
