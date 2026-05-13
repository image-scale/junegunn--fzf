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
