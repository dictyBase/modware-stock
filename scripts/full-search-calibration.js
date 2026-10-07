// Calibration harness for the stock full search.
//
// Run against a disposable database on a real ArangoDB 3.11 server:
//
//   docker exec -i arango311 arangosh \
//     --server.endpoint tcp://localhost:8529 \
//     --server.username root --server.password rootpass \
//     --javascript.execute /dev/stdin < scripts/full-search-calibration.js
//
// The script prints a JSON report with the 26 probe results, the
// EXPLAIN node list and the latency baseline. It drops its database
// when it finishes.

'use strict';

// ---------------------------------------------------------------- setup

const DBNAME = 'stock_calib_' + Date.now();
db._createDatabase(DBNAME);
db._useDatabase(DBNAME);
print('database: ' + DBNAME);

const stock = db._create('stock_calib');
const props = db._create('stock_properties_calib');
const edges = db._createEdgeCollection('stock_type_calib');
require('@arangodb/general-graph')._create('stock_prop_calib', [
  {
    collection: 'stock_type_calib',
    from: ['stock_calib'],
    to: ['stock_properties_calib']
  }
]);

const NORM = 'stock_search_norm';
const NGRAM = 'stock_search_ngram';
const TEXT = 'text_en';

function createAnalyzer(def) {
  const res = db._connection.POST('/_db/' + DBNAME + '/_api/analyzer', JSON.stringify(def));
  if (res.statusCode >= 400) {
    throw new Error('analyzer ' + def.name + ' failed: ' + JSON.stringify(res.body));
  }
}

createAnalyzer({
  name: NORM,
  type: 'norm',
  properties: { locale: 'en.utf-8', case: 'lower', accent: false },
  features: []
});
createAnalyzer({
  name: NGRAM,
  type: 'pipeline',
  properties: {
    pipeline: [
      { type: 'norm', properties: { locale: 'en.utf-8', case: 'lower', accent: false } },
      {
        type: 'ngram',
        properties: { min: 2, max: 3, preserveOriginal: true, streamType: 'utf8' }
      }
    ]
  },
  features: ['frequency', 'norm', 'position']
});

const both = [NORM, NGRAM];
// editable_summary carries the featureless norm analyzer only, so that
// probe 11 can measure PHRASE against an analyzer that lacks the
// frequency and position features.
db._createView('stock_full_search', 'arangosearch', {
  links: {
    stock_calib: {
      fields: {
        stock_id: { analyzers: both },
        genes: { analyzers: both },
        dbxrefs: { analyzers: both },
        summary: { analyzers: [TEXT] },
        depositor: { analyzers: [TEXT] },
        editable_summary: { analyzers: [NORM] }
      }
    },
    stock_properties_calib: {
      fields: {
        label: { analyzers: both },
        names: { analyzers: both },
        species: { analyzers: both },
        plasmid: { analyzers: both },
        name: { analyzers: both }
      }
    }
  }
});

// ------------------------------------------------------------- fixtures

// One stock document with _key == stock_id, one property document with
// an auto-generated _key, one edge that carries the entity type.
function addStock(stockId, kind, propDoc, stockDoc) {
  const doc = stockDoc || {};
  doc._key = stockId;
  doc.stock_id = stockId;
  stock.insert(doc);
  const p = props.insert(propDoc);
  edges.insert({
    _from: 'stock_calib/' + stockId,
    _to: 'stock_properties_calib/' + p._key,
    type: kind
  });
}

// The primary strain: identifier, label, names, species, plasmid,
// depositor, a 2-word phrase summary, and a unique editable_summary
// term that the feature must not search.
addStock('DBS0236126', 'strain',
  { label: 'yS13', names: ['Ax2', 'gammaS13'], species: 'Dictyostelium discoideum', plasmid: 'pDM304' },
  {
    genes: ['sadA', 'DDB_G0348394'],
    dbxrefs: ['d0319'],
    depositor: 'george@costanza.com',
    summary: 'the mutant forms culminants under starvation',
    editable_summary: 'editableuniqueterm'
  });
