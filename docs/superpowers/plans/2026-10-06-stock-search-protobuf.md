# Stock Search Protocol Buffer Specification Plan

> **For agentic workers:** Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task by task. The steps use checkboxes.

**Goal:** Define the complete wire contract for two new stock search RPCs, `AutocompleteStock` and `SearchStock`, in `dictybase/stock/stock.proto`. Publish the generated Go stubs through dictyBase/go-genproto. Update the modware-stock dependency, and prove the validation contract with a test that runs against the generated types.

**Architecture:** The proto file holds all request validation as `buf.validate` (protovalidate) rules. Each request uses the nested JSON:API shape of this repository: a `Data` message that holds a `type` string and a required `attributes` message. Each response is a collection message with a `repeated` data list and the existing `Meta` message. Two enums close the value space. `StockEntity` selects strains, plasmids or both. `StockSearchField` names the field that matched. The modware-stock service layer validates each request with `protovalidate.Validate`. That call is necessary, because the mwitkow `govalidators` plugin generates a no-op `Validate()` method for a message that carries no mwitkow rule.

**Tech Stack:** buf 1.72.0 locally and buf 1.47.2 in CI. The `buf.build/bufbuild/protovalidate` module dependency. protoc-gen-go v1.31.0, protoc-gen-go-grpc v1.3.0 and protoc-gen-govalidators v0.3.2 through `buf.gen.yaml`. `github.com/bufbuild/protovalidate-go` v0.10.0 in modware-stock. Go 1.26.

**Spec:** User request (2026-10-06): add type-ahead autocomplete and a full search that returns up to 50 ranked stocks. This plan owns only the protocol surface and the generated stubs. Two sibling plans own the two server implementations: `docs/superpowers/plans/2026-10-06-stock-autocomplete.md` and `docs/superpowers/plans/2026-10-06-stock-full-search.md`. Both of them are blocked until the handoff artifacts of this plan exist.

