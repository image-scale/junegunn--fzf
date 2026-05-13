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