// A strain whose summary holds the two words far apart and in the
// other order: the phrase branch must skip it, the token branch must
// catch it.
addStock('DBS0444001', 'strain', { label: 'farapart' },
  {
    genes: ['sadA'],
    summary: 'culminants zebra quartz ladder anchor tulip forms'
  });
// A strain whose summary holds only one of the two words: the single
// token tail.
addStock('DBS0444002', 'strain', { label: 'onlyforms' },
  { genes: ['sadA'], summary: 'forms' });
// The plasmid without a summary attribute.
addStock('DBP0000027', 'plasmid', { name: 'pDM304' },
  { genes: ['sadA'] });
// A strain whose property document has no names attribute.
addStock('DBS0444003', 'strain', { label: 'axlessnames' },
  { genes: ['axlessnamesgene'], summary: 'namesless marker strain' });
// A strain that matches label, genes and summary for one query, so the
// merge attribution probe can rank the three stages.
addStock('DBS0167777', 'strain',
  { label: 'cordax', names: ['cordax-alias'], species: 'Dictyostelium cordax' },
  {
    genes: ['cordaxin'],
    summary: 'the cordax strain of the cordax line'
  });

// A property document with no inbound edge, to probe the missing-owner
// filter.
props.insert({ label: 'orphanonly' });

// 60 strains and 5 plasmids that all carry a gene with the prefix
// xylose.
let i;
for (i = 0; i < 60; i++) {
  addStock('DBS05' + String(100000 + i), 'strain', { label: 'xyl-' + i },
    { genes: ['xylose_G' + i] });
}
for (i = 0; i < 5; i++) {
  addStock('DBP05' + String(100000 + i), 'plasmid', { name: 'xylp-' + i },
    { genes: ['xylose_P' + i] });
}

// About 1000 further stocks with random identifiers and random prose
// summaries, so the latency measurement is not taken on an empty index.
const CHARS = 'abcdefghijklmnopqrstuvwxyz';
function randWord() {
  let s = '';
  for (let j = 0; j < 6; j++) {
    s += CHARS[Math.floor(Math.random() * CHARS.length)];
  }
  return s;
}
function randId(prefix) {
  let s = '';
  for (let j = 0; j < 7; j++) {
    s += CHARS[Math.floor(Math.random() * CHARS.length)];
  }
  return prefix + s;
}
const bulkStock = [];
const bulkProps = [];
const bulkEdges = [];
for (i = 0; i < 1000; i++) {
  const sid = randId(i % 2 === 0 ? 'DBS0' : 'DBP0');
  const words = [randWord(), randWord(), randWord(), randWord()];
  // Seed a repeated word into a fraction of the bulk summaries, so the
  // BM25 range probe of probe 15 has a token that appears in many
  // documents.
  if (i % 30 === 0) { words.push('calibword'); }
  bulkStock.push({
    _key: sid,
    stock_id: sid,
    genes: [randWord()],
    dbxrefs: [],
    summary: words.join(' ')
  });
  bulkProps.push({ label: randWord() });
}
stock.insert(bulkStock);
const insertedProps = props.insert(bulkProps, { returnNew: true }).map(function (r) { return r.new; });
for (i = 0; i < 1000; i++) {
  bulkEdges.push({
    _from: 'stock_calib/' + bulkStock[i]._key,
    _to: 'stock_properties_calib/' + insertedProps[i]._key,
    type: i % 2 === 0 ? 'strain' : 'plasmid'
  });
}
edges.insert(bulkEdges);

// -------------------------------------------------------- AQL builders

const VIEW = 'stock_full_search';

function prefixExpr(field) {
  return 'ANALYZER(STARTS_WITH(d.' + field + ', @q), "' + NORM + '")';
}
function fuzzyExpr(field) {
  return 'NGRAM_MATCH(d.' + field + ', @q, @th, "' + NGRAM + '")';
}
function tokenExpr(field) {
  return 'ANALYZER(d.' + field + ' IN TOKENS(@q, "' + TEXT + '"), "' + TEXT + '")';
}
function phraseExpr(field) {
  return 'PHRASE(d.' + field + ', @q, "' + TEXT + '")';
}
function scalarDisplay(field) {
  return 'NOT_NULL(d.' + field + ', "")';
}
function arrayDisplay(field) {
  return 'NOT_NULL(FIRST(FOR item IN NOT_NULL(d.' + field + ', []) ' +
    'FILTER CONTAINS(LOWER(item), @q) RETURN item), ' +
    'CONCAT_SEPARATOR(", ", NOT_NULL(d.' + field + ', [])))';
}

