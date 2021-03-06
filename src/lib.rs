#![feature(custom_attribute)]
#![feature(box_syntax, box_patterns)]

#[macro_use]
extern crate lazy_static;

extern crate ordered_float;
extern crate rand;
extern crate regex;
extern crate unicode_normalization;

pub mod collections;
pub mod db;
pub mod syntax;
