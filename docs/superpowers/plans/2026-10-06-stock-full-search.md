# Stock Full Search Implementation Plan

> **For agentic workers:** Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task by task. The steps use checkboxes.

**Goal:** Add the `SearchStock` RPC to modware-stock. A client sends a search text of at least 2 characters and gets a ranked list of at most 50 stocks. The search covers 8 identifier and name fields plus 2 prose fields of strains and plasmids. Each result carries the complete stored summary and `strain_label`. ArangoSearch on ArangoDB 3.11 supplies the index.

**Architecture:** This plan owns its own search assets and shares nothing with the autocomplete plan. The assets are two custom analyzers, `stock_search_norm` and `stock_search_ngram`, the built-in analyzer `text_en`, and one classic `arangosearch` view, `stock_full_search`. The view links the stock collection and the stock property collection. One AQL statement runs 19 branches. Eight identifier and name fields get prefix and fuzzy branches. The `summary` and `depositor` fields get token branches. The `summary` field also gets a phrase branch. Do not index or search `editable_summary`. A branch over the stock collection reads the stock key directly. It then traverses the named graph `stock_prop_graph` once in the OUTBOUND direction, to learn the entity type. A branch over the property collection traverses once in the INBOUND direction. That direction is necessary, because the property `_key` is auto-generated and carries no relation to the stock key. Each branch filters by entity before its own `SORT` and `LIMIT`. A merge step keeps the best row per stock key. The tail reads the summary only for the rows that it returns. The repository constructor creates the analyzers and the view eagerly, so a broken setup stops the start of the service.

**Tech Stack:** ArangoDB 3.11 (CI image `arangodb:3.11`), go-driver v1.6.9, dictyBase/arangomanager v0.8.0, dictyBase/aphgrpc v1.4.2, `github.com/bufbuild/protovalidate-go` v0.10.0, IBM/fp-go v1.1.84, gRPC v1.83.2, Go 1.26, gotestsum, golangci-lint, gopls.