// A stock-collection branch. The row key is the stock key; one
// OUTBOUND traversal supplies the entity and the strain label.
function stockBranch(vname, field, expr, display, score) {
  return 'LET ' + vname + ' = (\n' +
    '  FOR d IN ' + VIEW + '\n' +
    '    SEARCH ' + expr + '\n' +
    '    LET meta = FIRST(\n' +
    '      FOR prop, edge IN 1..1 OUTBOUND d GRAPH @stock_prop_graph\n' +
    '        RETURN {\n' +
    '          entity: edge.type,\n' +
    '          strain_label: edge.type == "strain"\n' +
    '            ? NOT_NULL(prop.label, "")\n' +
    '            : ""\n' +
    '        }\n' +
    '    )\n' +
    '    FILTER meta != null\n' +
    '    FILTER @entity == "" OR @entity == meta.entity\n' +
    '    SORT BM25(d) DESC, d._key ASC\n' +
    '    LIMIT @limit\n' +
    '    RETURN { k: d._key, id: d.stock_id, entity: meta.entity, sl: meta.strain_label, f: ' + JSON.stringify(field) + ', v: ' + display + ', s: ' + score + ' }\n' +
    ')';
}
// A property-collection branch. The property _key is auto-generated,
// so the row resolves its owner with one INBOUND traversal.
function propBranch(vname, field, expr, display, score) {
  return 'LET ' + vname + ' = (\n' +
    '  FOR d IN ' + VIEW + '\n' +
    '    SEARCH ' + expr + '\n' +
    '    LET own = FIRST(\n' +
    '      FOR v, e IN 1..1 INBOUND d GRAPH @stock_prop_graph\n' +
    '        RETURN {\n' +
    '          k: v._key,\n' +
    '          id: v.stock_id,\n' +
    '          entity: e.type,\n' +
    '          strain_label: e.type == "strain"\n' +
    '            ? NOT_NULL(d.label, "")\n' +
    '            : ""\n' +
    '        }\n' +
    '    )\n' +
    '    FILTER own != null\n' +
    '    FILTER @entity == "" OR @entity == own.entity\n' +
    '    SORT BM25(d) DESC, own.k ASC\n' +
    '    LIMIT @limit\n' +
    '    RETURN { k: own.k, id: own.id, entity: own.entity, sl: own.strain_label, f: ' + JSON.stringify(field) + ', v: ' + display + ', s: ' + score + ' }\n' +
    ')';
}

// The 19 branches of the full search, in the order of the matrix.
const BRANCHES = [
  { v: 'p0', field: 'stock_id', coll: 'stock', stage: 'prefix', kind: 'scalar' },
  { v: 'n0', field: 'stock_id', coll: 'stock', stage: 'fuzzy', kind: 'scalar' },
  { v: 'p1', field: 'genes', coll: 'stock', stage: 'prefix', kind: 'array' },
  { v: 'n1', field: 'genes', coll: 'stock', stage: 'fuzzy', kind: 'array' },
  { v: 'p2', field: 'dbxrefs', coll: 'stock', stage: 'prefix', kind: 'array' },
  { v: 'n2', field: 'dbxrefs', coll: 'stock', stage: 'fuzzy', kind: 'array' },
  { v: 'p3', field: 'label', coll: 'prop', stage: 'prefix', kind: 'scalar' },
  { v: 'n3', field: 'label', coll: 'prop', stage: 'fuzzy', kind: 'scalar' },
  { v: 'p4', field: 'names', coll: 'prop', stage: 'prefix', kind: 'array' },
  { v: 'n4', field: 'names', coll: 'prop', stage: 'fuzzy', kind: 'array' },
  { v: 'p5', field: 'species', coll: 'prop', stage: 'prefix', kind: 'scalar' },
  { v: 'n5', field: 'species', coll: 'prop', stage: 'fuzzy', kind: 'scalar' },
  { v: 'p6', field: 'plasmid', coll: 'prop', stage: 'prefix', kind: 'scalar' },
  { v: 'n6', field: 'plasmid', coll: 'prop', stage: 'fuzzy', kind: 'scalar' },
  { v: 'p7', field: 'name', coll: 'prop', stage: 'prefix', kind: 'scalar' },
  { v: 'n7', field: 'name', coll: 'prop', stage: 'fuzzy', kind: 'scalar' },
  { v: 't0', field: 'summary', coll: 'stock', stage: 'token', kind: 'scalar' },
  { v: 't1', field: 'depositor', coll: 'stock', stage: 'token', kind: 'scalar' },
  { v: 'h0', field: 'summary', coll: 'stock', stage: 'phrase', kind: 'scalar' }
];

