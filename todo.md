# Todo

## Plan
Implement fzf's core functionality bottom-up by dependency order: start with foundational types (character handling, memory allocation, utilities), then build the matching algorithms, followed by input processing (ANSI, tokenization), then item management and caching, pattern search, result ranking, merging, and finally the coordination layers (history, reader, matcher).

## Tasks
- [x] Task 1: Implement fuzzy text matching with character handling and scoring (algo package with Chars, Slab, FuzzyMatchV1, FuzzyMatchV2, ExactMatch, PrefixMatch, SuffixMatch, EqualMatch, character normalization, and scoring scheme initialization)
- [x] Task 2: Implement ANSI escape sequence parsing that strips ANSI codes from input text while tracking color/attribute state across foreground, background, underline colors, text attributes, and hyperlinks
- [x] Task 3: Implement concurrent coordination primitives including a thread-safe boolean, an event bus with blocking wait and selective event watching, and general utility functions for display width calculation, string truncation, value clamping, and version comparison
- [x] Task 4: Implement text tokenization that splits input into fields using AWK-style whitespace splitting, string delimiters, or regex delimiters, with range-based field selection and reordering
- [x] Task 5: Implement chunk-based item storage that holds items in fixed-size arrays with thread-safe append, snapshot isolation, tail-trimming, and bitmap-based query result caching for incremental search refinement
- [x] Task 6: Implement result scoring and ranking that evaluates match quality using configurable sort criteria (score, length, begin position, end position, chunk length) and sorts results using radix sort with comparison sort fallback
- [x] Task 7: Implement pattern-based search with extended syntax supporting AND/OR operators, negation, fuzzy/exact/prefix/suffix/equal match types, smart case sensitivity, nth-field matching, and bitmap cache integration
- [x] Task 8: Implement result merging that combines sorted result lists from multiple workers using lazy k-way merge, with a pass-through mode for unfiltered results read directly from chunks
- [ ] Task 9: Implement file-backed command history with cursor navigation, in-memory overrides, maximum size enforcement, and persistent append
- [ ] Task 10: Implement input reading from stdin, shell commands, or filesystem walking with adaptive event polling, delimiter support, and asynchronous event signaling
- [ ] Task 11: Implement parallel matching coordination that distributes chunks across worker goroutines, manages match requests with cancellation support, caches final results, and reports progress
