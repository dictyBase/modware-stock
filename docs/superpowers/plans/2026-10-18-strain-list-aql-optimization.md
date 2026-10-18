# Strain List Filtered AQL Optimization Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Optimize filtered `ListStrains` AQL queries in `modware-stock` by classifying the generated AQL filter's collection dependencies (`cv`/`cvterm`, `stock_prop`, `s`) with `fp-go` predicates/pattern matching (`matchopt.Match`) and routing to three specialized AQL templates instead of a single worst-case 2-graph join-before-sort template.

**Architecture:**
Currently, every filtered `ListStrains` call uses `statement.StrainListFilter` (or `StrainListFilterWithCursor`), which always traverses `cvterm` → `cv` → `INBOUND s GRAPH stockonto` → `OUTBOUND stock_prop, etype GRAPH stockprop_type` **before** applying `SORT s.created_at DESC` and `LIMIT @limit`:
1. For ontology + stock-only filters (`listStrains` with `StrainTypeAll` / `StrainTypeRegular` / `StrainTypeGwdi`, or combined with `s.summary` / `s.depositor`), ArangoDB traverses `stockprop_type` for all `7,055`–`28,576` matched strains before taking `@limit` (`114,310` index scans, `768 ms`). Moving `SORT s.created_at DESC LIMIT @limit` before the `stockprop_type` traversal cuts `stockprop_type` lookups from `28,576` to `@limit` (`57,180` index scans, `224 ms` — **3.4× faster**).
2. For stock-only filters with no ontology clause (`listStrainsWithGene`: `gene@==...`, or `s.summary` / `s.depositor` alone), the current template still scans all `2,306` `cvterm` documents (`EnumerateCollectionNode`) and traverses `stockonto`. Starting from `@@stock_collection` and sorting/limiting on `s` before `stockprop_type` eliminates the ontology collections/graph entirely (`57 ms` vs `~1,000 ms` — **17× faster**).
3. For filters that reference `stock_prop.` (`label`, `species`, `plasmid`, `names`), `stock_prop` must remain in scope before `FILTER` and `SORT`/`LIMIT`:
   - If the filter also references `cv.`/`cvterm.`, retain the existing ontology + property join-before-limit template.
   - If the filter does not reference `cv.`/`cvterm.`, use a stock + property template without `cv`/`cvterm`.

**Tech Stack:** Go 1.26, ArangoDB 3.11 AQL, `github.com/IBM/fp-go/v2` (`option`, `predicate`, `function`), `github.com/dictyBase/fp-go-loom/matchopt`.

---

## Query Routing Matrix

| Route | Filter references `cv.` / `cvterm.`? | Filter references `stock_prop.`? | AQL Strategy | Bind Parameters |
|---|---|---|---|---|
| **1. OntologyOnly** | Yes | No | `cvterm` + `cv` → `INBOUND s GRAPH stockonto` → `FILTER` → `SORT s.created_at DESC LIMIT @limit` → `OUTBOUND stock_prop, etype GRAPH stockprop_type` | `@@cv_collection`, `@@cvterm_collection`, `@stock_cvterm_graph`, `@stock_prop_graph`, `@limit` (`+ @cursor`) |
| **2. OntologyAndProp** | Yes | Yes | Existing `StrainListFilter` / `StrainListFilterWithCursor` (`cvterm` + `cv` → `stockonto` → `stockprop_type` → `FILTER` → `SORT` → `LIMIT`) | `@@cv_collection`, `@@cvterm_collection`, `@stock_cvterm_graph`, `@stock_prop_graph`, `@limit` (`+ @cursor`) |
| **3. StockOnly** | No | No | `FOR s IN @@stock_collection` → `FILTER` → `SORT s.created_at DESC LIMIT @limit` → `OUTBOUND stock_prop, etype GRAPH stockprop_type FILTER etype.type == 'strain'` —Wait: `etype.type == 'strain'` must filter before `LIMIT` so plasmids in `@@stock_collection` do not consume `@limit` slots! Therefore: `FOR s IN @@stock_collection FOR stock_prop, etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph FILTER etype.type == 'strain' %s SORT s.created_at DESC LIMIT @limit` | `@@stock_collection`, `@stock_prop_graph`, `@limit` (`+ @cursor`) |
| **4. StockAndProp** | No | Yes | Same as Route 3 (`FOR s IN @@stock_collection FOR stock_prop, etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph FILTER etype.type == 'strain' %s SORT s.created_at DESC LIMIT @limit`) | `@@stock_collection`, `@stock_prop_graph`, `@limit` (`+ @cursor`) |

