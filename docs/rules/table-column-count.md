# table-column-count

> require every table row to match the header's column count

- **Default severity:** error
- **Fix:** no automatic fix — surfaced for a human to resolve

## How to fix

A GFM pipe table's column count is fixed by its header row. When a data row has more or fewer cells, the renderer silently drops or pads cells, so the table no longer means what the author wrote. This rule reports each row whose cell count differs from the header. It emits no fix because the correct cell boundaries cannot be inferred.

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
