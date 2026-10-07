# modware-stock: nil `attributes` panics the gRPC server

**Status:** open, not fixed
**Repo verified:** `github.com/dictyBase/modware-stock`, branch `develop`, commit `bf52343`
**Related fix already shipped elsewhere:** `modware-order` PR #269 (merge commit `fafc8c1`, released as `v1.2.1`)

## Summary

A gRPC request can arrive with its `data` object present but the `attributes` object inside it
missing. Validation accepts that shape, execution continues, and the first line that reads an
attribute dereferences a nil pointer. The panic is not contained: the server installs no recovery
interceptor, so one malformed request can terminate the process.

This matters because the affected RPCs are reachable by any authenticated caller of the stock
service. It is a denial-of-service bug, not a data-corruption bug.

## Root cause

Two separate things have to line up, and in this repo both do.

**1. The validator treats a missing nested object as fine.** Requests are validated with the
generated `Validate()` methods (the `mwitkow`/`protoc-gen-validate` family). Those validators only
inspect nested messages that are actually present; a nested message that is `nil` is skipped rather
than reported. Confirmed empirically with a throwaway test in `internal/app/service`:

```go
cases := map[string]func() error{
    "NewStrain{}":                      func() error { return (&stock.NewStrain{}).Validate() },
    "NewStrain{Data only}":             func() error { return (&stock.NewStrain{Data: &stock.NewStrain_Data{Type: "strain"}}).Validate() },
    "ExistingStrain{}":                 func() error { return (&stock.ExistingStrain{}).Validate() },
    "ExistingStrain{Data only}":        func() error { return (&stock.ExistingStrain{Data: &stock.ExistingStrain_Data{Type: "strain", Id: "1"}}).Validate() },
    "StrainUpdate{}":                   func() error { return (&stock.StrainUpdate{}).Validate() },
    "StrainUpdate{Data+Id, no attrs}":  func() error { return (&stock.StrainUpdate{Data: &stock.StrainUpdate_Data{Type: "strain", Id: "1"}}).Validate() },
    "NewPlasmid{Data only}":            func() error { return (&stock.NewPlasmid{Data: &stock.NewPlasmid_Data{Type: "plasmid"}}).Validate() },
    "ExistingPlasmid{Data only}":       func() error { return (&stock.ExistingPlasmid{Data: &stock.ExistingPlasmid_Data{Type: "plasmid", Id: "1"}}).Validate() },
    "PlasmidUpdate{Data+Id, no attrs}": func() error { return (&stock.PlasmidUpdate{Data: &stock.PlasmidUpdate_Data{Type: "plasmid", Id: "1"}}).Validate() },
}
for name, fn := range cases {
    t.Logf("PROBE %-40s err=%v", name, fn() != nil) // true means the validator rejected it
}
```

Observed result — every `Data`-only case returns `err=false`, meaning validation passed and the
request moves on:

```
PROBE NewStrain{}                        err=true    (rejected: whole message empty)
PROBE NewStrain{Data only}               err=false   <-- slips through
PROBE ExistingStrain{}                   err=true
PROBE ExistingStrain{Data only}          err=false   <-- slips through
PROBE StrainUpdate{}                     err=true
PROBE StrainUpdate{Data+Id, no attrs}    err=false   <-- slips through
PROBE NewPlasmid{Data only}              err=false   <-- slips through
PROBE ExistingPlasmid{Data only}         err=false   <-- slips through
PROBE PlasmidUpdate{Data+Id, no attrs}   err=false   <-- slips through
```

Only the fully empty messages are rejected. That is the whole gap: there is no check that
`data.attributes` exists.

**2. The code then assumes the object is there.** Attribute reads happen in the service layer and,
for the update and plasmid paths, in the repository layer as well.

**3. Nothing catches the resulting panic.** `internal/app/server/server.go` builds the server with
tag and logging interceptors only:

```go
grpcS := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        grpc_ctxtags.UnaryServerInterceptor(),
        grpc_logrus.UnaryServerInterceptor(getLogger(c)),
    ),
)
```

`grpc-go` does not recover panics in handlers — the process dies. There is no
`grpc_recovery`/`recovery.UnaryServerInterceptor` anywhere in the tree.

## Affected RPCs

| RPC | File:line | Risk (dereferenced read) |
|---|---|---|
| `CreateStrain` | `internal/app/service/strain.go:71-72` | `r.Data.Attributes.DictyStrainProperty`; then `internal/repository/arangodb/strain_write.go:18-19` (`ns.Data.Attributes.Parent`, `.DictyStrainProperty`) |
| `LoadStrain` | `internal/app/service/strain.go:46-47` | `r.Data.Attributes.DictyStrainProperty`; then `strain_write.go:124-125` (`es.Data.Attributes.Parent`, `.DictyStrainProperty`) |
| `UpdateStrain` | `internal/repository/arangodb/strain_write.go:78,83-85` | `ar.checkStock(us.Data.Id)` then `us.Data.Attributes.DictyStrainProperty` and `getUpdatableStrainBindParams(us.Data.Attributes)` |
| `CreatePlasmid` | `internal/app/service/plasmid_workflow.go:177-184` | `req.Data.Attributes.DictyPlasmidProperty` inside `applyDefaultPlasmidProperty`, called from `validateNewPlasmidRequest` (line 457) |
| `LoadPlasmid` | `internal/app/service/plasmid_workflow.go:273-280` | same read inside `applyDefaultExistingPlasmidProperty` |
| `LoadPlasmid` (repository) | `internal/app/service/plasmid_workflow.go:659-663` | `data.Attributes.DictyPlasmidProperty` inside `loadPlasmidInRepository` |