const SCORES = {
  prefix: '1000 + BM25(d)',
  phrase: '500 + BM25(d)',
  token: '250 + BM25(d)',
  fuzzy: 'BM25(d)'
};
const EXPRS = {
  prefix: prefixExpr,
  fuzzy: fuzzyExpr,
  token: tokenExpr,
  phrase: phraseExpr
};

// The full 19-branch statement with the merge and the summary tail.
function fullStatement() {
  const branches = [];
  const names = [];
  BRANCHES.forEach(function (b) {
    const display = b.kind === 'scalar' ? scalarDisplay(b.field) : arrayDisplay(b.field);
    const tpl = b.coll === 'stock' ? stockBranch : propBranch;
    branches.push(tpl(b.v, b.field, EXPRS[b.stage](b.field), display, SCORES[b.stage]));
    names.push(b.v);
  });
  const merge =
    'LET hits = FLATTEN([' + names.join(', ') + '])\n' +
    'LET best = (\n' +
    '  FOR x IN hits\n' +
    '    COLLECT stkey = x.k INTO grp = x\n' +
    '    LET top = FIRST(FOR m IN grp SORT m.s DESC, m.f ASC RETURN m)\n' +
    '    RETURN top\n' +
    ')\n' +
    'FOR x IN best\n' +
    '  SORT x.s DESC, x.k ASC\n' +
    '  LIMIT @limit\n' +
    '  LET stk = DOCUMENT(@stock_collection, x.k)\n' +
    '  RETURN { k: x.k, id: x.id, entity: x.entity, f: x.f, v: x.v, s: x.s, sm: NOT_NULL(stk.summary, ""), sl: x.sl }';
  return branches.join('\n') + '\n' + merge;
}

function bindVars(q, extra) {
  const b = {
    q: q,
    th: 0.45,
    limit: 50,
    entity: '',
    stock_prop_graph: 'stock_prop_calib',
    stock_collection: 'stock_calib'
  };
  if (extra) {
    Object.keys(extra).forEach(function (k) { b[k] = extra[k]; });
  }
  return b;
}

function sleep(ms) {
  const end = Date.now() + ms;
  while (Date.now() < end) { /* busy wait */ }
}

// Wait until the view answers a known query.
let waited = 0;
let first = null;
while (waited < 60000) {
  const r = db._query(fullStatement(), bindVars('dbs023')).toArray();
  if (r.length > 0) {
    first = r;
    break;
  }
  sleep(500);
  waited += 500;
}

// ---------------------------------------------------------------- probes

const results = {};

function probe(name, stmt, bvars) {
  try {
    const bv = bvars || bindVars('dbs023');
    // ArangoDB rejects bind parameters that the query never declares.
    const used = {};
    Object.keys(bv).forEach(function (k) {
      if (stmt.indexOf('@' + k) !== -1) { used[k] = bv[k]; }
    });
    results[name] = db._query(stmt, used).toArray();
  } catch (err) {
    results[name] = 'ERROR: ' + err.message;
  }
}

// 1. TOKENS with a bind parameter: shows what the stemmer does.
probe('p01_tokens_bind',
  'RETURN TOKENS(@q, "' + TEXT + '")', bindVars('forms culminants'));

