// Calibration harness for the stock autocomplete search.
//
// Run against a disposable database on a real ArangoDB 3.11 server:
//
//   docker exec -i arango311 arangosh \
//     --server.endpoint tcp://localhost:8529 \
//     --server.username root --server.password rootpass \
//     --javascript.execute /dev/stdin < scripts/autocomplete-calibration.js
//
// The script prints a JSON report with the 20 probe results, the
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

const NORM = 'stock_autocomplete_norm';
const NGRAM = 'stock_autocomplete_ngram';

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
db._createView('stock_autocomplete', 'arangosearch', {
  links: {
    stock_calib: {
      fields: {
        stock_id: { analyzers: both },
        genes: { analyzers: both },
        dbxrefs: { analyzers: both }
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
function addStock(stockId, kind, propDoc, genes, dbxrefs) {
  const doc = { _key: stockId, stock_id: stockId, genes: genes, dbxrefs: dbxrefs };
  stock.insert(doc);
  const p = props.insert(propDoc);
  edges.insert({
    _from: 'stock_calib/' + stockId,
    _to: 'stock_properties_calib/' + p._key,
    type: kind
  });
}

addStock('DBS0236126', 'strain',
  { label: 'yS13', names: ['Ax2', 'gammaS13'], species: 'Dictyostelium discoideum', plasmid: 'pDM304' },
  ['sadA', 'DDB_G0348394'], ['d0319']);
addStock('DBP0000027', 'plasmid', { name: 'pDM304' }, ['sadA'], []);
addStock('DBS0777001', 'strain', { label: 'nonames' }, [], []);
// A property document with no inbound edge, to probe the missing-owner filter.
props.insert({ label: 'orphanprop' });
// A stock that matches both label (property branch) and genes (stock branch).
addStock('DBS0167777', 'strain', { label: 'cordax', names: ['cordax-alias'] }, ['cordaxin'], []);

// 60 strains and 5 plasmids that all carry a gene with the prefix xylose.
let i;
for (i = 0; i < 60; i++) {
  addStock('DBS05' + String(100000 + i), 'strain', { label: 'xyl-' + i },
    ['xylose_G' + i], []);
}
for (i = 0; i < 5; i++) {
  addStock('DBP05' + String(100000 + i), 'plasmid', { name: 'xylp-' + i },
    ['xylose_P' + i], []);
}

// About 1000 further stocks with random identifiers, for a warm index.
const CHARS = 'abcdefghijklmnopqrstuvwxyz0123456789';
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
  bulkStock.push({ _key: sid, stock_id: sid, genes: [randId('g')], dbxrefs: [] });
  bulkProps.push({ label: randId('l') });
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

const VIEW = 'stock_autocomplete';

function prefixExpr(field) {
  return 'ANALYZER(STARTS_WITH(d.' + field + ', @q), "' + NORM + '")';
}
function fuzzyExpr(field) {
  return 'NGRAM_MATCH(d.' + field + ', @q, @th, "' + NGRAM + '")';
}
function scalarDisplay(field) {
  return 'NOT_NULL(d.' + field + ', "")';
}
function arrayDisplay(field) {
  return 'NOT_NULL(FIRST(FOR item IN NOT_NULL(d.' + field + ', []) ' +
    'FILTER CONTAINS(LOWER(item), @q) RETURN item), ' +
    'CONCAT_SEPARATOR(", ", NOT_NULL(d.' + field + ', [])))';
}

// A stock-collection branch. The row key is the stock key.
function stockBranch(vname, field, expr, display, score) {
  return 'LET ' + vname + ' = (\n' +
    '  FOR d IN ' + VIEW + '\n' +
    '    SEARCH ' + expr + '\n' +
    '    LET ent = FIRST(\n' +
    '      FOR v, e IN 1..1 OUTBOUND d GRAPH @stock_prop_graph\n' +
    '        RETURN e.type\n' +
    '    )\n' +
    '    FILTER ent != null\n' +
    '    FILTER @entity == "" OR @entity == ent\n' +
    '    SORT BM25(d) DESC, d._key ASC\n' +
    '    LIMIT @limit\n' +
    '    RETURN { k: d._key, id: d.stock_id, entity: ent, f: ' + JSON.stringify(field) + ', v: ' + display + ', s: ' + score + ' }\n' +
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
    '        RETURN { k: v._key, id: v.stock_id, entity: e.type }\n' +
    '    )\n' +
    '    FILTER own != null\n' +
    '    FILTER @entity == "" OR @entity == own.entity\n' +
    '    SORT BM25(d) DESC, own.k ASC\n' +
    '    LIMIT @limit\n' +
    '    RETURN { k: own.k, id: own.id, entity: own.entity, f: ' + JSON.stringify(field) + ', v: ' + display + ', s: ' + score + ' }\n' +
    ')';
}

const FIELDS = [
  { field: 'stock_id', coll: 'stock', kind: 'scalar' },
  { field: 'genes', coll: 'stock', kind: 'array' },
  { field: 'dbxrefs', coll: 'stock', kind: 'array' },
  { field: 'label', coll: 'prop', kind: 'scalar' },
  { field: 'names', coll: 'prop', kind: 'array' },
  { field: 'species', coll: 'prop', kind: 'scalar' },
  { field: 'plasmid', coll: 'prop', kind: 'scalar' },
  { field: 'name', coll: 'prop', kind: 'scalar' }
];

// The full 16-branch statement: prefix and fuzzy per field, merged.
function fullStatement() {
  const branches = [];
  const names = [];
  FIELDS.forEach(function (f, idx) {
    const display = f.kind === 'scalar' ? scalarDisplay(f.field) : arrayDisplay(f.field);
    const pn = 'p' + idx;
    const nn = 'n' + idx;
    branches.push((f.coll === 'stock' ? stockBranch : propBranch)(pn, f.field, prefixExpr(f.field), display, '1000 + BM25(d)'));
    branches.push((f.coll === 'stock' ? stockBranch : propBranch)(nn, f.field, fuzzyExpr(f.field), display, 'BM25(d)'));
    names.push(pn, nn);
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
    '  RETURN { k: x.k, id: x.id, entity: x.entity, f: x.f, v: x.v, s: x.s }';
  return branches.join('\n') + '\n' + merge;
}

function bindVars(q, extra) {
  const b = {
    q: q,
    th: 0.45,
    limit: 5,
    entity: '',
    stock_prop_graph: 'stock_prop_calib'
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

// 1. prefix dbs023 on stock_id, stock branch.
probe('p1_stockid_prefix',
  stockBranch('p0', 'stock_id', prefixExpr('stock_id'), scalarDisplay('stock_id'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p0])');
// 2. prefix ys on label, property branch: the row key must be the stock key.
probe('p2_label_prefix',
  propBranch('p3', 'label', prefixExpr('label'), scalarDisplay('label'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p3])', bindVars('ys'));
// 3. prefix pdm on the plasmid name.
probe('p3_plasmid_name_prefix',
  propBranch('p7', 'name', prefixExpr('name'), scalarDisplay('name'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p7])', bindVars('pdm'));
// 4. the same prefix without the ANALYZER() wrapper.
probe('p4_no_analyzer_wrapper',
  propBranch('p7', 'name', 'STARTS_WITH(d.name, @q)', scalarDisplay('name'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p7])', bindVars('pdm'));
// 5. fuzzy dbs0236127 on stock_id at a sweep of thresholds.
[0.2, 0.3, 0.45, 0.55, 0.65, 0.8, 1.0].forEach(function (th) {
  probe('p5_fuzzy_th_' + String(th).replace('.', '_'),
    stockBranch('n0', 'stock_id', fuzzyExpr('stock_id'), scalarDisplay('stock_id'), 'BM25(d)') +
    '\nRETURN FLATTEN([n0])', bindVars('dbs0236127', { th: th }));
});
// 6. fuzzy sdaa on genes.
probe('p6_fuzzy_gene_sdaa',
  stockBranch('n1', 'genes', fuzzyExpr('genes'), arrayDisplay('genes'), 'BM25(d)') +
  '\nRETURN FLATTEN([n1])', bindVars('sdaa'));
// 7. prefix dictyo on species.
probe('p7_species_prefix',
  propBranch('p5', 'species', prefixExpr('species'), scalarDisplay('species'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p5])', bindVars('dictyo'));
// 8. prefix d031 on dbxrefs.
probe('p8_dbxrefs_prefix',
  stockBranch('p2', 'dbxrefs', prefixExpr('dbxrefs'), arrayDisplay('dbxrefs'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p2])', bindVars('d031'));
// 9. uppercase query DBS023, lowercased in the client.
probe('p9_uppercase_lowered',
  stockBranch('p0', 'stock_id', prefixExpr('stock_id'), scalarDisplay('stock_id'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p0])', bindVars('dbs023'.toLowerCase()));
// 10. accented query Ax2: plain ToLower versus TOKENS through the norm analyzer.
probe('p10_accent_tolower',
  stockBranch('p0', 'stock_id', prefixExpr('stock_id'), scalarDisplay('stock_id'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p0])', bindVars('Áx2'.toLowerCase()));
probe('p10_accent_tokens',
  propBranch('p4', 'names', 'ANALYZER(STARTS_WITH(d.names, FIRST(TOKENS(@q, "' + NORM + '"))), "' + NORM + '")',
    arrayDisplay('names'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p4])', bindVars('Áx2'));
// 11. array display on names for prefix ax.
probe('p11_array_display_prefix',
  propBranch('p4', 'names', prefixExpr('names'), arrayDisplay('names'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p4])', bindVars('ax'));
// 12. array display for a fuzzy-only hit (no CONTAINS match).
probe('p12_array_display_fuzzy',
  propBranch('n4', 'names', fuzzyExpr('names'), arrayDisplay('names'), 'BM25(d)') +
  '\nRETURN FLATTEN([n4])', bindVars('gammas'));
// 13. array display on the property document without names.
probe('p13_array_display_absent',
  propBranch('n4', 'names', fuzzyExpr('names'), arrayDisplay('names'), 'BM25(d)') +
  '\nRETURN FLATTEN([n4])', bindVars('nonam'));
// 14. the property document with no inbound edge.
probe('p14_orphan_property',
  propBranch('p3', 'label', prefixExpr('label'), scalarDisplay('label'), '1000 + BM25(d)') +
  '\nRETURN FLATTEN([p3])', bindVars('orphan'));
// 15a. entity filter inside the genes branch.
const xyloseBranch = stockBranch('n1', 'genes', fuzzyExpr('genes'), arrayDisplay('genes'), 'BM25(d)');
probe('p15_entity_filter_in_branch',
  xyloseBranch + '\nRETURN FLATTEN([n1])',
  bindVars('xylose', { entity: 'plasmid', limit: 10 }));
// 15b. the same, with the filter moved after the merge.
const postMerge =
  'LET n1 = (\n' +
  '  FOR d IN ' + VIEW + '\n' +
  '    SEARCH ' + fuzzyExpr('genes') + '\n' +
  '    LET ent = FIRST(FOR v, e IN 1..1 OUTBOUND d GRAPH @stock_prop_graph RETURN e.type)\n' +
  '    FILTER ent != null\n' +
  '    SORT BM25(d) DESC, d._key ASC\n' +
  '    LIMIT @limit\n' +
  '    RETURN { k: d._key, id: d.stock_id, entity: ent, f: "genes", v: ' + arrayDisplay('genes') + ', s: BM25(d) }\n' +
  ')\n' +
  'LET filtered = (FOR x IN FLATTEN([n1]) FILTER @entity == "" OR @entity == x.entity RETURN x)\n' +
  'RETURN filtered';
probe('p15_entity_filter_after_merge', postMerge, bindVars('xylose', { entity: 'plasmid', limit: 10 }));
// 16. cross-collection merge: label and genes match one stock.
probe('p16_cross_collection_merge', fullStatement(), bindVars('corda'));
// 17. whitespace-only query.
probe('p17_whitespace_query', fullStatement(), bindVars('   '));
// 18. punctuation-only query.
probe('p18_punctuation_query', fullStatement(), bindVars('---'));
// 19. 1-character and 2-character queries.
probe('p19_one_char', fullStatement(), bindVars('d'));
probe('p19_two_char', fullStatement(), bindVars('db'));
// 20. FLATTEN over the 16 branch variables returns a flat list.
const flat = results._flat_check = db._query(fullStatement(), bindVars('dbs023')).toArray();
results.p20_flatten_is_flat =
  (Array.isArray(flat) && flat.every(function (r) { return r !== null && typeof r === 'object' && !Array.isArray(r); })) ?
  'flat: ' + flat.length + ' rows' : 'NOT FLAT';

// -------------------------------------------------------------- EXPLAIN

const stmt = fullStatement();
const explRes = db._connection.POST('/_db/' + DBNAME + '/_api/explain',
  JSON.stringify({ query: stmt, bindVars: bindVars('dbs023') }));
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
  db._query(stmt, bindVars('dbs023')).toArray();
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
