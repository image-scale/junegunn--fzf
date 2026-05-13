# Acceptance Criteria

## Task 1: Fuzzy text matching with character handling and scoring

### Acceptance Criteria
- [ ] Chars type stores pure ASCII text as bytes (inBytes=true) and mixed/Unicode text as runes (inBytes=false)
- [ ] ToChars correctly detects ASCII vs non-ASCII input and produces the right internal representation
- [ ] Chars.Get(i) returns the correct rune at position i for both byte and rune modes
- [ ] Chars.Length() returns correct character count for both modes
- [ ] Chars.TrimLength() returns length after stripping leading/trailing whitespace, cached after first call
- [ ] Slab pre-allocates int16 and int32 slices for reuse during matching
- [ ] Init("default") configures scoring scheme with correct boundary bonuses and character class tables
- [ ] Init("path") configures path-specific scoring with delimiter bonus for path separators
- [ ] FuzzyMatchV1 finds first fuzzy occurrence using forward scan then backward scan for shortest match
- [ ] FuzzyMatchV2 finds optimal fuzzy match using modified Smith-Waterman scoring matrix
- [ ] FuzzyMatchV2 falls back to V1 when input*pattern exceeds slab capacity or pattern > 1000
- [ ] ExactMatchNaive finds the exact substring occurrence with highest first-char bonus
- [ ] PrefixMatch matches pattern at start of text (after leading whitespace)
- [ ] SuffixMatch matches pattern at end of text (before trailing whitespace)
- [ ] EqualMatch requires full-string equality after trimming whitespace
- [ ] All match functions support case-insensitive mode (pattern given in lowercase)
- [ ] All match functions support Unicode normalization (accented chars match ASCII equivalents)
- [ ] NormalizeRunes maps accented Latin characters and fullwidth ASCII to their base forms
- [ ] Scoring: matches get +16, gap start -3, gap extension -1, boundary bonus +8, camelCase bonus +7
- [ ] First character match bonus is doubled (bonusFirstCharMultiplier=2)
- [ ] Empty pattern returns Result{0,0,0} for fuzzy/exact/prefix, Result{len,len,0} for suffix
- [ ] Non-matching input returns Result{-1,-1,0}
- [ ] Long strings (>MaxUint16) are handled correctly

## Task 2: ANSI escape sequence parsing

