# Stock Autocomplete Implementation Plan

> **For agentic workers:** Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task by task. The steps use checkboxes.

**Goal:** Add the `AutocompleteStock` RPC to modware-stock. The user types at least 3 characters. The server returns at most 50 short suggestions, 5 by default, over 8 identifier and name fields of strains and plasmids. ArangoSearch on ArangoDB 3.11 supplies the index.

**Architecture:** This plan owns its own search assets and shares nothing with the full-search plan. The assets are two analyzers, `stock_autocomplete_norm` and `stock_autocomplete_ngram`, plus one classic `arangosearch` view, `stock_autocomplete`. The view links the stock collection and the stock property collection. One AQL statement runs 16 branches: a prefix branch and a fuzzy branch per field. A branch over the stock collection reads the stock key directly. It then traverses the named graph `stock_prop_graph` once in the OUTBOUND direction, to learn the entity type. A branch over the property collection traverses once in the INBOUND direction. That direction is necessary, because the property `_key` is auto-generated and carries no relation to the stock key. Each branch filters by entity before its own `SORT` and `LIMIT`, so a small result group cannot be cut off. A merge step keeps the best row per stock key. The repository constructor creates the analyzers and the view eagerly, so a broken setup stops the start of the service.

**Tech Stack:** ArangoDB 3.11 (CI image `arangodb:3.11`), go-driver v1.6.9, dictyBase/arangomanager v0.8.0, dictyBase/aphgrpc v1.4.2, `github.com/bufbuild/protovalidate-go` v0.10.0, IBM/fp-go v1.1.84, gRPC v1.83.2, Go 1.26, gotestsum, golangci-lint, gopls.

**Spec:** User request (2026-10-06): the same autocomplete that modware-order has, applied to stocks. The untracked file `AUTOCOMPLETE.md` in the repository root holds the modware-order work record and the calibration results that this plan re-verifies. The web user interface is out of scope; this repository delivers the RPC.