// 2. prefix dbs023 on stock_id, stock branch.
probe('p02_stockid_prefix',
  stockBranch('p0', 'stock_id', prefixExpr('stock_id'), scalarDisplay('stock_id'), SCORES.prefix) +
  '\nRETURN FLATTEN([p0])');

// 3. prefix ys on label, property branch: the row key must be the
// stock key, which proves the INBOUND direction.
probe('p03_label_prefix',
  propBranch('p3', 'label', prefixExpr('label'), scalarDisplay('label'), SCORES.prefix) +
  '\nRETURN FLATTEN([p3])', bindVars('ys'));

// 4. prefix pdm on the plasmid name.
probe('p04_plasmid_name_prefix',
  propBranch('p7', 'name', prefixExpr('name'), scalarDisplay('name'), SCORES.prefix) +
  '\nRETURN FLATTEN([p7])', bindVars('pdm'));

// 5. the same prefix without the ANALYZER() wrapper.
probe('p05_no_analyzer_wrapper',
  propBranch('p7', 'name', 'STARTS_WITH(d.name, @q)', scalarDisplay('name'), SCORES.prefix) +
  '\nRETURN FLATTEN([p7])', bindVars('pdm'));

// 6. fuzzy dbs0236127 on stock_id at a sweep of thresholds.
[0.2, 0.3, 0.45, 0.55, 0.65, 0.8, 1.0].forEach(function (th) {
  probe('p06_fuzzy_th_' + String(th).replace('.', '_'),
    stockBranch('n0', 'stock_id', fuzzyExpr('stock_id'), scalarDisplay('stock_id'), SCORES.fuzzy) +
    '\nRETURN FLATTEN([n0])', bindVars('dbs0236127', { th: th }));
});

// 7. token expression with the wrapper, 1-word query culminants.
probe('p07_token_wrapped',
  stockBranch('t0', 'summary', tokenExpr('summary'), scalarDisplay('summary'), SCORES.token) +
  '\nRETURN FLATTEN([t0])', bindVars('culminants'));

// 8. the same token expression without the wrapper.
probe('p08_token_unwrapped',
  stockBranch('t0', 'summary', 'd.summary IN TOKENS(@q, "' + TEXT + '")', scalarDisplay('summary'), SCORES.token) +
  '\nRETURN FLATTEN([t0])', bindVars('culminants'));

// 9. PHRASE with the 2-word bind parameter, in stored order.
probe('p09_phrase_in_order',
  stockBranch('h0', 'summary', phraseExpr('summary'), scalarDisplay('summary'), SCORES.phrase) +
  '\nRETURN FLATTEN([h0])', bindVars('forms culminants'));

// 10. PHRASE against the other-order strain: zero rows expected.
probe('p10_phrase_other_order',
  stockBranch('h0', 'summary', phraseExpr('summary'), scalarDisplay('summary'), SCORES.phrase) +
  '\nRETURN FLATTEN([h0])', bindVars('culminants forms'));

// 11. PHRASE against a field whose analyzer lacks frequency and
// position (editable_summary is linked with the featureless norm
// analyzer only).
probe('p11_phrase_no_features',
  stockBranch('h0', 'editable_summary', 'PHRASE(d.editable_summary, @q, "' + NORM + '")', scalarDisplay('editable_summary'), SCORES.phrase) +
  '\nRETURN FLATTEN([h0])', bindVars('editableuniqueterm'));

// 12. prefix dictyo on species.
probe('p12_species_prefix',
  propBranch('p5', 'species', prefixExpr('species'), scalarDisplay('species'), SCORES.prefix) +
  '\nRETURN FLATTEN([p5])', bindVars('dictyo'));

// 13. token query costanza on depositor.
probe('p13_depositor_token',
  stockBranch('t1', 'depositor', tokenExpr('depositor'), scalarDisplay('depositor'), SCORES.token) +
  '\nRETURN FLATTEN([t1])', bindVars('costanza'));

// 14. the full statement with the query dbs023.
probe('p14_full_dbs023', fullStatement());

