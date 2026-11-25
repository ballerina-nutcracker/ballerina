---
name: filling-subset-doc
description: Update doc/lang/subset*.md for compiler language features and fixes.
---

- Use the subset specified in the request. If none is specified, use the open subset's `doc/lang/subset<N>.md`: the highest numbered one if the latest `vX.Y.Z` tag doesn't contain it, otherwise a new `subset<N+1>.md`. Don't edit a subset doc in that tag: it is released, and the [Released Subset check](../../../.github/workflows/released-subset.yml) fails any change to it.
- Identify the relevant specification sections using the target document's existing references.
- Verify support against the implementation and relevant corpus cases. Document remaining restrictions or implementation-specific behaviour under the corresponding statement or expression bullet.
- Remove restrictions only when implementation and corpus evidence show they no longer apply. When the implementation supports the full relevant specification, remove the restrictions instead of enumerating each supported case. Do not list unsupported cases that an existing "only supports" restriction already excludes.
- Validate links in the changed subset documents with `python3 .github/scripts/check_doc_links.py doc/lang/subset<N>.md`, using the same [link checker](../../../.github/scripts/check_doc_links.py) as [CI](../../../.github/workflows/doc-links.yml).
