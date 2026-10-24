# Standard conformance

Conformance uses shared property and report/relationship checks at both unit and
integration levels, against the declared CTRF specification version. These are
coverage families, not an exhaustive set of cases. Existing reporter tests remain
separate.

```sh
bash tests/run.sh unit
bash tests/run.sh integration
# Run both, even if unit fails:
bash tests/run.sh conformance
```

Unit tests capture serialized output from controlled Go events. Integration uses
`go install` into a temporary consumer and runs real `go test -json`, including
pass/fail/skip, subtests, parallel tests, panic, empty runs, explicit reruns,
repeated runs and coordinated shard labels. Reruns are separate native test
executions; ordinary `-count` repetition is not treated as retry evidence. Go
module dependencies must be available. Missing setup fails the run.

Temporary consumers are cleaned automatically. Set `CTRF_STANDARD_ARTIFACTS` to an
absolute directory outside the checkout to retain assessment CTRF, native fixture
streams, source expectations, subject reports and diagnostics;
otherwise result artifacts are temporary too.

Both suites generate schema-validated CTRF under `$CTRF_STANDARD_ARTIFACTS/reports`.
CI publishes built-in summaries and folded suite tables, with separate unit and
integration identities, schema properties and specification relationships. The
stable Go coverage job maintains one updating `reporter-conformance` PR comment;
all matrix jobs retain evidence. Fork PRs retain summaries and artifacts.

- `shared/catalogue.json`: common family IDs, copied from the reviewed snapshot.
- `shared/checks.go`: native schema and specification assertions.
- `shared/expected_failures.go`: native strict expected-failure engine.
- `go.json`: explicit capabilities and reasons.
- `../../native/fixtures`: native inputs, independent expectations and defect matchers.
- `../shared-files.json`: native file hashes and reference/schema provenance.

Unsupported optional features remain skipped with reasons. Supported fields need
positive evidence; every emitted document receives full schema validation.
Mandatory violations fail, recommendations warn, and missing evidence cannot pass.
Temporary expected failures still execute and narrowly match a recorded defect;
different errors and unexpected or partial passes fail CI. Remove markers with
fixes. Issue references live in native records; local follow-up keys must become
public issue links before publication.

Expected failures use **Other**, explicit labels and defect metadata, in separate
tracked-defect groups. They are not conformance passes. The top summary separates
skipped and Other; the built-in folded table combines skipped/pending/Other.
Expected-failure groups contain tracked defects only.

`bash tests/run.sh checker` runs independent controls under `tests/standard/checker`
and writes only to `checker-reports`, publishing a separate `reporter-checker`
comment. `bash tests/run.sh standard` runs shared controls and both contract levels
even if one fails. `bash tests/run.sh ordinary` runs existing reporter tests and
native fixture/matcher/reporting controls, publishing `reporter-tests` separately.
`bash tests/run.sh all` runs all scopes.

This is a native Go implementation of the shared IDs and assertion semantics;
it does not claim byte identity with JavaScript helpers. The nested `tests/go.mod`
keeps validation dependencies out of the reporter's runtime module and binaries.