> **Important correctness note on Route 3 & 4:** Because `@@stock_collection` holds both strains and plasmids (`e.type == 'strain'` vs `'plasmid'`), `FILTER etype.type == 'strain'` must execute **before** `LIMIT @limit` whenever `stockonto` (`dicty_strain_property`) is not filtering to strain terms. Thus Routes 3 and 4 collapse into one **Stock-First Filter** route (`StrainListStockFilter` / `StrainListStockFilterWithCursor`), giving **three** distinct routes total:
> 1. **Ontology-First Deferred Property** (`hasOntology && !hasStockProp`): `SORT` + `LIMIT` before `stockprop_type`.
> 2. **Ontology-First Joined Property** (`hasOntology && hasStockProp`): existing `StrainListFilter` / `StrainListFilterWithCursor`.
> 3. **Stock-First Filter** (`!hasOntology`): starts from `@@stock_collection` + `stockprop_type` without `cv`/`cvterm`/`stockonto`.

---

## File Map

- **Modify:** `internal/repository/arangodb/statement/read.go`
  - Add `StrainListOntologyFilter`, `StrainListOntologyFilterWithCursor`, `StrainListStockFilter`, `StrainListStockFilterWithCursor`.
- **Modify:** `internal/repository/arangodb/strain_read.go`
  - Replace imperative `strainStmtNoFilter` / `strainStmtWithFilter` selection with `fp-go` predicates (`hasOntologyFilter`, `hasStockPropFilter`, `hasCursor`) and `matchopt.Match` statement/bind-param builders.
- **Modify:** `internal/repository/arangodb/strain_read_test.go` & `internal/repository/arangodb/strain_test.go`
  - Add unit tests for route classification + bind-var sets, `EXPLAIN` plan assertions against TestArango, and pagination/filter parity tests across all three routes.

---

## Step-by-Step Tasks

### Task 1: Add AQL Templates in `internal/repository/arangodb/statement/read.go`

- [x] **Step 1.1: Add `StrainListOntologyFilter` and `StrainListOntologyFilterWithCursor`**
  In `internal/repository/arangodb/statement/read.go`, add templates that apply ontology + `s.*` filters, `SORT s.created_at DESC`, and `LIMIT @limit` **before** traversing `@stock_prop_graph`:

```go
	StrainListOntologyFilter = `
		FOR cvterm in @@cvterm_collection
			FOR cv IN @@cv_collection
				FILTER cvterm.graph_id == cv._id
				FOR s IN 1..1 INBOUND cvterm GRAPH @stock_cvterm_graph
					%s
					SORT s.created_at DESC
					LIMIT @limit
					FOR stock_prop,etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
						FILTER etype.type == 'strain'
						RETURN MERGE(s,{
								strain_properties: {
									label: stock_prop.label,
									species: stock_prop.species,
									plasmid: stock_prop.plasmid,
									names: stock_prop.names
								}
							}
						)
	`
	StrainListOntologyFilterWithCursor = `
		FOR cvterm in @@cvterm_collection
			FOR cv IN @@cv_collection
				FILTER cvterm.graph_id == cv._id
				FOR s IN 1..1 INBOUND cvterm GRAPH @stock_cvterm_graph
					%s
					FILTER s.created_at <= DATE_ISO8601(@cursor)
					SORT s.created_at DESC
					LIMIT @limit
					FOR stock_prop,etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
						FILTER etype.type == 'strain'
						RETURN MERGE(s,{
								strain_properties: {
									label: stock_prop.label,
									species: stock_prop.species,
									plasmid: stock_prop.plasmid,
									names: stock_prop.names
								}
							}
						)
	`
```

- [x] **Step 1.2: Add `StrainListStockFilter` and `StrainListStockFilterWithCursor`**
  In `internal/repository/arangodb/statement/read.go`, add stock-first filtered templates that omit `@@cv_collection`, `@@cvterm_collection`, and `@stock_cvterm_graph` when the filter does not reference `cv.` or `cvterm.`:

```go
	StrainListStockFilter = `
		FOR s IN @@stock_collection
			FOR stock_prop,etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
				FILTER etype.type == 'strain'
				%s
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(s,{
						strain_properties: {
							label: stock_prop.label,
							species: stock_prop.species,
							plasmid: stock_prop.plasmid,
							names: stock_prop.names
						}
					}
				)
	`
	StrainListStockFilterWithCursor = `
		FOR s IN @@stock_collection
			FOR stock_prop,etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
				FILTER etype.type == 'strain'
				%s
				FILTER s.created_at <= DATE_ISO8601(@cursor)
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(s,{
						strain_properties: {
							label: stock_prop.label,
							species: stock_prop.species,
							plasmid: stock_prop.plasmid,
							names: stock_prop.names
						}
					}
				)
	`
```

---

### Task 2: Write Failing Route, Explain, and Parity Tests (TDD Red)

- [x] **Step 2.1: Add unit tests in `internal/repository/arangodb/strain_read_test.go`**
  Test pure query classification and bind-parameter generation across all routes:
  1. Empty filter (`""`) → `StrainList` / `StrainListWithCursor` with `@stock_collection` + `stock_prop_graph`.
  2. Ontology-only filter (`FILTER cv.metadata.namespace == 'dicty_strain_property' AND cvterm.label == 'general strain'`) → `StrainListOntologyFilter` / `StrainListOntologyFilterWithCursor` (`stockprop_type` traversal appears **after** `LIMIT @limit`).
  3. Ontology + stock filter (`FILTER s.summary =~ 'mutant' AND cv.metadata.namespace == 'dicty_strain_property' AND cvterm.label == 'general strain'`) → `StrainListOntologyFilter`.
  4. Ontology + `stock_prop` filter (`FILTER stock_prop.label =~ 'axe' AND cvterm.label == 'general strain'`) → `StrainListFilter` (join-before-limit).
  5. Stock-only filter (`LET g = (FILTER 'DDB_G0287317' IN s.genes[*] RETURN 1) FILTER LENGTH(g) > 0` or `FILTER s.depositor == 'george@costanza.com'`) → `StrainListStockFilter` with `@stock_collection` and **no** `@cv_collection` / `@cvterm_collection` / `stock_cvterm_graph` bind vars.
  6. `stock_prop`-only filter (`FILTER stock_prop.label =~ 'yS13'`) → `StrainListStockFilter`.

- [x] **Step 2.2: Add `EXPLAIN` plan assertions in `internal/repository/arangodb/strain_test.go`**
  Against `testarango`:
  - Assert that `ExplainQuery` on the Ontology-Only query places the `stockprop_type` `TraversalNode` **after** the `LimitNode`.
  - Assert that `ExplainQuery` on the Stock-First gene filter contains **zero** `cv` or `cvterm` `IndexNode`/`EnumerateCollectionNode` and **zero** `stockonto` `TraversalNode`.
  - Assert that cursor pagination, ordering, page overlap, and total counts remain identical across pages for all routes.

- [x] **Step 2.3: Run tests and confirm RED**
  ```bash
  go test -v ./internal/repository/arangodb -run 'TestStrainStmt|TestListStrains'
  ```

---

### Task 3: Implement Functional Route Classifier in `internal/repository/arangodb/strain_read.go` (TDD Green)

- [x] **Step 3.1: Define fp-go predicates and `matchopt.Match` classifier**
  Follow house `fp-go` rules (no imperative `if`/`switch` branching in pure selection helpers; single `type`/`const`/`var` blocks for `grouper`/`decorder`; no nested calls `f(g(x))` or `)(` direct combinator application):
  - Predicates on `filter string`:
    - `hasCvRef`: `predstrings.Contains("cv.")`
    - `hasCvtermRef`: `predstrings.Contains("cvterm.")`
    - `hasOntologyFilter = F.Pipe1(hasCvRef, P.Or(hasCvtermRef))`
    - `hasStockPropFilter = predstrings.Contains("stock_prop.")`
    - `isOntologyDeferredProp = F.Pipe1(hasOntologyFilter, P.And(P.Not(hasStockPropFilter)))`
  - Classify `*stock.StockParameters` into a route value using `MO.Match` (`github.com/dictyBase/fp-go-loom/matchopt`), selecting both the statement template (with cursor variant via `P.Fold` on `hasCursor`) and the exact bind-parameter map (`ontologyFirstBindParams` vs `stockFirstBindParams`) so ArangoDB never receives unused bind variables.