**Spec:** User request (2026-10-06): a complete search that returns a list of 50 retrieved items for the web drop-down, on top of the type-ahead autocomplete. The search must work on the ArangoDB 3.11 series. The web user interface is out of scope; this repository delivers the RPC.

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
  - [Scoring and tie order](#scoring-and-tie-order)
  - [Multi-token semantics](#multi-token-semantics)
  - [AQL branch templates](#aql-branch-templates)
  - [Merge and tail](#merge-and-tail)
  - [Bind parameters](#bind-parameters)
  - [Limit rules](#limit-rules)
  - [Handler signature and mapping](#handler-signature-and-mapping)
- [File map](#file-map)
- [Implementation tasks](#implementation-tasks)
  - [Task 1: Calibration against real ArangoDB 3.11](#task-1-calibration-against-real-arangodb-311)
  - [Task 2: Repository types and interface method](#task-2-repository-types-and-interface-method)
  - [Task 3: Search assets in the constructor](#task-3-search-assets-in-the-constructor)
  - [Task 4: Query builder and SearchStock](#task-4-query-builder-and-searchstock)
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

Deliver a working `SearchStock` RPC with a repository implementation, its own ArangoSearch assets, and tests that run against a real disposable ArangoDB 3.11 database. A client that sends 2 or more characters gets at most 50 ranked results, 50 by default. Each result names the stock, the kind of stock, and the field that matched. It also carries the text to show, a score, the complete stored summary, and `strain_label`. For strains, `strain_label` is the property label shown as Descriptor in the stock center. For plasmids and strains with no label, it is an empty string.

## Scope

- 2 new repository types, plus the shared entity filter type, and 1 new method on `repository.StockRepository`.
- 2 new custom analyzers and 1 new classic `arangosearch` view, created eagerly in `NewStockRepo`. The third analyzer, `text_en`, is built in and is not created.
- Reconciliation of an existing view whose links differ from the definition, with `ArangoSearchView.Properties` and `ArangoSearchView.SetProperties`.
- 1 AQL statement with 19 branches, built from 4 templates.
- 1 gRPC handler on `StockService`.
- Repository tests against a real database, handler unit tests against a stub repository, and one handler test over a real buffer connection.
- A calibration harness that proves every AQL assumption on ArangoDB 3.11, including the behavior of `PHRASE` and `TOKENS` with bind parameters.
- An EXPLAIN check and a latency gate.
- Documentation of the RPC contract.

## Non-goals

- No proto edit. The protocol plan `docs/superpowers/plans/2026-10-06-stock-search-protobuf.md` owns the wire contract, and this plan consumes the released stubs.
- No autocomplete work. The plan `docs/superpowers/plans/2026-10-06-stock-autocomplete.md` owns the `AutocompleteStock` RPC, the analyzers `stock_autocomplete_norm` and `stock_autocomplete_ngram`, and the view `stock_autocomplete`. This plan must run correctly whether or not those assets exist.
- No n-gram analyzer on prose. `summary` and `depositor` use `text_en` only. Do not index or search `editable_summary`. An n-gram index over long prose grows fast and returns weak matches.
- No stemmer on `species`. A Latin binomial such as `Dictyostelium discoideum` breaks under an English stemmer, so `species` stays on the prefix and fuzzy stages.
- No cursor pagination. The spec asks for one list of 50 items. `Meta.NextCursor` is always 0.
- No snippet and no highlight. `display_text` and `summary` carry complete stored values. `strain_label` carries the strain property label, or an empty string for a plasmid or a strain without a label.
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
| `summary`, `editable_summary` and `depositor` live on the **stock** document, never on a property document. This plan searches `summary` and `depositor` only. It does not search `editable_summary`. The 3 prose branches use the stock branch template. | `statement/insert.go` |
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
| Collection and graph names are **not** constant. They come from `CollectionParams`. The repository tests use `stock_test`, `stock_properties_test`, `stock_type_test` and the graph `stockprop_type_test`. Build the view links from `ar.stockc.stock.Name()` and `ar.stockc.stockProp.Name()`, the traversal from `ar.stockc.stockPropType.Name()`, and the tail document lookup from `ar.stockc.stock.Name()`. Never hardcode a production name. | `database.go` `docCollections`, `internal/repository/arangodb/arangodb_test.go` `getCollectionParams` |
| Existing bind-parameter names live in `consts.go`: `nameStockPropGraph = "stock_prop_graph"`, `nameStockCollection = "stock_collection"`, `paramLimit = "limit"`, and the field names `fieldSummary`, `fieldDepositor`, `fieldGenes`, `fieldDbxrefs`, `fieldLabel`, `fieldSpecies`, `fieldPlasmid`, `paramName`, `paramStockID`. Reuse them. | `consts.go` |
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
| `SearchRows` returns `&Resultset{empty: true}, nil` when the cursor has no rows. `Resultset.Scan()` then returns false at once. A loop of `for result.Scan() { result.Read(&row) }` is therefore safe for an empty result. | `arangomanager/database.go` line 110, `arangomanager/resultset.go` |
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
| Repository test fixtures already exist: `newTestStrain(createdby string, stype StrainType) *stock.NewStrain` with `Summary` and `EditableSummary` set from `testRadiationSummary`, `Depositor: testEmailCostanza`, `Label: "yS13"`, `Genes: []string{"DDB_G0348394", "DDB_G098058933"}`, `Plasmid: "DBP0000027"` and `Names: []string{"gammaS13", "gammaS-13", "γS-13"}`, and `newTestPlasmid(createdby string) *stock.NewPlasmid` with `Name: "p123456"` and `Summary: testPlasmidSummary`. | `internal/repository/arangodb/arangodb_test.go` lines 110-196 |
| `repo.AddStrain` returns `(*model.StockDoc, error)`. `repo.AddPlasmid` returns `IOE.IOEither[error, *model.StockDoc]`. The existing tests run it with `F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)`, where `ToEither` and `toStockDocResult` are defined in `internal/repository/arangodb/plasmid_test.go` lines 37-54 and are available to every test file of the same package. | `internal/repository/arangodb/plasmid_test.go` lines 204-218 |
| `NewPlasmidAttributes` carries `genes` as field 5 and `summary` as field 3, so a plasmid holds both. A `genes` query therefore matches both kinds of stock, which makes `genes` the right field for an entity-filter regression test. | `dictybaseapis/dictybase/stock/stock.proto` lines 205-231 |
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
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockSearchParameters
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockSearchAttributes
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockSearchResult
go doc github.com/dictyBase/go-genproto/dictybaseapis/stock StockSearchResultCollection
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
git switch develop && git pull --rebase && git switch -c feat/stock-full-search
```

## Consumed protocol contract

This section repeats the full contract that this plan consumes. Treat it as normative; it is not a summary.

Generated Go types in package `github.com/dictyBase/go-genproto/dictybaseapis/stock`:

```go
type StockSearchParameters struct {
	Data *StockSearchParameters_Data
}

type StockSearchParameters_Data struct {
	Type       string
	Attributes *StockSearchAttributes
}

type StockSearchAttributes struct {
	Query  string      // buf.validate string.min_len = 2
	Limit  int64       // buf.validate int64 {gte: 0, lte: 100}
	Entity StockEntity // buf.validate enum.defined_only = true
}

type StockSearchResult struct {
	Id          string
	Entity      StockEntity
	Field       StockSearchField
	DisplayText string
	Score       float64
	Summary     string
	StrainLabel string
}

type StockSearchResultCollection struct {
	Data []*StockSearchResult
	Meta *Meta
}
```

Generated server method that this plan implements:

```go
SearchStock(context.Context, *stock.StockSearchParameters) (*stock.StockSearchResultCollection, error)
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
| `StockSearchField` | `STOCK_SEARCH_FIELD_SUMMARY` | 9 |
| `StockSearchField` | reserved number and name `STOCK_SEARCH_FIELD_EDITABLE_SUMMARY` | 10 |
| `StockSearchField` | `STOCK_SEARCH_FIELD_DEPOSITOR` | 11 |

`StockSearchResult.Summary` carries the **exact stored value** of the `summary` attribute of the stock document. It is not a snippet, it is not truncated, and it is not highlighted. An absent attribute gives an empty string. `StockSearchResult.StrainLabel` carries the strain property `label`, shown as Descriptor in the stock center. For plasmids and strains with no label, it is an empty string. This field is result metadata, not a searchable field.

Two gaps in the proto rules that this plan must close in the handler:

1. `string.min_len = 2` counts characters, so the query `"  "` passes validation. The handler must trim the query and reject an empty result with an invalid-argument error.
2. The generated `Validate()` method is a no-op for these messages. The handler must call `protovalidate.Validate`.

## Exact interfaces

### Repository types and method

Add to `internal/repository/repository.go`.

**Shared declaration rule.** The autocomplete plan declares the same entity filter type and the same three constants, with identical text. Before you add the block, check whether it already exists:

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

Then add the full-search types and the interface method. These are owned by this plan only. `FullSearchResult` embeds nothing, so it compiles whether or not the autocomplete types exist:

```go
// FullSearchQuery holds the input of a full stock search. The repository
// normalizes Query and clamps Limit.
type FullSearchQuery struct {
	// Query is the search text. The caller sends it untrimmed; the
	// repository trims it and lowercases it.
	Query string
	// Entity restricts the search to one kind of stock.
	Entity StockEntityFilter
	// Limit is the maximum number of results. A value at or below 0
	// becomes 50. A value above 50 becomes 50.
	Limit int
}

// FullSearchResult is one full search match.
type FullSearchResult struct {
	// ID is the stock_id of the matched stock, for example DBS0236126.
	ID string
	// Field is the field path that matched, for example summary or label.
	Field string
	// DisplayText is the complete stored value of the matched field.
	DisplayText string
	// Summary is the complete stored summary of the stock document. It is
	// empty when the attribute is absent.
	Summary string
	// StrainLabel is the strain property label shown as Descriptor in the
	// stock center. It is empty for plasmids and missing labels.
	StrainLabel string
	// Entity is the kind of the matched stock. It is never EntityBoth.
	Entity StockEntityFilter
	// Score ranks the match. A prefix match scores above a phrase match, a
	// phrase match above a token match, and a token match above a fuzzy
	// match.
	Score float64
}
```

Add one method to the `StockRepository` interface, after `RemoveStock(id string) error`:

```go
	SearchStock(params *FullSearchQuery) ([]*FullSearchResult, error)
```

The method takes a parameter struct, not ordered scalars, so a later field cannot be passed in the wrong position.

### Search asset names and constants

Add to `internal/repository/arangodb/fullsearch.go`. Every custom name belongs to this plan alone, so the autocomplete assets can be absent, present or stale without any effect.

```go
const (
	// fullSearchViewName is the classic arangosearch view of this
	// feature. The AQL templates name it literally, because ArangoDB
	// does not accept a bind parameter for a view name in SEARCH.
	fullSearchViewName = "stock_full_search"
	// fullSearchNormAnalyzer lowercases a value and removes accents. It
	// serves the prefix branches.
	fullSearchNormAnalyzer = "stock_search_norm"
	// fullSearchNgramAnalyzer chains norm and ngram. It serves the fuzzy
	// branches.
	fullSearchNgramAnalyzer = "stock_search_ngram"
	// fullSearchTextAnalyzer is the built-in English text analyzer. It
	// serves the token branches and the phrase branches. It is built in,
	// so the setup must not try to create it.
	fullSearchTextAnalyzer = "text_en"
	// defaultFullSearchLimit is the list length for a non-positive limit.
	defaultFullSearchLimit = 50
	// maxFullSearchLimit is the hard cap of the returned list length. The
	// proto accepts a limit up to 100, and the repository clamps it to
	// this value, because the spec asks for one list of 50 items.
	maxFullSearchLimit = 50
	// fullSearchNgramThreshold is the minimum n-gram similarity of a
	// fuzzy match. Task 1 calibration on ArangoDB 3.11.14 corrected the
	// assumed 0.45 to 0.30: 0.45 misses the one-character typo
	// dbs0236127 against DBS0236126 entirely, while 0.30 matches it and
	// still admits no junk. This mirrors the autocomplete calibration.
	fullSearchNgramThreshold = 0.3
)
```

Analyzer definitions. Create only the two custom analyzers, both with `EnsureCreatedAnalyzer`:

| Analyzer | Type | Properties | Features |
| --- | --- | --- | --- |
| `stock_search_norm` | `driver.ArangoSearchAnalyzerTypeNorm` | `Locale: "en.utf-8"`, `Case: driver.ArangoSearchCaseLower`, `Accent: new(false)` | none |
| `stock_search_ngram` | `driver.ArangoSearchAnalyzerTypePipeline` | pipeline step 1 `Norm` with the same three properties; pipeline step 2 `NGram` with `Min: new(int64(2))`, `Max: new(int64(3))`, `PreserveOriginal: new(true)`, `StreamType: new(driver.ArangoSearchNGramStreamUTF8)` | `Frequency`, `Norm`, `Position` |
| `text_en` | built in | not created by this code | supplied by the server |

The three features on the pipeline analyzer are mandatory. `NGRAM_MATCH` fails without them. A plain `ngram` analyzer, without the pipeline, returned zero matches on a real server; see `AUTOCOMPLETE.md` section 3 Step 1. `PHRASE` needs the `frequency` and `position` features on its analyzer; `text_en` carries them, so no custom text analyzer is needed. Task 1 probe 11 confirms this.

Go 1.26 accepts `new(expr)`, so `new(false)` and `new(int64(2))` need no helper function.

### View definition

One classic `arangosearch` view, `stock_full_search`, with 2 links. Build both collection names from the repository struct.

| Link key | Field | Analyzers |
| --- | --- | --- |
| `ar.stockc.stock.Name()` | `stock_id` | `stock_search_norm`, `stock_search_ngram` |
| `ar.stockc.stock.Name()` | `genes` | `stock_search_norm`, `stock_search_ngram` |
| `ar.stockc.stock.Name()` | `dbxrefs` | `stock_search_norm`, `stock_search_ngram` |
| `ar.stockc.stock.Name()` | `summary` | `text_en` |
| `ar.stockc.stock.Name()` | `depositor` | `text_en` |
| `ar.stockc.stockProp.Name()` | `label` | `stock_search_norm`, `stock_search_ngram` |
| `ar.stockc.stockProp.Name()` | `names` | `stock_search_norm`, `stock_search_ngram` |
| `ar.stockc.stockProp.Name()` | `species` | `stock_search_norm`, `stock_search_ngram` |
| `ar.stockc.stockProp.Name()` | `plasmid` | `stock_search_norm`, `stock_search_ngram` |
| `ar.stockc.stockProp.Name()` | `name` | `stock_search_norm`, `stock_search_ngram` |

No field carries both the n-gram analyzer and `text_en`. An n-gram index over long prose grows fast and returns weak matches, and a stemmer breaks a Latin binomial. The split is deliberate.

**Reconciliation rule.** Do not delete and create a view when it already exists.

1. `ViewExists(ctx, "stock_full_search")`. When it is false, call `CreateArangoSearchView` and tolerate `driver.IsConflict(err)`, because a second repository instance can win the race.
2. When it is true, call `View(ctx, name)`, then `ArangoSearchView()`, then `Properties(ctx)`.
3. Compare **only** the shape that this plan controls: the set of link keys, the set of field names per link, and the sorted set of analyzer names per field. Do not compare the whole `ArangoSearchElementProperties` value. The server fills defaults such as `includeAllFields`, `trackListPositions` and `storeValues` into the response. A deep comparison therefore always reports a difference, and it rewrites the view at every start.
4. When the compared shape differs, call `SetProperties(ctx, driver.ArangoSearchViewProperties{Links: desired})`. The driver sends a PUT, which replaces the properties. The index rebuilds in the background.
5. When `ArangoSearchView()` fails because a view of another type holds the name, return an error. Do not delete the other view.

### Field and branch matrix

19 branches: 8 identifier fields with 2 stages each, 2 token branches, and 1 phrase branch.

| # | Field | Collection | Branch template | Stage | Branch variable | Display expression kind |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `stock_id` | stock | stock | prefix | `p0` | scalar |
| 2 | `stock_id` | stock | stock | fuzzy | `n0` | scalar |
| 3 | `genes` | stock | stock | prefix | `p1` | array |
| 4 | `genes` | stock | stock | fuzzy | `n1` | array |
| 5 | `dbxrefs` | stock | stock | prefix | `p2` | array |
| 6 | `dbxrefs` | stock | stock | fuzzy | `n2` | array |
| 7 | `label` | property | property | prefix | `p3` | scalar |
| 8 | `label` | property | property | fuzzy | `n3` | scalar |
| 9 | `names` | property | property | prefix | `p4` | array |
| 10 | `names` | property | property | fuzzy | `n4` | array |
| 11 | `species` | property | property | prefix | `p5` | scalar |
| 12 | `species` | property | property | fuzzy | `n5` | scalar |
| 13 | `plasmid` | property | property | prefix | `p6` | scalar |
| 14 | `plasmid` | property | property | fuzzy | `n6` | scalar |
| 15 | `name` | property | property | prefix | `p7` | scalar |
| 16 | `name` | property | property | fuzzy | `n7` | scalar |
| 17 | `summary` | stock | stock | token | `t0` | scalar |
| 18 | `depositor` | stock | stock | token | `t1` | scalar |
| 19 | `summary` | stock | stock | phrase | `h0` | scalar |

Branch family counts, used by the builder test: 9 branches use the stock template and the OUTBOUND traversal. Those are rows 1 to 6 and rows 17 to 19. The other 10 branches use the property template and the INBOUND traversal.

Match expression per stage:

| Stage | `SEARCH` expression |
| --- | --- |
| prefix | `ANALYZER(STARTS_WITH(d.<field>, @q), "stock_search_norm")` |
| fuzzy | `NGRAM_MATCH(d.<field>, @q, @th, "stock_search_ngram")` |
| token | `ANALYZER(d.<field> IN TOKENS(@q, "text_en"), "text_en")` |
| phrase | `PHRASE(d.<field>, @q, "text_en")` |

`STARTS_WITH` inside `SEARCH` needs the `ANALYZER()` wrapper. Without it the comparison uses the `identity` analyzer and matches nothing. `NGRAM_MATCH` takes the analyzer as its fourth argument and needs no wrapper. The `IN TOKENS` form **does** need the wrapper. The comparison of the indexed value against the token list must run under the same analyzer that produced the index. `PHRASE` takes the analyzer as its last argument.

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

For the 3 prose branches the display expression is the scalar form, so `display_text` of a summary match is the complete stored summary. There is no snippet.

### Scoring and tie order

| Stage | Score expression | Rank band |
| --- | --- | --- |
| prefix | `1000 + BM25(d)` | highest |
| phrase | `500 + BM25(d)` | second |
| token | `250 + BM25(d)` | third |
| fuzzy | `BM25(d)` | lowest |

`BM25(d)` of a pure `STARTS_WITH` match can be 0, so the constant bands carry the order and the deterministic tiebreak carries the rest. The bands are 250 apart, and `BM25` on this data set stays far below 250, which Task 1 probe 15 measures and records. If a measured `BM25` ever reaches 250, the bands must be widened; record the measurement.

Tie order is fully determined:

1. Inside a branch: `SORT BM25(d) DESC, <key> ASC`, where `<key>` is `d._key` for a stock branch and `own.k` for a property branch.
2. In the merge: one row per stock key, chosen by `SORT m.s DESC, m.f ASC`. When two branches give the same score for one stock, the field label in ascending string order wins. The order of the 10 field labels is therefore `dbxrefs`, `depositor`, `genes`, `label`, `name`, `names`, `plasmid`, `species`, `stock_id`, `summary`.
3. In the tail: `SORT x.s DESC, x.k ASC`. Two stocks with the same score come back in ascending stock key order.

### Multi-token semantics

The behavior for a query of more than one word is deliberate and must be pinned by tests.

| Query shape | Branch that matches | Effect |
| --- | --- | --- |
| one word that is a prefix of an identifier | prefix | the row leads the list with at least 1000 points |
| two or more words in the stored order | phrase | the row gets 500 points and ranks above any token-only row |
| two or more words in another order, or with other words between them | token | each row that holds **at least one** token gets 250 points |
| one word that appears in prose | token | 250 points |
| a misspelled identifier | fuzzy | below 250 points, so it fills the tail |

The token stage uses `IN TOKENS`, which matches **any** token, not all of them. A document that holds only one word of a two-word query therefore appears in the result, with 250 points. This is intentional for version 1: the phrase band and the prefix band keep the strong rows at the head, and the single-token rows fill the tail of the 50.

Task 1 probe 17 can show that the tail is too noisy. The fallback is then to generate one `IN TOKENS` condition per token in Go, and to join the conditions with `AND`. That change makes the token stage an all-token stage. Make the decision from the probe, not from an assumption. Record the decision in the [Execution notes](#execution-notes).

Stopword behavior is **not verified** and Task 1 probe 18 decides it. The built-in `text_en` analyzer of ArangoDB applies lowercasing, accent removal and English stemming; whether it also removes English stopwords is the open question. Two outcomes:

- `TOKENS("the and", "text_en")` returns an empty list. Then a stopword-only query matches nothing, and the result is an empty list with a nil error.
- `TOKENS("the and", "text_en")` returns tokens. Then a stopword-only query matches many documents through the token stage, all at 250 points plus a small `BM25`. The result is noisy but bounded by the limit. The documented mitigation is a short Go-side stopword list that rejects a query whose tokens are all stopwords, with an invalid-argument error.

Record the probe output and the chosen behavior, and pin it with `TestSearchStockStopwordOnlyQuery`.

### AQL branch templates

Put all four templates in `internal/repository/arangodb/statement/fullsearch.go` as `const` strings, next to the existing statement files. The builder fills them with `fmt.Sprintf`. The two branch families differ only in the ownership resolution. The stage differences therefore live entirely in the `SEARCH` expression and the score expression that the builder passes in.

**Stock collection branch.** The row key is the stock key. One OUTBOUND traversal supplies the entity and strain label. Rows 1 to 6 and 17 to 19 of the matrix use it.

```aql
LET %s = (
  FOR d IN stock_full_search
    SEARCH %s
    LET meta = FIRST(
      FOR prop, edge IN 1..1 OUTBOUND d GRAPH @stock_prop_graph
        RETURN {
          entity: edge.type,
          strain_label: edge.type == "strain"
            ? NOT_NULL(prop.label, "")
            : ""
        }
    )
    FILTER meta != null
    FILTER @entity == "" OR @entity == meta.entity
    SORT BM25(d) DESC, d._key ASC
    LIMIT @limit
    RETURN {
      k: d._key,
      id: d.stock_id,
      entity: meta.entity,
      sl: meta.strain_label,
      f: %q,
      v: %s,
      s: %s
    }
)
```

**Property collection branch.** The property `_key` is auto-generated, so the row must resolve its owner with one INBOUND traversal. The same row reads `d.label` for a strain. A plasmid row sets the label to an empty string. Rows 7 to 16 of the matrix use this template.

```aql
LET %s = (
  FOR d IN stock_full_search
    SEARCH %s
    LET own = FIRST(
      FOR v, e IN 1..1 INBOUND d GRAPH @stock_prop_graph
        RETURN {
          k: v._key,
          id: v.stock_id,
          entity: e.type,
          strain_label: e.type == "strain"
            ? NOT_NULL(d.label, "")
            : ""
        }
    )
    FILTER own != null
    FILTER @entity == "" OR @entity == own.entity
    SORT BM25(d) DESC, own.k ASC
    LIMIT @limit
    RETURN {
      k: own.k,
      id: own.id,
      entity: own.entity,
      sl: own.strain_label,
      f: %q,
      v: %s,
      s: %s
    }
)
```

Every branch returns the owning strain label in `sl`. It returns an empty string for a plasmid or a strain property document with no `label`. The label is result metadata, not a search field, so it does not add an AQL branch or a `StockSearchField` enum value.

The five verbs of both templates are: branch variable, `SEARCH` expression, field label, display expression, score expression.

The entity filter runs **inside** every branch, before the branch `SORT` and `LIMIT`. A design that filters after the merge loses a small result group, because the per-branch cap keeps rows of the wrong entity. A review of modware-order rejected the post-merge form; see `AUTOCOMPLETE.md` section 3 Step 4.

### Merge and tail

One `const` string in `statement/fullsearch.go`, filled with the comma-separated list of the 19 branch variables.

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
  LET stk = DOCUMENT(@stock_collection, x.k)
  RETURN {
    k: x.k,
    id: x.id,
    entity: x.entity,
    f: x.f,
    v: x.v,
    s: x.s,
    sm: NOT_NULL(stk.summary, ""),
    sl: x.sl
  }
```

The merge variable must not be named `all`, because `all` is a reserved word in AQL; the recorded compile error is in `AUTOCOMPLETE.md` section 4. The collect variable is `stkey` for the same reason.

The `DOCUMENT` lookup sits **after** the tail `LIMIT`, so it reads at most 50 documents per request, never one per candidate row. `NOT_NULL(stk.summary, "")` guards a stock document without a summary attribute. The lookup is a function call in a calculation step, not a collection scan, so it adds no `EnumerateCollectionNode` to the plan; Task 1 probe 20 records the real node list.

The row type, in `fullsearch.go`:

```go
// searchResultRow mirrors one object row of the full search projection.
// go-driver v1 cannot decode an array row, so the projection returns one
// object per row.
type searchResultRow struct {
	Key         string  `json:"k"`
	ID          string  `json:"id"`
	Entity      string  `json:"entity"`
	Field       string  `json:"f"`
	Value       string  `json:"v"`
	Summary     string  `json:"sm"`
	StrainLabel string  `json:"sl"`
	Score       float64 `json:"s"`
}
```

### Bind parameters

| Name | Go type | Value |
| --- | --- | --- |
| `@q` | `string` | the normalized query: `strings.ToLower(strings.TrimSpace(params.Query))`, subject to the Task 1 decision on accents |
| `@th` | `float64` | `fullSearchNgramThreshold` |
| `@limit` | `int` | the clamped limit |
| `@entity` | `string` | `string(params.Entity)`: `""`, `"strain"` or `"plasmid"` |
| `@stock_prop_graph` | `string` | `ar.stockc.stockPropType.Name()` |
| `@stock_collection` | `string` | `ar.stockc.stock.Name()` |

The view name is **not** a bind parameter. It is written into the template from `fullSearchViewName`.

`@limit` serves both the per-branch cap and the tail cap. A branch can therefore return up to `@limit` rows. The 19 branches together can produce up to `19 * @limit` rows before the merge. With the cap of 50, that is 950 rows at most. The merge reduces them in memory.

### Limit rules

| Request `limit` | protovalidate | Effective limit used by the handler and the repository | Rows returned at most |
| --- | --- | --- | --- |
| below 0 | rejected, `codes.InvalidArgument` | — | — |
| 0 | accepted | 50 | 50 |
| 1 to 50 | accepted | the requested value | the requested value |
| 51 to 100 | accepted | 50 | 50 |
| above 100 | rejected, `codes.InvalidArgument` | — | — |

The proto cap of 100 and the result cap of 50 are different on purpose. The proto cap keeps a very large request out of the server. The result cap carries the product rule from the spec: one list of 50 items. The handler and the repository clamp with the same rule. `Meta.Limit` therefore always equals the number of rows that the request can return.

| Meta field | Value |
| --- | --- |
| `Meta.Limit` | the effective limit from the table above |
| `Meta.Total` | the number of rows in `Data` |
| `Meta.NextCursor` | always 0, because there is no pagination |

### Handler signature and mapping

Add to `internal/app/service/fullsearch.go`:

```go
func (s *StockService) SearchStock(
	ctx context.Context,
	r *stock.StockSearchParameters,
) (*stock.StockSearchResultCollection, error)
```

Order of operations, exactly:

1. **Nil guard.** `if r.GetData() == nil || r.GetData().GetAttributes() == nil { return nil, aphgrpc.HandleInvalidParamError(ctx, errors.New("search request needs data and attributes")) }`. This runs **before** validation, because this repository has no recovery interceptor.
2. **Validation.** `if err := protovalidate.Validate(r); err != nil { return nil, aphgrpc.HandleInvalidParamError(ctx, err) }`. Do not call `r.Validate()`; it is a no-op.
3. **Trim guard.** Trim the query. When the trimmed query is empty, return an invalid-argument error. The proto rule counts characters, so `"  "` reaches this point.
4. **Effective limit.** Apply the table in [Limit rules](#limit-rules): 0 becomes 50, and a value from 51 to 100 becomes 50.
5. **Entity mapping.** `STOCK_ENTITY_UNSPECIFIED` to `repository.EntityBoth`, `STOCK_ENTITY_STRAIN` to `repository.EntityStrain`, `STOCK_ENTITY_PLASMID` to `repository.EntityPlasmid`. A value outside the three is unreachable after step 2, because of `enum.defined_only`; map it to an invalid-argument error anyway, so a future proto change cannot open a silent hole.
6. **Repository call.** `s.repo.SearchStock(&repository.FullSearchQuery{Query: trimmed, Entity: ent, Limit: int(effLimit)})`. A non-nil error maps to `aphgrpc.HandleGetError(ctx, err)`, like the other read handlers. An empty result is **not** a not-found error.
7. **Response.** Map each `*repository.FullSearchResult` to a `*stock.StockSearchResult`, including `Summary` and `StrainLabel`. Use fp-go composition in the style of `plasmid_mappers.go`. Set `Meta: &stock.Meta{NextCursor: 0, Limit: effLimit, Total: int64(len(rows))}.

Field mapping, repository string to proto enum:

| `FullSearchResult.Field` | `stock.StockSearchField` |
| --- | --- |
| `stock_id` | `StockSearchField_STOCK_SEARCH_FIELD_STOCK_ID` |
| `genes` | `StockSearchField_STOCK_SEARCH_FIELD_GENES` |
| `dbxrefs` | `StockSearchField_STOCK_SEARCH_FIELD_DBXREFS` |
| `label` | `StockSearchField_STOCK_SEARCH_FIELD_LABEL` |
| `names` | `StockSearchField_STOCK_SEARCH_FIELD_NAMES` |
| `species` | `StockSearchField_STOCK_SEARCH_FIELD_SPECIES` |
| `plasmid` | `StockSearchField_STOCK_SEARCH_FIELD_PLASMID` |
| `name` | `StockSearchField_STOCK_SEARCH_FIELD_NAME` |
| `summary` | `StockSearchField_STOCK_SEARCH_FIELD_SUMMARY` |
| `depositor` | `StockSearchField_STOCK_SEARCH_FIELD_DEPOSITOR` |
| anything else | `StockSearchField_STOCK_SEARCH_FIELD_UNSPECIFIED` |

Entity mapping, repository string to proto enum:

| `FullSearchResult.Entity` | `stock.StockEntity` |
| --- | --- |
| `strain` | `StockEntity_STOCK_ENTITY_STRAIN` |
| `plasmid` | `StockEntity_STOCK_ENTITY_PLASMID` |
| anything else | `StockEntity_STOCK_ENTITY_UNSPECIFIED` |

## File map

| File | Action | Content |
| --- | --- | --- |
| `scripts/full-search-calibration.js` | create | arangosh calibration harness: disposable database, prose and identifier fixtures, the 2 custom analyzers, the view, a threshold sweep, the probe matrix, EXPLAIN and latency output |
| `internal/repository/repository.go` | modify | `StockEntityFilter` and its 3 constants (guarded, see the shared declaration rule), `FullSearchQuery`, `FullSearchResult`, and the interface method `SearchStock` |
| `internal/repository/arangodb/fullsearch.go` | create | constants, analyzer setup, view creation and reconciliation, query builder, `searchResultRow`, row conversion, `SearchStock` |
| `internal/repository/arangodb/statement/fullsearch.go` | create | the 3 AQL `const` templates: stock branch, property branch, merge and tail |
| `internal/repository/arangodb/database.go` | modify | call `ar.ensureFullSearch(context.Background())` at the end of the `createDbStruct` chain |
| `internal/repository/arangodb/fullsearch_test.go` | create | repository tests against a real disposable database, plus builder unit tests |
| `internal/app/service/fullsearch.go` | create | the handler and the two mapping functions |
| `internal/app/service/fullsearch_test.go` | create | handler unit tests against a stub repository |
| `internal/app/service/fullsearch_arango_test.go` | create | one buffer-connection round trip against a real database, with the safe bufconn pattern |
| `README.md` | modify | the RPC contract in Simplified Technical English |
| `docs/superpowers/plans/2026-10-06-stock-full-search.md` | modify | fill the [Execution notes](#execution-notes) |

Note on `database.go`: the autocomplete plan adds its own call, `ar.ensureAutocompleteSearch(context.Background())`, to the same chain. Keep both calls. Do not replace the other call.

## Implementation tasks

### Task 1: Calibration against real ArangoDB 3.11

Run this task before any Go or AQL code is written. Every later task depends on its recorded output.

**Files:** create `scripts/full-search-calibration.js`.

**Produces:** these 9 recorded results.

1. The confirmed or corrected n-gram threshold.
2. The query normalization decision.
3. The proof of both traversal directions.
4. The behavior of `PHRASE` and of `IN TOKENS` with a bind parameter.
5. The stopword decision.
6. The multi-token noise measurement.
7. The measured `BM25` range.
8. The behavior of the array display expression.
9. The EXPLAIN node list and the latency baseline of the final 19-branch statement.

- [ ] **Step 1: Start the server.** The version must be 3.11, the same series as CI:

```bash
docker run -d --name arango311 -p 8529:8529 -e ARANGO_ROOT_PASSWORD=rootpass arangodb:3.11
docker exec arango311 arangosh --server.password rootpass --javascript.execute-string 'print(db._version())'
```

Record the exact version string in the [Execution notes](#execution-notes).

- [ ] **Step 2: Write `scripts/full-search-calibration.js`.** The script must:
  - create a disposable database with a random name;
  - create the collections `stock_calib` and `stock_properties_calib`, and the edge collection `stock_type_calib`;
  - create the named graph `stock_prop_calib`. Give it one edge definition, from `stock_calib` to `stock_properties_calib`. This definition mirrors `createNamedGraph` in `database.go`;
  - insert fixtures with the same shape as `statement/insert.go`: a stock document that sets `_key` equal to `stock_id`, a property document with an auto-generated `_key`, and an edge that carries `type`;
  - insert one strain with `stock_id` `DBS0236126`, `genes: ["sadA","DDB_G0348394"]`, `dbxrefs: ["d0319"]`, `depositor: "george@costanza.com"`, `summary: "the mutant forms culminants under starvation"`, `editable_summary` with a unique term that this feature must not search, property `label: "yS13"`, `names: ["Ax2","gammaS13"]`, `species: "Dictyostelium discoideum"`, `plasmid: "pDM304"`;
  - insert one strain whose summary holds the two words `forms` and `culminants` far apart and in the other order, for the phrase against token probe;
  - insert one strain whose summary holds only the word `forms`, for the single-token tail probe;
  - insert one plasmid with `stock_id` `DBP0000027`, `genes: ["sadA"]`, property `name: "pDM304"`, and **no** `summary` attribute, for the missing-summary probe;
  - insert one strain whose property document has **no** `names` attribute, for the absent-array probe;
  - insert one property document with **no** inbound edge, for the missing-owner probe;
  - insert 60 strains and 5 plasmids that all carry a gene with the prefix `xylose`, for the entity-filter probe;
  - insert about 1000 further stocks with random identifiers and random prose summaries, so the latency measurement is not taken on an empty index;
  - create the analyzers `stock_search_norm` and `stock_search_ngram` with the exact definitions of [Search asset names and constants](#search-asset-names-and-constants), and create **no** `text_en` analyzer;
  - create the view `stock_full_search` with the exact links of [View definition](#view-definition);
  - poll until the view answers a known query, because arangosh in `--javascript.execute` mode has no `wait()` helper.

- [ ] **Step 3: Run the probe matrix and record every result.** One small AQL statement per probe.

| # | Probe | Expected result |
| --- | --- | --- |
| 1 | `TOKENS("forms culminants", "text_en")` with a bind parameter | a non-empty token list; record the exact tokens, which show whether the stemmer changed them |
| 2 | prefix `dbs023` on `stock_id`, stock branch | one row, score above 1000, `k` equal to `DBS0236126`, and `strain_label` equal to `yS13` |
| 3 | prefix `ys` on `label`, property branch | one row whose `k` is the **stock** key `DBS0236126`, not the property key; `strain_label` is `yS13`; this proves the INBOUND direction |
| 4 | prefix `pdm` on the plasmid `name` | one row whose `k` is `DBP0000027`, whose `entity` is `plasmid`, and whose `strain_label` is empty |
| 5 | the same prefix without the `ANALYZER()` wrapper | zero rows; this pins the wrapper requirement |
| 6 | fuzzy `dbs0236127` at thresholds 0.2, 0.3, 0.45, 0.55, 0.65, 0.8, 1.0 | record the match at each threshold; 0.45 must match and 0.2 must admit junk that outranks the real row |
| 7 | `ANALYZER(d.summary IN TOKENS(@q, "text_en"), "text_en")` with the 1-word query `culminants` | one or more rows; record the count |
| 8 | the same token expression **without** the `ANALYZER()` wrapper | record the count; this shows whether the wrapper is required for the `IN TOKENS` form |
| 9 | `PHRASE(d.summary, @q, "text_en")` with the 2-word bind parameter `forms culminants` | exactly the strain whose summary holds the words in that order |
| 10 | `PHRASE` on the strain that holds the words in the other order | zero rows |
| 11 | `PHRASE` against a field linked with an analyzer that lacks `frequency` and `position` | an error or zero rows; this confirms why `text_en` is used |
| 12 | prefix `dictyo` on `species` | one row |
| 13 | token query `costanza` on `depositor` | one row |
| 14 | the full 19-branch statement with the query `dbs023` | the prefix row leads, and its score is above 1000 |
| 15 | the measured `BM25(d)` range over all branches on the 1000-document fixture | record the minimum and the maximum; the maximum must stay far below 250, the width of a score band |
| 16 | the full statement with the query `forms culminants` | the phrase strain leads, the token strain follows, and the single-word strain is present but last among the three |
| 17 | multi-token noise: the number of rows that match only one token of a 2-word query | record the count; this decides whether the token stage must become an all-token stage |
| 18 | `TOKENS("the and", "text_en")` | record whether the list is empty; this decides the stopword behavior |
| 19 | the full statement with the query `"  "` after trimming, and with the query `"---"` | zero rows, and no AQL error |
| 20 | entity filter inside the `genes` branch: query `xylose`, `@entity` `plasmid`, `@limit` 50, over the 60 strains and 5 plasmids | 5 plasmid rows. Run the same probe with the filter moved after the merge and record the smaller count; this measures the regression that the in-branch filter prevents. |
| 21 | one stock that matches `label` in a property branch, `genes` in a stock branch and `summary` in a token branch | one merged row, with the field attribution of the highest score |
| 22 | the array display expression on `names` for the prefix query `ax`, for a fuzzy-only hit, and on the document without `names` | the matched element, the joined list, and an empty string; never `null` and never an error |
| 23 | the property document with no inbound edge | dropped by `FILTER own != null` |
| 24 | the plasmid without a `summary` attribute | `sm` is an empty string, and no AQL error |
| 25 | `FLATTEN([...])` over the 19 branch variables | one flat list, not a list of lists |
| 26 | the full 19-branch query for a unique term that exists only in `editable_summary` | zero rows and no error; this field does not take part in search |

- [ ] **Step 4: EXPLAIN the final statement.** Assemble the full 19-branch statement by hand in the script and run `db._explain(stmt, bindVars)`. Assert both counts:

```text
number of EnumerateViewNode entries  == 19
number of EnumerateCollectionNode entries == 0
```

The `DOCUMENT()` call in the tail is a function in a calculation step, not a collection scan, so it must not add an `EnumerateCollectionNode`. If one appears anyway, record the node list and find which branch lost its view index.

- [ ] **Step 5: Measure the latency baseline.** Run the full statement 20 times against the warmed fixture of about 1000 documents, with the query `forms culminants` and `@limit` 50. Record the minimum, the median and the p95 in milliseconds. This p95 is the baseline that Task 6 compares against. Use the same query and the same limit in Task 6; a different query is not a comparison.

- [ ] **Step 6: Fold every correction into this plan.** Update [Search asset names and constants](#search-asset-names-and-constants), [Field and branch matrix](#field-and-branch-matrix), [Scoring and tie order](#scoring-and-tie-order), [Multi-token semantics](#multi-token-semantics) and [Bind parameters](#bind-parameters) when a probe contradicts them. Record each change in the [Execution notes](#execution-notes). Do not start Task 2 with an open contradiction.

Run command:

```bash
docker exec -i arango311 arangosh \
  --server.endpoint tcp://localhost:8529 \
  --server.username root --server.password rootpass \
  --javascript.execute /dev/stdin < scripts/full-search-calibration.js
```

- [ ] **Step 7: Commit.**

```bash
git add scripts/full-search-calibration.js
git commit -m "feat: add arangosearch calibration harness for stock full search"
```

### Task 2: Repository types and interface method

**Files:** modify `internal/repository/repository.go`.

- [ ] **Step 1: Apply the shared declaration rule** from [Repository types and method](#repository-types-and-method): grep for `StockEntityFilter`, and add the type and the 3 constants only when they are absent.

- [ ] **Step 2: Add `FullSearchQuery` and `FullSearchResult`** with the exact field names, types, order and doc comments of [Repository types and method](#repository-types-and-method). The field order places the four strings before the `float64`, which satisfies the field-alignment check.

- [ ] **Step 3: Add the interface method** `SearchStock(params *FullSearchQuery) ([]*FullSearchResult, error)` to `StockRepository`.

- [ ] **Step 4: Prove that the build breaks where it must.**

```bash
go build ./... 2>&1 | head
```

Expected: `*arangorepository` no longer satisfies `repository.StockRepository`. That failure is the signal that Task 4 must close. Record it.

### Task 3: Search assets in the constructor

**Files:** create `internal/repository/arangodb/fullsearch.go`; modify `internal/repository/arangodb/database.go`.

- [ ] **Step 1: Write the failing tests** in `internal/repository/arangodb/fullsearch_test.go`. Write flat sequential functions. Do not use a `t.Parallel` subtest. A parallel subtest resumes after the parent deferred functions run, and it then queries a dropped database.
  - `TestEnsureFullSearchCreatesAssets`: build the repository with `setUp(t)`. Assert through `repo.Dbh().Handler()` that both custom analyzers exist, and that `ViewExists` reports the view. Read the view properties. Assert the 2 link keys and the 10 field names. Assert the 2 custom analyzers on the 8 identifier fields, and `text_en` on the 2 prose fields. Assert also that the setup created **no** analyzer named `text_en`.
  - `TestEnsureFullSearchReconcilesStaleView`: create a disposable database with `testarango.NewTestArangoFromEnv(true)`. Create the view `stock_full_search` **before** the repository, with one wrong link and one wrong analyzer list. Call `NewStockRepo`. Read the properties, and assert the correct shape. Assert also that the view identifier did not change. An unchanged identifier proves that the code used `SetProperties` and not a delete.
  - `TestNewStockRepoFailsWhenFullSearchSetupFails`: point `manager.ConnectParams` at a database name that does not exist. Assert that `NewStockRepo` returns an error. Create that database with the arangomanager session. Assert that a second `NewStockRepo` call succeeds. This test pins the eager, retryable constructor contract. A pod restart is the retry.

- [ ] **Step 2: Run the tests and make sure that they fail.**

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'TestEnsureFullSearch|TestNewStockRepoFailsWhenFullSearchSetupFails' ./internal/repository/arangodb/
```

Expected: FAIL with an undefined symbol.

- [ ] **Step 3: Write the setup code** in `fullsearch.go`:
  - the constant block of [Search asset names and constants](#search-asset-names-and-constants);
  - `func (ar *arangorepository) ensureFullSearch(ctx context.Context) error`, which calls `EnsureCreatedAnalyzer` twice, for the norm analyzer and the pipeline analyzer only, and then `ensureFullSearchView(ctx)`;
  - `func (ar *arangorepository) fullSearchLinks() driver.ArangoSearchLinks`, which builds the links from `ar.stockc.stock.Name()` and `ar.stockc.stockProp.Name()`, with the analyzer lists of [View definition](#view-definition);
  - `func (ar *arangorepository) ensureFullSearchView(ctx context.Context) error`, which follows the 5 numbered rules of [View definition](#view-definition);
  - a link-shape comparison function that compares only the link keys, the field names and the sorted analyzer names. When the autocomplete plan already added such a function, reuse it instead of writing a second one; grep for `sameLinkShape` first. Wrap every error with `errors.Errorf` from `github.com/cockroachdb/errors`, as the rest of the package does.

- [ ] **Step 4: Wire the constructor.** In `database.go`, add the call at the end of the `createDbStruct` chain. When the autocomplete plan already landed, the chain ends with both calls:

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
	// Keep the autocomplete call when that plan has landed.
	if err := ar.ensureAutocompleteSearch(context.Background()); err != nil {
		return err
	}
	return ar.ensureFullSearch(context.Background())
}
```

When the autocomplete plan has **not** landed, drop the middle block and keep only the `ensureFullSearch` call. The setup is eager and idempotent. A failure stops the start of the service, so an operator sees the problem at once. Do not use `sync.Once`, which consumes its single run even on an error, and do not lock the request path.

- [ ] **Step 5: Make the tests pass**, then run the gates of [Quality gates](#quality-gates).

- [ ] **Step 6: Commit.**

```bash
git add internal/repository/repository.go internal/repository/arangodb/fullsearch.go internal/repository/arangodb/database.go internal/repository/arangodb/fullsearch_test.go
git commit -m "feat: create arangosearch assets for stock full search in the constructor"
```

### Task 4: Query builder and SearchStock

**Files:** create `internal/repository/arangodb/statement/fullsearch.go`; modify `internal/repository/arangodb/fullsearch.go` and `fullsearch_test.go`.

- [ ] **Step 1: Write the failing tests.** Add every row of the [Test matrix](#test-matrix) that belongs to the repository layer. Two groups:

  **Builder unit tests, no fixture.**
  - `TestBuildFullSearchQueryBranchCount`: the built statement holds 19 branch variables and 19 occurrences of `FOR d IN stock_full_search`. It holds 9 `OUTBOUND` traversals and 10 `INBOUND` traversals. It holds 8 `ANALYZER(STARTS_WITH`, 8 `NGRAM_MATCH`, 2 `IN TOKENS` and 1 `PHRASE(`. It has no `editable_summary` text.
  - `TestBuildFullSearchQueryHasNoNgramOnProse`: the statement holds no `NGRAM_MATCH` on `summary` or `depositor`.
  - `TestBuildFullSearchQueryIsValidAQL`: run `repo.Dbh().Handler().ValidateQuery(ctx, built)` and require no error.
  - `TestBuildFullSearchQueryFilterOrder`: in every branch, the index of `FILTER @entity` is lower than the index of `SORT BM25` and of `LIMIT @limit`.
  - `TestBuildFullSearchQueryDocumentLookupIsAfterLimit`: the index of `LIMIT @limit` in the tail is lower than the index of `DOCUMENT(@stock_collection`.

  **Repository tests against a real database.** Each one calls `setUp(t)`, inserts fixtures with `repo.AddStrain` or the `F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)` form, waits for the view commit with `require.Eventually`, then calls `repo.SearchStock`. The names are listed in the [Test matrix](#test-matrix).

  Wait helper, used by every database test:

```go
	assert.Eventually(func() bool {
		rows, err := repo.SearchStock(&repository.FullSearchQuery{Query: probe, Limit: 50})
		return err == nil && len(rows) > 0
	}, 20*time.Second, 500*time.Millisecond, "the view must commit the new documents")
```

The view commits in the background, about one second after an insert. A test that queries at once fails without this poll.

- [ ] **Step 2: Run the tests and make sure that they fail.**

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'TestSearchStock|TestBuildFullSearchQuery' ./internal/repository/arangodb/
```

Expected: FAIL with an undefined method.

- [ ] **Step 3: Write the templates** in `statement/fullsearch.go`. Three exported `const` strings, with a doc comment that lists the `fmt.Sprintf` verbs and the required bind parameters:
  - `FullSearchStockBranch` — the stock collection branch of [AQL branch templates](#aql-branch-templates);
  - `FullSearchPropBranch` — the property collection branch;
  - `FullSearchMerge` — the merge and tail of [Merge and tail](#merge-and-tail).

- [ ] **Step 4: Write the builder and the method** in `fullsearch.go`:
  - a package-level table of the 19 branches, in the exact order of [Field and branch matrix](#field-and-branch-matrix). Each entry holds the field label, the branch family, the stage, and the display-expression kind. Derive the `SEARCH` expression and the score expression from the stage, so a new stage needs one table row and no new template;
  - `func buildFullSearchQuery() string`, which loops over the table, emits one branch per entry, collects the branch variable names, and appends the merge template. Assign the result to a package-level `var fullSearchQuery = buildFullSearchQuery()`, so the statement is built once per process;
  - `func (ar *arangorepository) SearchStock(params *repository.FullSearchQuery) ([]*repository.FullSearchResult, error)`:
    1. reject a nil `params` with an error;
    2. normalize the query per the Task 1 decision; the baseline is `strings.ToLower(strings.TrimSpace(params.Query))`;
    3. return an error when the normalized query is empty;
    4. clamp the limit: at or below 0 becomes `defaultFullSearchLimit`, above `maxFullSearchLimit` becomes `maxFullSearchLimit`;
    5. accept only `repository.EntityBoth`, `repository.EntityStrain` and `repository.EntityPlasmid`; any other value returns an error;
    6. apply the stopword rule that Task 1 probe 18 decided, if that probe asked for one;
    7. call `ar.database.SearchRows(fullSearchQuery, bindVars)` with the 6 bind parameters of [Bind parameters](#bind-parameters);
    8. `defer result.Close()`;
    9. loop `for result.Scan()`, read one `searchResultRow` at a time, and append;
    10. convert the rows to `[]*repository.FullSearchResult` and return. Map `searchResultRow.StrainLabel` to `FullSearchResult.StrainLabel`. Return an empty, non-nil slice when there is no row.
  - Use fp-go style for the conversion pipeline, as the rest of this repository does. Load the `fp-go`, `fp-go-pipe-flow` and `fp-go-predicates` skills before you write it. Keep the alias `IOE` for `ioeither`.

- [ ] **Step 5: Make the tests pass**, then run the gates of [Quality gates](#quality-gates).

- [ ] **Step 6: Commit.**

```bash
git add internal/repository/arangodb/fullsearch.go internal/repository/arangodb/statement/fullsearch.go internal/repository/arangodb/fullsearch_test.go
git commit -m "feat: add full stock search with 50 item retrieval to the repository"
```

### Task 5: Service handler

**Files:** create `internal/app/service/fullsearch.go`, `fullsearch_test.go` and `fullsearch_arango_test.go`.

- [ ] **Step 1: Write the failing tests.**

  **Unit tests with a stub repository, in `fullsearch_test.go`.** The stub must implement every method of `repository.StockRepository` that exists on the branch. The current set is `GetStrain`, `GetPlasmid`, `AddStrain`, `AddPlasmid`, `EditStrain`, `EditPlasmid`, `ListStrains`, `ListStrainsByIDs`, `ListPlasmids`, `LoadStrain`, `LoadPlasmid`, `RemoveStock`, `Dbh`, `LoadOboJSON`, plus `SearchStock` from this plan. If the autocomplete plan already landed, `AutocompleteStock` is also required. Discover the real set with `go build ./...` and add the missing stubs. Name the type `fullSearchStubRepo`, so it cannot collide with the stub of the autocomplete plan. Give it the fields `hits []*repository.FullSearchResult`, `err error`, and the captured input `got *repository.FullSearchQuery`.
  - `TestSearchStockHandlerMapsResults`: two results map to two `StockSearchResult` values, with the right enum field, the right enum entity, the display text, the score, the complete summary and `strain_label`. Verify that a strain label passes through and a plasmid result keeps an empty label. `Meta.Total` is 2, `Meta.Limit` is the effective limit, `Meta.NextCursor` is 0.
  - `TestSearchStockHandlerPassesThroughQueryAndEntity`: the captured `got.Query` is the trimmed query, and `got.Entity` is the mapped filter.
  - `TestSearchStockHandlerDefaultsLimitToFifty`: a request with `Limit` 0 gives `got.Limit == 50` and `Meta.Limit == 50`.
  - `TestSearchStockHandlerClampsLimitAboveFifty`: a request with `Limit` 80 is accepted by protovalidate and gives `got.Limit == 50` and `Meta.Limit == 50`.
  - `TestSearchStockHandlerRejectsNilData`: `&stock.StockSearchParameters{}` gives `codes.InvalidArgument` and no panic.
  - `TestSearchStockHandlerRejectsNilAttributes`: `Data` present and `Attributes` nil gives `codes.InvalidArgument` and no panic.
  - `TestSearchStockHandlerRejectsShortQuery`: a 1-character query gives `codes.InvalidArgument`.
  - `TestSearchStockHandlerRejectsWhitespaceQuery`: the query `"  "` gives `codes.InvalidArgument`. This closes the proto gap.
  - `TestSearchStockHandlerRejectsLimitAboveHundred`: `Limit` 101 gives `codes.InvalidArgument`.
  - `TestSearchStockHandlerRejectsUndefinedEntity`: `stock.StockEntity(9)` gives `codes.InvalidArgument`.
  - `TestSearchStockHandlerReturnsEmptyCollection`: an empty repository result gives no error, an empty `Data` list and `Meta.Total` 0.
  - `TestSearchStockHandlerKeepsEmptySummary`: a result with an empty `Summary` maps to an empty string, not to a missing field or a panic.
  - `TestSearchStockHandlerMapsRepositoryError`: a repository error gives `codes.Internal`, which is what `aphgrpc.HandleGetError` produces for the other read handlers. Assert the code that the helper really returns, and record it.

  **One round trip, in `fullsearch_arango_test.go`.** Use the safe bufconn pattern. Do **not** reuse `setupGrpcServer` or `setupGrpcClient` from `strain_test_helpers.go`; both are unsafe, as recorded in [Service and test plumbing](#service-and-test-plumbing).

```go
func newFullSearchArangoEnv(t *testing.T) (stock.StockServiceClient, *require.Assertions) {
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
  - `TestSearchStockEndToEnd`: create a strain with a known summary and label through `client.CreateStrain`. Poll with `require.Eventually` until `client.SearchStock` returns a row. Then assert the identifier, the entity enum, the field enum, the display text, the complete summary, `strain_label` and `Meta`. The response label must equal the strain property `label`.

- [ ] **Step 2: Run the tests and make sure that they fail.**

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'SearchStock' ./internal/app/service/
```

Expected: FAIL, because `StockService` has no such method. The embedded `stock.UnimplementedStockServiceServer` makes the gRPC call return `codes.Unimplemented` instead of a compile error, so assert on the method result, not on compilation.

- [ ] **Step 3: Write the handler** in `internal/app/service/fullsearch.go`, in the exact order of [Handler signature and mapping](#handler-signature-and-mapping). Put the two mapping functions in the same file, and write them as pure functions, so the unit tests can cover them directly. When the autocomplete plan already landed, its mapping functions can cover the same 8 identifier labels. Reuse them, and extend the field map with the 2 prose labels. Map `FullSearchResult.StrainLabel` to `StockSearchResult.strain_label`. Do not add a `StockSearchField` value for this metadata. Do not write a second map.

- [ ] **Step 4: Make the tests pass**, then run the gates of [Quality gates](#quality-gates).

- [ ] **Step 5: Commit.**

```bash
git add internal/app/service/fullsearch.go internal/app/service/fullsearch_test.go internal/app/service/fullsearch_arango_test.go
git commit -m "feat: add the full stock search grpc handler"
```

### Task 6: EXPLAIN and latency gate

- [ ] **Step 1: Print the statement that the Go builder produced.** Add a short temporary program or a test with the `-v` flag that writes `fullSearchQuery` to standard output, and save it to `/tmp/fullsearch.aql`. The statement under test must be the built one, not the hand-written one of Task 1.

- [ ] **Step 2: EXPLAIN it** against the Task 1 fixture database:

```bash
docker exec -i arango311 arangosh --server.username root --server.password rootpass \
  --server.database <calib-db> --javascript.execute-string \
  'var q=require("fs").read("/tmp/fullsearch.aql"); print(JSON.stringify(db._explain(q, {q:"forms culminants", th:0.45, limit:50, entity:"", stock_prop_graph:"stock_prop_calib", stock_collection:"stock_calib"}), null, 2));'
```

Assert 19 `EnumerateViewNode` entries and 0 `EnumerateCollectionNode` entries. Record the node list.

- [ ] **Step 3: Measure the latency** of the same statement, on the same warmed fixture, with the same query `forms culminants` and the same limit 50, over 20 runs. The gate is: **p95 at or below 125 percent of the Task 1 baseline p95**. Record both numbers.

- [ ] **Step 4: When the gate fails**, do not raise the threshold of the gate. Find the cause. These 4 causes are known:
  1. a lost view index on a branch;
  2. a traversal that runs before the branch `LIMIT`;
  3. a `DOCUMENT` lookup that moved before the tail `LIMIT`;
  4. a display expression that the optimizer pushed into the index.

  Record the cause and the fix.

### Task 7: Documentation

**Files:** modify `README.md`.

- [ ] **Step 1: Describe the RPC** in Simplified Technical English. Cover these 11 items:
  1. the method name;
  2. the request message and the response message;
  3. the floor of 2 characters on the query;
  4. the default of 50, the result cap of 50, and the proto cap of 100;
  5. the 10 searched fields;
  6. the 4 match stages and their rank bands;
  7. the multi-token rule, which is that the token stage matches any token;
  8. the meaning of the entity filter;
  9. the complete `summary` value and the `strain_label` rule for strains, plasmids and missing labels;
  10. the absence of pagination, and the rule that an empty list is valid;
  11. the 4 environment variables that the tests read: `ARANGO_HOST`, `ARANGO_USER`, `ARANGO_PASS`, and the optional `ARANGO_PORT`.

- [ ] **Step 2: Name the search assets**, so an operator can find them. The names are the analyzers `stock_search_norm` and `stock_search_ngram`, the built-in analyzer `text_en`, and the view `stock_full_search`. State that the repository constructor creates the two custom analyzers and the view. State that `text_en` is built in. State also that a failed creation stops the start of the service.

- [ ] **Step 3: Commit.**

```bash
git add README.md
git commit -m "docs: describe the full stock search rpc"
```

## Test matrix

### Repository tests, real disposable ArangoDB 3.11

| # | Test | Fixture | Assertion |
| --- | --- | --- | --- |
| 1 | `TestSearchStockTokenMatchInSummary` | 1 strain with label `yS13` whose summary holds the word `culminants` | the query `culminants` returns 1 row; `Field` is `summary`; `Score` is between 250 and 500; `StrainLabel` is `yS13` |
| 2 | `TestSearchStockDoesNotSearchEditableSummary` | 1 strain whose `editable_summary` holds a unique word that appears nowhere else | the query returns an empty list and a nil error |
| 3 | `TestSearchStockTokenMatchInDepositor` | 1 strain with `depositor` `george@costanza.com` | the query `costanza` returns 1 row; `Field` is `depositor` |
| 4 | `TestSearchStockPhraseOutranksToken` | 1 strain whose summary holds `forms culminants` in order, and 1 whose summary holds both words apart and in the other order | the query `forms culminants` puts the phrase strain first; its score is between 500 and 1000, and the other score is between 250 and 500 |
| 5 | `TestSearchStockPrefixOutranksPhrase` | 1 strain whose `stock_id` starts with the query, and 1 whose summary holds the query as a phrase | the prefix row is first, with a score above 1000 |
| 6 | `TestSearchStockMultiTokenMatchesAnyToken` | 1 strain whose summary holds only `forms` | the 2-word query `forms culminants` returns that strain too, below the phrase row and the full-token row |
| 7 | `TestSearchStockFuzzyRanksLast` | 1 strain with a misspelled identifier match, and 1 with a token match | the token row ranks above the fuzzy row |
| 8 | `TestSearchStockReturnsCompleteSummary` | 1 strain with a long summary | `Summary` equals the stored summary exactly, with no truncation and no ellipsis |
| 9 | `TestSearchStockPlasmidMetadataAndMissingSummary` | 1 plasmid with a unique `name` and no `summary` attribute | the name match has `Summary` empty and `StrainLabel` empty |
| 10 | `TestSearchStockIdentifierPrefixFields` | 1 strain with label `yS13` and 1 plasmid | one assertion block per identifier field proves a prefix match on all 8 fields; the strain gene hit has `StrainLabel` `yS13`, and the plasmid name hit has an empty `StrainLabel` |
| 11 | `TestSearchStockDefaultLimitIsFifty` | 60 strains that match | `Limit` 0 returns 50 rows |
| 12 | `TestSearchStockClampsLimitToFifty` | 60 strains that match | `Limit` 100 returns 50 rows |
| 13 | `TestSearchStockEntityFilterKeepsSmallGroup` | 60 strains and 5 plasmids, each with a gene that starts with `xylose` | `Entity` `plasmid` and `Limit` 50 return exactly the 5 plasmids. This fails when the entity filter runs after the merge. |
| 14 | `TestSearchStockCrossCollectionMerge` | 1 strain that matches `label`, `genes` and `summary` for one query | exactly 1 row, with the field of the highest score |
| 15 | `TestSearchStockDuplicateHitKeepsBestField` | 1 strain that matches a prefix branch and a token branch | 1 row, and `Field` is the prefix field, not the prose field |
| 16 | `TestSearchStockEmptyResultIsNotAnError` | 1 strain | the query `zzzqqq` returns an empty, non-nil slice and a nil error |
| 17 | `TestSearchStockOneTokenQuery` | 1 strain | a 2-character query such as `ax` returns the row that holds it, and raises no error |
| 18 | `TestSearchStockPunctuationOnlyQuery` | 1 strain | the query `---` returns an empty list and a nil error |
| 19 | `TestSearchStockStopwordOnlyQuery` | 1 strain with prose | the query `the and` behaves as Task 1 probe 18 recorded: either an empty list, or a bounded noisy list, or an error from the Go-side stopword rule. Assert the recorded behavior, not an assumption. |
| 20 | `TestSearchStockRejectsWhitespaceQuery` | none | the query `"  "` returns an error, and no AQL runs |
| 21 | `TestSearchStockRejectsUnknownEntity` | none | `Entity` `"vector"` returns an error |
| 22 | `TestSearchStockHandlesMissingPropertyMetadata` | 1 orphan property with `label: "orphanonly"` and no edge; 1 strain stock with `summary: "missinglabelmarker"` and a linked property document without `label` | query `orphanonly` returns no row; query `missinglabelmarker` returns the linked strain with an empty `StrainLabel` and no error |
| 23 | `TestSearchStockArrayDisplayFallback` | 1 strain whose `names` match only the fuzzy stage | `DisplayText` is the joined list and is never empty or `null` |
| 24 | `TestEnsureFullSearchCreatesAssets` | a fresh repository | both custom analyzers exist; no `text_en` analyzer was created; the view exists with 2 links and 10 fields, with the right analyzer list per field |
| 25 | `TestEnsureFullSearchReconcilesStaleView` | a view pre-created with wrong links | after `NewStockRepo` the link shape matches the definition, and the view was not deleted |
| 26 | `TestNewStockRepoFailsWhenFullSearchSetupFails` | a missing database | the first call fails; after the database is created the second call succeeds |

### Builder unit tests, no fixture

| # | Test | Assertion |
| --- | --- | --- |
| 27 | `TestBuildFullSearchQueryBranchCount` | 19 branch variables; 19 `FOR d IN stock_full_search`; 9 `OUTBOUND`; 10 `INBOUND`; 8 `ANALYZER(STARTS_WITH`; 8 `NGRAM_MATCH`; 2 `IN TOKENS`; 1 `PHRASE(`; no `editable_summary` text in the query |
| 28 | `TestBuildFullSearchQueryHasNoNgramOnProse` | the statement holds no `NGRAM_MATCH` over `summary` or `depositor` |
| 29 | `TestBuildFullSearchQueryIsValidAQL` | `ValidateQuery` returns no error for the built statement |
| 30 | `TestBuildFullSearchQueryFilterOrder` | in each branch the `FILTER @entity` index is lower than the `SORT BM25` index and the `LIMIT @limit` index |
| 31 | `TestBuildFullSearchQueryDocumentLookupIsAfterLimit` | the tail `LIMIT @limit` index is lower than the `DOCUMENT(@stock_collection` index |
| 32 | `TestFullSearchFieldMapping` | every one of the 10 searched field labels maps to a defined `StockSearchField`, and an unknown label maps to `UNSPECIFIED` |

### Handler tests, stub repository

Rows 33 to 45 are the 13 unit tests named in Task 5, Step 1.

### Handler test, one real round trip

| # | Test | Assertion |
| --- | --- | --- |
| 46 | `TestSearchStockEndToEnd` | a strain with label `yS13`, created over the buffer connection, is found by `client.SearchStock` | the response carries the identifier, entity enum, field enum, display text, complete summary, `strain_label` `yS13`, `Meta.Total` 1, `Meta.Limit` 50 and `Meta.NextCursor` 0 |

## Failure behavior

| Input or condition | Layer | Behavior |
| --- | --- | --- |
| `data` is nil | handler | `codes.InvalidArgument`, no panic, no repository call |
| `attributes` is nil | handler | `codes.InvalidArgument`, no panic, no repository call |
| query shorter than 2 characters | handler, protovalidate | `codes.InvalidArgument` |
| query of 2 or more characters that trims to empty, for example `"  "` | handler, trim guard | `codes.InvalidArgument`. The proto rule accepts it, so the handler must reject it. |
| punctuation-only query, for example `"---"` | repository and ArangoDB | an empty list and a nil error. The prefix stage finds nothing, `TOKENS` gives no usable token, and the fuzzy stage finds nothing above the threshold. |
| stopword-only query, for example `"the and"` | repository and ArangoDB | the behavior that Task 1 probe 18 recorded. Either an empty list, or a bounded noisy list in the 250-point band, or an invalid-argument error from the documented Go-side stopword rule. The chosen behavior is pinned by `TestSearchStockStopwordOnlyQuery`. |
| one-token query | repository | the token stage, the prefix stage and the fuzzy stage all apply. The phrase stage degenerates to a single-word phrase and gives the same rows as the token stage, at a higher band. This is harmless. |
| multi-token query | repository | the token stage matches **any** token, the phrase stage matches the ordered sequence, and the prefix stage matches an identifier prefix. The bands keep the strong rows at the head. |
| `limit` below 0 or above 100 | handler, protovalidate | `codes.InvalidArgument` |
| `limit` 0 | handler and repository | the effective limit is 50 |
| `limit` from 51 to 100 | handler and repository | the effective limit is 50, and `Meta.Limit` is 50 |
| `entity` outside 0, 1, 2 | handler, protovalidate and the explicit map default | `codes.InvalidArgument` |
| `Entity` outside the 3 repository constants | repository | an error, before any AQL runs |
| no stock matches | repository and handler | an empty, non-nil `Data` list, `Meta.Total` 0, and a nil error. This is never a not-found error. |
| a property document with no inbound edge | AQL | dropped by `FILTER own != null` |
| a stock document with no outbound edge | AQL | dropped by `FILTER ent != null`. Such a stock cannot be classified, so it cannot be filtered or labeled. |
| the stock document has no `summary` attribute | AQL | `NOT_NULL(stk.summary, "")` gives an empty string |
| the result is a plasmid | AQL | `strain_label` is an empty string |
| a strain property document has no `label` | AQL | `strain_label` is an empty string through `NOT_NULL` |
| an array attribute is absent | AQL | the display expression gives an empty string through `NOT_NULL(d.<field>, [])` and `CONCAT_SEPARATOR` |
| a fuzzy array hit whose elements all fail `CONTAINS` | AQL | the display expression falls back to the joined list |
| one stock matches in several branches | AQL | the merge keeps one row, with the highest score, and breaks a tie on the field label in ascending order |
| the view exists with the wrong links | constructor | `SetProperties` rewrites the links. The view is not deleted. The index rebuilds in the background, so the first queries after a restart can return fewer rows. |
| a non-ArangoSearch view holds the name `stock_full_search` | constructor | an error from `ArangoSearchView()`. The start of the service stops. Do not delete the other view. |
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
  internal/repository/arangodb/fullsearch.go \
  internal/repository/arangodb/statement/fullsearch.go \
  internal/repository/arangodb/database.go \
  internal/repository/arangodb/fullsearch_test.go \
  internal/app/service/fullsearch.go \
  internal/app/service/fullsearch_test.go \
  internal/app/service/fullsearch_arango_test.go
```

Focused runs during development:

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'TestSearchStock|TestBuildFullSearchQuery|TestEnsureFullSearch' ./internal/repository/arangodb/
gotestsum --format-hide-empty-pkg --format dots -- -run 'SearchStock' ./internal/app/service/
```

Rules that hold for every gate:

- Repository code is tested against a real disposable ArangoDB database, one per test, created by `testarango.NewTestArangoFromEnv(true)` and dropped in `t.Cleanup` or `tearDown`.
- Handlers are tested twice: against a stub repository for every input class, and over one real buffer connection for the full round trip.
- Write flat sequential tests. Do not use `t.Parallel` in a subtest that shares a database with its parent.
- The server version under test is ArangoDB 3.11, the same series as CI.
- CI needs no change. The workflow already starts `arangodb:3.11` and exports the four variables.

## Completion criteria

- [ ] `scripts/full-search-calibration.js` exists, runs against ArangoDB 3.11, and its 26 probe results plus the EXPLAIN and latency baseline are recorded in the [Execution notes](#execution-notes).
- [ ] Every contradiction that calibration found is folded into this plan, and the fold is recorded.
- [ ] The stopword decision and the multi-token decision are recorded, and a test pins each one.
- [ ] `internal/repository/repository.go` declares `StockEntityFilter` with its 3 constants, `FullSearchQuery`, `FullSearchResult`, and the interface method `SearchStock(params *FullSearchQuery) ([]*FullSearchResult, error)`.
- [ ] `NewStockRepo` creates both custom analyzers and the view `stock_full_search` eagerly, creates no `text_en` analyzer, and fails at start when the setup fails.
- [ ] A stale view converges to the definition through `SetProperties`, with no delete.
- [ ] The built AQL statement holds 19 branches. It does not name `editable_summary`, and it has no n-gram match over a prose field.
- [ ] The entity filter sits before every `SORT` and `LIMIT`, and the `DOCUMENT` lookup sits after the tail `LIMIT`.
- [ ] `SearchStock` returns at most 50 rows, 50 by default, and an empty list is not an error.
- [ ] Every result carries the complete stored summary, and a missing summary gives an empty string.
- [ ] Each strain result carries the property `label` in `strain_label`. A plasmid result or a strain without a label carries an empty string.
- [ ] `StockService.SearchStock` guards nil `data` and nil `attributes`. It validates with `protovalidate.Validate`, and it rejects a whitespace-only query.
- [ ] `StockService.SearchStock` clamps a limit from 51 to 100 down to 50.
- [ ] `StockService.SearchStock` fills `Meta` with `Total` as the row count, `Limit` as the effective limit, and `NextCursor` 0.
- [ ] Every row of the [Test matrix](#test-matrix) has a test, and all of them pass.
- [ ] The EXPLAIN check reports 19 `EnumerateViewNode` entries and 0 `EnumerateCollectionNode` entries.
- [ ] The measured p95 latency is at or below 125 percent of the calibration baseline p95. The measurement uses the same warmed fixture, the same query and the same limit.
- [ ] Every gate in [Quality gates](#quality-gates) passes.
- [ ] `README.md` describes the RPC, the limits, the 10 searched fields, the 4 stages, the `strain_label` metadata rule and the asset names.

## Handoff artifacts

| # | Artifact | Where it lives | Who consumes it |
| --- | --- | --- | --- |
| 1 | The confirmed n-gram threshold, the query normalization decision, the stopword decision and the multi-token decision | [Execution notes](#execution-notes) | a later tuning task, and a reviewer |
| 2 | The 26 probe results of Task 1, including the measured `BM25` range | [Execution notes](#execution-notes) | a reviewer, and any later change of a score band |
| 3 | The EXPLAIN node list and the latency baseline p95 | [Execution notes](#execution-notes) | Task 6 of this plan, and any later change of the statement |
| 4 | `repository.StockEntityFilter`, `repository.EntityBoth`, `repository.EntityStrain`, `repository.EntityPlasmid` | `internal/repository/repository.go` | the autocomplete plan, which reuses the same declarations under its own shared declaration rule |
| 5 | The safe bufconn pattern in `internal/app/service/fullsearch_arango_test.go` | the test file | the autocomplete plan, which needs the same pattern in its own file |
| 6 | The asset names `stock_search_norm`, `stock_search_ngram`, `text_en` and `stock_full_search` | `README.md` and `internal/repository/arangodb/fullsearch.go` | an operator, and the autocomplete plan, which must not reuse them |
| 7 | A list of follow-up issues: an evaluation of `optimizeTopK`, an all-token mode for the token stage if the tail is noisy, snippet support, nil guards for the existing handlers, a replacement of the unsafe `setupGrpcServer` and `setupGrpcClient` helpers, and a shared bufconn helper for the two search test files | the issue tracker | a later session |

## Review focus

Input classes and risks that no single task fully owns. Each has a named owner test.

1. **Multi-word queries.** The token stage matches any token by design, so a document with one word of a two-word query appears in the result. The phrase band and the prefix band must own the head of the list, and the single-token rows must fill the tail. Owner: `TestSearchStockPhraseOutranksToken` and `TestSearchStockMultiTokenMatchesAnyToken`.
2. **Do not search `editable_summary`.** The field stays stored, but the view and query do not use it. Owners: `TestSearchStockDoesNotSearchEditableSummary` and `TestBuildFullSearchQueryBranchCount`.
3. **No n-gram on prose.** An n-gram index over a long summary grows fast and returns weak matches. A reviewer checks that `NGRAM_MATCH` only searches identifier and name fields. Owner: `TestBuildFullSearchQueryHasNoNgramOnProse`.
4. **Score bands against the measured `BM25` range.** The bands are 250 apart. If `BM25` on real data ever reaches 250, a fuzzy row can outrank a token row. Owner: Task 1 probe 15, plus `TestSearchStockFuzzyRanksLast`.
5. **Property ownership through the graph.** A property `_key` is auto-generated. Every property-branch row must resolve its stock through the INBOUND traversal, or the branch must return nothing. A row whose `ID` is a property key is a defect. Owner: `TestSearchStockIdentifierPrefixFields` and `TestSearchStockCrossCollectionMerge`.
6. **Entity-filter loss.** The filter must sit inside each branch, before the branch `SORT` and `LIMIT`. Probe 20 of Task 1 measures the loss of the wrong design, and the test pins the right one. Owner: `TestSearchStockEntityFilterKeepsSmallGroup`.
7. **The summary is complete, not a snippet.** The tail must read the stored value and return it unchanged, and it must read it only for the rows that it returns. Owner: `TestSearchStockReturnsCompleteSummary` and `TestBuildFullSearchQueryDocumentLookupIsAfterLimit`.
8. **Missing summary and label.** A plasmid without a `summary` attribute must give an empty string. A plasmid or a strain without a property `label` must give an empty `strain_label`. Owner: `TestSearchStockPlasmidMetadataAndMissingSummary`, `TestSearchStockHandlesMissingPropertyMetadata` and `TestSearchStockHandlerKeepsEmptySummary`.
9. **Nil `data` and nil `attributes`.** This repository has no recovery interceptor, so a nil dereference kills the process. Owner: `TestSearchStockHandlerRejectsNilData` and `TestSearchStockHandlerRejectsNilAttributes`.
10. **Whitespace-only query.** `string.min_len` counts characters, so the proto accepts `"  "`. Owner: `TestSearchStockHandlerRejectsWhitespaceQuery` and `TestSearchStockRejectsWhitespaceQuery`.
11. **The two limit caps.** The proto cap is 100 and the result cap is 50. A request of 80 must be accepted and must return at most 50 rows, with `Meta.Limit` 50. Owner: `TestSearchStockHandlerClampsLimitAboveFifty` and `TestSearchStockClampsLimitToFifty`.
12. **View commit delay and a stale view.** New documents are invisible for about one second, and an old view definition must converge without a delete. Owner: the `require.Eventually` wrapper in every database test, plus `TestEnsureFullSearchReconcilesStaleView`.
13. **Independence from the autocomplete plan.** This plan must build, test and run with the autocomplete assets absent. A reviewer checks that no identifier of this plan names `stock_autocomplete_norm`, `stock_autocomplete_ngram` or `stock_autocomplete`. A reviewer also checks that `database.go` keeps both setup calls when both plans have landed.
14. **The no-op generated validator.** A handler that calls `r.Validate()` validates nothing. Owner: `TestSearchStockHandlerRejectsShortQuery`, which fails when the handler uses the wrong validator.

## Execution notes

Fill these during implementation. They are part of the handoff.

- ArangoDB version string from Task 1 Step 1: **3.11.14** (container `arango311`, host port 8530; a 3.12 server holds 8529 locally).
- Probe results 1 to 26 (harness `scripts/full-search-calibration.js`, disposable db `stock_calib_1791402364516`, 1071 stock docs):
  1. `TOKENS(@q, "text_en")` with bind param returns `["form", "culmin"]` — the stemmer rewrites both words; the bind-parameter form works.
  2. Prefix `dbs023` on `stock_id`: 1 row, `s` 1000, `k` `DBS0236126`, `strain_label` `yS13`. Matches expectation.
  3. Prefix `ys` on `label`: 1 row whose `k` is the stock key `DBS0236126`, `strain_label` `yS13`. INBOUND direction proven.
  4. Prefix `pdm` on plasmid `name`: `k` `DBP0000027`, entity `plasmid`, `strain_label` empty. Matches expectation.
  5. Same prefix without `ANALYZER()`: 0 rows. Wrapper required, as planned.
  6. Fuzzy `dbs0236127` sweep: **0.2 → 15 rows of junk plus the real row; 0.3 → exactly the real row; 0.45, 0.55, 0.65, 0.8, 1.0 → 0 rows.** The assumed 0.45 fails; folded to **0.30** (see the constants block). Same conclusion as the autocomplete calibration.
  7. `ANALYZER(d.summary IN TOKENS(@q, "text_en"), "text_en")` with `culminants`: 2 rows (the phrase strain and the far-apart strain).
  8. Same without the wrapper: 0 rows. The `IN TOKENS` form needs the `ANALYZER()` wrapper, as planned.
  9. `PHRASE(d.summary, @q, "text_en")` with `forms culminants`: exactly the in-order strain (score 509.68). Bind-parameter phrase works.
  10. `PHRASE` with the other-order bind value `culminants forms`: 0 rows.
  11. `PHRASE` on `editable_summary` linked with the featureless `stock_search_norm` analyzer: **returned 1 row with score 500, no error.** The plan's expectation (error or 0 rows) is wrong — PHRASE still matched on the norm analyzer. The reason to keep `text_en` on prose stands regardless: prose needs tokenization and stemming, which a norm analyzer does not supply.
  12. Prefix `dictyo` on `species`: 2 rows (both strains share the `Dictyostelium` genus). Prefix match works.
  13. Token `costanza` on `depositor`: 1 row, score 250.29.
  14. Full statement with `dbs023`: prefix row leads at 1000; a fuzzy `stock_id` row follows at 16.6.
  15. Measured BM25 range: token branch 3.07–8.18 (`calibword` in 34 docs, `forms` in 3 docs); fuzzy branch 18.95–39.52 (`dbs0236127` at threshold 0.2, 15 docs); pure-prefix rows score BM25 0. **Maximum 39.5, far below the 250-point band width; the bands hold.**
  16. Full statement with `forms culminants`: phrase strain leads (509.7), both-words strain follows (259.0), single-word strain last (258.2). Matches the plan.
  17. Multi-token noise: **1 of 3 rows matched only one token** (the `forms`-only strain). The tail stays small and bounded by the limit; decision: keep the **any-token** form of `IN TOKENS`.
  18. `TOKENS("the and", "text_en")` returns `["the", "and"]` — **`text_en` does NOT remove stopwords**. The full statement with `the and` returned 2 rows in the 250-point band (bounded noise). Decision: the repository applies a **Go-side stopword list** and rejects an all-stopword query with an error, so a stopword-only request never reaches AQL. `TestSearchStockStopwordOnlyQuery` pins the error.
  19. Whitespace query (trimmed empty) and `---`: 0 rows, no AQL error.
  20. Entity filter: in-branch `plasmid` filter over 65 `xylose` gene docs returns exactly the 5 plasmids. Post-merge variant also returned 5 here (at equal BM25 the `DBP…` keys sort before the `DBS…` keys, so the 50-row cap kept them), and the unfiltered variant kept 50 rows of which 5 were plasmids — **no loss measurable on this fixture ordering**, but the loss depends on key order and BM25 ties, so the in-branch filter stays (design already decided; `TestSearchStockEntityFilterKeepsSmallGroup` pins it).
  21. One stock matching `label` + `genes` + `summary` for `cordax`: exactly 1 merged row, field attributed to the highest band (`genes` prefix, 1000).
  22. Array display: prefix `ax` → `Ax2` (matched element); fuzzy-only `gammas` → `gammaS13` (matched element, `CONTAINS` passes); document without `names` → empty string. Never `null`, never an error.
  23. The orphan property document: 0 rows — dropped by `FILTER own != null`.
  24. The plasmid without a summary attribute: `sm` empty string, no error.
  25. `FLATTEN([...])` over the 19 branch variables: flat list, one object per row.
  26. Full statement for the `editable_summary`-only term: 0 rows, no error.
- Final n-gram threshold: **0.30** (0.45 folded out; see probe 6).
- Query normalization decision, Go lowercase or `TOKENS`: **Go side** — trim, lowercase, strip combining diacritical marks (same `normalizeAutocompleteQuery` shape as the autocomplete feature). A punctuation-only query keeps its tokens safe, where an AQL `TOKENS` normalization would produce an empty token list and a null prefix argument.
- `TOKENS("the and", "text_en")` output `["the", "and"]`, and the chosen stopword behavior: text_en keeps stopwords; **Go-side stopword rejection** (invalid-argument error) chosen; see probe 18.
- Multi-token noise count, and the decision between any-token and all-token: 1 of 3 rows; **any-token retained**; see probe 17.
- Measured `BM25` minimum and maximum: token 3.07–8.18, fuzzy 18.95–39.52, prefix 0. **The 250-point bands hold** (see probe 15).
- Whether the `IN TOKENS` form needs the `ANALYZER()` wrapper, from probe 8: **yes** — 0 rows without it.
- EXPLAIN node list and the baseline p95 in milliseconds: 19 `EnumerateViewNode`, 0 `EnumerateCollectionNode`; full node counts `{SingletonNode:1, SubqueryStartNode:46, EnumerateViewNode:19, TraversalNode:19, LimitNode:46, CalculationNode:116, SubqueryEndNode:46, FilterNode:25, SortNode:22, EnumerateListNode:9, CollectNode:1, ReturnNode:1}`. Latency over 20 runs on the warmed 1071-doc fixture with `forms culminants`, limit 50: **min 10 ms, median 10 ms, p95 11 ms**.
- Task 6 measured p95, and the ratio against the baseline: (none yet)
- Build break recorded in Task 2 Step 4: (none yet)
- Pre-existing test failures on the branch before any change: **none** — `go test ./...` green on `feat/stock-full-search` (493 tests, 1 skipped integration test) before any change.
- The gRPC code that `aphgrpc.HandleGetError` really returns: (none yet)
- Deviations from [Exact interfaces](#exact-interfaces), with the reason:
  - `fullSearchNgramThreshold` 0.45 → 0.30 (Task 1 probe 6; the assumed 0.45 misses real one-character typos).
  - Probe 11 contradicted the plan's PHRASE feature expectation (PHRASE matched on a featureless norm analyzer); `text_en` stays on prose because prose needs tokenization and stemming.
  - The repository adds a Go-side stopword check that rejects an all-stopword query with an error (Task 1 probe 18; text_en does not remove stopwords). This is the documented mitigation the plan anticipated, not a deviation from an interface.