---

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Goal](#goal)
- [Scope](#scope)
- [Non-goals](#non-goals)
- [Verified repository facts](#verified-repository-facts)
  - [Data model](#data-model)
  - [Repository plumbing](#repository-plumbing)
  - [Driver and manager limits](#driver-and-manager-limits)
  - [Service and test plumbing](#service-and-test-plumbing)
  - [Continuous integration](#continuous-integration)
- [Prerequisite gate](#prerequisite-gate)
- [Consumed protocol contract](#consumed-protocol-contract)
- [Exact interfaces](#exact-interfaces)
  - [Repository types and method](#repository-types-and-method)
  - [Search asset names and constants](#search-asset-names-and-constants)
  - [View definition](#view-definition)
  - [Field and branch matrix](#field-and-branch-matrix)
  - [AQL branch templates](#aql-branch-templates)
  - [Merge and tail](#merge-and-tail)
  - [Bind parameters](#bind-parameters)
  - [Handler signature and mapping](#handler-signature-and-mapping)
- [File map](#file-map)
- [Implementation tasks](#implementation-tasks)
  - [Task 1: Calibration against real ArangoDB 3.11](#task-1-calibration-against-real-arangodb-311)
  - [Task 2: Repository types and interface method](#task-2-repository-types-and-interface-method)
  - [Task 3: Search assets in the constructor](#task-3-search-assets-in-the-constructor)
  - [Task 4: Query builder and AutocompleteStock](#task-4-query-builder-and-autocompletestock)
  - [Task 5: Service handler](#task-5-service-handler)
  - [Task 6: EXPLAIN and latency gate](#task-6-explain-and-latency-gate)
  - [Task 7: Documentation](#task-7-documentation)
- [Test matrix](#test-matrix)
  - [Repository tests, real disposable ArangoDB 3.11](#repository-tests-real-disposable-arangodb-311)
  - [Builder unit tests, no fixture](#builder-unit-tests-no-fixture)
  - [Handler tests, stub repository](#handler-tests-stub-repository)
  - [Handler test, one real round trip](#handler-test-one-real-round-trip)
- [Failure behavior](#failure-behavior)
- [Quality gates](#quality-gates)
- [Completion criteria](#completion-criteria)
- [Handoff artifacts](#handoff-artifacts)
- [Review focus](#review-focus)
- [Execution notes](#execution-notes)

---

## Goal

Deliver a working `AutocompleteStock` RPC with a repository implementation, its own ArangoSearch assets, and tests that run against a real disposable ArangoDB 3.11 database. A client that types 3 or more characters gets at most 50 suggestions, 5 by default. Each suggestion names the stock, the kind of stock, the field that matched, the text to show, and a score.

## Scope

- 3 new repository types and 1 new method on `repository.StockRepository`.
- 2 new analyzers and 1 new classic `arangosearch` view, created eagerly in `NewStockRepo`.
- Reconciliation of an existing view whose links differ from the definition, with `ArangoSearchView.Properties` and `ArangoSearchView.SetProperties`.
- 1 AQL statement with 16 branches, built from 2 templates.
- 1 gRPC handler on `StockService`.
- Repository tests against a real database, handler unit tests against a stub repository, and one handler test over a real buffer connection.
- A calibration harness that proves every AQL assumption on ArangoDB 3.11.
- An EXPLAIN check and a latency gate.
- Documentation of the RPC contract.

## Non-goals

- No proto edit. The protocol plan `docs/superpowers/plans/2026-10-06-stock-search-protobuf.md` owns the wire contract, and this plan consumes the released stubs.
- No full-search work. The plan `docs/superpowers/plans/2026-10-06-stock-full-search.md` owns the `SearchStock` RPC, the analyzers `stock_search_norm` and `stock_search_ngram`, the built-in `text_en` analyzer, and the view `stock_full_search`. This plan must run correctly whether or not those assets exist.
- No prose search. `summary`, `editable_summary` and `depositor` are not indexed by the view of this plan.
- No cursor pagination. A suggestion list is short by design.
- No `search-alias` view and no `MIN_HASH_MATCH()`. This plan uses classic `arangosearch` views only.
- No change to `.github/workflows/ci.yml`.

## Verified repository facts

### Data model

Checked in `internal/repository/arangodb/statement/insert.go`, `statement/read.go`, `statement/update.go` and `internal/repository/arangodb/database.go`.

| Fact | Evidence |
| --- | --- |
| Strains and plasmids share one stock collection. A stock document holds `created_at`, `updated_at`, `created_by`, `updated_by`, `summary`, `editable_summary`, `depositor`, `genes`, `dbxrefs`, `publications` and `stock_id`. | `statement/insert.go`, `StockStrainIns` and `StockPlasmidIns` |
| The stock `_key` is explicit and equal to `stock_id`: `CONCAT("DBS0", kg[0]._key)` for a new strain, `CONCAT("DBP0", kg[0]._key)` for a new plasmid, and `@stock_id` for a load. | `statement/insert.go`, all six insert statements |
| Strains and plasmids share one stock property collection. A strain property document holds `label`, `species`, `plasmid` and `names`. A plasmid property document holds `image_map`, `sequence` and `name`. | `statement/insert.go`, the `LET o = (INSERT ...)` blocks |
| The property `_key` is **auto-generated**. No insert statement sets it. It is unrelated to the stock `_key`. | `statement/insert.go`, the `LET o` blocks set no `_key` |
| The only link from a stock document to its property document is the edge `{ _from: <stock>, _to: <property>, type: 'strain' or 'plasmid' }` in the stock type edge collection. | `statement/insert.go`, the `INSERT { _from: n[0]._id, _to: o[0]._id, type: ... }` lines |
| The named graph has exactly one edge definition: the edge collection runs **from** the stock collection **to** the stock property collection. So OUTBOUND from a stock document reaches its property document, and INBOUND from a property document reaches its stock document. | `database.go`, `createNamedGraph`, the first `driver.EdgeDefinition` |
| `StockFindIDQ` returns the auto-generated property `_key`, and the update statements consume it as `@propkey`. This is a second proof that the two keys differ. | `statement/read.go` `StockFindIDQ`, `statement/update.go` lines 17, 36, 60, 95 |
| An existing read query resolves the entity from the edge attribute, for example `FILTER e.type == 'strain'`. The same attribute carries the entity for a search row. | `statement/read.go`, `StockGetStrain` and `StockGetPlasmid` |
| A persistent unique index `stock_id_idx` already exists on the stock collection over `stock_id`. | `database.go`, `createIndex` |

An `arangosearch` view indexes each element of an array field on its own, so `STARTS_WITH` and `NGRAM_MATCH` work per element on `genes`, `dbxrefs` and `names`.

### Repository plumbing

Checked in `internal/repository/arangodb/arangodb.go`, `database.go`, `consts.go`.

| Fact | Evidence |
| --- | --- |
| `NewStockRepo(connP *manager.ConnectParams, collP *CollectionParams, ontoP *ontoarango.CollectionParams) (repository.StockRepository, error)` builds the struct, validates `collP`, opens the session, creates the ontology collections, and ends with `createDbStruct(ar, collP)`. It takes **no** `context.Context`. | `arangodb.go` lines 34-61 |
| `createDbStruct` runs `docCollections`, then `graphAndEdgeCollections`, then `return createIndex(ar)`. The search setup call belongs at the end of this chain. | `database.go`, `createDbStruct` |
| The struct `arangorepository` holds `ontoc`, `sess *manager.Session`, `database *manager.Database`, `stockc *stockc`, `strainOnto` and `plasmidOnto`. | `arangodb.go` lines 24-31 |
| The struct `stockc` holds `stock`, `stockProp`, `stockKey`, `stockType`, `parentStrain`, `stockTerm` as `driver.Collection`, and `stockPropType`, `strain2Parent`, `stockOnto` as `driver.Graph`. | `database.go`, `type stockc struct` |
| Collection and graph names are **not** constant. They come from `CollectionParams`. The repository tests use `stock_test`, `stock_properties_test`, `stock_type_test` and the graph `stockprop_type_test`. Build the view links from `ar.stockc.stock.Name()` and `ar.stockc.stockProp.Name()`, and the traversal from `ar.stockc.stockPropType.Name()`. Never hardcode a production name. | `database.go` `docCollections`, `internal/repository/arangodb/arangodb_test.go` `getCollectionParams` |
| Existing bind-parameter names live in `consts.go`: `nameStockPropGraph = "stock_prop_graph"`, `nameStockCollection = "stock_collection"`, `paramLimit = "limit"`, and the field names `fieldGenes`, `fieldDbxrefs`, `fieldLabel`, `fieldSpecies`, `fieldPlasmid`, `paramName`, `paramStockID`. Reuse them. | `consts.go` |
| `RemoveStock` already uses `context.Background()` inside the repository, so the same is acceptable in `createDbStruct`. | `internal/repository/arangodb` package, `RemoveStock` |

### Driver and manager limits

Checked in the module cache for go-driver v1.6.9 and arangomanager v0.8.0.

| Fact | Evidence |
| --- | --- |
| `driver.Database` offers `EnsureCreatedAnalyzer(ctx, *ArangoSearchAnalyzerDefinition) (ArangoSearchAnalyzer, bool, error)`. It is idempotent. | `database_arangosearch_analyzers.go` line 60 |
| `driver.Database` offers `ViewExists(ctx, name) (bool, error)`, `View(ctx, name) (View, error)` and `CreateArangoSearchView(ctx, name, *ArangoSearchViewProperties) (ArangoSearchView, error)`. | `database_views.go` lines 33 and 41, `database_views_impl.go` line 45 |
| `View` offers `ArangoSearchView() (ArangoSearchView, error)`, which fails when the view has another type. | `view.go` lines 38-40 |
| `ArangoSearchView` offers `Properties(ctx) (ArangoSearchViewProperties, error)` as a `GET .../properties`, and `SetProperties(ctx, ArangoSearchViewProperties) error` as a **PUT** `.../properties`. A PUT replaces the properties, so `SetProperties` is a full reconciliation and needs no delete. | `view_arangosearch.go` lines 34 and 37, `view_arangosearch_impl.go` lines 36-74 |
| `ArangoSearchLinks` is `map[string]ArangoSearchElementProperties` keyed by collection name. `ArangoSearchFields` is `map[string]ArangoSearchElementProperties` keyed by field name. `ArangoSearchElementProperties` carries `Analyzers []string`, `Fields ArangoSearchFields`, `IncludeAllFields *bool`, `TrackListPositions *bool` and `StoreValues`. | `view_arangosearch.go`, `ArangoSearchLinks` and following types |
| The analyzer constants are `driver.ArangoSearchAnalyzerTypeNorm`, `...TypeNGram`, `...TypePipeline`, the case constant `driver.ArangoSearchCaseLower`, the stream constant `driver.ArangoSearchNGramStreamUTF8`, and the features `driver.ArangoSearchAnalyzerFeatureFrequency`, `...FeatureNorm`, `...FeaturePosition`. | `view_arangosearch.go` lines 50-124 |
| `manager.Database` offers `Handler() driver.Database` and `SearchRows(query string, bindVars map[string]any) (*Resultset, error)`. | `arangomanager/database.go` lines 82 and 88 |
| `SearchRows` calls `ValidateQuery` **before** it runs the query. A statement with an AQL syntax error therefore fails with `error in validating the query ...` and never reaches the server as a query. | `arangomanager/database.go` lines 88-104 |
| `SearchRows` returns `&Resultset{empty: true}, nil` when the cursor has no rows. `Resultset.Scan()` then returns false at once. A direct call to `Resultset.Read` returns the error `cannot read from empty resultset`. A loop of `for result.Scan() { result.Read(&row) }` is therefore safe for an empty result. | `arangomanager/database.go` line 110, `arangomanager/resultset.go` |
| `Resultset.Read` calls `cursor.ReadDocument`. go-driver v1 **cannot** decode an array row into a Go slice. Every projection must return one object per row. | `arangomanager/resultset.go` line 44, plus the recorded failure `cannot unmarshal object into []suggestionRow` in `AUTOCOMPLETE.md` section 4 |
| `testarango.NewTestArangoFromEnv(true)` reads `ARANGO_USER`, `ARANGO_HOST`, `ARANGO_PASS` and the optional `ARANGO_PORT`, connects, and creates a random disposable database of 6 to 8 characters. | `arangomanager/testarango/testarango.go` |

### Service and test plumbing

Checked in `internal/app/service`.

| Fact | Evidence |
| --- | --- |
| `StockService` embeds `*aphgrpc.Service` and `stock.UnimplementedStockServiceServer`, and holds `repo repository.StockRepository` and `publisher message.Publisher`. A new method on the generated interface therefore needs no change in `internal/app/server`. | `service.go`, `type StockService struct` |
| `NewStockService(repo, pub, opt ...aphgrpc.Option) *StockService`. | `service.go` |
| Existing handlers map errors with `aphgrpc.HandleInvalidParamError`, `aphgrpc.HandleGetError` and `aphgrpc.HandleNotFoundError`. | `strain.go`, `GetStrain` |
| Existing handlers call the generated `r.Validate()`. That method is a **no-op** for a message without mwitkow rules, so a new handler must call `protovalidate.Validate` instead. | `strain.go` line 20, and `dictybaseapis/stock/stock.validator.pb.go` line 431 in go-genproto, where `func (this *Meta) Validate() error { return nil }` |
| modware-stock has **no** gRPC recovery interceptor, and existing handlers dereference `r.Data.Attributes` without a nil check. A panic in a handler kills the process. A new handler must guard a nil `data` and a nil `attributes` first. | `internal/app/server`, absence of a recovery interceptor; `strain.go` `LoadStrain` line 46 |
| The repository test helper `setUp(t *testing.T) (*require.Assertions, repository.StockRepository)` creates a disposable database, builds the repository, and loads the ontology. `tearDown(repo)` drops the database. | `internal/repository/arangodb/arangodb_test.go` lines 198-222 |
| Repository test fixtures already exist: `newTestStrain(createdby string, stype StrainType) *stock.NewStrain` with `Label: "yS13"`, `Genes: []string{"DDB_G0348394", "DDB_G098058933"}`, `Plasmid: "DBP0000027"` and `Names: []string{"gammaS13", "gammaS-13", "γS-13"}`, and `newTestPlasmid(createdby string) *stock.NewPlasmid` with `Name: "p123456"`. | `internal/repository/arangodb/arangodb_test.go` lines 110-196 |
| `repo.AddStrain` returns `(*model.StockDoc, error)`. `repo.AddPlasmid` returns `IOE.IOEither[error, *model.StockDoc]`. The existing tests run it with `F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)`, where `ToEither` and `toStockDocResult` are defined in `internal/repository/arangodb/plasmid_test.go` lines 37-54 and are available to every test file of the same package. | `internal/repository/arangodb/plasmid_test.go` lines 204-218 |
| `NewPlasmidAttributes` carries `genes` as field 5, so a plasmid can hold genes. A `genes` query therefore matches both kinds of stock, which makes `genes` the right field for an entity-filter regression test. | `dictybaseapis/dictybase/stock/stock.proto` lines 205-231 |
| The current service test helper `setupGrpcClient` calls `resolver.SetDefaultScheme("passthrough")`, which mutates global gRPC state and races with parallel tests, and `setupGrpcServer` calls `t.Logf` and `os.Exit(1)` from the serving goroutine, which races with the testing package and can kill the test binary. New tests must not copy this helper. | `internal/app/service/strain_test_helpers.go`, `setupGrpcServer` and `setupGrpcClient` |
| The safe pattern used by modware-order dials `grpc.NewClient("passthrough:///bufnet", ...)` with a context dialer, changes no global resolver, and logs nothing from the serving goroutine: `go func() { _ = srv.Serve(lis) }()`. | `modware-order/internal/app/service/service_arango_test.go` lines 39-78 |
| A no-op publisher already exists for service tests: `MockPublisher` with `PublishStrain`, `PublishPlasmid` and `Close`. | `internal/app/service/strain_test_helpers.go` |

### Continuous integration

| Fact | Evidence |
| --- | --- |
| The `test` job runs the service `arangodb:3.11` with `ARANGO_ROOT_PASSWORD: rootpass` and port `8529/tcp`, and the test step exports `ARANGO_USER=root`, `ARANGO_PASS=rootpass`, `ARANGO_HOST=localhost` and `ARANGO_PORT` from the mapped service port. | `.github/workflows/ci.yml` |
| The test step runs `go test -covermode=atomic -coverprofile=profile.cov -v ./...`. New tests in existing packages are picked up with no workflow change. | `.github/workflows/ci.yml` |
| Go 1.26 is installed in CI, and `go.mod` declares `go 1.26.0`. | `.github/workflows/ci.yml`, `go.mod` |

## Prerequisite gate

Every item is a hard gate. Do not start Task 2 before the gate passes. Task 1 can run before the gate, because it needs no generated stubs.

- [ ] The protocol plan is complete, and `docs/superpowers/plans/2026-10-06-stock-search-protobuf.md` records a real `github.com/dictyBase/go-genproto` pseudo-version in its Execution notes.
- [ ] `go.mod` of modware-stock requires that pseudo-version. Confirm with:

```bash
cd /Users/sba964/Projects/devenv/golang/modware-stock
grep -n 'go-genproto' go.mod
```

- [ ] The generated stubs hold every type that this plan consumes. Confirm with:

```bash
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockAutocompleteParameters
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockAutocompleteAttributes
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockSuggestion
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockSuggestionCollection
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockEntity
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockSearchField
```

- [ ] `go.mod` requires `github.com/bufbuild/protovalidate-go v0.10.0`.
- [ ] A local ArangoDB 3.11 server runs, and the environment carries the four variables:

```bash
docker run -d -p 8529:8529 -e ARANGO_ROOT_PASSWORD=rootpass arangodb:3.11
export ARANGO_HOST=localhost ARANGO_USER=root ARANGO_PASS=rootpass ARANGO_PORT=8529
```

- [ ] `go test ./...` passes on the branch before any change of this plan. A pre-existing failure must be recorded in the [Execution notes](#execution-notes), so a later failure is not mistaken for a regression.
- [ ] The branch is cut from `develop`:

```bash
git switch develop && git pull --rebase && git switch -c feat/stock-autocomplete
```

## Consumed protocol contract

This section repeats the full contract that this plan consumes. Treat it as normative; it is not a summary.

Generated Go types in package `github.com/dictyBase/go-genproto/dictybaseapis/stock`:

```go
type StockAutocompleteParameters struct {
	Data *StockAutocompleteParameters_Data
}

type StockAutocompleteParameters_Data struct {
	Type       string
	Attributes *StockAutocompleteAttributes
}

type StockAutocompleteAttributes struct {
	Query  string      // buf.validate string.min_len = 3
	Limit  int64       // buf.validate int64 {gte: 0, lte: 50}
	Entity StockEntity // buf.validate enum.defined_only = true
}

type StockSuggestion struct {
	Id          string
	Entity      StockEntity
	Field       StockSearchField
	DisplayText string
	Score       float64
}

type StockSuggestionCollection struct {
	Data []*StockSuggestion
	Meta *Meta
}
```

Generated server method that this plan implements:

```go
AutocompleteStock(context.Context, *stock.StockAutocompleteParameters) (*stock.StockSuggestionCollection, error)
```

`Meta` already exists in the same package with the fields `NextCursor int64`, `Limit int64` and `Total int64`.

Enum values:

| Enum | Value | Number |
| --- | --- | --- |
| `StockEntity` | `STOCK_ENTITY_UNSPECIFIED` (both kinds) | 0 |
| `StockEntity` | `STOCK_ENTITY_STRAIN` | 1 |
| `StockEntity` | `STOCK_ENTITY_PLASMID` | 2 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_UNSPECIFIED` | 0 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_STOCK_ID` | 1 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_GENES` | 2 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_DBXREFS` | 3 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_LABEL` | 4 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_NAMES` | 5 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_SPECIES` | 6 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_PLASMID` | 7 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_NAME` | 8 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_SUMMARY` | 9, never returned by this RPC |
| `StockSearchField` | reserved number and name `STOCK_SEARCH_FIELD_EDITABLE_SUMMARY` | 10 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_DEPOSITOR` | 11, never returned by this RPC |

Limits and defaults:

| Item | Value |
| --- | --- |
| Minimum query length enforced by protovalidate | 3 characters |
| `limit` accepted range | 0 to 50 |
| Effective limit when `limit` is 0 | 5 |
| Repository clamp | a limit at or below 0 becomes 5; a limit above 50 becomes 50 |
| `Meta.Limit` | the effective limit of the request |
| `Meta.Total` | the number of rows in `Data` |
| `Meta.NextCursor` | always 0 |
| Empty `Data` list | valid, and never an error |

Two gaps in the proto rules that this plan must close in the handler:

1. `string.min_len = 3` counts characters, so the query `"   "` passes validation. The handler must trim the query and reject an empty result with an invalid-argument error.
2. The generated `Validate()` method is a no-op for these messages. The handler must call `protovalidate.Validate`.

## Exact interfaces

### Repository types and method

Add to `internal/repository/repository.go`.

**Shared declaration rule.** The full-search plan declares the same entity filter type and the same three constants, with identical text. Before you add the block, check whether it already exists:

```bash
grep -n 'StockEntityFilter\|EntityStrain' internal/repository/repository.go
```

If the grep finds the block, reuse it and add nothing. If the grep finds nothing, add exactly this:

```go
// StockEntityFilter restricts a stock search to one kind of stock. The
// zero value covers both kinds. The values match the type attribute of
// the edge in the stock property graph.
type StockEntityFilter string

const (
	// EntityBoth covers strains and plasmids.
	EntityBoth StockEntityFilter = ""
	// EntityStrain covers strains only.
	EntityStrain StockEntityFilter = "strain"
	// EntityPlasmid covers plasmids only.
	EntityPlasmid StockEntityFilter = "plasmid"
)
```

Then add the autocomplete types and the interface method. These are owned by this plan only:

```go
// AutocompleteQuery holds the input of an autocomplete search. The
// repository normalizes Query and clamps Limit.
type AutocompleteQuery struct {
	// Query is the partial search text. The caller sends it untrimmed;
	// the repository trims it and lowercases it.
	Query string
	// Entity restricts the search to one kind of stock.
	Entity StockEntityFilter
	// Limit is the maximum number of suggestions. A value at or below 0
	// becomes 5. A value above 50 becomes 50.
	Limit int
}

// Suggestion is one autocomplete match.
type Suggestion struct {
	// ID is the stock_id of the matched stock, for example DBS0236126.
	ID string
	// Field is the field path that matched, for example label or genes.
	Field string
	// DisplayText is the text that the user interface shows.
	DisplayText string
	// Entity is the kind of the matched stock. It is never EntityBoth.
	Entity StockEntityFilter
	// Score ranks the match. A prefix match scores above a fuzzy match.
	Score float64
}
```

Add one method to the `StockRepository` interface, after `RemoveStock(id string) error`:

```go
	AutocompleteStock(params *AutocompleteQuery) ([]*Suggestion, error)
```

The method takes a parameter struct, not ordered scalars, so a later field cannot be passed in the wrong position.

### Search asset names and constants

Add to `internal/repository/arangodb/autocomplete.go`. Every name belongs to this plan alone, so the full-search assets can be absent, present or stale without any effect.

```go
const (
	// autocompleteViewName is the classic arangosearch view of this
	// feature. The AQL templates name it literally, because ArangoDB
	// does not accept a bind parameter for a view name in SEARCH.
	autocompleteViewName = "stock_autocomplete"
	// autocompleteNormAnalyzer lowercases a value and removes accents.
	// It serves the prefix branches.
	autocompleteNormAnalyzer = "stock_autocomplete_norm"
	// autocompleteNgramAnalyzer chains norm and ngram. It serves the
	// fuzzy branches.
	autocompleteNgramAnalyzer = "stock_autocomplete_ngram"
	// defaultAutocompleteLimit is the list length for a non-positive
	// limit.
	defaultAutocompleteLimit = 5
	// maxAutocompleteLimit is the hard cap of the list length.
	maxAutocompleteLimit = 50
	// autocompleteNgramThreshold is the minimum n-gram similarity of a
	// fuzzy match. Task 1 calibration on ArangoDB 3.11 confirmed 0.30:
	// 0.45 misses one-character typos in 4-character values (ys14 vs
	// yS13) and in 10-character identifiers (dbs0236127 vs
	// DBS0236126), while 0.30 still admits no junk.
	autocompleteNgramThreshold = 0.3
)
```

Analyzer definitions, both created with `EnsureCreatedAnalyzer`:

| Analyzer | Type | Properties | Features |
| --- | --- | --- | --- |
| `stock_autocomplete_norm` | `driver.ArangoSearchAnalyzerTypeNorm` | `Locale: "en.utf-8"`, `Case: driver.ArangoSearchCaseLower`, `Accent: new(false)` | none |
| `stock_autocomplete_ngram` | `driver.ArangoSearchAnalyzerTypePipeline` | pipeline step 1 `Norm` with the same three properties; pipeline step 2 `NGram` with `Min: new(int64(2))`, `Max: new(int64(3))`, `PreserveOriginal: new(true)`, `StreamType: new(driver.ArangoSearchNGramStreamUTF8)` | `Frequency`, `Norm`, `Position` |

The three features are mandatory. `NGRAM_MATCH` fails without them. A plain `ngram` analyzer, without the pipeline, returned zero matches on a real server; see `AUTOCOMPLETE.md` section 3 Step 1.

Go 1.26 accepts `new(expr)`, so `new(false)` and `new(int64(2))` need no helper function.

### View definition

One classic `arangosearch` view, `stock_autocomplete`, with 2 links. Build both collection names from the repository struct.

| Link key | Field | Analyzers |
| --- | --- | --- |
| `ar.stockc.stock.Name()` | `stock_id` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |
| `ar.stockc.stock.Name()` | `genes` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |
| `ar.stockc.stock.Name()` | `dbxrefs` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |
| `ar.stockc.stockProp.Name()` | `label` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |
| `ar.stockc.stockProp.Name()` | `names` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |
| `ar.stockc.stockProp.Name()` | `species` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |
| `ar.stockc.stockProp.Name()` | `plasmid` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |
| `ar.stockc.stockProp.Name()` | `name` | `stock_autocomplete_norm`, `stock_autocomplete_ngram` |

**Reconciliation rule.** Do not delete and create a view when it already exists.

1. `ViewExists(ctx, "stock_autocomplete")`. When it is false, call `CreateArangoSearchView` and tolerate `driver.IsConflict(err)`, because a second repository instance can win the race.
2. When it is true, call `View(ctx, name)`, then `ArangoSearchView()`, then `Properties(ctx)`.
3. Compare **only** the shape that this plan controls: the set of link keys, the set of field names per link, and the sorted set of analyzer names per field. Do not compare the whole `ArangoSearchElementProperties` value. The server fills defaults such as `includeAllFields`, `trackListPositions` and `storeValues` into the response. A deep comparison therefore always reports a difference, and it rewrites the view at every start.
4. When the compared shape differs, call `SetProperties(ctx, driver.ArangoSearchViewProperties{Links: desired})`. The driver sends a PUT, which replaces the properties. The index rebuilds in the background.
5. When `ArangoSearchView()` fails because a view of another type holds the name, return an error. Do not delete the other view.

### Field and branch matrix

8 fields, 2 stages per field, 16 branches.

| # | Field | Collection | Branch template | Prefix branch variable | Fuzzy branch variable | Display expression kind |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `stock_id` | stock | stock | `p0` | `n0` | scalar |
| 2 | `genes` | stock | stock | `p1` | `n1` | array |
| 3 | `dbxrefs` | stock | stock | `p2` | `n2` | array |
| 4 | `label` | property | property | `p3` | `n3` | scalar |
| 5 | `names` | property | property | `p4` | `n4` | array |
| 6 | `species` | property | property | `p5` | `n5` | scalar |
| 7 | `plasmid` | property | property | `p6` | `n6` | scalar |
| 8 | `name` | property | property | `p7` | `n7` | scalar |

Match expression and score per stage:

| Stage | `SEARCH` expression | Score expression |
| --- | --- | --- |
| prefix | `ANALYZER(STARTS_WITH(d.<field>, @q), "stock_autocomplete_norm")` | `1000 + BM25(d)` |
| fuzzy | `NGRAM_MATCH(d.<field>, @q, @th, "stock_autocomplete_ngram")` | `BM25(d)` |

`STARTS_WITH` inside `SEARCH` needs the `ANALYZER()` wrapper. Without it the comparison uses the `identity` analyzer and matches nothing. `NGRAM_MATCH` takes the analyzer as its fourth argument and needs no wrapper. `STARTS_WITH` does not normalize its prefix argument, so the caller must trim and lowercase the query.

A prefix match scores at least 1000, and `BM25` of a pure `STARTS_WITH` match can be 0. The deterministic tiebreak on the stock key is therefore required, not optional.

Display expressions:

- scalar field: `NOT_NULL(d.<field>, "")`
- array field: 

```aql
NOT_NULL(
  FIRST(
    FOR item IN NOT_NULL(d.<field>, [])
      FILTER CONTAINS(LOWER(item), @q)
      RETURN item
  ),
  CONCAT_SEPARATOR(", ", NOT_NULL(d.<field>, []))
)
```

`NOT_NULL`, `FIRST`, `CONTAINS`, `LOWER` and `CONCAT_SEPARATOR` are plain AQL functions. They run only on the rows that survive the branch `LIMIT`, never inside the index. A fuzzy array hit that passes `NGRAM_MATCH` but fails `CONTAINS` falls back to the joined list, so the result is never `null`. The inner `NOT_NULL(d.<field>, [])` guards a document whose array attribute is absent.

### AQL branch templates

Put both templates in `internal/repository/arangodb/statement/autocomplete.go` as `const` strings, next to the existing statement files. The builder fills them with `fmt.Sprintf`.

**Stock collection branch.** The row key is the stock key. One OUTBOUND traversal supplies the entity.

```aql
LET %s = (
  FOR d IN stock_autocomplete
    SEARCH %s
    LET ent = FIRST(
      FOR v, e IN 1..1 OUTBOUND d GRAPH @stock_prop_graph
        RETURN e.type
    )
    FILTER ent != null
    FILTER @entity == "" OR @entity == ent
    SORT BM25(d) DESC, d._key ASC
    LIMIT @limit
    RETURN {
      k: d._key,
      id: d.stock_id,
      entity: ent,
      f: %q,
      v: %s,
      s: %s
    }
)
```

The five verbs are: branch variable, `SEARCH` expression, field label, display expression, score expression.

**Property collection branch.** The property `_key` is auto-generated, so the row must resolve its owner with one INBOUND traversal.

```aql
LET %s = (
  FOR d IN stock_autocomplete
    SEARCH %s
    LET own = FIRST(
      FOR v, e IN 1..1 INBOUND d GRAPH @stock_prop_graph
        RETURN { k: v._key, id: v.stock_id, entity: e.type }
    )
    FILTER own != null
    FILTER @entity == "" OR @entity == own.entity
    SORT BM25(d) DESC, own.k ASC
    LIMIT @limit
    RETURN {
      k: own.k,
      id: own.id,
      entity: own.entity,
      f: %q,
      v: %s,
      s: %s
    }
)
```

The entity filter runs **inside** every branch, before the branch `SORT` and `LIMIT`. A design that filters after the merge loses a small result group, because the per-branch cap keeps rows of the wrong entity. A review of modware-order rejected the post-merge form; see `AUTOCOMPLETE.md` section 3 Step 4.

### Merge and tail

One `const` string in `statement/autocomplete.go`, filled with the comma-separated list of the 16 branch variables.

```aql
LET hits = FLATTEN([%s])
LET best = (
  FOR x IN hits
    COLLECT stkey = x.k INTO grp = x
    LET top = FIRST(
      FOR m IN grp
        SORT m.s DESC, m.f ASC
        RETURN m
    )
    RETURN top
)
FOR x IN best
  SORT x.s DESC, x.k ASC
  LIMIT @limit
  RETURN {
    k: x.k,
    id: x.id,
    entity: x.entity,
    f: x.f,
    v: x.v,
    s: x.s
  }
```

The merge variable must not be named `all`, because `all` is a reserved word in AQL; the recorded compile error is in `AUTOCOMPLETE.md` section 4. The collect variable is `stkey` for the same reason.

Tie order is fully determined:

1. The merge picks one row per stock key by `m.s DESC, m.f ASC`. When two branches give the same score for one stock, the field label in ascending string order wins. The order of the 8 field labels is therefore `dbxrefs`, `genes`, `label`, `name`, `names`, `plasmid`, `species`, `stock_id`.
2. The tail sorts by `x.s DESC, x.k ASC`. Two stocks with the same score come back in ascending stock key order.

The row type, in `autocomplete.go`:

```go
// suggestionRow mirrors one object row of the autocomplete projection.
// go-driver v1 cannot decode an array row, so the projection returns one
// object per row.
type suggestionRow struct {
	Key    string  `json:"k"`
	ID     string  `json:"id"`
	Entity string  `json:"entity"`
	Field  string  `json:"f"`
	Value  string  `json:"v"`
	Score  float64 `json:"s"`
}
```

### Bind parameters

| Name | Go type | Value |
| --- | --- | --- |
| `@q` | `string` | the normalized query. Task 1 decision: normalization runs in Go — `strings.ToLower(strings.TrimSpace(params.Query))` plus removal of combining diacritical marks (Unicode category Mn). `TOKENS` in AQL was rejected: a punctuation-only query yields an empty token list, and a null prefix argument would poison the `STARTS_WITH` branch. |
| `@th` | `float64` | `autocompleteNgramThreshold` |
| `@limit` | `int` | the clamped limit |
| `@entity` | `string` | `string(params.Entity)`: `""`, `"strain"` or `"plasmid"` |
| `@stock_prop_graph` | `string` | `ar.stockc.stockPropType.Name()` |

The view name is **not** a bind parameter. It is written into the template from `autocompleteViewName`.

`@limit` serves both the per-branch cap and the tail cap. A branch can therefore return up to `@limit` rows. The 16 branches together can produce up to `16 * @limit` rows before the merge. With the cap of 50, that is 800 rows at most. The merge reduces them in memory.

### Handler signature and mapping

Add to `internal/app/service/autocomplete.go`:

```go
func (s *StockService) AutocompleteStock(
	ctx context.Context,
	r *stock.StockAutocompleteParameters,
) (*stock.StockSuggestionCollection, error)
```

Order of operations, exactly:

1. **Nil guard.** `if r.GetData() == nil || r.GetData().GetAttributes() == nil { return nil, aphgrpc.HandleInvalidParamError(ctx, errors.New("autocomplete request needs data and attributes")) }`. This runs **before** validation, because this repository has no recovery interceptor.
2. **Validation.** `if err := protovalidate.Validate(r); err != nil { return nil, aphgrpc.HandleInvalidParamError(ctx, err) }`. Do not call `r.Validate()`; it is a no-op.
3. **Trim guard.** Trim the query. When the trimmed query is empty, return an invalid-argument error. The proto rule counts characters, so `"   "` reaches this point.
4. **Effective limit.** `effLimit := attr.GetLimit()`; when it is 0, set it to 5. The repository clamps again, so the two values always agree.
5. **Entity mapping.** `STOCK_ENTITY_UNSPECIFIED` to `repository.EntityBoth`, `STOCK_ENTITY_STRAIN` to `repository.EntityStrain`, `STOCK_ENTITY_PLASMID` to `repository.EntityPlasmid`. A value outside the three is unreachable after step 2, because of `enum.defined_only`; map it to an invalid-argument error anyway, so a future proto change cannot open a silent hole.
6. **Repository call.** `s.repo.AutocompleteStock(&repository.AutocompleteQuery{Query: trimmed, Entity: ent, Limit: int(effLimit)})`. A non-nil error maps to `aphgrpc.HandleGetError(ctx, err)`, like the other read handlers. An empty result is **not** a not-found error.
7. **Response.** Map each `*repository.Suggestion` to a `*stock.StockSuggestion`. Use fp-go composition in the style of `plasmid_mappers.go`. Set `Meta: &stock.Meta{NextCursor: 0, Limit: effLimit, Total: int64(len(rows))}`.

Field mapping, repository string to proto enum:

| `Suggestion.Field` | `stock.StockSearchField` |
| --- | --- |
| `stock_id` | `StockSearchField_STOCK_SEARCH_FIELD_STOCK_ID` |
| `genes` | `StockSearchField_STOCK_SEARCH_FIELD_GENES` |
| `dbxrefs` | `StockSearchField_STOCK_SEARCH_FIELD_DBXREFS` |
| `label` | `StockSearchField_STOCK_SEARCH_FIELD_LABEL` |
| `names` | `StockSearchField_STOCK_SEARCH_FIELD_NAMES` |
| `species` | `StockSearchField_STOCK_SEARCH_FIELD_SPECIES` |
| `plasmid` | `StockSearchField_STOCK_SEARCH_FIELD_PLASMID` |
| `name` | `StockSearchField_STOCK_SEARCH_FIELD_NAME` |
| anything else | `StockSearchField_STOCK_SEARCH_FIELD_UNSPECIFIED` |

Entity mapping, repository string to proto enum:

| `Suggestion.Entity` | `stock.StockEntity` |
| --- | --- |
| `strain` | `StockEntity_STOCK_ENTITY_STRAIN` |
| `plasmid` | `StockEntity_STOCK_ENTITY_PLASMID` |
| anything else | `StockEntity_STOCK_ENTITY_UNSPECIFIED` |

## File map

| File | Action | Content |
| --- | --- | --- |
| `scripts/autocomplete-calibration.js` | create | arangosh calibration harness: disposable database, fixtures, the 2 analyzers, the view, a threshold sweep, the probe matrix, EXPLAIN and latency output |
| `internal/repository/repository.go` | modify | `StockEntityFilter` and its 3 constants (guarded, see the shared declaration rule), `AutocompleteQuery`, `Suggestion`, and the interface method `AutocompleteStock` |
| `internal/repository/arangodb/autocomplete.go` | create | constants, analyzer setup, view creation and reconciliation, query builder, `suggestionRow`, row conversion, `AutocompleteStock` |
| `internal/repository/arangodb/statement/autocomplete.go` | create | the 3 AQL `const` templates: stock branch, property branch, merge and tail |
| `internal/repository/arangodb/database.go` | modify | call `ar.ensureAutocompleteSearch(context.Background())` at the end of the `createDbStruct` chain |
| `internal/repository/arangodb/autocomplete_test.go` | create | repository tests against a real disposable database, plus builder unit tests |
| `internal/app/service/autocomplete.go` | create | the handler and the two mapping functions |
| `internal/app/service/autocomplete_test.go` | create | handler unit tests against a stub repository |
| `internal/app/service/autocomplete_arango_test.go` | create | one buffer-connection round trip against a real database, with the safe bufconn pattern |
| `README.md` | modify | the RPC contract in Simplified Technical English |
| `docs/superpowers/plans/2026-10-06-stock-autocomplete.md` | modify | fill the [Execution notes](#execution-notes) |

Note on `database.go`: the full-search plan adds its own call, `ar.ensureFullSearch(context.Background())`, to the same chain. Keep both calls. Do not replace the other call.

## Implementation tasks

### Task 1: Calibration against real ArangoDB 3.11

Run this task before any Go or AQL code is written. Every later task depends on its recorded output.

**Files:** create `scripts/autocomplete-calibration.js`.

**Produces:** these 7 recorded results.

1. The confirmed or corrected n-gram threshold.
2. The query normalization decision.
3. The proof of both traversal directions.
4. The behavior of the array display expression.
5. The behavior of an absent array attribute.
6. The behavior for a punctuation-only query and a whitespace-only query.
7. The EXPLAIN node list and the latency baseline of the final 16-branch statement.

- [ ] **Step 1: Start the server.** The version must be 3.11, the same series as CI:

```bash
docker run -d --name arango311 -p 8529:8529 -e ARANGO_ROOT_PASSWORD=rootpass arangodb:3.11
docker exec arango311 arangosh --server.password rootpass --javascript.execute-string 'print(db._version())'
```

Record the exact version string in the [Execution notes](#execution-notes).

- [ ] **Step 2: Write `scripts/autocomplete-calibration.js`.** Model it on the modware-order harness described in `AUTOCOMPLETE.md` section 3 Step 1. The script must:
  - create a disposable database with a random name;
  - create the collections `stock_calib` and `stock_properties_calib`, and the edge collection `stock_type_calib`;
  - create the named graph `stock_prop_calib`. Give it one edge definition, from `stock_calib` to `stock_properties_calib`. This definition mirrors `createNamedGraph` in `database.go`;
  - insert fixtures with the same shape as `statement/insert.go`: a stock document that sets `_key` equal to `stock_id`, a property document with an auto-generated `_key`, and an edge that carries `type`;
  - insert one strain with `stock_id` `DBS0236126`, `genes: ["sadA","DDB_G0348394"]`, `dbxrefs: ["d0319"]`, property `label: "yS13"`, `names: ["Ax2","gammaS13"]`, `species: "Dictyostelium discoideum"`, `plasmid: "pDM304"`;
  - insert one plasmid with `stock_id` `DBP0000027`, `genes: ["sadA"]`, property `name: "pDM304"`;
  - insert one strain whose property document has **no** `names` attribute, to probe the absent-array fallback;
  - insert one property document with **no** inbound edge, to probe the missing-owner filter;
  - insert 60 strains and 5 plasmids that all carry a gene with the prefix `xylose`, for the entity-filter probe;
  - insert about 1000 further stocks with random identifiers, so the latency measurement is not taken on an empty index;
  - create the analyzers `stock_autocomplete_norm` and `stock_autocomplete_ngram` with the exact definitions of [Search asset names and constants](#search-asset-names-and-constants);
  - create the view `stock_autocomplete` with the exact links of [View definition](#view-definition);
  - poll until the view answers a known query, because arangosh in `--javascript.execute` mode has no `wait()` helper.

- [ ] **Step 3: Run the probe matrix and record every result.** One small AQL statement per probe.

| # | Probe | Expected result |
| --- | --- | --- |
| 1 | prefix `dbs023` on `stock_id`, stock branch | one row, score above 1000, `k` equal to `DBS0236126` |
| 2 | prefix `ys` on `label`, property branch | one row whose `k` is the **stock** key `DBS0236126`, not the property key; this proves the INBOUND direction |
| 3 | prefix `pdm` on the plasmid `name` | one row whose `k` is `DBP0000027` and whose `entity` is `plasmid` |
| 4 | the same prefix without the `ANALYZER()` wrapper | zero rows; this pins the wrapper requirement |
| 5 | fuzzy `dbs0236127` at thresholds 0.2, 0.3, 0.45, 0.55, 0.65, 0.8, 1.0 | record the match at each threshold; 0.45 must match and 0.2 must admit junk that outranks the real row |
| 6 | fuzzy `sdaa` on `genes` | one row for the gene `sadA` at 0.45 |
| 7 | prefix `dictyo` on `species` | one row |
| 8 | prefix `d031` on `dbxrefs` | one row |
| 9 | uppercase query `DBS023`, lowercased in the client | one row |
| 10 | accented query `Áx2` on `names`, once as `strings.ToLower` output and once as `FIRST(TOKENS(@q, "stock_autocomplete_norm"))` | record which form finds `Ax2`; this decides the normalization step of `AutocompleteStock` |
| 11 | the array display expression on `names` for the prefix query `ax` | returns the element `Ax2`, not the joined list |
| 12 | the array display expression for a fuzzy-only hit | falls back to the joined list, and is never `null` |
| 13 | the array display expression on the property document without `names` | returns an empty string, and raises no AQL error |
| 14 | the property document with no inbound edge | dropped by `FILTER own != null` |
| 15 | entity filter inside the `genes` branch: query `xylose`, `@entity` `plasmid`, `@limit` 10, over the 60 strains and 5 plasmids | 5 plasmid rows. Run the same probe with the filter moved after the merge and record the smaller count; this measures the regression that the in-branch filter prevents. |
| 16 | one stock that matches `label` in a property branch and `genes` in a stock branch | one merged row, with the field attribution of the higher score |
| 17 | whitespace-only query `"   "` | zero rows, and no AQL error |
| 18 | punctuation-only query `"---"` | zero rows, and no AQL error |
| 19 | 1-character and 2-character queries | record the row count; the handler rejects them, so this probe only proves that no error is raised |
| 20 | `FLATTEN([...])` over the 16 branch variables | one flat list, not a list of lists |

- [ ] **Step 4: EXPLAIN the final statement.** Assemble the full 16-branch statement by hand in the script and run `db._explain(stmt, bindVars)`. Assert both counts:

```text
number of EnumerateViewNode entries  == 16
number of EnumerateCollectionNode entries == 0
```

A collection scan node means that a branch lost its view index. Record the node list.

- [ ] **Step 5: Measure the latency baseline.** Run the full statement 20 times against the warmed fixture of about 1000 documents, with the query `dbs023` and `@limit` 5. Record the minimum, the median and the p95 in milliseconds. This p95 is the baseline that Task 6 compares against.

- [ ] **Step 6: Fold every correction into this plan.** Update [Search asset names and constants](#search-asset-names-and-constants), [AQL branch templates](#aql-branch-templates) and [Bind parameters](#bind-parameters) when a probe contradicts them. Record each change in the [Execution notes](#execution-notes). Do not start Task 2 with an open contradiction.

Run command:

```bash
docker exec -i arango311 arangosh \
  --server.endpoint tcp://localhost:8529 \
  --server.username root --server.password rootpass \
  --javascript.execute /dev/stdin < scripts/autocomplete-calibration.js
```

- [ ] **Step 7: Commit.**

```bash
git add scripts/autocomplete-calibration.js
git commit -m "feat: add arangosearch calibration harness for stock autocomplete"
```

### Task 2: Repository types and interface method

**Files:** modify `internal/repository/repository.go`.

- [ ] **Step 1: Apply the shared declaration rule** from [Repository types and method](#repository-types-and-method): grep for `StockEntityFilter`, and add the type and the 3 constants only when they are absent.

- [ ] **Step 2: Add `AutocompleteQuery` and `Suggestion`** with the exact field names, types, order and doc comments of [Repository types and method](#repository-types-and-method). The field order places the three strings before the `float64`, which satisfies the field-alignment check.

- [ ] **Step 3: Add the interface method** `AutocompleteStock(params *AutocompleteQuery) ([]*Suggestion, error)` to `StockRepository`.

- [ ] **Step 4: Prove that the build breaks where it must.**

```bash
go build ./... 2>&1 | head
```

Expected: `*arangorepository` no longer satisfies `repository.StockRepository`. That failure is the signal that Task 4 must close. Record it.

### Task 3: Search assets in the constructor

**Files:** create `internal/repository/arangodb/autocomplete.go`; modify `internal/repository/arangodb/database.go`.

- [ ] **Step 1: Write the failing tests** in `internal/repository/arangodb/autocomplete_test.go`. Write flat sequential functions. Do not use a `t.Parallel` subtest. A parallel subtest resumes after the parent deferred functions run, and it then queries a dropped database.
  - `TestEnsureAutocompleteSearchCreatesAssets`: build the repository with `setUp(t)`, then assert through `repo.Dbh().Handler()` that both analyzers exist and that `ViewExists` reports the view. Read the view properties and assert the 2 link keys, the 8 field names and the 2 analyzers per field.
  - `TestEnsureAutocompleteSearchReconcilesStaleView`: create a disposable database with `testarango.NewTestArangoFromEnv(true)`. Create the view `stock_autocomplete` **before** the repository, with one wrong link and one wrong analyzer list. Call `NewStockRepo`. Read the properties, and assert the correct shape. Assert also that the view identifier did not change. An unchanged identifier proves that the code used `SetProperties` and not a delete.
  - `TestNewStockRepoFailsWhenSetupFails`: point `manager.ConnectParams` at a database name that does not exist. Assert that `NewStockRepo` returns an error. Create that database with the arangomanager session. Assert that a second `NewStockRepo` call succeeds. This test pins the eager, retryable constructor contract. A pod restart is the retry.

- [ ] **Step 2: Run the tests and make sure that they fail.**

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'TestEnsureAutocompleteSearch|TestNewStockRepoFailsWhenSetupFails' ./internal/repository/arangodb/
```

Expected: FAIL with an undefined symbol.

- [ ] **Step 3: Write the setup code** in `autocomplete.go`:
  - the constant block of [Search asset names and constants](#search-asset-names-and-constants);
  - `func (ar *arangorepository) ensureAutocompleteSearch(ctx context.Context) error`, which calls `EnsureCreatedAnalyzer` twice and then `ensureAutocompleteView(ctx)`;
  - `func (ar *arangorepository) autocompleteLinks() driver.ArangoSearchLinks`, which builds the links from `ar.stockc.stock.Name()` and `ar.stockc.stockProp.Name()`;
  - `func (ar *arangorepository) ensureAutocompleteView(ctx context.Context) error`, which follows the 5 numbered rules of [View definition](#view-definition);
  - `func sameLinkShape(got, want driver.ArangoSearchLinks) bool`, which compares only the link keys, the field names and the sorted analyzer names. Wrap every error with `errors.Errorf` from `github.com/cockroachdb/errors`, as the rest of the package does.

- [ ] **Step 4: Wire the constructor.** In `database.go`, change `createDbStruct` so that the chain ends with the search setup:

```go
func createDbStruct(ar *arangorepository, collP *CollectionParams) error {
	if err := docCollections(ar, collP); err != nil {
		return err
	}
	if err := graphAndEdgeCollections(ar, collP); err != nil {
		return err
	}
	if err := createIndex(ar); err != nil {
		return err
	}
	return ar.ensureAutocompleteSearch(context.Background())
}
```

The setup is eager and idempotent. A failure stops the start of the service, so an operator sees the problem at once. Do not use `sync.Once`, which consumes its single run even on an error, and do not lock the request path.

- [ ] **Step 5: Make the tests pass**, then run the gates of [Quality gates](#quality-gates).

- [ ] **Step 6: Commit.**

```bash
git add internal/repository/repository.go internal/repository/arangodb/autocomplete.go internal/repository/arangodb/database.go internal/repository/arangodb/autocomplete_test.go
git commit -m "feat: create arangosearch assets for stock autocomplete in the constructor"
```

### Task 4: Query builder and AutocompleteStock

**Files:** create `internal/repository/arangodb/statement/autocomplete.go`; modify `internal/repository/arangodb/autocomplete.go` and `autocomplete_test.go`.

- [ ] **Step 1: Write the failing tests.** Add every row of the [Test matrix](#test-matrix) that belongs to the repository layer. Two groups:

  **Builder unit tests, no database.**
  - `TestBuildAutocompleteQueryBranchCount`: the built statement holds 16 branch variables and 16 occurrences of `FOR d IN stock_autocomplete`. It holds 6 `OUTBOUND` traversals, one per stage of the 3 stock fields. It holds 10 `INBOUND` traversals, one per stage of the 5 property fields.
  - `TestBuildAutocompleteQueryIsValidAQL`: run `repo.Dbh().Handler().ValidateQuery(ctx, built)` and require no error. This is a cheap syntax gate that needs a connection but no fixture.
  - `TestBuildAutocompleteQueryFilterOrder`: in every branch, the index of `FILTER @entity` is lower than the index of `SORT BM25` and of `LIMIT @limit`.

  **Repository tests against a real database.** Each one calls `setUp(t)`, inserts fixtures with `repo.AddStrain` or the `F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)` form, waits for the view commit with `require.Eventually`, then calls `repo.AutocompleteStock`.
  - `TestAutocompleteStockLabelPrefix`
  - `TestAutocompleteStockPlasmidNamePrefix`
  - `TestAutocompleteStockGenePrefix`
  - `TestAutocompleteStockDbxrefPrefix`
  - `TestAutocompleteStockStockIDPrefix`
  - `TestAutocompleteStockSpeciesPrefix`
  - `TestAutocompleteStockStrainPlasmidPrefix`
  - `TestAutocompleteStockNamesPrefix`
  - `TestAutocompleteStockTypoFuzzy`
  - `TestAutocompleteStockEntityFilterKeepsSmallGroup`
  - `TestAutocompleteStockCrossCollectionMerge`
  - `TestAutocompleteStockDefaultLimitIsFive`
  - `TestAutocompleteStockLimitCapIsFifty`
  - `TestAutocompleteStockRankingUnderSmallLimit`
  - `TestAutocompleteStockEmptyResultIsNotAnError`
  - `TestAutocompleteStockArrayDisplayFallback`
  - `TestAutocompleteStockRejectsWhitespaceQuery`
  - `TestAutocompleteStockRejectsUnknownEntity`
  - `TestAutocompleteStockSkipsStockWithoutEdge`

  Wait helper, used by every database test:

```go
	assert.Eventually(func() bool {
		rows, err := repo.AutocompleteStock(&repository.AutocompleteQuery{Query: probe, Limit: 5})
		return err == nil && len(rows) > 0
	}, 20*time.Second, 500*time.Millisecond, "the view must commit the new documents")
```

The view commits in the background, about one second after an insert. A test that queries at once fails without this poll.

- [ ] **Step 2: Run the tests and make sure that they fail.**

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'TestAutocompleteStock|TestBuildAutocompleteQuery' ./internal/repository/arangodb/
```

Expected: FAIL with an undefined method.

- [ ] **Step 3: Write the templates** in `statement/autocomplete.go`. Three exported `const` strings, with a doc comment that lists the `fmt.Sprintf` verbs and the required bind parameters:
  - `AutocompleteStockBranch` — the stock collection branch of [AQL branch templates](#aql-branch-templates);
  - `AutocompletePropBranch` — the property collection branch;
  - `AutocompleteMerge` — the merge and tail of [Merge and tail](#merge-and-tail).

- [ ] **Step 4: Write the builder and the method** in `autocomplete.go`:
  - a package-level table of the 8 fields, in the order of [Field and branch matrix](#field-and-branch-matrix). Each entry holds the field label, the branch family and the display-expression kind;
  - `func buildAutocompleteQuery() string`. It loops over the table. It emits a prefix branch and a fuzzy branch per field. It collects the branch variable names, and it appends the merge template. Assign the result to a package-level `var autocompleteQuery = buildAutocompleteQuery()`, so the statement is built once per process;
  - `func (ar *arangorepository) AutocompleteStock(params *repository.AutocompleteQuery) ([]*repository.Suggestion, error)`:
    1. reject a nil `params` with an error;
    2. normalize the query per the Task 1 decision; the baseline is `strings.ToLower(strings.TrimSpace(params.Query))`;
    3. return an error when the normalized query is empty;
    4. clamp the limit: at or below 0 becomes `defaultAutocompleteLimit`, above `maxAutocompleteLimit` becomes `maxAutocompleteLimit`;
    5. accept only `repository.EntityBoth`, `repository.EntityStrain` and `repository.EntityPlasmid`; any other value returns an error;
    6. call `ar.database.SearchRows(autocompleteQuery, bindVars)` with the 5 bind parameters of [Bind parameters](#bind-parameters);
    7. `defer result.Close()`;
    8. loop `for result.Scan()`, read one `suggestionRow` at a time, and append;
    9. convert the rows to `[]*repository.Suggestion` and return. Return an empty, non-nil slice when there is no row.
  - Use fp-go style for the conversion pipeline, as the rest of this repository does. Load the `fp-go`, `fp-go-pipe-flow` and `fp-go-predicates` skills before you write it. Keep the alias `IOE` for `ioeither`.

- [ ] **Step 5: Make the tests pass**, then run the gates of [Quality gates](#quality-gates).

- [ ] **Step 6: Commit.**

```bash
git add internal/repository/arangodb/autocomplete.go internal/repository/arangodb/statement/autocomplete.go internal/repository/arangodb/autocomplete_test.go
git commit -m "feat: add autocomplete stock search to the arangodb repository"
```

### Task 5: Service handler

**Files:** create `internal/app/service/autocomplete.go`, `autocomplete_test.go` and `autocomplete_arango_test.go`.

- [ ] **Step 1: Write the failing tests.**

  **Unit tests with a stub repository, in `autocomplete_test.go`.** The stub must implement every method of `repository.StockRepository` that exists on the branch. The current set is `GetStrain`, `GetPlasmid`, `AddStrain`, `AddPlasmid`, `EditStrain`, `EditPlasmid`, `ListStrains`, `ListStrainsByIDs`, `ListPlasmids`, `LoadStrain`, `LoadPlasmid`, `RemoveStock`, `Dbh`, `LoadOboJSON`, plus `AutocompleteStock` from this plan. If the full-search plan already landed, `SearchStock` is also required. Discover the real set with `go build ./...` and add the missing stubs. Name the type `autocompleteStubRepo`, so it cannot collide with the stub of the full-search plan. Give it the fields `sugs []*repository.Suggestion`, `err error`, and the captured input `got *repository.AutocompleteQuery`.
  - `TestAutocompleteStockHandlerMapsSuggestions`: two suggestions map to two `StockSuggestion` values, with the right enum field, the right enum entity, the display text and the score. `Meta.Total` is 2, `Meta.Limit` is the effective limit, `Meta.NextCursor` is 0.
  - `TestAutocompleteStockHandlerPassesThroughQueryAndLimit`: the captured `got.Query` is the trimmed query, `got.Limit` is the request limit, and `got.Entity` is the mapped filter.
  - `TestAutocompleteStockHandlerDefaultsLimitToFive`: a request with `Limit` 0 gives `got.Limit == 5` and `Meta.Limit == 5`.
  - `TestAutocompleteStockHandlerRejectsNilData`: `&stock.StockAutocompleteParameters{}` gives `codes.InvalidArgument` and no panic.
  - `TestAutocompleteStockHandlerRejectsNilAttributes`: `Data` present and `Attributes` nil gives `codes.InvalidArgument` and no panic.
  - `TestAutocompleteStockHandlerRejectsShortQuery`: a 2-character query gives `codes.InvalidArgument`.
  - `TestAutocompleteStockHandlerRejectsWhitespaceQuery`: the query `"   "` gives `codes.InvalidArgument`. This closes the proto gap.
  - `TestAutocompleteStockHandlerRejectsLimitAboveCap`: `Limit` 51 gives `codes.InvalidArgument`.
  - `TestAutocompleteStockHandlerRejectsUndefinedEntity`: `stock.StockEntity(7)` gives `codes.InvalidArgument`.
  - `TestAutocompleteStockHandlerReturnsEmptyCollection`: an empty repository result gives no error, an empty `Data` list and `Meta.Total` 0.
  - `TestAutocompleteStockHandlerMapsRepositoryError`: a repository error gives `codes.Internal`, which is what `aphgrpc.HandleGetError` produces for the other read handlers. Assert the code that the helper really returns, and record it.

  **One round trip, in `autocomplete_arango_test.go`.** Use the safe bufconn pattern. Do **not** reuse `setupGrpcServer` or `setupGrpcClient` from `strain_test_helpers.go`; both are unsafe, as recorded in [Service and test plumbing](#service-and-test-plumbing).

```go
func newAutocompleteArangoEnv(t *testing.T) (stock.StockServiceClient, *require.Assertions) {
	t.Helper()
	assert := require.New(t)
	tra, err := testarango.NewTestArangoFromEnv(true)
	assert.NoErrorf(err, "expect no error creating a test database, received %s", err)
	repo, err := arangodb.NewStockRepo(
		getConnectParamsFromDb(tra),
		getCollectionParams(),
		getOntoParams(),
	)
	assert.NoErrorf(err, "expect no error building the repository, received %s", err)
	assert.NoError(loadData(tra), "expect no error loading the ontology")
	svc := setupTestService(repo)
	srv := grpc.NewServer()
	stock.RegisterStockServiceServer(srv, svc)
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		// Serve returns grpc.ErrServerStopped during cleanup. The
		// goroutine can outlive the test, so it must not log and must
		// not exit the process.
		_ = srv.Serve(lis)
	}()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
	)
	assert.NoErrorf(err, "expect no error creating a grpc client, received %s", err)
	t.Cleanup(func() {
		_ = conn.Close()
		_ = lis.Close()
		srv.Stop()
		_ = repo.Dbh().Drop()
	})

	return stock.NewStockServiceClient(conn), assert
}
```

The target `"passthrough:///bufnet"` carries its own scheme, so no call to `resolver.SetDefaultScheme` is needed. A global resolver change races with every other test in the package. The helpers `getConnectParamsFromDb`, `getCollectionParams`, `getOntoParams`, `loadData` and `setupTestService`, and the no-op `MockPublisher`, already exist in `strain_test_helpers.go` of the same package, so reuse them.
  - `TestAutocompleteStockEndToEnd`: create a strain through `client.CreateStrain`. Poll with `require.Eventually` until `client.AutocompleteStock` returns a row. Then assert the identifier, the entity enum, the field enum, the display text and `Meta`.

- [ ] **Step 2: Run the tests and make sure that they fail.**

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'AutocompleteStock' ./internal/app/service/
```

Expected: FAIL, because `StockService` has no such method. The embedded `stock.UnimplementedStockServiceServer` makes the gRPC call return `codes.Unimplemented` instead of a compile error, so assert on the method result, not on compilation.

- [ ] **Step 3: Write the handler** in `internal/app/service/autocomplete.go`, in the exact order of [Handler signature and mapping](#handler-signature-and-mapping). Put the two mapping functions in the same file, and write them as pure functions, so the unit tests can cover them directly.

- [ ] **Step 4: Make the tests pass**, then run the gates of [Quality gates](#quality-gates).

- [ ] **Step 5: Commit.**

```bash
git add internal/app/service/autocomplete.go internal/app/service/autocomplete_test.go internal/app/service/autocomplete_arango_test.go
git commit -m "feat: add the autocomplete stock grpc handler"
```

### Task 6: EXPLAIN and latency gate

- [ ] **Step 1: Print the statement that the Go builder produced.** Add a short temporary program or a test with the `-v` flag that writes `autocompleteQuery` to standard output, and save it to `/tmp/autocomplete.aql`. The statement under test must be the built one, not the hand-written one of Task 1.

- [ ] **Step 2: EXPLAIN it** against the Task 1 fixture database:

```bash
docker exec -i arango311 arangosh --server.username root --server.password rootpass \
  --server.database <calib-db> --javascript.execute-string \
  'var q=require("fs").read("/tmp/autocomplete.aql"); print(JSON.stringify(db._explain(q, {q:"dbs023", th:0.45, limit:5, entity:"", stock_prop_graph:"stock_prop_calib"}), null, 2));'
```

Assert 16 `EnumerateViewNode` entries and 0 `EnumerateCollectionNode` entries. Record the node list.

- [ ] **Step 3: Measure the latency** of the same statement, on the same warmed fixture, with the same query and limit, over 20 runs. The gate is: **p95 at or below 125 percent of the Task 1 baseline p95**. Record both numbers.

- [ ] **Step 4: When the gate fails**, do not raise the threshold of the gate. Find the cause: a lost view index on a branch, a traversal that runs before the branch `LIMIT`, or a display expression that was pushed into the index. Record the cause and the fix.

### Task 7: Documentation

**Files:** modify `README.md`.

- [ ] **Step 1: Describe the RPC** in Simplified Technical English. Cover these 8 items:
  1. the method name;
  2. the request message and the response message;
  3. the floor of 3 characters on the query;
  4. the default of 5 and the cap of 50;
  5. the 8 searched fields;
  6. the meaning of the entity filter, and the meaning of the score;
  7. the rule that an empty list is valid;
  8. the 4 environment variables that the tests read: `ARANGO_HOST`, `ARANGO_USER`, `ARANGO_PASS`, and the optional `ARANGO_PORT`.

- [ ] **Step 2: Name the search assets**, so an operator can find them. The names are the analyzers `stock_autocomplete_norm` and `stock_autocomplete_ngram`, and the view `stock_autocomplete`. State that the repository constructor creates them. State also that a failed creation stops the start of the service.

- [ ] **Step 3: Commit.**

```bash
git add README.md
git commit -m "docs: describe the stock autocomplete rpc"
```

## Test matrix

### Repository tests, real disposable ArangoDB 3.11

| # | Test | Fixture | Assertion |
| --- | --- | --- | --- |
| 1 | `TestAutocompleteStockLabelPrefix` | 1 strain with `label` `yS13` | query `ys1` returns 1 row; `Field` is `label`; `ID` is the stock identifier, not the property key; `Entity` is `strain`; `Score` is at least 1000 |
| 2 | `TestAutocompleteStockPlasmidNamePrefix` | 1 plasmid with `name` `p123456` | query `p123` returns 1 row; `Field` is `name`; `Entity` is `plasmid` |
| 3 | `TestAutocompleteStockGenePrefix` | 1 strain with `genes` `DDB_G0348394` | query `ddb_g03` returns 1 row; `Field` is `genes`; `DisplayText` is the matched element |
| 4 | `TestAutocompleteStockDbxrefPrefix` | the `newTestStrain` dbxrefs | query `d031` returns 1 row; `Field` is `dbxrefs` |
| 5 | `TestAutocompleteStockStockIDPrefix` | 1 strain | query with the first 6 characters of the returned `stock_id` returns that stock; `Field` is `stock_id` |
| 6 | `TestAutocompleteStockSpeciesPrefix` | 1 strain | query `dictyo` returns at least 1 row; `Field` is `species` |
| 7 | `TestAutocompleteStockStrainPlasmidPrefix` | 1 strain with `plasmid` `DBP0000027` | query `dbp00` returns the strain; `Field` is `plasmid` |
| 8 | `TestAutocompleteStockNamesPrefix` | 1 strain with `names` `gammaS13` | query `gammas` returns 1 row; `Field` is `names` |
| 9 | `TestAutocompleteStockTypoFuzzy` | 1 strain with `label` `yS13` | query `ys14`, one edit away, returns the row at the threshold 0.3; `Score` is below 1000, which proves the fuzzy stage |
| 10 | `TestAutocompleteStockEntityFilterKeepsSmallGroup` | 60 strains and 5 plasmids, each with a gene that starts with `xylose` | `Entity` `plasmid` and `Limit` 10 return exactly the 5 plasmids. This fails when the entity filter runs after the merge. |
| 11 | `TestAutocompleteStockCrossCollectionMerge` | 1 strain whose `label` and whose `genes` both match one query | exactly 1 row, with the field of the higher score, the stock identifier and the entity `strain` |
| 12 | `TestAutocompleteStockDefaultLimitIsFive` | 8 strains that match | `Limit` 0 returns 5 rows |
| 13 | `TestAutocompleteStockLimitCapIsFifty` | 60 strains that match | `Limit` 500 returns 50 rows |
| 14 | `TestAutocompleteStockRankingUnderSmallLimit` | 7 strains that match the fuzzy stage and 1 that matches the prefix stage | `Limit` 3 returns the prefix match first. This pins the per-branch `SORT` before `LIMIT`. |
| 15 | `TestAutocompleteStockEmptyResultIsNotAnError` | 1 strain | query `zzzqqq` returns an empty, non-nil slice and a nil error |
| 16 | `TestAutocompleteStockArrayDisplayFallback` | 1 strain whose `names` match only the fuzzy stage | `DisplayText` is the joined list and is never empty or `null` |
| 17 | `TestAutocompleteStockRejectsWhitespaceQuery` | none | the query `"   "` returns an error, and no AQL runs |
| 18 | `TestAutocompleteStockRejectsUnknownEntity` | none | `Entity` `"vector"` returns an error |
| 19 | `TestAutocompleteStockSkipsStockWithoutEdge` | 1 property document inserted directly with no edge | a query that matches it returns no row and no error |
| 20 | `TestEnsureAutocompleteSearchCreatesAssets` | a fresh repository | both analyzers exist; the view exists with 2 links, 8 fields and 2 analyzers per field |
| 21 | `TestEnsureAutocompleteSearchReconcilesStaleView` | a view pre-created with wrong links | after `NewStockRepo` the link shape matches the definition, and the view was not deleted |
| 22 | `TestNewStockRepoFailsWhenSetupFails` | a missing database | the first call fails; after the database is created the second call succeeds |

### Builder unit tests, no fixture

| # | Test | Assertion |
| --- | --- | --- |
| 23 | `TestBuildAutocompleteQueryBranchCount` | 16 branch variables; 16 `FOR d IN stock_autocomplete`; 6 `OUTBOUND`; 10 `INBOUND`; 8 `ANALYZER(STARTS_WITH`; 8 `NGRAM_MATCH` |
| 24 | `TestBuildAutocompleteQueryIsValidAQL` | `ValidateQuery` returns no error for the built statement |
| 25 | `TestBuildAutocompleteQueryFilterOrder` | in each branch the `FILTER @entity` index is lower than the `SORT BM25` index and the `LIMIT @limit` index |
| 26 | `TestAutocompleteFieldMapping` | every one of the 8 field labels maps to a defined `StockSearchField`, and an unknown label maps to `UNSPECIFIED` |

### Handler tests, stub repository

Rows 27 to 37 are the 11 unit tests named in Task 5, Step 1.

### Handler test, one real round trip

| # | Test | Assertion |
| --- | --- | --- |
| 38 | `TestAutocompleteStockEndToEnd` | a strain created over the buffer connection is found by `client.AutocompleteStock`; the response carries the identifier, the entity enum, the field enum, the display text, a score of at least 1000, `Meta.Total` 1, `Meta.Limit` 5 and `Meta.NextCursor` 0 |

## Failure behavior

| Input or condition | Layer | Behavior |
| --- | --- | --- |
| `data` is nil | handler | `codes.InvalidArgument`, no panic, no repository call |
| `attributes` is nil | handler | `codes.InvalidArgument`, no panic, no repository call |
| query shorter than 3 characters | handler, protovalidate | `codes.InvalidArgument` |
| query of 3 or more characters that trims to empty, for example `"   "` | handler, trim guard | `codes.InvalidArgument`. The proto rule accepts it, so the handler must reject it. |
| query that normalizes to only punctuation | repository and ArangoDB | an empty list and a nil error. The prefix stage finds nothing, and the fuzzy stage finds nothing above the threshold. |
| `limit` below 0 or above 50 | handler, protovalidate | `codes.InvalidArgument` |
| `limit` 0 | handler and repository | the effective limit is 5 |
| `entity` outside 0, 1, 2 | handler, protovalidate and the explicit map default | `codes.InvalidArgument` |
| `Entity` outside the 3 repository constants | repository | an error, before any AQL runs |
| no stock matches | repository and handler | an empty, non-nil `Data` list, `Meta.Total` 0, and a nil error. This is never a not-found error. |
| a property document with no inbound edge | AQL | dropped by `FILTER own != null` |
| a stock document with no outbound edge | AQL | dropped by `FILTER ent != null`. Such a stock cannot be classified, so it cannot be filtered or labeled. |
| an array attribute is absent | AQL | the display expression gives an empty string through `NOT_NULL(d.<field>, [])` and `CONCAT_SEPARATOR` |
| a fuzzy array hit whose elements all fail `CONTAINS` | AQL | the display expression falls back to the joined list |
| the view exists with the wrong links | constructor | `SetProperties` rewrites the links. The view is not deleted. The index rebuilds in the background, so the first queries after a restart can return fewer rows. |
| a non-ArangoSearch view holds the name `stock_autocomplete` | constructor | an error from `ArangoSearchView()`. The start of the service stops. Do not delete the other view. |
| an analyzer with the same name exists with other properties | constructor | `EnsureCreatedAnalyzer` returns an error on a conflicting definition. The start of the service stops, and an operator must drop the old analyzer. Record the exact error text in the [Execution notes](#execution-notes) when it happens. |
| the database is missing at start | constructor | `NewStockRepo` returns an error. The pod restart is the retry. No `sync.Once`, so a later attempt is not poisoned. |
| the documents are newer than the last view commit | repository | the query returns fewer rows than expected for about one second. Tests poll with `require.Eventually`. |
| the generated AQL has a syntax error | repository | `SearchRows` fails in `ValidateQuery` with `error in validating the query ...`, before the server runs anything |
| a projection returns an array row | repository | `Resultset.Read` fails with `cannot unmarshal object into ...`. Every projection must return one object per row. |
| the repository returns an error | handler | `aphgrpc.HandleGetError`, like the other read handlers |

## Quality gates

Run every command after each task that changes a `.go` file. Do not hand over work while a diagnostic remains.

```bash
cd /Users/sba964/Projects/devenv/golang/modware-stock
export ARANGO_HOST=localhost ARANGO_USER=root ARANGO_PASS=rootpass ARANGO_PORT=8529

go test ./...
go test -race ./...
golangci-lint fmt
golangci-lint run ./...
gopls check -severity=hint \
  internal/repository/repository.go \
  internal/repository/arangodb/autocomplete.go \
  internal/repository/arangodb/statement/autocomplete.go \
  internal/repository/arangodb/database.go \
  internal/repository/arangodb/autocomplete_test.go \
  internal/app/service/autocomplete.go \
  internal/app/service/autocomplete_test.go \
  internal/app/service/autocomplete_arango_test.go
```

Focused runs during development:

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'TestAutocompleteStock|TestBuildAutocompleteQuery|TestEnsureAutocompleteSearch' ./internal/repository/arangodb/
gotestsum --format-hide-empty-pkg --format dots -- -run 'AutocompleteStock' ./internal/app/service/
```

Rules that hold for every gate:

- Repository code is tested against a real disposable ArangoDB database, one per test, created by `testarango.NewTestArangoFromEnv(true)` and dropped in `t.Cleanup` or `tearDown`.
- Handlers are tested twice: against a stub repository for every input class, and over one real buffer connection for the full round trip.
- Write flat sequential tests. Do not use `t.Parallel` in a subtest that shares a database with its parent.
- The server version under test is ArangoDB 3.11, the same series as CI.
- CI needs no change. The workflow already starts `arangodb:3.11` and exports the four variables.

## Completion criteria

- [ ] `scripts/autocomplete-calibration.js` exists, runs against ArangoDB 3.11, and its 20 probe results plus the EXPLAIN and latency baseline are recorded in the [Execution notes](#execution-notes).
- [ ] Every contradiction that calibration found is folded into this plan, and the fold is recorded.
- [ ] `internal/repository/repository.go` declares `StockEntityFilter` with its 3 constants, `AutocompleteQuery`, `Suggestion`, and the interface method `AutocompleteStock(params *AutocompleteQuery) ([]*Suggestion, error)`.
- [ ] `NewStockRepo` creates both analyzers and the view `stock_autocomplete` eagerly, and fails at start when the setup fails.
- [ ] A stale view converges to the definition through `SetProperties`, with no delete.
- [ ] The built AQL statement holds 16 branches, with the entity filter before every `SORT` and `LIMIT`.
- [ ] `AutocompleteStock` returns at most 50 rows, 5 by default, and an empty list is not an error.
- [ ] `StockService.AutocompleteStock` guards nil `data` and nil `attributes`. It validates with `protovalidate.Validate`, and it rejects a whitespace-only query.
- [ ] `StockService.AutocompleteStock` fills `Meta` with `Total` as the row count, `Limit` as the effective limit, and `NextCursor` 0.
- [ ] Every row of the [Test matrix](#test-matrix) has a test, and all of them pass.
- [ ] The EXPLAIN check reports 16 `EnumerateViewNode` entries and 0 `EnumerateCollectionNode` entries.
- [ ] The measured p95 latency is at or below 125 percent of the calibration baseline p95, on the same warmed fixture.
- [ ] Every gate in [Quality gates](#quality-gates) passes.
- [ ] `README.md` describes the RPC, the limits, the 8 fields and the asset names.

## Handoff artifacts

| # | Artifact | Where it lives | Who consumes it |
| --- | --- | --- | --- |
| 1 | The confirmed n-gram threshold, and the query normalization decision | [Execution notes](#execution-notes) | a later tuning task, and the full-search plan, which calibrates its own threshold and can compare |
| 2 | The 20 probe results of Task 1 | [Execution notes](#execution-notes) | a reviewer, and any later change of an analyzer |
| 3 | The EXPLAIN node list and the latency baseline p95 | [Execution notes](#execution-notes) | Task 6 of this plan, and any later change of the statement |
| 4 | `repository.StockEntityFilter`, `repository.EntityBoth`, `repository.EntityStrain`, `repository.EntityPlasmid` | `internal/repository/repository.go` | the full-search plan, which reuses the same declarations under its own shared declaration rule |
| 5 | The safe bufconn pattern in `internal/app/service/autocomplete_arango_test.go` | the test file | the full-search plan, which needs the same pattern in its own file |
| 6 | The asset names `stock_autocomplete_norm`, `stock_autocomplete_ngram` and `stock_autocomplete` | `README.md` and `internal/repository/arangodb/autocomplete.go` | an operator, and the full-search plan, which must not reuse them |
| 7 | A list of follow-up issues: an evaluation of `optimizeTopK`, nil guards for the existing handlers, a replacement of the unsafe `setupGrpcServer` and `setupGrpcClient` helpers, and a shared bufconn helper for the two search test files | the issue tracker | a later session |

## Review focus

Input classes and risks that no single task fully owns. Each has a named owner test.

1. **Property ownership through the graph.** A property `_key` is auto-generated. Every property-branch row must resolve its stock through the INBOUND traversal, or the branch must return nothing. A row whose `ID` is a property key is a defect. Owner: `TestAutocompleteStockLabelPrefix` and `TestAutocompleteStockCrossCollectionMerge`.
2. **Entity-filter loss.** The filter must sit inside each branch, before the branch `SORT` and `LIMIT`. Probe 15 of Task 1 measures the loss of the wrong design, and the test pins the right one. Owner: `TestAutocompleteStockEntityFilterKeepsSmallGroup`.
3. **Ranking under a small limit.** A branch that truncates in index order loses a stronger match. This was a real review defect in modware-order. Owner: `TestAutocompleteStockRankingUnderSmallLimit`.
4. **Uppercase and accented input.** `STARTS_WITH` does not normalize its prefix argument. The Go lowercase step is the baseline, and probe 10 of Task 1 decides whether `TOKENS` is needed for accents. Owner: probe 10, plus `TestAutocompleteStockHandlerPassesThroughQueryAndLimit`.
5. **Nil `data` and nil `attributes`.** This repository has no recovery interceptor, so a nil dereference kills the process. Owner: `TestAutocompleteStockHandlerRejectsNilData` and `TestAutocompleteStockHandlerRejectsNilAttributes`.
6. **Whitespace-only query.** `string.min_len` counts characters, so the proto accepts `"   "`. Owner: `TestAutocompleteStockHandlerRejectsWhitespaceQuery` and `TestAutocompleteStockRejectsWhitespaceQuery`.
7. **View commit delay and a stale view.** New documents are invisible for about one second, and an old view definition must converge without a delete. Owner: the `require.Eventually` wrapper in every database test, plus `TestEnsureAutocompleteSearchReconcilesStaleView`.
8. **Array display extraction.** A fuzzy array hit that fails `CONTAINS`, and an absent array attribute, must both give a string, never `null`. Owner: `TestAutocompleteStockArrayDisplayFallback` and probe 13 of Task 1.
9. **Independence from the full-search plan.** This plan must build, test and run with the full-search assets absent. A reviewer checks that no identifier of this plan names `stock_search_norm`, `stock_search_ngram`, `text_en` or `stock_full_search`. A reviewer also checks that `database.go` keeps both setup calls when both plans have landed.
10. **The no-op generated validator.** A handler that calls `r.Validate()` validates nothing. Owner: `TestAutocompleteStockHandlerRejectsShortQuery`, which fails when the handler uses the wrong validator.

## Execution notes

Filled 2026-10-07. Server: local docker `arango311`, **ArangoDB 3.11.14** community (this container maps host port 8530; host port 8529 is held by an unrelated 3.12 container). Harness: `scripts/autocomplete-calibration.js` plus two follow-up probe runs (threshold sweeps); disposable databases dropped at exit.

- ArangoDB version string from Task 1 Step 1: `3.11.14`
- Probe results 1 to 20:
  1. prefix `dbs023` on `stock_id`: 1 row, `k` = DBS0236126, score 1000 (BM25 of a pure prefix match is 0, so the score is exactly 1000 — assertions must use `>= 1000`).
  2. prefix `ys` on `label`: 1 row, `k` = the **stock** key DBS0236126 → INBOUND direction proven.
  3. prefix `pdm` on plasmid `name`: 1 row, `k` = DBP0000027, entity `plasmid`.
  4. same query without the `ANALYZER()` wrapper: 0 rows → wrapper required.
  5. fuzzy `dbs0236127` sweep: 0.2 matches junk; 0.3–0.4 match only DBS0236126; 0.45+ match nothing. (Published threshold semantics do not hold.)
  6. fuzzy `sdaa` on `genes` (transposition): 0 rows at every threshold 0.25–0.5 — transpositions are invisible to n-gram similarity. Follow-up `sada` (appended char) matches at 0.25–0.5. Fuzzy test cases must avoid transpositions.
  7. prefix `dictyo` on `species`: 1 row.
  8. prefix `d031` on `dbxrefs`: 1 row.
  9. uppercase `DBS023` lowercased client-side: 1 row.
  10. accented `Áx2`: plain `strings.ToLower` output finds nothing; `FIRST(TOKENS(@q, "stock_autocomplete_norm"))` finds Ax2 in both `STARTS_WITH` and `NGRAM_MATCH`. Decision: normalize in **Go** (trim, lowercase, strip Unicode Mn combining marks) and keep AQL free of `TOKENS` — a punctuation-only query gives an empty token list and a null prefix argument, which would poison the prefix branches.
  11. array display on `names`, prefix `ax`: returns element `Ax2`, not the joined list.
  12. array display for a fuzzy hit `gammas`: returns `gammaS13` (the `CONTAINS` filter matched, so the joined-list fallback path was not directly observed; the fallback is still guarded by `NOT_NULL`).
  13. array display on the property document without `names` (matched via `label` `nonam`): returns `""`, no AQL error.
  14. property document with no inbound edge: dropped by `FILTER own != null`.
  15. entity filter inside the `genes` branch, query `xylose`, `@entity` `plasmid`, `@limit` 10: **5 plasmid rows**. With the filter moved after the merge: also 5, because all xylose rows tie on BM25 and the `d._key ASC` tiebreak sorts `DBP*` before `DBS*`. The regression did not reproduce on this fixture, but the in-branch filter stays — it is the only design that cannot lose a small group, per the modware-order review (AUTOCOMPLETE.md section 3 Step 4).
  16. cross-collection merge (`cordaxin` gene + `cordax` label on one stock, query `corda`): exactly 1 merged row, field attribution from the higher score.
  17. whitespace-only query `"   "`: 0 rows, no AQL error.
  18. punctuation-only query `"---"`: 0 rows, no AQL error.
  19. 1-char and 2-char queries: rows return, no error (the handler rejects short queries; AQL itself is safe).
  20. `FLATTEN([...])` over the 16 branch variables: flat object list, not nested.
  Extra finding: ArangoDB 3.11 rejects **unused bind parameters** (`AQL: bind parameter 'th' was not declared in the query`). The full statement declares all 5 parameters and uses them all, so the repository is unaffected; probe statements had to filter unused parameters.
- Final n-gram threshold: **0.3** (confirmed by sweeps; 0.45, the value assumed from the 3.12 calibration of modware-order, misses one-character typos on 3.11).
- Query normalization decision: Go-side — `strings.TrimSpace`, `strings.ToLower`, strip Unicode Mn combining marks. No `TOKENS` in AQL.
- EXPLAIN node list and the baseline p95 in milliseconds: 16 `EnumerateViewNode`, 0 `EnumerateCollectionNode`; full node-type counts: SingletonNode 1, SubqueryStartNode 40, EnumerateViewNode 16, TraversalNode 16, LimitNode 40, CalculationNode 100, SubqueryEndNode 40, FilterNode 22, SortNode 19, EnumerateListNode 9, CollectNode 1, ReturnNode 1. Latency over 20 runs on the ~1000-document fixture, query `dbs023`, limit 5: min 7, median 8, **p95 8 ms**.
- Task 6 measured p95, and the ratio against the baseline: p95 **9 ms** vs baseline 8 ms on the same warmed fixture (query `dbs023`, limit 5, 20 runs) — ratio 1.125, inside the 125% gate. EXPLAIN of the built statement: 16 `EnumerateViewNode`, 0 `EnumerateCollectionNode`; node-type counts identical to Task 1 (SingletonNode 1, SubqueryStartNode 40, EnumerateViewNode 16, TraversalNode 16, LimitNode 40, CalculationNode 100, SubqueryEndNode 40, FilterNode 22, SortNode 19, EnumerateListNode 9, CollectNode 1, ReturnNode 1). Calibration fixture database kept for Task 6: `stock_calib_1791390788601`.
- Build break recorded in Task 2 Step 4: (pending)
- Pre-existing test failures on the branch before any change: none on the plain run — full suite 448 tests, 1 skipped, 0 failures (with ARANGO_PORT=8530). Under `go test -race`, `cmd/modware-stock` flag tests (TestMain*, TestServerFlags, TestDbCollectionFlags, TestAllFlags) fail pre-existing; verified by stashing the branch — unrelated to this change. go-genproto bumped to v0.0.0-20261007153809-00e41d0050e2 (user ruling: latest master, which already ships all autocomplete stubs) and protovalidate-go v0.10.0 added before Task 1.
- The gRPC code that `aphgrpc.HandleGetError` really returns: `codes.Internal` (verified in aphgrpc v1.4.2 `error.go`)
- Deviations from [Exact interfaces](#exact-interfaces), with the reason: threshold constant 0.45 → 0.3 (calibration); `@q` normalization is Go-side trim+lower+diacritic-strip (calibration, see probe 10).