Two things to note about this table:

- `UpdateStrain` has **no service-layer dereference**; it passes the request straight to
  `repo.EditStrain`, which panics. A service-level grep alone will miss it.
- The plasmid dereferences run inside `IOE.Map` steps of the workflow pipeline, which sit
  **outside** the surrounding `IOE.TryCatchError` that only wraps `request.Validate()`. So a guard
  placed inside the try/catch does not protect them; the check has to happen before the workflow's
  mapping steps run, or the panic propagates regardless.

This list was derived by reading the code and probing the validators. It has not been confirmed
end-to-end by sending the malformed requests to a running server — do that as part of the fix, per
RPC, and treat any path not listed here as unverified rather than safe.

## Fix

Add an explicit guard after validation in each affected RPC, before any attribute access. The
pattern used and reviewed in `modware-order` PR #269:

```go
if err := r.Validate(); err != nil {
    return st, aphgrpc.HandleInvalidParamError(ctx, err)
}
if r.Data == nil || r.Data.Attributes == nil {
    return st, aphgrpc.HandleInvalidParamError(
        ctx,
        errors.New("expect strain attributes"),
    )
}
```

Apply the same shape with the matching noun for plasmids (`"expect plasmid attributes"`).

For the plasmid workflows, put the guard in the step that validates the request
(`validateNewPlasmidRequest`, `validateExistingPlasmidRequest`) and return the invalid-argument
error from there, so nothing reaches `applyDefaultPlasmidProperty` with a nil attributes pointer.

Two design notes worth preserving:

- Returning the error is the point. A guard that silently substitutes an empty attributes object
  would hide a malformed client instead of reporting it.
- Services in this family already build their own error before calling
  `aphgrpc.HandleNotFoundError`, because that helper calls `err.Error()` and panics on a nil error.
  The same care applies here: pass a constructed error.

Optionally, and separately, add a recovery interceptor to the server as defence in depth. That
changes behaviour for every handler and deserves its own review; it does not replace the guards,
because a recovered panic still returns an internal error to a caller who sent a malformed request.

## Tests to add

The service-level harness for this repo lives in
`internal/app/service/strain_test_helpers.go` and `internal/app/service/plasmid_crud_test_helpers.go`
(`*_test_helpers.go` files are compiled into normal builds, not test-only). Following that pattern,
each test gets a fresh disposable ArangoDB database, a real repository, a no-op publisher, and a
gRPC server on `bufconn` dialled as `passthrough:///bufnet`.

For each affected RPC, send the `Data`-only form and assert an `codes.InvalidArgument` status
rather than a dead test process:

```go
_, err := client.CreateStrain(ctx, &stock.NewStrain{
    Data: &stock.NewStrain_Data{Type: "strain"},
})
require.Equal(t, codes.InvalidArgument, status.Code(err))
```

Also keep one valid-request assertion per RPC so the guard cannot be written so broadly that it
rejects good input. Confirm that a valid create still persists and returns its generated ID.

`TestCreateOrderPersistsUserInfo` and its siblings in `modware-order` show the shape of these
service-level tests.

## Verification steps for whoever fixes this

1. Confirm the current behaviour first, so the fix has a baseline: run the probe above, then send a
   `Data`-only request to a locally running server and observe the panic.
2. Add the guards, add the tests.
3. Run the suite locally against a real ArangoDB. The environment provides `ARANGO_HOST`,
   `ARANGO_USER` and `ARANGO_PASS`; tests create and drop their own databases, so an existing local
   instance is enough.
4. Run `golangci-lint` at the version pinned in `.github/workflows/lint.yml` (`v2.14.0-alpine`) — a
   locally installed older version can report a different result.
5. Run the full test suite and check `gopls check -severity=hint` on every file you touched.
6. CI needs no change: `ci.yml` and `testcov.yml` already start an `arangodb` service and export the
   `ARANGO_*` variables.

## Open questions

- Do the update paths for plasmids dereference attributes in the repository layer the way
  `EditStrain` does? Not confirmed here; check `internal/repository/arangodb/plasmid_write_helpers.go`.
- Are there other RPCs in the `ListStrains*`/`ListPlasmids*` family that read a request attribute
  without a guard? The filters looked safe, but this was a read-through, not a proof.
- Should the recovery interceptor be added in the same change or held back as its own PR?

## Reference implementation

`modware-order` had the same bug in `CreateOrder`, `UpdateOrder` and `LoadOrder`. It was fixed with
the guard pattern above and released in `v1.2.1`; `PR #269` contains both the fix and its
service-level tests.