- [x] **Step 3.2: Run unit and ArangoDB integration tests**
  ```bash
  go test -v ./internal/repository/arangodb -run 'TestStrainStmt|TestListStrains'
  ```

---

### Task 4: Run Full Quality Gates

- [x] **Step 4.1: Format, lint, gopls, nested-call audit, and full test suite**
  ```bash
  golangci-lint fmt
  golangci-lint run ./...
  git ls-files '*.go' | xargs gopls check -severity=hint
  rg -n '[A-Z][A-Za-z0-9_.]*\([a-z][A-Za-z0-9_.]*\(.*\)' internal/repository/arangodb/strain_read.go
  rg -n '\)\([a-z]' internal/repository/arangodb/strain_read.go
  go test -count=1 ./...
  ```
- [ ] **Step 4.2: Performance acceptance check**
  - Ontology-only `StrainTypeAll` query scans `<= 60,000` index entries (down from `114,310`) and completes `>= 2x` faster than the `0.768s` baseline (`~0.22s`).
  - Stock-only `gene@==...` query executes zero `cv`/`cvterm` scans.

---

## Execution Notes

**Status:** Tasks 1–4.1 are complete on the branch `feat/strain-list-aql-routes`. Task 4.2
is not measured, because the local test database holds a few dozen strains (see below).

### Deviations from the plan

1. **`predstrings.Contains` does not exist.** The package
   `github.com/dictyBase/fp-go-loom/predicate/strings` exports only `LastIndexOf`,
   `HasSuffix`, `ContainsRuneClass`, `HasAtSign` and `StrLenBetween`. The classifier uses
   `S.Includes` from `github.com/IBM/fp-go/v2/string`, which is the curried
   `strings.Contains`.
2. **The classifier blanks string literals first.** A substring test alone sends a filter
   such as `FILTER s.depositor == 'cv.user@example.com'` to the ontology route, because
   the value contains `cv.`. `aqlLiteral` replaces every single quoted literal with `''`
   before the predicates run, and the routing table has a case for this filter.
3. **`matchopt.Match` is confirmed** in the pinned loom version
   `v0.0.0-20261010160534-3c971335f51e`.
4. **One entrypoint replaced two helpers.** `strainStmtNoFilter` and `strainStmtWithFilter`
   are gone. `strainListPage` resolves all four template pairs, so the empty filter reaches
   the unfiltered route through the same classification as the filtered routes.
5. **`paramCursor` was added** to `internal/repository/arangodb/consts.go`. The new code
   uses it instead of the `"cursor"` literal.
6. **A bind parameter coverage test was added.** `TestStrainListBindsCoverStatement` asserts
   that the statement and its bind map agree in both directions, which is the property
   ArangoDB enforces with its unused bind parameter error.
7. **The Route 1 invariant is documented** in the `StrainListOntologyFilter` doc comment:
   the route relies on every strain having exactly one `type: 'strain'` property edge, so
   the limit stays a strain count and the limit+1 page probe of the caller stays exact.

### Task 4.2: not measured

The plan asks for an index scan count of `<= 60,000` (down from `114,310`) and a runtime of
`>= 2x` better than `0.768s`. The local ArangoDB test database holds a few dozen strains, so
those numbers are not reproducible here. What the tests assert instead is the mechanism,
against ArangoDB 3.12.12:

- The ontology-only route explains to `SortNode` → `LimitNode` → `TraversalNode`
  (`stockprop_type`), so the limit runs before the property traversal.
- The stock-only route explains with no `cv` and no `cvterm` collection and no `stockonto`
  traversal node.
- The joined route explains to `TraversalNode` (`stockprop_type`) → `SortNode` →
  `LimitNode`, the join-before-limit order it must keep.

Run the acceptance measurement against a database of production size before release.
