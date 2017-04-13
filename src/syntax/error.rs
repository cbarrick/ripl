use std::error::Error;
use std::fmt;

/// A type alias for results with possible `SyntaxError`s.
pub type Result<T> = ::std::result::Result<T, SyntaxError>;

/// The location and description of syntax errors.
#[derive(Debug)]
pub struct SyntaxError {
    line: usize,
    col: usize,
    kind: Kind,
}

#[derive(Debug)]
enum Kind {
    PrioirtyClash,
    Unbalanced(char),
    Unexpected(&'static str),
    Wrapper(Box<Error + Send + Sync>),

    // Emitted when using an incomplete feature.
    TODO,
}

impl SyntaxError {
    fn new(line: usize, col: usize, kind: Kind) -> SyntaxError {
        SyntaxError {
            line: line,
            col: col,
            kind: kind,
        }
    }

    pub fn wrap<E>(line: usize, col: usize, err: E) -> SyntaxError
    where
        E: Into<Box<Error + Send + Sync>>,
    {
        SyntaxError::new(line, col, Kind::Wrapper(err.into()))
    }

    pub fn priority_clash(line: usize, col: usize) -> SyntaxError {
        SyntaxError::new(line, col, Kind::PrioirtyClash)
    }

    pub fn unbalanced(line: usize, col: usize, ch: char) -> SyntaxError {
        SyntaxError::new(line, col, Kind::Unbalanced(ch))
    }

    pub fn unexpected(line: usize, col: usize, s: &'static str) -> SyntaxError {
        SyntaxError::new(line, col, Kind::Unexpected(s))
    }

    pub fn todo(line: usize, col: usize) -> SyntaxError {
        SyntaxError::new(line, col, Kind::TODO)
    }

    /// Returns the line at which the error occurs.
    pub fn line(&self) -> usize {
        self.line
    }

    /// Returns the column at which the error occurs.
    pub fn col(&self) -> usize {
        self.col
    }
}

impl Error for SyntaxError {
    fn description(&self) -> &str {
        match &self.kind {
            &Kind::PrioirtyClash => "operator priority clash",
            &Kind::Unbalanced(_) => "unbalanced quote or paren",
            &Kind::Unexpected(_) => "unexpected token",
            &Kind::TODO => "not yet implemented",
            &Kind::Wrapper(ref e) => e.description(),
        }
    }

    fn cause(&self) -> Option<&Error> {
        if let &Kind::Wrapper(ref e) = &self.kind {
            e.cause()
        } else {
            None
        }
    }
}

impl<'ctx> fmt::Display for SyntaxError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{}:{}: ", self.line, self.col)?;
        match &self.kind {
            &Kind::PrioirtyClash => write!(f, "operator priority clash"),
            &Kind::Unbalanced(ch) => write!(f, "unbalanced grouping character: '{}'", ch),
            &Kind::Unexpected(tok) => write!(f, "unexpected token: {}", tok),
            &Kind::TODO => write!(f, "not yet implemented"),
            &Kind::Wrapper(ref e) => write!(f, "{}", e),
        }
    }
}

impl PartialEq for SyntaxError {
    fn eq(&self, other: &SyntaxError) -> bool {
        self.line == other.line && self.col == other.col
    }
}