### Acceptance Criteria
- [ ] nextAnsiEscapeSequence finds CSI sequences (\x1b[...m), OSC sequences (\x1b]...), SO/SI (\x0e/\x0f), backspace overstrike (.\x08), and newlines
- [ ] extractColor strips ANSI codes from input and returns the clean string, color offset array, and final state
- [ ] Foreground colors are parsed from SGR 30-37 (basic), 90-97 (bright), 38;5;N (256-color), 38;2;R;G;B (24-bit)
- [ ] Background colors are parsed from SGR 40-47 (basic), 100-107 (bright), 48;5;N (256-color), 48;2;R;G;B (24-bit)
- [ ] Text attributes: bold (1), dim (2), italic (3), underline (4), blink (5), reverse (7), strikethrough (9)
- [ ] Underline styles via colon sub-parameters: 4:0 (none), 4:1 (single), 4:2 (double), 4:3 (curly), 4:4 (dotted), 4:5 (dashed)
- [ ] Underline color via SGR 58 (58;5;N for 256, 58;2;R;G;B for 24-bit)
- [ ] SGR 0 or empty sequence resets all attributes
- [ ] State carries over across calls when previous state is passed in
- [ ] OSC 8 hyperlink sequences are parsed (params and URI extracted)
- [ ] Erase to end of line ([K, [0K) sets line background from current bg
- [ ] parseAnsiCode correctly extracts numeric values with semicolon/colon separators

## Task 3: Concurrent coordination primitives and utility functions

### Acceptance Criteria
- [ ] AtomicBool provides thread-safe Get() and Set() using atomic int32 operations
- [ ] EventBox provides Set/Wait/Peek/Watch/Unwatch for event coordination with mutex+condvar
- [ ] EventBox.Wait blocks until events are set, then calls callback with event map
- [ ] EventBox.Set broadcasts to waiters unless event is in the ignore list
- [ ] EventBox.Watch/Unwatch controls which events trigger broadcasts
- [ ] Constrain clamps a value between min and max
- [ ] AsUint16 clamps an int to [0, MaxUint16]
- [ ] Once returns a function that returns a given bool value once, then the opposite forever
- [ ] RunesWidth calculates display width of runes with tab stop support and overflow detection
- [ ] Truncate truncates a string to fit within a display width limit
- [ ] CompareVersions compares dot-separated version strings numerically

## Task 4: Text tokenization with field selection

### Acceptance Criteria
- [ ] Range type represents nth-expressions with begin/end fields; rangeEllipsis=0 is the sentinel for open-ended ranges
- [ ] Range.IsFull() returns true when both begin and end are rangeEllipsis (i.e., ".." meaning all fields)
- [ ] ParseRange parses ".." as full range, "N.." as begin-open, "..N" as end-open, "N..M" as closed range, "N" as single field
- [ ] ParseRange normalizes begin=1 to rangeEllipsis and end=-1 to rangeEllipsis
- [ ] ParseRange rejects invalid inputs: zero index, mixed negative/positive ranges, non-numeric, too many ".." separators
- [ ] Token type holds a *Chars text pointer and int32 prefixLength tracking position in original string
- [ ] Delimiter type has optional regex and str fields; IsAwk() returns true when both are nil
- [ ] AWK-style tokenization splits on whitespace runs (tab/space/newline), each token includes trailing whitespace, leading whitespace is tracked as prefix length
- [ ] String delimiter tokenization splits after each occurrence of the delimiter string
- [ ] Regex delimiter tokenization splits at regex match boundaries, each token includes the matched delimiter
- [ ] Transform selects and reorders tokens according to Range specifications, supporting negative indices (counting from end)
- [ ] Transform handles single field, field ranges, open-ended ranges, and full range (rangeEllipsis)
- [ ] JoinTokens concatenates all token texts into a single string
- [ ] StripLastDelimiter removes trailing delimiter (string, regex match, or whitespace for AWK mode)
- [ ] SplitNth parses comma-separated nth-expressions into a slice of Ranges
- [ ] DelimiterFromString creates a Delimiter: single chars and non-regex strings use str field, valid regex patterns use regex field
- [ ] RangesToString converts Range slice back to string representation

## Task 5: Chunk-based item storage with bitmap query caching

### Acceptance Criteria
- [ ] Item stores text as charutil.Chars, optional origText for raw bytes, optional colors for ANSI ranges, and an Index piggybacked in Chars
- [ ] Item.AsString(stripAnsi) returns origText (with optional ANSI stripping) or falls back to text.ToString()
- [ ] Item.TrimLength() delegates to Chars.TrimLength() for cached trimmed length
- [ ] Item.Colors() returns color ranges or empty slice if nil
- [ ] Chunk stores items in a fixed-size array [chunkSize]Item with a count field
- [ ] Chunk.IsFull() returns true when count equals chunkSize (1024)
- [ ] ItemBuilder closure populates an Item from raw bytes, returns success
- [ ] ChunkList provides thread-safe Push that appends items, creating new Chunks as needed
- [ ] ChunkList.Snapshot returns an immutable copy of the chunk slice with boundary chunks duplicated
- [ ] Snapshot with tail>0 truncates from the front, keeping only the last tail items, retiring evicted chunks from cache
- [ ] Snapshot immutability: pushing more items after snapshot does not affect previous snapshot
- [ ] CountItems efficiently counts total items assuming middle chunks are full
- [ ] GetItems collects the first n items across all chunks
- [ ] ChunkList.Clear sets chunks to nil under lock
- [ ] ChunkList.ForEachItem iterates all items under lock, calling done callback while locked
- [ ] ChunkBitmap is a fixed-size [chunkBitWords]uint64 array (16 words for 1024-bit bitmap)
- [ ] ChunkCache.Add only caches full chunks with non-empty keys and matchCount <= queryCacheMax
- [ ] ChunkCache.Lookup returns exact bitmap match for a (chunk, key) pair
- [ ] ChunkCache.Search finds bitmap for longest prefix or suffix of the key (incremental refinement)
- [ ] ChunkCache.Clear and retire remove cache entries under lock

## Task 6: Result scoring and ranking with radix sort

### Acceptance Criteria
- [ ] Offset type is [2]int32 representing [begin, end) of a matched substring
- [ ] Result type holds *Item pointer and [4]uint16 points array for lexicographic ranking
- [ ] Criterion enum: ByScore, ByChunk, ByLength, ByBegin, ByEnd, ByPathname
- [ ] SortCriteria configurable slice of criteria, max 4; first criterion maps to points[3] (most significant)
- [ ] BuildResult sorts offsets, computes minBegin/minEnd/maxEnd, delegates to BuildResultFromBounds
- [ ] ByScore: points = MaxUint16 - score (higher score sorts first)
- [ ] ByChunk: expands match to whitespace boundaries, stores chunk length
- [ ] ByLength: stores item TrimLength
- [ ] ByBegin: distance from whitespace prefix to minEnd
- [ ] ByEnd: inverted proportion of maxEnd position within trimmed length
- [ ] ByPathname: distance from last path separator to minBegin
- [ ] CompareRanks compares points[3] down to points[0], tiebreaks on item index (lower wins normal, higher wins tac)
- [ ] SortKey packs [4]uint16 into uint64 with points[3] in high bits
- [ ] RadixSortResults uses LSD radix sort (8 passes over 64-bit key) for n>=128, falls back to comparison sort below 128
- [ ] Radix sort skips passes where all items share the same byte value
- [ ] In tac mode, runs of equal sort keys are reversed after sorting
- [ ] ByRelevance and ByRelevanceTac implement sort.Interface
- [ ] MinRank returns worst possible rank (MaxUint16 points[0], MinInt32 index)
- [ ] CompareOffsets sorts offsets lexicographically by begin then end

## Task 7: Pattern-based search with extended syntax

### Acceptance Criteria
- [ ] Term types: fuzzy, exact, exactBoundary, prefix, suffix, equal
- [ ] ParseTerms parses extended query syntax with AND (space) and OR (|) operators
- [ ] Negation with ! prefix inverts match logic (match found = condition fails, no match = condition succeeds)
- [ ] Smart case: per-term in extended mode, per-pattern in non-extended; uppercase triggers case-sensitive
- [ ] BuildPattern constructs Pattern with procFun mapping each term type to its algo function
- [ ] Pattern.Match integrates with ChunkCache for bitmap-based incremental refinement
- [ ] MatchItem handles both extended (AND-of-OR term matching) and basic (single fuzzy/exact) modes
- [ ] extendedMatch returns offsets only when all termSets match (AND semantics)
- [ ] Within each termSet, terms are OR alternatives (at least one must match)
- [ ] Nth-field matching via transformInput tokenizes and transforms item text, caching results per revision
- [ ] Direct fast path bypasses MatchItem for single non-inverse fuzzy term with no nth
- [ ] Cache key built from cacheable terms only (non-inverse, single-element termSets of default type)
- [ ] Pattern caching via patternCache map keyed by normalized string
- [ ] Denylist and startIndex support for skipping specific items
- [ ] iter adjusts match offsets by token prefixLength for correct positioning

## Task 8: Result merging with lazy k-way merge

### Acceptance Criteria
- [ ] Merger holds multiple sorted result lists and provides a single globally-sorted view via lazy k-way merge
- [ ] mergedGet lazily advances cursors across lists, picking the minimum-rank result at each step
- [ ] NewMerger creates a merge-mode merger with cursors for each list
- [ ] PassMerger creates a pass-through merger that reads items directly from chunks in original order
- [ ] EmptyMerger returns a merger with zero items
- [ ] Get returns items by index, supporting both sorted merge and unsorted pass-through modes
- [ ] Tac mode reverses the index mapping for pass-through and unsorted modes
- [ ] FindIndex locates an item by its item index (O(1) in pass mode, O(n) scan otherwise)
- [ ] ToMap converts all items to a map keyed by item index
- [ ] Cacheable returns true when count is below MergerCacheMax (100000)
- [ ] Length returns total count across all lists

## Task 9: File-backed command history

### Acceptance Criteria
- [ ] NewHistory reads existing history file or creates a new one with 0600 permissions
- [ ] NewHistory returns error for invalid paths (permission denied, directory paths)
- [ ] History lines are split on newline, with an empty sentinel line appended at the end
- [ ] Append adds a non-empty line to history, enforces maxSize by dropping oldest lines, and writes to file
- [ ] Empty lines are silently ignored by Append
- [ ] Override modifies the current line in memory without writing to file
- [ ] Current returns the overridden value if present, otherwise the original line
- [ ] Previous/Next navigate the cursor within bounds and return the current line
- [ ] Cursor starts at the last position (empty sentinel line)

## Task 10: Input reading with adaptive event polling

### Acceptance Criteria
- [ ] Reader reads from shell commands via ReadFromCommand, executing with $SHELL -c
- [ ] Reader reads from channels via ReadChannel
- [ ] Reader reads from stdin via ReadFromStdin
- [ ] Feed splits input on newline (or null byte if delimNil) using slab-based buffering
- [ ] Leftover bytes from partial reads are accumulated and pushed when delimiter is found or EOF
- [ ] StartEventPoller polls for EvtReadNew with adaptive interval (min 10ms, step 5ms, max 50ms)
- [ ] Fin signals EvtReadFin and reports command failure (non-nil command pointer) unless killed or successful
- [ ] Terminate kills the running process and sets killed flag under lock
- [ ] Wait mode blocks Fin until the event poller acknowledges via finChan

## Task 11: Parallel matching coordination

### Acceptance Criteria
- [ ] Matcher distributes chunks across worker goroutines using atomic counter for work stealing
- [ ] NewMatcher configures partitions from runtime.NumCPU() or explicit thread count
- [ ] Loop processes MatchRequests from reqBox, blocking until events arrive
- [ ] Loop exits cleanly on reqQuit event
- [ ] Cache invalidation on sort mode change or revision change
- [ ] Major revision change clears ChunkCache
- [ ] Merger cache hit returns cached result when item count unchanged and pattern matches
- [ ] Item count change invalidates merger cache
- [ ] scan returns empty merger for zero chunks
- [ ] scan returns pass merger for empty pattern
- [ ] scan spawns min(partitions, numChunks) workers
- [ ] Workers use atomic counter to claim chunks without contention
- [ ] Each worker accumulates matches and sends partial results via channel
- [ ] Workers apply radix sort when sort mode is enabled and pattern is sortable
- [ ] Progress events emitted after ProgressMinDuration (200ms)
- [ ] Scan cancellation via cancelScan flag or reqReset in reqBox
- [ ] CancelScan blocks new scans via scanMutex until ResumeScan
- [ ] Reset sends MatchRequest as reqReset (cancel=true) or reqRetry (cancel=false)
- [ ] Cacheable results stored in mergerCache keyed by pattern string
- [ ] EvtSearchFin event set with MatchResult on successful completion
- [ ] Pre-allocated Slab and sortBuf arrays reused across scans