// 15. the measured BM25 range: the raw BM25 of the token branch over a
// repeated word that appears in many documents, plus the raw BM25 of
// the fuzzy and prefix branches. The band width is 250, so the maximum
// must stay far below 250.
function rawBm25(vname, field, expr, q, extra) {
  const stmt =
    'LET r = (\n' +
    '  FOR d IN ' + VIEW + '\n' +
    '    SEARCH ' + expr + '\n' +
    '    SORT BM25(d) DESC\n' +
    '    LIMIT 100\n' +
    '    RETURN BM25(d)\n' +
    ')\n' +
    'RETURN { max: FIRST(r), min: LAST(r), n: LENGTH(r) }';
  probe(vname, stmt, bindVars(q, extra));
}
rawBm25('p15_bm25_token_calibword', 'summary', tokenExpr('summary'), 'calibword');
rawBm25('p15_bm25_token_forms', 'summary', tokenExpr('summary'), 'forms');
rawBm25('p15_bm25_fuzzy_stockid', 'stock_id', fuzzyExpr('stock_id'), 'dbs0236127', { th: 0.2 });
rawBm25('p15_bm25_prefix_stockid', 'stock_id', prefixExpr('stock_id'), 'dbs0');
rawBm25('p15_bm25_prefix_genes', 'genes', prefixExpr('genes'), 'xylose');
probe('p15_full_scores', fullStatement(), bindVars('quartz ladder'));
probe('p15_full_forms', fullStatement(), bindVars('forms culminants'));

// 16. multi-token semantics through the full statement.
probe('p16_full_multiword', fullStatement(), bindVars('forms culminants'));

// 17. multi-token noise: computed in the report from the multi-word
// full-statement rows, because a LET assignment of a statement that
// starts with LET is not valid AQL.

// 18. stopword behavior.
probe('p18_tokens_stopwords', 'RETURN TOKENS(@q, "' + TEXT + '")', bindVars('the and'));
probe('p18_full_stopwords', fullStatement(), bindVars('the and'));

// 19. whitespace-only and punctuation-only queries.
probe('p19_whitespace', fullStatement(), bindVars('   '));
probe('p19_punctuation', fullStatement(), bindVars('---'));

// 20. entity filter inside the genes branch versus after the merge.
const xyloseFuzzy = stockBranch('n1', 'genes', fuzzyExpr('genes'), arrayDisplay('genes'), SCORES.fuzzy);
probe('p20_entity_in_branch',
  xyloseFuzzy + '\nRETURN FLATTEN([n1])', bindVars('xylose', { entity: 'plasmid' }));
const postMerge =
  'LET n1 = (\n' +
  '  FOR d IN ' + VIEW + '\n' +
  '    SEARCH ' + fuzzyExpr('genes') + '\n' +
  '    LET meta = FIRST(FOR prop, edge IN 1..1 OUTBOUND d GRAPH @stock_prop_graph RETURN edge.type)\n' +
  '    FILTER meta != null\n' +
  '    SORT BM25(d) DESC, d._key ASC\n' +
  '    LIMIT @limit\n' +
  '    RETURN { k: d._key, id: d.stock_id, entity: meta, f: "genes", v: ' + arrayDisplay('genes') + ', s: ' + SCORES.fuzzy + ' }\n' +
  ')\n' +
  'LET filtered = (FOR x IN FLATTEN([n1]) FILTER @entity == "" OR @entity == x.entity RETURN x)\n' +
  'RETURN filtered';
probe('p20_entity_after_merge', postMerge, bindVars('xylose', { entity: 'plasmid' }));
// 20c. the same branch with no filter at all: how many of the 50 kept
// rows are plasmids, which measures the loss of the wrong design.
const postMergeUnfiltered =
  'LET n1 = (\n' +
  '  FOR d IN ' + VIEW + '\n' +
  '    SEARCH ' + fuzzyExpr('genes') + '\n' +
  '    LET meta = FIRST(FOR prop, edge IN 1..1 OUTBOUND d GRAPH @stock_prop_graph RETURN edge.type)\n' +
  '    FILTER meta != null\n' +
  '    SORT BM25(d) DESC, d._key ASC\n' +
  '    LIMIT @limit\n' +
  '    RETURN { k: d._key, id: d.stock_id, entity: meta, f: "genes", v: ' + arrayDisplay('genes') + ', s: ' + SCORES.fuzzy + ' }\n' +
  ')\n' +
  'LET kept = FLATTEN([n1])\n' +
  'LET plasmids = (FOR x IN kept FILTER x.entity == "plasmid" RETURN x.k)\n' +
  'RETURN { kept: LENGTH(kept), plasmids: LENGTH(plasmids) }';