---

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Goal](#goal)
- [Scope](#scope)
- [Non-goals](#non-goals)
- [Verified repository facts](#verified-repository-facts)
  - [dictybaseapis facts](#dictybaseapis-facts)
  - [go-genproto facts](#go-genproto-facts)
  - [modware-stock facts](#modware-stock-facts)
- [Prerequisite gate](#prerequisite-gate)
- [Exact interfaces](#exact-interfaces)
  - [Enums](#enums)
  - [Service methods](#service-methods)
  - [Request and response messages](#request-and-response-messages)
  - [Field number and validation table](#field-number-and-validation-table)
  - [Response semantics](#response-semantics)
  - [Field enum to repository field path mapping](#field-enum-to-repository-field-path-mapping)
- [File map](#file-map)
- [Implementation tasks](#implementation-tasks)
  - [Task 1: Edit the proto file](#task-1-edit-the-proto-file)
  - [Task 2: Lint and build the proto file](#task-2-lint-and-build-the-proto-file)
  - [Task 3: Merge the proto change and generate the stubs](#task-3-merge-the-proto-change-and-generate-the-stubs)
  - [Task 4: Record the released go-genproto pseudo-version](#task-4-record-the-released-go-genproto-pseudo-version)
  - [Task 5: Update the modware-stock dependencies](#task-5-update-the-modware-stock-dependencies)
  - [Task 6: Write the contract test](#task-6-write-the-contract-test)
- [Test matrix](#test-matrix)
- [Failure behavior](#failure-behavior)
- [Quality gates](#quality-gates)
- [Completion criteria](#completion-criteria)
- [Handoff artifacts](#handoff-artifacts)
- [Review focus](#review-focus)
- [Execution notes](#execution-notes)

---

## Goal

Deliver a frozen, lint-clean wire contract for stock autocomplete and stock full search, plus compiled Go stubs that modware-stock can import. After this plan ends, an implementer of either server plan needs no second source. This file gives every message name, field name, field number, validation rule, default value and enum value.

## Scope

- Edit `dictybase/stock/stock.proto` in the dictybaseapis repository: 2 RPC methods, 2 enums, 8 messages.
- Run `buf lint` and `buf build` on the changed file.
- Merge the proto change to the `master` branch of dictybaseapis, so the `Generate go code` workflow produces the stubs and pushes them to dictyBase/go-genproto.
- Merge the generated branch in go-genproto, and record the exact released pseudo-version.
- Raise the `github.com/dictyBase/go-genproto` requirement in `go.mod` and `go.sum` of modware-stock.
- Add `github.com/bufbuild/protovalidate-go` v0.10.0 to modware-stock as a direct requirement.
- Add one contract test in modware-stock that asserts the validation behavior of the generated types.

## Non-goals

- No repository code. No ArangoDB analyzers, views or AQL. The autocomplete plan and the full-search plan own that work.
- No service handler. This plan adds no method to `StockService`.
- No web user interface.
- No change to existing messages in `stock.proto`. No field number of an existing message changes, and no existing field number is reused.
- No second `Meta` message. The file already defines `Meta` at line 525, and both new collection messages reuse it.

## Verified repository facts

### dictybaseapis facts

Checked in `/Users/sba964/Projects/devenv/golang/dictybaseapis`.

| Fact | Evidence |
| --- | --- |
| The proto package is `dictybase.stock`; the Go package is `github.com/dictyBase/go-genproto/dictybaseapis/stock;stock`. | `dictybase/stock/stock.proto` lines 3 and 11 |
| `stock.proto` is 534 lines long. The service block `StockService` runs from line 18 to line 45 and holds 13 RPC methods. The last method is `OboJSONFileUpload`. The file holds 23 top-level messages and 0 enums. | `dictybase/stock/stock.proto` lines 18-45, `grep -c '^message '` and `grep -c '^enum '` |
| `stock.proto` currently imports `dictybase/api/upload/file.proto`, `github.com/mwitkow/go-proto-validators/validator.proto`, `google/protobuf/empty.proto` and `google/protobuf/timestamp.proto`. It does **not** import `buf/validate/validate.proto` yet. | `dictybase/stock/stock.proto` lines 5-8 |
| `message Meta` exists with `next_cursor = 1`, `limit = 2`, `total = 3`. | `dictybase/stock/stock.proto` lines 525-534 |
| None of the 23 messages in `stock.proto` is named `StockEntity`, `StockSearchField`, `StockAutocompleteParameters`, `StockAutocompleteAttributes`, `StockSuggestion`, `StockSuggestionCollection`, `StockSearchParameters`, `StockSearchAttributes`, `StockSearchResult` or `StockSearchResultCollection`. The new names collide with nothing. | full list of `^message`/`^enum` lines in `dictybase/stock/stock.proto` |
| `buf.yaml` declares the dependencies `buf.build/googleapis/googleapis` and `buf.build/bufbuild/protovalidate`, and the lint rule set `BASIC` with the ignore path `github.com/mwitkow`. | `buf.yaml` |
| The `BASIC` rule set does not include `ENUM_VALUE_PREFIX` or `ENUM_ZERO_VALUE_SUFFIX`. The prefixed names in this plan therefore pass lint and also satisfy the stricter `DEFAULT` set, in case the rule set changes later. | `buf.yaml` lint block |
| `buf.gen.yaml` runs four plugins: `buf.build/protocolbuffers/go:v1.31.0` and `buf.build/grpc/go:v1.3.0` and the local `govalidators` plugin into `generated`, plus `buf.build/community/pseudomuto-doc` into `docs`. | `buf.gen.yaml` |
| The repository has no `develop` branch. The generation workflow triggers on a push to `master`. | `.github/workflows/buf-generate.yml` trigger block |
| `dictybase/order/order.proto` already uses the exact pattern this plan copies: `import "buf/validate/validate.proto";` next to the mwitkow import, a nested `message Data` inside the request, `(buf.validate.field).required = true` on both `data` and `attributes`, and `(buf.validate.field).string.min_len = 3` on `query`. | `dictybase/order/order.proto` lines 5-6, 302-341 |

### go-genproto facts

Checked in `/Users/sba964/Projects/devenv/golang/go-genproto`.

| Fact | Evidence |
| --- | --- |
| The `Generate go code` workflow checks out dictyBase/go-genproto, deletes `target/dictybaseapis`, copies `generated/github.com/dictyBase/go-genproto/dictybaseapis` over it, commits with the message `update on <date>`, and pushes to the branch `chore/update-<short-sha>`. The workflow does not merge that branch. A human must merge it. | `.github/workflows/buf-generate.yml` steps `copy generated files` and `push changes` |
| go-genproto `go.mod` already requires `buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.4-20241127180247-a33202765966.1`, so generated code that references `buf.validate` extensions compiles without a `go.mod` edit. | `go-genproto/go.mod` |
| `annotation`, `organism` and `feature_annotation` stubs already reference protovalidate. This proves the generation path works for `buf.validate` annotations. | `grep -rl protovalidate go-genproto/dictybaseapis/` |
| The `govalidators` plugin emits a **no-op** `Validate()` method for every message that carries no mwitkow rule. Example: `func (this *Meta) Validate() error { return nil }`. A handler that calls `Validate()` on a new request therefore validates nothing. | `go-genproto/dictybaseapis/stock/stock.validator.pb.go` line 431 |
| The HEAD of go-genproto is commit `163a42b` (`update on 09-28-2026:17:12:46`). `dictybaseapis/order/order.pb.go` at that commit contains **no** `AutocompleteParameters`. The order autocomplete proto work is therefore not released either. This plan must not assume any existing released version that carries stock search types. | `git log --oneline -3`, `grep -n AutocompleteParameters dictybaseapis/order/*.go` |

### modware-stock facts

Checked in `/Users/sba964/Projects/devenv/golang/modware-stock`.

| Fact | Evidence |
| --- | --- |
| `go.mod` requires `github.com/dictyBase/go-genproto v0.0.0-20211001224012-6cf691015622`. This version predates every search type in this plan. | `go.mod` |
| `go.mod` does **not** require `github.com/bufbuild/protovalidate-go`. The module must be added. The known good version in this family is v0.10.0, as used by modware-annotation. | `go.mod` of modware-stock, `go.mod` of modware-annotation line 5 |
| The module declares `go 1.26.0`. CI installs Go 1.26. | `go.mod`, `.github/workflows/ci.yml` |
| `go.mod` carries `exclude google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1`. Keep the exclude line. It prevents ambiguous imports with the split genproto modules. | `go.mod` |
| The validation pattern in this family is the package-level call `protovalidate.Validate(req)` with the import `"github.com/bufbuild/protovalidate-go"`. | `modware-annotation/internal/app/service/read_service.go` lines 1-23 |
| Existing stock handlers call the generated `r.Validate()`, for example `GetStrain` and `CreateStrain`. New handlers must not copy that call, because of the no-op fact above. | `internal/app/service/strain.go` lines 13-35 and 60-80 |
| CI runs `arangodb:3.11` and exports `ARANGO_USER`, `ARANGO_PASS`, `ARANGO_HOST` and `ARANGO_PORT`. The contract test in this plan needs no database, so CI needs no change. | `.github/workflows/ci.yml` |

## Prerequisite gate

Satisfy every item before Task 1 starts.

- [ ] Write access to the dictybaseapis repository, and the right to merge to `master`.
- [ ] Write access to the dictyBase/go-genproto repository, and the right to merge a `chore/update-*` branch.
- [ ] `buf --version` prints 1.72.0 or later. The repository is a buf workspace, so every `buf lint` and `buf build` call in this plan needs the `--path` flag.
- [ ] A local clone of dictybaseapis at `/Users/sba964/Projects/devenv/golang/dictybaseapis`, with `master` up to date.
- [ ] A local clone of modware-stock at `/Users/sba964/Projects/devenv/golang/modware-stock`, on the branch `develop` or a feature branch off `develop`.
- [ ] `go version` prints go1.26 or later.

## Exact interfaces

Every name, number and rule below is normative. The autocomplete plan and the full-search plan consume exactly these names.

### Enums

```proto
// StockEntity selects the kind of stock that a search covers.
enum StockEntity {
  // Both strains and plasmids. This is the default, because an absent
  // field carries the value 0.
  STOCK_ENTITY_UNSPECIFIED = 0;
  // Strains only.
  STOCK_ENTITY_STRAIN = 1;
  // Plasmids only.
  STOCK_ENTITY_PLASMID = 2;
}

// StockSearchField names the stock field that produced a match.
enum StockSearchField {
  STOCK_SEARCH_FIELD_UNSPECIFIED = 0;
  // Do not reuse the field number or enum name from an earlier plan draft.
  reserved 10;
  reserved "STOCK_SEARCH_FIELD_EDITABLE_SUMMARY";
  // stock_id of the stock document, for example DBS0236126.
  STOCK_SEARCH_FIELD_STOCK_ID = 1;
  // genes array of the stock document.
  STOCK_SEARCH_FIELD_GENES = 2;
  // dbxrefs array of the stock document.
  STOCK_SEARCH_FIELD_DBXREFS = 3;
  // label of the strain property document.
  STOCK_SEARCH_FIELD_LABEL = 4;
  // names array of the strain property document.
  STOCK_SEARCH_FIELD_NAMES = 5;
  // species of the strain property document.
  STOCK_SEARCH_FIELD_SPECIES = 6;
  // plasmid of the strain property document.
  STOCK_SEARCH_FIELD_PLASMID = 7;
  // name of the plasmid property document.
  STOCK_SEARCH_FIELD_NAME = 8;
  // summary of the stock document.
  STOCK_SEARCH_FIELD_SUMMARY = 9;
  // depositor of the stock document.
  STOCK_SEARCH_FIELD_DEPOSITOR = 11;
}
```

`STOCK_SEARCH_FIELD_SUMMARY` and `STOCK_SEARCH_FIELD_DEPOSITOR` never appear in an `AutocompleteStock` response. `editable_summary` remains stored stock data, but neither search RPC indexes or searches it.

### Service methods

Add both methods inside the existing `service StockService` block, after `rpc LoadPlasmid(ExistingPlasmid) returns (Plasmid) {}` and before `rpc OboJSONFileUpload(...)`:

```proto
  // AutocompleteStock returns short type-ahead suggestions for a partial
  // search text. The default list length is 5.
  rpc AutocompleteStock(StockAutocompleteParameters) returns (StockSuggestionCollection) {}

  // SearchStock returns a ranked list of stocks for a search text. The
  // default list length is 50.
  rpc SearchStock(StockSearchParameters) returns (StockSearchResultCollection) {}
```

The generated server interface is embedded by `stock.UnimplementedStockServiceServer`, which `StockService` already embeds. No change to `internal/app/server` is needed for compilation.

### Request and response messages

Place the block after `message Meta` at the end of the file.

```proto
// StockAutocompleteParameters is the input of an AutocompleteStock call.
message StockAutocompleteParameters {
  message Data {
    // Resource name. By default it is stock.
    string type = 1;
    StockAutocompleteAttributes attributes = 2 [(buf.validate.field).required = true];
  }
  Data data = 1 [(buf.validate.field).required = true];
}

// StockAutocompleteAttributes holds the search text of an autocomplete
// request. The match runs against stock_id, genes, dbxrefs, the strain
// label, the strain names, the species, the strain plasmid and the
// plasmid name.
message StockAutocompleteAttributes {
  // Search text. The client sends it when at least 3 characters are
  // typed. The server trims the text and lowercases it.
  string query = 1 [(buf.validate.field).string.min_len = 3];
  // Maximum number of suggestions. The server returns 5 when this field
  // is absent or zero, and it never returns more than 50.
  int64 limit = 2 [(buf.validate.field).int64 = {
    gte: 0,
    lte: 50
  }];
  // Kind of stock to search. An absent field covers both kinds.
  StockEntity entity = 3 [(buf.validate.field).enum.defined_only = true];
}

// StockSuggestion is one autocomplete match.
message StockSuggestion {
  // stock_id of the matched stock, for example DBS0236126.
  string id = 1;
  // Kind of the matched stock. The server never returns
  // STOCK_ENTITY_UNSPECIFIED here.
  StockEntity entity = 2;
  // Field that produced the match.
  StockSearchField field = 3;
  // Text that the user interface shows for this suggestion.
  string display_text = 4;
  // Rank score. A prefix match scores above a fuzzy match.
  double score = 5;
}

// StockSuggestionCollection returns the suggestions of an autocomplete
// request. An empty data list is valid, because a query can match no
// stock.
message StockSuggestionCollection {
  repeated StockSuggestion data = 1;
  // next_cursor is always 0, limit is the effective limit of the
  // request, and total is the number of rows in data.
  Meta meta = 2;
}

// StockSearchParameters is the input of a SearchStock call.
message StockSearchParameters {
  message Data {
    // Resource name. By default it is stock.
    string type = 1;
    StockSearchAttributes attributes = 2 [(buf.validate.field).required = true];
  }
  Data data = 1 [(buf.validate.field).required = true];
}

// StockSearchAttributes holds the search text of a full search request.
// The match runs against stock_id, genes, dbxrefs, the strain label, the
// strain names, the species, the strain plasmid, the plasmid name, the
// summary and the depositor. The search does not use editable_summary.
message StockSearchAttributes {
  // Search text. The minimum length is 2 characters, because gene names
  // such as csA are short.
  string query = 1 [(buf.validate.field).string.min_len = 2];
  // Maximum number of results. The server returns 50 when this field is
  // absent or zero, and it never returns more than 100.
  int64 limit = 2 [(buf.validate.field).int64 = {
    gte: 0,
    lte: 100
  }];
  // Kind of stock to search. An absent field covers both kinds.
  StockEntity entity = 3 [(buf.validate.field).enum.defined_only = true];
}

// StockSearchResult is one full search match.
message StockSearchResult {
  // stock_id of the matched stock, for example DBS0236126.
  string id = 1;
  // Kind of the matched stock. The server never returns
  // STOCK_ENTITY_UNSPECIFIED here.
  StockEntity entity = 2;
  // Field that produced the match.
  StockSearchField field = 3;
  // Text that the user interface shows for the matched field.
  string display_text = 4;
  // Rank score. A prefix match scores above a phrase match, a phrase
  // match scores above a token match, and a token match scores above a
  // fuzzy match.
  double score = 5;
  // Complete stored summary of the stock document. It is not a snippet,
  // and it is not highlighted. An absent stored summary gives an empty
  // string.
  string summary = 6;
  // Strain label, shown as Descriptor in the stock center. Plasmid results
  // and strains without a label return an empty string.
  string strain_label = 7;
}

// StockSearchResultCollection returns the results of a full search
// request. An empty data list is valid, because a query can match no
// stock.
message StockSearchResultCollection {
  repeated StockSearchResult data = 1;
  // next_cursor is always 0, limit is the effective limit of the
  // request, and total is the number of rows in data.
  Meta meta = 2;
}
```

### Field number and validation table

| Message | Field | Number | Type | Validation rule |
| --- | --- | --- | --- | --- |
| `StockAutocompleteParameters` | `data` | 1 | `Data` | `required = true` |
| `StockAutocompleteParameters.Data` | `type` | 1 | `string` | none |
| `StockAutocompleteParameters.Data` | `attributes` | 2 | `StockAutocompleteAttributes` | `required = true` |
| `StockAutocompleteAttributes` | `query` | 1 | `string` | `string.min_len = 3` |
| `StockAutocompleteAttributes` | `limit` | 2 | `int64` | `int64 = {gte: 0, lte: 50}` |
| `StockAutocompleteAttributes` | `entity` | 3 | `StockEntity` | `enum.defined_only = true` |
| `StockSuggestion` | `id` | 1 | `string` | none |
| `StockSuggestion` | `entity` | 2 | `StockEntity` | none |
| `StockSuggestion` | `field` | 3 | `StockSearchField` | none |
| `StockSuggestion` | `display_text` | 4 | `string` | none |
| `StockSuggestion` | `score` | 5 | `double` | none |
| `StockSuggestionCollection` | `data` | 1 | `repeated StockSuggestion` | none, an empty list is valid |
| `StockSuggestionCollection` | `meta` | 2 | `Meta` | none |
| `StockSearchParameters` | `data` | 1 | `Data` | `required = true` |
| `StockSearchParameters.Data` | `type` | 1 | `string` | none |
| `StockSearchParameters.Data` | `attributes` | 2 | `StockSearchAttributes` | `required = true` |
| `StockSearchAttributes` | `query` | 1 | `string` | `string.min_len = 2` |
| `StockSearchAttributes` | `limit` | 2 | `int64` | `int64 = {gte: 0, lte: 100}` |
| `StockSearchAttributes` | `entity` | 3 | `StockEntity` | `enum.defined_only = true` |
| `StockSearchResult` | `id` | 1 | `string` | none |
| `StockSearchResult` | `entity` | 2 | `StockEntity` | none |
| `StockSearchResult` | `field` | 3 | `StockSearchField` | none |
| `StockSearchResult` | `display_text` | 4 | `string` | none |
| `StockSearchResult` | `score` | 5 | `double` | none |
| `StockSearchResult` | `summary` | 6 | `string` | none |
| `StockSearchResult` | `strain_label` | 7 | `string` | none |
| `StockSearchResultCollection` | `data` | 1 | `repeated StockSearchResult` | none, an empty list is valid |
| `StockSearchResultCollection` | `meta` | 2 | `Meta` | none |

No response field carries a validation rule. The server produces the response. Do not add a `repeated_count_min` rule, because it rejects a legal empty result.

### Response semantics

| Item | Autocomplete | Full search |
| --- | --- | --- |
| Effective limit when `limit` is 0 | 5 | 50 |
| Effective limit when `limit` is above the cap | rejected by protovalidate at 51 and above | rejected by protovalidate at 101 and above |
| Server-side cap after validation | 50 | 100, but the result list never holds more than 50 rows, because the spec asks for one list of 50 items |
| `Meta.limit` | the effective limit of the request | the effective limit of the request |
| `Meta.total` | number of rows in `data` | number of rows in `data` |
| `Meta.next_cursor` | 0 | 0 |
| `data` length | 0 to the effective limit | 0 to the effective limit |
| `StockSearchResult.summary` | not present in this message | the exact stored value of `stock.summary`, or an empty string when the attribute is absent |
| `StockSearchResult.strain_label` | not present in this message | the strain property `label` (shown as Descriptor in the stock center), or an empty string for a plasmid or a strain without a label |

### Field enum to repository field path mapping

Both server plans map a repository field path string to `StockSearchField`. This table is the single source of that mapping.

| Repository field path | Source collection | `StockSearchField` value |
| --- | --- | --- |
| `stock_id` | stock document | `STOCK_SEARCH_FIELD_STOCK_ID` |
| `genes` | stock document | `STOCK_SEARCH_FIELD_GENES` |
| `dbxrefs` | stock document | `STOCK_SEARCH_FIELD_DBXREFS` |
| `label` | stock property document | `STOCK_SEARCH_FIELD_LABEL` |
| `names` | stock property document | `STOCK_SEARCH_FIELD_NAMES` |
| `species` | stock property document | `STOCK_SEARCH_FIELD_SPECIES` |
| `plasmid` | stock property document | `STOCK_SEARCH_FIELD_PLASMID` |
| `name` | stock property document | `STOCK_SEARCH_FIELD_NAME` |
| `summary` | stock document | `STOCK_SEARCH_FIELD_SUMMARY` |
| `depositor` | stock document | `STOCK_SEARCH_FIELD_DEPOSITOR` |
| any other value | — | `STOCK_SEARCH_FIELD_UNSPECIFIED` |

The entity mapping is symmetric:

| `StockEntity` | Repository entity filter string | ArangoDB edge `type` attribute |
| --- | --- | --- |
| `STOCK_ENTITY_UNSPECIFIED` | `""` | not filtered |
| `STOCK_ENTITY_STRAIN` | `"strain"` | `strain` |
| `STOCK_ENTITY_PLASMID` | `"plasmid"` | `plasmid` |

## File map

| Repository | File | Action |
| --- | --- | --- |
| dictybaseapis | `dictybase/stock/stock.proto` | modify: 1 import, 2 RPC methods, 2 enums, 8 messages |
| dictyBase/go-genproto | `dictybaseapis/stock/stock.pb.go` | regenerated by the workflow; do not hand-edit |
| dictyBase/go-genproto | `dictybaseapis/stock/stock_grpc.pb.go` | regenerated by the workflow; do not hand-edit |
| dictyBase/go-genproto | `dictybaseapis/stock/stock.validator.pb.go` | regenerated by the workflow; do not hand-edit |
| modware-stock | `go.mod`, `go.sum` | modify: raise go-genproto, add protovalidate-go v0.10.0 |
| modware-stock | `internal/app/service/search_contract_test.go` | create: the validation contract test |
| modware-stock | `docs/superpowers/plans/2026-10-06-stock-search-protobuf.md` | modify: fill the [Execution notes](#execution-notes) |

## Implementation tasks

### Task 1: Edit the proto file

**Files:** modify `dictybase/stock/stock.proto` in dictybaseapis.

- [ ] **Step 1: Create the branch.** The repository has no `develop` branch, so branch off `master`:

```bash
cd /Users/sba964/Projects/devenv/golang/dictybaseapis
git switch master
git pull --rebase
git switch -c feat/stock-search
```

- [ ] **Step 2: Add the protovalidate import.** Insert `import "buf/validate/validate.proto";` above the existing mwitkow import, so the import block reads in this order:

```proto
import "buf/validate/validate.proto";
import "dictybase/api/upload/file.proto";
import "github.com/mwitkow/go-proto-validators/validator.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";
```

Keep the mwitkow import. Existing messages still carry mwitkow rules. If you remove the import, those messages break.

- [ ] **Step 3: Add the two RPC methods** inside `service StockService`, in the position given in [Service methods](#service-methods).

- [ ] **Step 4: Add the two enums** from [Enums](#enums). Place them after the `service StockService` block and before `message StockId`.

- [ ] **Step 5: Add the eight messages** from [Request and response messages](#request-and-response-messages), after `message Meta` at the end of the file.

- [ ] **Step 6: Check that no existing line changed.** `git diff` must show additions only, apart from the one new import line:

```bash
git diff --stat dictybase/stock/stock.proto
git diff dictybase/stock/stock.proto | grep '^-' | grep -v '^---'
```

Expected: the second command prints nothing.

### Task 2: Lint and build the proto file

- [ ] **Step 1: Lint.** The repository is a buf workspace, so the `--path` flag is required:

```bash
cd /Users/sba964/Projects/devenv/golang/dictybaseapis
buf lint --path dictybase/stock/stock.proto
```

Expected: no output and exit status 0.

- [ ] **Step 2: Build.**

```bash
buf build --path dictybase/stock/stock.proto
```

Expected: no output and exit status 0. A failure here almost always means a missing `buf.validate` dependency. In that case run `buf dep update` and commit the changed `buf.lock`.

- [ ] **Step 3: Check the definition counts.** The file holds 23 top-level messages and 0 enums before the change, measured with `grep -c '^message '` and `grep -c '^enum '`. This plan adds 8 top-level messages, because the 2 `Data` messages are nested and are indented. So after the change:

```bash
grep -c '^message ' dictybase/stock/stock.proto   # expected: 31
grep -c '^enum ' dictybase/stock/stock.proto      # expected: 2
grep '^message \|^enum ' dictybase/stock/stock.proto | sort | uniq -d
```

Record the real numbers in the [Execution notes](#execution-notes) if the baseline differs, and make sure that the third command prints nothing.

- [ ] **Step 4: Commit.**

```bash
git add dictybase/stock/stock.proto buf.lock
git commit -m "feat: add stock autocomplete and search rpc definitions"
```

### Task 3: Merge the proto change and generate the stubs

- [ ] **Step 1: Push the branch and open a pull request.** Pushing a new branch runs `create-pull-request.yml`. Confirm that the `Buf checkup` workflow passes on the pull request. That workflow runs buf 1.47.2, an older version than the local buf, so a rule difference shows up here.

- [ ] **Step 2: Merge to `master` after review.** The merge triggers `.github/workflows/buf-generate.yml`, because the push touches `dictybase/**/*.proto`.

- [ ] **Step 3: Watch the workflow.** It must finish all steps: `buf generate`, the checkout of dictyBase/go-genproto, the copy of `generated/github.com/dictyBase/go-genproto/dictybaseapis` into `target`, the commit, and the push of the branch `chore/update-<short-sha>`.

- [ ] **Step 4: Verify the generated stubs** on that branch in go-genproto. All three assertions must hold:

```bash
cd <go-genproto clone>
git fetch origin
git switch chore/update-<short-sha>
grep -c 'StockAutocompleteParameters\|StockSearchParameters' dictybaseapis/stock/stock.pb.go   # must be > 0
grep -n 'AutocompleteStock\|SearchStock' dictybaseapis/stock/stock_grpc.pb.go | head
go build ./...
```

Expected: the grep commands find the new types and both client and server methods, and the build passes. The build can fail with a missing `buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go` symbol. In that case, raise that requirement in the go-genproto `go.mod`. Then run `go mod tidy` in a separate commit on the same branch.

- [ ] **Step 5: Merge the `chore/update-<short-sha>` branch into the default branch of go-genproto.** The pseudo-version that modware-stock consumes is derived from the merge commit, so this step cannot be skipped.

### Task 4: Record the released go-genproto pseudo-version

This task produces the hard handoff that both server plans gate on. Do not guess or invent a version string.

- [ ] **Step 1: Read the version that the Go module proxy serves.**

```bash
cd /Users/sba964/Projects/devenv/golang/modware-stock
GOPROXY=https://proxy.golang.org go list -m -versions github.com/dictyBase/go-genproto
GOPROXY=https://proxy.golang.org go list -m github.com/dictyBase/go-genproto@latest
```

The second command prints the exact pseudo-version, for example `github.com/dictyBase/go-genproto v0.0.0-<timestamp>-<12-char-commit>`. go-genproto carries no semantic version tags, so a pseudo-version is expected.

- [ ] **Step 2: Confirm that the served version holds the new types.**

```bash
GOPROXY=https://proxy.golang.org go mod download github.com/dictyBase/go-genproto@<pseudo-version>
grep -c 'StockAutocompleteParameters' "$(go env GOMODCACHE)/github.com/dicty!base/go-genproto@<pseudo-version>/dictybaseapis/stock/stock.pb.go"
```

Expected: a count above 0. The module cache escapes the capital letters of `dictyBase` as `dicty!base`.

- [ ] **Step 3: Write the exact version string into the [Execution notes](#execution-notes)** of this file, under `Released go-genproto pseudo-version`. Both server plans read it from there.

### Task 5: Update the modware-stock dependencies

**Files:** modify `go.mod` and `go.sum` in modware-stock.

- [ ] **Step 1: Raise go-genproto and add protovalidate-go.** Use the version recorded in Task 4, Step 3:

```bash
cd /Users/sba964/Projects/devenv/golang/modware-stock
go get github.com/dictyBase/go-genproto@<pseudo-version>
go get github.com/bufbuild/protovalidate-go@v0.10.0
go mod tidy
```

- [ ] **Step 2: Check the result.** `github.com/bufbuild/protovalidate-go v0.10.0` must appear in a `require` block without the `// indirect` comment after Task 6 adds the first import. The `exclude google.golang.org/genproto` line must still be present.

```bash
grep -n 'go-genproto\|protovalidate\|^exclude' go.mod
```

- [ ] **Step 3: Build.**

```bash
go build ./...
go test ./... 2>&1 | tail -20
```

Expected: the build passes. The raise of go-genproto changes generated types that existing code uses, so a compile break here is a real signal. Fix it in this task, and record every fix in the [Execution notes](#execution-notes).

- [ ] **Step 4: Commit.**

```bash
git add go.mod go.sum
git commit -m "chore(deps): bump go-genproto for stock search rpcs and add protovalidate"
```

### Task 6: Write the contract test

The test asserts the validation contract against the real generated types. It needs no database and no handler, so it belongs to this plan.

**Files:** create `internal/app/service/search_contract_test.go`.

- [ ] **Step 1: Write the failing test.** Package `service`. Imports: `testing`, `github.com/bufbuild/protovalidate-go`, `github.com/dictyBase/go-genproto/dictybaseapis/stock`, `github.com/stretchr/testify/require`. Write flat sequential test functions, one per row of the [Test matrix](#test-matrix). Use two small builders:

```go
func autocompleteRequest(query string, limit int64, entity stock.StockEntity) *stock.StockAutocompleteParameters {
	return &stock.StockAutocompleteParameters{
		Data: &stock.StockAutocompleteParameters_Data{
			Type: "stock",
			Attributes: &stock.StockAutocompleteAttributes{
				Query:  query,
				Limit:  limit,
				Entity: entity,
			},
		},
	}
}

func searchRequest(query string, limit int64, entity stock.StockEntity) *stock.StockSearchParameters {
	return &stock.StockSearchParameters{
		Data: &stock.StockSearchParameters_Data{
			Type: "stock",
			Attributes: &stock.StockSearchAttributes{
				Query:  query,
				Limit:  limit,
				Entity: entity,
			},
		},
	}
}
```

Every assertion calls `protovalidate.Validate(req)` and uses `require.NoError` or `require.Error` with a message that names the rule.

- [ ] **Step 2: Run the test and make sure that it fails** before the dependency raise lands, or right after it if Task 5 already ran:

```bash
gotestsum --format-hide-empty-pkg --format dots -- -run 'TestStockSearchContract' ./internal/app/service/
```

Expected before Task 5: a compile failure, because the generated types do not exist. Expected after Task 5: PASS. A failure after Task 5 means that the proto rules do not match this plan. Fix the proto file, not the test.

- [ ] **Step 3: Add the two explicit no-op assertions.** They pin the facts that the server plans depend on:

```go
func TestStockSearchContractGeneratedValidateIsNoOp(t *testing.T) {
	req := require.New(t)
	// The govalidators plugin emits an empty Validate() for messages
	// without mwitkow rules. A handler must not rely on it.
	req.NoError(
		autocompleteRequest("ab", 999, stock.StockEntity(77)).Validate(),
		"the generated Validate must accept a request that protovalidate rejects",
	)
	req.Error(
		protovalidate.Validate(autocompleteRequest("ab", 999, stock.StockEntity(77))),
		"protovalidate must reject the same request",
	)
}
```

If the generated `Validate()` method does not exist on the new types, delete the first assertion and record the deviation in the [Execution notes](#execution-notes).

- [ ] **Step 4: Run the quality gates** from [Quality gates](#quality-gates).

- [ ] **Step 5: Commit.**

```bash
git add internal/app/service/search_contract_test.go
git commit -m "test: pin the stock search request validation contract"
```

## Test matrix

Every row is one test function in `internal/app/service/search_contract_test.go`. Prefix every function name with `TestStockSearchContract`.

| # | Input | Expected result | Rule under test |
| --- | --- | --- | --- |
| 1 | autocomplete, `query` `"dbs"`, `limit` 0, entity unspecified | valid | happy path, default limit path |
| 2 | autocomplete, `query` `"db"` | invalid | `string.min_len = 3` |
| 3 | autocomplete, `query` `""` | invalid | `string.min_len = 3` |
| 4 | autocomplete, `query` `"dbs"`, `limit` 50 | valid | `lte: 50` boundary |
| 5 | autocomplete, `query` `"dbs"`, `limit` 51 | invalid | `lte: 50` |
| 6 | autocomplete, `query` `"dbs"`, `limit` -1 | invalid | `gte: 0` |
| 7 | autocomplete, `entity` `stock.StockEntity(3)` | invalid | `enum.defined_only = true` |
| 8 | autocomplete, `&stock.StockAutocompleteParameters{}` | invalid | `data` `required = true` |
| 9 | autocomplete, `Data` present with `Attributes` nil | invalid | `attributes` `required = true` |
| 10 | autocomplete, `query` `"   "` (three spaces) | **valid** | documents the gap: `min_len` counts characters, so a whitespace-only query passes the proto rule and the handler must trim and reject it |
| 11 | full search, `query` `"cs"`, `limit` 0, entity unspecified | valid | happy path, 2 character floor |
| 12 | full search, `query` `"c"` | invalid | `string.min_len = 2` |
| 13 | full search, `query` `"cs"`, `limit` 100 | valid | `lte: 100` boundary |
| 14 | full search, `query` `"cs"`, `limit` 101 | invalid | `lte: 100` |
| 15 | full search, `query` `"cs"`, `limit` -5 | invalid | `gte: 0` |
| 16 | full search, `entity` `stock.StockEntity(9)` | invalid | `enum.defined_only = true` |
| 17 | full search, `&stock.StockSearchParameters{}` | invalid | `data` `required = true` |
| 18 | full search, `Data` present with `Attributes` nil | invalid | `attributes` `required = true` |
| 19 | both, every defined `StockEntity` value 0, 1, 2 | valid | the enum accepts all three defined values |
| 20 | response messages, an empty `data` list with a `Meta` that holds `total` 0 | valid under `protovalidate.Validate` | no `repeated_count_min` rule exists on a response |
| 21 | `StockSearchField` values | `stock.StockSearchField_name` holds exactly 11 entries, with values 0 through 9 and 11; `stock.StockSearchField_name` has no key 10; `stock.StockSearchField_value` has no `STOCK_SEARCH_FIELD_EDITABLE_SUMMARY` key | value 10 and its enum name are reserved; the enum covers all 10 searched fields plus unspecified |
| 22 | generated `Validate()` against an invalid request | returns nil | the no-op fact from [go-genproto facts](#go-genproto-facts) |
| 23 | `StockSearchResult` field shape | `strain_label` has number 7 and type `string` | the field is response metadata, not a search field |

Row 10 is the reason both server plans must trim the query and reject an empty result before they call the repository. Row 22 is the reason both server plans must call `protovalidate.Validate` and not `Validate()`.

## Failure behavior

| Situation | Expected behavior |
| --- | --- |
| `buf lint` reports a rule break | fix the proto file. Do not relax `buf.yaml`. |
| `buf build` cannot resolve `buf/validate/validate.proto` | run `buf dep update`, commit the changed `buf.lock`, and build again |
| `Buf checkup` fails on the pull request while the local lint passes | buf 1.47.2 in CI is older than the local buf. Fix the proto file so that both versions pass. |
| The `Generate go code` workflow fails at the `generate code` step | read the job log. The `govalidators` plugin is installed at v0.3.2 in that job; a new mwitkow annotation is not needed by this plan, so a failure here points at the proto file. |
| The workflow pushes no branch to go-genproto | the push step needs `secrets.REPO_ACCESS_TOKEN`. Report the missing secret; do not hand-copy generated files into go-genproto. |
| `go list -m github.com/dictyBase/go-genproto@latest` still serves the old version | the proxy caches for a short time. Retry, or fetch the version directly with the commit hash of the go-genproto merge. |
| `go build ./...` in modware-stock fails after the raise | the raise moves every generated stock type forward, not only the new ones. Fix the call sites in this plan and list each fix in the [Execution notes](#execution-notes). |
| A contract test row fails | the proto rule does not match this plan. Change the proto file, not the test. |
| The generated types have no `Validate()` method | drop row 22, and record the deviation. The server plans then need no warning about the no-op method. |

## Quality gates

Run every gate in modware-stock after Task 5 and Task 6. All of them must pass before the plan is complete.

```bash
cd /Users/sba964/Projects/devenv/golang/modware-stock
go test ./...
go test -race ./...
golangci-lint fmt
golangci-lint run ./...
gopls check -severity=hint internal/app/service/search_contract_test.go
```

The ArangoDB-backed tests in this repository need `ARANGO_HOST`, `ARANGO_USER`, `ARANGO_PASS` and the optional `ARANGO_PORT`. Start a local server that matches CI:

```bash
docker run -d -p 8529:8529 -e ARANGO_ROOT_PASSWORD=rootpass arangodb:3.11
export ARANGO_HOST=localhost ARANGO_USER=root ARANGO_PASS=rootpass ARANGO_PORT=8529
```

In dictybaseapis run:

```bash
buf lint --path dictybase/stock/stock.proto
buf build --path dictybase/stock/stock.proto
```

## Completion criteria

- [ ] `dictybase/stock/stock.proto` on `master` of dictybaseapis holds 2 new RPC methods, 2 new enums and 8 new messages.
- [ ] Every new name, field name, field number and rule matches [Exact interfaces](#exact-interfaces).
- [ ] No existing message, field name or field number in `stock.proto` changed.
- [ ] `buf lint` and `buf build` pass locally and in the `Buf checkup` workflow.
- [ ] The `Generate go code` workflow finished, and the generated branch is merged into dictyBase/go-genproto.
- [ ] `go list -m github.com/dictyBase/go-genproto@latest` serves a version whose `dictybaseapis/stock/stock.pb.go` holds `StockAutocompleteParameters` and `StockSearchParameters`, and whose `stock_grpc.pb.go` holds `AutocompleteStock` and `SearchStock`.
- [ ] The exact pseudo-version is written in the [Execution notes](#execution-notes).
- [ ] `go.mod` of modware-stock requires that version plus `github.com/bufbuild/protovalidate-go v0.10.0`, and still carries the `exclude google.golang.org/genproto` line.
- [ ] `internal/app/service/search_contract_test.go` holds one test per row of the [Test matrix](#test-matrix), including `strain_label` number and type, and all of them pass.
- [ ] Every gate in [Quality gates](#quality-gates) passes.

## Handoff artifacts

The autocomplete plan and the full-search plan consume these artifacts. Each one must exist, with a concrete value, before either plan starts its Go work.

| # | Artifact | Where it lives | Consumer |
| --- | --- | --- | --- |
| 1 | The exact released `github.com/dictyBase/go-genproto` pseudo-version | [Execution notes](#execution-notes) of this file, and the `require` block of `go.mod` | both server plans, Task "Prerequisite gate" |
| 2 | Generated Go type names: `stock.StockAutocompleteParameters`, `stock.StockAutocompleteParameters_Data`, `stock.StockAutocompleteAttributes`, `stock.StockSuggestion`, `stock.StockSuggestionCollection`, `stock.StockSearchParameters`, `stock.StockSearchParameters_Data`, `stock.StockSearchAttributes`, `stock.StockSearchResult`, `stock.StockSearchResultCollection`, `stock.StockEntity`, `stock.StockSearchField` | `dictybaseapis/stock/stock.pb.go` in go-genproto | both server plans, handler tasks |
| 3 | Generated server method signatures `AutocompleteStock(context.Context, *stock.StockAutocompleteParameters) (*stock.StockSuggestionCollection, error)` and `SearchStock(context.Context, *stock.StockSearchParameters) (*stock.StockSearchResultCollection, error)` | `dictybaseapis/stock/stock_grpc.pb.go` in go-genproto | both server plans, handler tasks |
| 4 | The limit and default table in [Response semantics](#response-semantics) | this file | both server plans, clamp logic |
| 5 | The field and entity mapping tables in [Field enum to repository field path mapping](#field-enum-to-repository-field-path-mapping) | this file | both server plans, response mapping |
| 6 | The two facts that the generated `Validate()` is a no-op and that a whitespace-only query passes `min_len` | [go-genproto facts](#go-genproto-facts) and row 10 of the [Test matrix](#test-matrix) | both server plans, handler guard logic |
| 7 | `github.com/bufbuild/protovalidate-go v0.10.0` as a requirement of modware-stock | `go.mod` | both server plans, handler validation |

## Review focus

1. **No field number reuse and no silent rename.** The diff of `stock.proto` must be additive, apart from the one new import line. A reviewer runs `git diff dictybase/stock/stock.proto | grep '^-' | grep -v '^---'` and expects empty output.
2. **One `Meta` message only.** The file must still hold exactly one `message Meta`. Both new collection messages reference it.
3. **`protovalidate` only on the new messages.** No new message carries a mwitkow `validator.field` option, and no existing message loses one.
4. **Whitespace-only query.** Row 10 of the test matrix asserts that the proto accepts it. A reviewer checks that the gap is written down as a handoff artifact, so neither server plan forgets the trim guard.
5. **Enum completeness.** `StockSearchField` must hold 11 named values: `UNSPECIFIED` plus the 10 searched fields. Value 10 stays reserved. The enum must cover every field that either server plan searches. Do not reuse value 10 or its reserved name.
6. **The pseudo-version is real.** A reviewer rejects a placeholder such as `v0.0.0-TBD`. The version must resolve through `go list -m`, and `go build ./...` must pass with it.
7. **`enum.defined_only` on both requests.** Without the rule, an unknown integer entity reaches the repository. There it filters nothing, and the server gives no error.

## Execution notes

Fill these during implementation. They are part of the handoff.

- Released go-genproto pseudo-version: (not yet released; Task 4 records it)
- dictybaseapis merge commit on `master`: (none yet)
- go-genproto merge commit: (none yet)
- `buf lint` and `buf build` output: (none yet)
- Message and enum counts after the edit: (none yet)
- Call sites that the go-genproto raise broke, and their fixes: (none yet)
- Deviations from the [Exact interfaces](#exact-interfaces) section, with the reason: (none yet)
