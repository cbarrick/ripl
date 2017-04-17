use std::collections::HashMap;
use std::sync::Arc;

use syntax::{Structure, Symbol};

pub struct DataBase<'ns> {
    preds: HashMap<Symbol<'ns>, Vec<Rule<'ns>>>,
}

#[derive(Clone)]
pub struct Rule<'ns> {
    head: Arc<Structure<'ns>>,
    body: Option<Arc<Structure<'ns>>>,
}

impl<'ns> DataBase<'ns> {
    pub fn new() -> DataBase<'ns> {
        DataBase { preds: HashMap::new() }
    }

    pub fn assert(&mut self, head: Arc<Structure<'ns>>, body: Option<Arc<Structure<'ns>>>) {
        let functor = head.functor();
        let rules = self.preds.entry(functor).or_insert(vec![]);
        rules.push(Rule::new(head, body));
    }

    pub fn query(&self, head: Arc<Structure<'ns>>) -> Vec<Rule<'ns>> {
        let functor = head.functor();
        match self.preds.get(&functor) {
            Some(rules) => rules.clone(),
            None => vec![],
        }
    }
}


impl<'ns> Rule<'ns> {
    fn new(head: Arc<Structure<'ns>>, body: Option<Arc<Structure<'ns>>>) -> Rule<'ns> {
        Rule {
            head: head,
            body: body,
        }
    }
}