probe('p20c_unfiltered_after_merge', postMergeUnfiltered, bindVars('xylose'));

// 21. cross-collection merge attribution: one stock matching label,
// genes and summary.
probe('p21_cross_collection_merge', fullStatement(), bindVars('cordax'));

// 22. array display: prefix hit, fuzzy-only hit, and the expression
// over a document without the array attribute.
probe('p22_array_prefix',
  propBranch('p4', 'names', prefixExpr('names'), arrayDisplay('names'), SCORES.prefix) +
  '\nRETURN FLATTEN([p4])', bindVars('ax'));
probe('p22_array_fuzzy',
  propBranch('n4', 'names', fuzzyExpr('names'), arrayDisplay('names'), SCORES.fuzzy) +
  '\nRETURN FLATTEN([n4])', bindVars('gammas'));
probe('p22_array_absent_expr',
  'FOR d IN stock_properties_calib\n' +
  '  FILTER d.label == "axlessnames"\n' +
  '  LET v = ' + arrayDisplay('names').replace(/@q/g, '"ax"') + '\n' +
  '  RETURN { label: d.label, v: v }');

// 23. the property document with no inbound edge.
probe('p23_orphan_property',
  propBranch('p3', 'label', prefixExpr('label'), scalarDisplay('label'), SCORES.prefix) +
  '\nRETURN FLATTEN([p3])', bindVars('orphan'));

// 24. the plasmid without a summary attribute, through the tail.
probe('p24_plasmid_missing_summary', fullStatement(), bindVars('pdm'));

// 25. FLATTEN over the 19 branch variables returns a flat list.
const flat = db._query(fullStatement(), bindVars('dbs023')).toArray();
results.p25_flatten_is_flat =
  (Array.isArray(flat) && flat.every(function (r) { return r !== null && typeof r === 'object' && !Array.isArray(r); })) ?
  'flat: ' + flat.length + ' rows' : 'NOT FLAT';

// 26. the full statement for the unique editable_summary term.
probe('p26_editable_summary_not_searched', fullStatement(), bindVars('editableuniqueterm'));

// -------------------------------------------------------------- EXPLAIN

const stmt = fullStatement();
const explRes = db._connection.POST('/_db/' + DBNAME + '/_api/explain',
  JSON.stringify({ query: stmt, bindVars: bindVars('forms culminants') }));
const explPlan = explRes.plan || (explRes.result && explRes.result.plan) || null;
if (!explPlan) {
  throw new Error('explain failed: ' + JSON.stringify(explRes).slice(0, 500));
}
const nodeTypes = {};
explPlan.nodes.forEach(function (n) {
  nodeTypes[n.type] = (nodeTypes[n.type] || 0) + 1;
});
results.explain_view_nodes = nodeTypes['EnumerateViewNode'] || 0;
results.explain_collection_nodes = nodeTypes['EnumerateCollectionNode'] || 0;
results.explain_node_types = nodeTypes;

// -------------------------------------------------------------- latency

const timings = [];
for (i = 0; i < 20; i++) {
  const t0 = Date.now();
  db._query(stmt, bindVars('forms culminants')).toArray();
  timings.push(Date.now() - t0);
}
timings.sort(function (a, b) { return a - b; });
function pct(arr, p) {
  return arr[Math.min(arr.length - 1, Math.ceil((p / 100) * arr.length) - 1)];
}
results.latency_ms = {
  min: timings[0],
  median: pct(timings, 50),
  p95: pct(timings, 95),
  all: timings
};

// --------------------------------------------------------------- report

results.database = DBNAME;
results.arango_version = db._version();
results.fixture_counts = {
  stock: stock.count(),
  properties: props.count(),
  edges: edges.count()
};
print(JSON.stringify(results, null, 2));

db._useDatabase('_system');
db._dropDatabase(DBNAME);
print('dropped ' + DBNAME);