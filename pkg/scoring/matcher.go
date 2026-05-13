package scoring

import (
	"bytes"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/fzf/finder/pkg/charutil"
)

type MatchResult struct {
	Start int
	End   int
	Score int
}

const (
	ScoreMatch        = 16
	ScoreGapStart     = -3
	ScoreGapExtension = -1

	BonusBoundary             = ScoreMatch / 2
	BonusNonWord              = ScoreMatch / 2
	BonusCamelCase            = BonusBoundary + ScoreGapExtension
	BonusConsecutive          = -(ScoreGapStart + ScoreGapExtension)
	BonusFirstCharMultiplier  = 2
)

var (
	BonusBoundaryWhite     int16 = BonusBoundary + 2
	BonusBoundaryDelimiter int16 = BonusBoundary + 1

	startCharClass = classWhite

	asciiClasses [unicode.MaxASCII + 1]charCategory
	bonusLookup  [classNumber + 1][classNumber + 1]int16
)

var separatorChars = "/,:;|"

const whitespaceChars = " \t\n\v\f\r\x85\xA0"

type charCategory int

const (
	classWhite charCategory = iota
	classNonWord
	classDelimiter
	classLower
	classUpper
	classLetter
	classNumber
)

func Setup(scheme string) bool {
	switch scheme {
	case "default":
		BonusBoundaryWhite = BonusBoundary + 2
		BonusBoundaryDelimiter = BonusBoundary + 1
	case "path":
		BonusBoundaryWhite = BonusBoundary
		BonusBoundaryDelimiter = BonusBoundary + 1
		if os.PathSeparator == '/' {
			separatorChars = "/"
		} else {
			separatorChars = string([]rune{os.PathSeparator, '/'})
		}
		startCharClass = classDelimiter
	case "history":
		BonusBoundaryWhite = BonusBoundary
		BonusBoundaryDelimiter = BonusBoundary
	default:
		return false
	}
	for i := 0; i <= unicode.MaxASCII; i++ {
		ch := rune(i)
		cat := classNonWord
		if ch >= 'a' && ch <= 'z' {
			cat = classLower
		} else if ch >= 'A' && ch <= 'Z' {
			cat = classUpper
		} else if ch >= '0' && ch <= '9' {
			cat = classNumber
		} else if strings.ContainsRune(whitespaceChars, ch) {
			cat = classWhite
		} else if strings.ContainsRune(separatorChars, ch) {
			cat = classDelimiter
		}
		asciiClasses[i] = cat
	}
	for i := 0; i <= int(classNumber); i++ {
		for j := 0; j <= int(classNumber); j++ {
			bonusLookup[i][j] = computeBonus(charCategory(i), charCategory(j))
		}
	}
	return true
}

func computeBonus(prev charCategory, cur charCategory) int16 {
	if cur > classNonWord {
		switch prev {
		case classWhite:
			return BonusBoundaryWhite
		case classDelimiter:
			return BonusBoundaryDelimiter
		case classNonWord:
			return BonusBoundary
		}
	}
	if prev == classLower && cur == classUpper || prev != classNumber && cur == classNumber {
		return BonusCamelCase
	}
	switch cur {
	case classNonWord, classDelimiter:
		return BonusNonWord
	case classWhite:
		return BonusBoundaryWhite
	}
	return 0
}

func classifyNonAscii(ch rune) charCategory {
	if unicode.IsLower(ch) {
		return classLower
	} else if unicode.IsUpper(ch) {
		return classUpper
	} else if unicode.IsNumber(ch) {
		return classNumber
	} else if unicode.IsLetter(ch) {
		return classLetter
	} else if unicode.IsSpace(ch) {
		return classWhite
	} else if strings.ContainsRune(separatorChars, ch) {
		return classDelimiter
	}
	return classNonWord
}

func classify(ch rune) charCategory {
	if ch <= unicode.MaxASCII {
		return asciiClasses[ch]
	}
	return classifyNonAscii(ch)
}

func bonusAt(input *charutil.Chars, idx int) int16 {
	if idx == 0 {
		return BonusBoundaryWhite
	}
	return bonusLookup[classify(input.Get(idx-1))][classify(input.Get(idx))]
}

func normalizeChar(r rune) rune {
	if r < 0x00C0 || r > 0xFF61 {
		return r
	}
	if n, ok := normalizedMap[r]; ok {
		return n
	}
	return r
}

type MatchFunc func(caseSensitive bool, normalize bool, forward bool, input *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int)

func makePositions(withPos bool, length int) *[]int {
	if withPos {
		pos := make([]int, 0, length)
		return &pos
	}
	return nil
}

func grabSlab16(offset int, slab *charutil.Slab, size int) (int, []int16) {
	if slab != nil && cap(slab.I16) > offset+size {
		sl := slab.I16[offset : offset+size]
		return offset + size, sl
	}
	return offset, make([]int16, size)
}

func grabSlab32(offset int, slab *charutil.Slab, size int) (int, []int32) {
	if slab != nil && cap(slab.I32) > offset+size {
		sl := slab.I32[offset : offset+size]
		return offset + size, sl
	}
	return offset, make([]int32, size)
}

func flipIndex(index int, total int, forward bool) int {
	if forward {
		return index
	}
	return total - index - 1
}

func allAsciiRunes(runes []rune) bool {
	for _, r := range runes {
		if r >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func findByteEither(data []byte, a, b byte) int {
	i1 := bytes.IndexByte(data, a)
	if i1 == 0 {
		return 0
	}
	scope := data
	if i1 > 0 {
		scope = data[:i1]
	}
	if i2 := bytes.IndexByte(scope, b); i2 >= 0 {
		return i2
	}
	return i1
}

func findLastByteEither(data []byte, a, b byte) int {
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == a || data[i] == b {
			return i
		}
	}
	return -1
}

func skipToMatch(input *charutil.Chars, caseSensitive bool, b byte, from int) int {
	arr := input.Bytes()[from:]
	if !caseSensitive && b >= 'a' && b <= 'z' {
		idx := findByteEither(arr, b, b-32)
		if idx < 0 {
			return -1
		}
		return from + idx
	}
	idx := bytes.IndexByte(arr, b)
	if idx < 0 {
		return -1
	}
	return from + idx
}

func findAsciiRange(input *charutil.Chars, pattern []rune, caseSensitive bool) (int, int) {
	if !input.IsBytes() {
		return 0, input.Length()
	}
	if !allAsciiRunes(pattern) {
		return -1, -1
	}
	firstIdx, idx, lastIdx := 0, 0, 0
	var b byte
	for pidx := range pattern {
		b = byte(pattern[pidx])
		idx = skipToMatch(input, caseSensitive, b, idx)
		if idx < 0 {
			return -1, -1
		}
		if pidx == 0 && idx > 0 {
			firstIdx = idx - 1
		}
		lastIdx = idx
		idx++
	}
	scope := input.Bytes()[lastIdx:]
	if len(scope) > 1 {
		tail := scope[1:]
		var end int
		if !caseSensitive && b >= 'a' && b <= 'z' {
			end = findLastByteEither(tail, b, b-32)
		} else {
			end = bytes.LastIndexByte(tail, b)
		}
		if end >= 0 {
			return firstIdx, lastIdx + 1 + end + 1
		}
	}
	return firstIdx, lastIdx + 1
}

func computeMatchScore(caseSensitive bool, normalize bool, text *charutil.Chars, pattern []rune, sidx int, eidx int, withPos bool) (int, *[]int) {
	pidx, score, inGap, consecutive, firstBonus := 0, 0, false, 0, int16(0)
	pos := makePositions(withPos, len(pattern))
	prev := startCharClass
	if sidx > 0 {
		prev = classify(text.Get(sidx - 1))
	}
	for idx := sidx; idx < eidx; idx++ {
		ch := text.Get(idx)
		cat := classify(ch)
		if !caseSensitive {
			if ch >= 'A' && ch <= 'Z' {
				ch += 32
			} else if ch > unicode.MaxASCII {
				ch = unicode.To(unicode.LowerCase, ch)
			}
		}
		if normalize {
			ch = normalizeChar(ch)
		}
		if ch == pattern[pidx] {
			if withPos {
				*pos = append(*pos, idx)
			}
			score += ScoreMatch
			bonus := bonusLookup[prev][cat]
			if consecutive == 0 {
				firstBonus = bonus
			} else {
				if bonus >= BonusBoundary && bonus > firstBonus {
					firstBonus = bonus
				}
				bonus = max(bonus, firstBonus, BonusConsecutive)
			}
			if pidx == 0 {
				score += int(bonus * BonusFirstCharMultiplier)
			} else {
				score += int(bonus)
			}
			inGap = false
			consecutive++
			pidx++
		} else {
			if inGap {
				score += ScoreGapExtension
			} else {
				score += ScoreGapStart
			}
			inGap = true
			consecutive = 0
			firstBonus = 0
		}
		prev = cat
	}
	return score, pos
}

func FuzzyMatchV1(caseSensitive bool, normalize bool, forward bool, text *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	if len(pattern) == 0 {
		return MatchResult{0, 0, 0}, nil
	}
	idx, _ := findAsciiRange(text, pattern, caseSensitive)
	if idx < 0 {
		return MatchResult{-1, -1, 0}, nil
	}

	pidx := 0
	sidx := -1
	eidx := -1
	textLen := text.Length()
	patLen := len(pattern)

	for index := range textLen {
		ch := text.Get(flipIndex(index, textLen, forward))
		if !caseSensitive {
			if ch >= 'A' && ch <= 'Z' {
				ch += 32
			} else if ch > unicode.MaxASCII {
				ch = unicode.To(unicode.LowerCase, ch)
			}
		}
		if normalize {
			ch = normalizeChar(ch)
		}
		pch := pattern[flipIndex(pidx, patLen, forward)]
		if ch == pch {
			if sidx < 0 {
				sidx = index
			}
			pidx++
			if pidx == patLen {
				eidx = index + 1
				break
			}
		}
	}

	if sidx >= 0 && eidx >= 0 {
		pidx--
		for index := eidx - 1; index >= sidx; index-- {
			tidx := flipIndex(index, textLen, forward)
			ch := text.Get(tidx)
			if !caseSensitive {
				if ch >= 'A' && ch <= 'Z' {
					ch += 32
				} else if ch > unicode.MaxASCII {
					ch = unicode.To(unicode.LowerCase, ch)
				}
			}
			if normalize {
				ch = normalizeChar(ch)
			}
			pch := pattern[flipIndex(pidx, patLen, forward)]
			if ch == pch {
				pidx--
				if pidx < 0 {
					sidx = index
					break
				}
			}
		}
		if !forward {
			sidx, eidx = textLen-eidx, textLen-sidx
		}
		score, pos := computeMatchScore(caseSensitive, normalize, text, pattern, sidx, eidx, withPos)
		return MatchResult{sidx, eidx, score}, pos
	}
	return MatchResult{-1, -1, 0}, nil
}

func FuzzyMatchV2(caseSensitive bool, normalize bool, forward bool, input *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	M := len(pattern)
	if M == 0 {
		return MatchResult{0, 0, 0}, makePositions(withPos, M)
	}
	N := input.Length()
	if M > N {
		return MatchResult{-1, -1, 0}, nil
	}

	if slab != nil && N*M > cap(slab.I16) || M > 1000 {
		return FuzzyMatchV1(caseSensitive, normalize, forward, input, pattern, withPos, slab)
	}

	minIdx, maxIdx := findAsciiRange(input, pattern, caseSensitive)
	if minIdx < 0 {
		return MatchResult{-1, -1, 0}, nil
	}
	N = maxIdx - minIdx

	off16 := 0
	off32 := 0
	off16, H0 := grabSlab16(off16, slab, N)
	off16, C0 := grabSlab16(off16, slab, N)
	off16, B := grabSlab16(off16, slab, N)
	off32, F := grabSlab32(off32, slab, M)
	_, T := grabSlab32(off32, slab, N)
	input.CopyRunes(T, minIdx)

	maxScore, maxScorePos := int16(0), 0
	pidx, lastIdx := 0, 0
	pchar0, pchar := pattern[0], pattern[0]
	prevH0, prev, inGap := int16(0), startCharClass, false

	for off, ch := range T {
		var cat charCategory
		if ch <= unicode.MaxASCII {
			cat = asciiClasses[ch]
			if !caseSensitive && cat == classUpper {
				ch += 32
				T[off] = ch
			}
		} else {
			cat = classifyNonAscii(ch)
			if !caseSensitive && cat == classUpper {
				ch = unicode.To(unicode.LowerCase, ch)
			}
			if normalize {
				ch = normalizeChar(ch)
			}
			T[off] = ch
		}
		bonus := bonusLookup[prev][cat]
		B[off] = bonus
		prev = cat

		if ch == pchar {
			if pidx < M {
				F[pidx] = int32(off)
				pidx++
				pchar = pattern[min(pidx, M-1)]
			}
			lastIdx = off
		}

		if ch == pchar0 {
			sc := ScoreMatch + bonus*BonusFirstCharMultiplier
			H0[off] = sc
			C0[off] = 1
			if M == 1 && (forward && sc > maxScore || !forward && sc >= maxScore) {
				maxScore, maxScorePos = sc, off
				if forward && bonus >= BonusBoundary {
					break
				}
			}
			inGap = false
		} else {
			if inGap {
				H0[off] = max(prevH0+ScoreGapExtension, 0)
			} else {
				H0[off] = max(prevH0+ScoreGapStart, 0)
			}
			C0[off] = 0
			inGap = true
		}
		prevH0 = H0[off]
	}
	if pidx != M {
		return MatchResult{-1, -1, 0}, nil
	}
	if M == 1 {
		result := MatchResult{minIdx + maxScorePos, minIdx + maxScorePos + 1, int(maxScore)}
		if !withPos {
			return result, nil
		}
		pos := []int{minIdx + maxScorePos}
		return result, &pos
	}

	f0 := int(F[0])
	width := lastIdx - f0 + 1
	off16, H := grabSlab16(off16, slab, width*M)
	copy(H, H0[f0:lastIdx+1])
	_, C := grabSlab16(off16, slab, width*M)
	copy(C, C0[f0:lastIdx+1])

	Fsub := F[1:]
	Psub := pattern[1:][:len(Fsub)]
	for off, f := range Fsub {
		f := int(f)
		pch := Psub[off]
		pidx := off + 1
		row := pidx * width
		inGap := false
		Tsub := T[f : lastIdx+1]
		Bsub := B[f:][:len(Tsub)]
		Csub := C[row+f-f0:][:len(Tsub)]
		Cdiag := C[row+f-f0-1-width:][:len(Tsub)]
		Hsub := H[row+f-f0:][:len(Tsub)]
		Hdiag := H[row+f-f0-1-width:][:len(Tsub)]
		Hleft := H[row+f-f0-1:][:len(Tsub)]
		Hleft[0] = 0
		for off, ch := range Tsub {
			col := off + f
			var s1, s2, consecutive int16
			if inGap {
				s2 = Hleft[off] + ScoreGapExtension
			} else {
				s2 = Hleft[off] + ScoreGapStart
			}
			if pch == ch {
				s1 = Hdiag[off] + ScoreMatch
				b := Bsub[off]
				consecutive = Cdiag[off] + 1
				if consecutive > 1 {
					fb := B[col-int(consecutive)+1]
					if b >= BonusBoundary && b > fb {
						consecutive = 1
					} else {
						b = max(b, BonusConsecutive, fb)
					}
				}
				if s1+b < s2 {
					s1 += Bsub[off]
					consecutive = 0
				} else {
					s1 += b
				}
			}
			Csub[off] = consecutive
			inGap = s1 < s2
			sc := max(s1, s2, 0)
			if pidx == M-1 && (forward && sc > maxScore || !forward && sc >= maxScore) {
				maxScore, maxScorePos = sc, col
			}
			Hsub[off] = sc
		}
	}

	pos := makePositions(withPos, M)
	j := f0
	if withPos {
		i := M - 1
		j = maxScorePos
		preferMatch := true
		for {
			I := i * width
			j0 := j - f0
			s := H[I+j0]
			var s1, s2 int16
			if i > 0 && j >= int(F[i]) {
				s1 = H[I-width+j0-1]
			}
			if j > int(F[i]) {
				s2 = H[I+j0-1]
			}
			if s > s1 && (s > s2 || s == s2 && preferMatch) {
				*pos = append(*pos, j+minIdx)
				if i == 0 {
					break
				}
				i--
			}
			preferMatch = C[I+j0] > 1 || I+width+j0+1 < len(C) && C[I+width+j0+1] > 0
			j--
		}
	}
	return MatchResult{minIdx + j, minIdx + maxScorePos + 1, int(maxScore)}, pos
}

func ExactMatchNaive(caseSensitive bool, normalize bool, forward bool, text *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	return doExactMatch(caseSensitive, normalize, forward, false, text, pattern, withPos, slab)
}

func ExactMatchBoundary(caseSensitive bool, normalize bool, forward bool, text *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	return doExactMatch(caseSensitive, normalize, forward, true, text, pattern, withPos, slab)
}

func doExactMatch(caseSensitive bool, normalize bool, forward bool, boundaryCheck bool, text *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	if len(pattern) == 0 {
		return MatchResult{0, 0, 0}, nil
	}
	textLen := text.Length()
	patLen := len(pattern)
	if textLen < patLen {
		return MatchResult{-1, -1, 0}, nil
	}
	idx, _ := findAsciiRange(text, pattern, caseSensitive)
	if idx < 0 {
		return MatchResult{-1, -1, 0}, nil
	}

	pidx := 0
	bestPos, bonus, bbonus, bestBonus := -1, int16(0), int16(0), int16(-1)

	for index := 0; index < textLen; index++ {
		realIdx := flipIndex(index, textLen, forward)
		ch := text.Get(realIdx)
		if !caseSensitive {
			if ch >= 'A' && ch <= 'Z' {
				ch += 32
			} else if ch > unicode.MaxASCII {
				ch = unicode.To(unicode.LowerCase, ch)
			}
		}
		if normalize {
			ch = normalizeChar(ch)
		}
		patIdx := flipIndex(pidx, patLen, forward)
		pch := pattern[patIdx]
		ok := pch == ch
		if ok {
			if patIdx == 0 {
				bonus = bonusAt(text, realIdx)
			}
			if boundaryCheck {
				if forward && patIdx == 0 {
					bbonus = bonus
				} else if !forward && patIdx == patLen-1 {
					if realIdx < textLen-1 {
						bbonus = bonusAt(text, realIdx+1)
					} else {
						bbonus = BonusBoundaryWhite
					}
				}
				ok = bbonus >= BonusBoundary
				if ok && patIdx == 0 {
					ok = realIdx == 0 || classify(text.Get(realIdx-1)) <= classDelimiter
				}
				if ok && patIdx == len(pattern)-1 {
					ok = realIdx == textLen-1 || classify(text.Get(realIdx+1)) <= classDelimiter
				}
			}
		}
		if ok {
			pidx++
			if pidx == patLen {
				if bonus > bestBonus {
					bestPos, bestBonus = index, bonus
				}
				if bonus >= BonusBoundary {
					break
				}
				index -= pidx - 1
				pidx, bonus = 0, 0
			}
		} else {
			index -= pidx
			pidx, bonus = 0, 0
		}
	}
	if bestPos >= 0 {
		var sidx, eidx int
		if forward {
			sidx = bestPos - patLen + 1
			eidx = bestPos + 1
		} else {
			sidx = textLen - (bestPos + 1)
			eidx = textLen - (bestPos - patLen + 1)
		}
		var score int
		if boundaryCheck {
			score = int(bonus)
			deduct := int(bonus-BonusBoundary) + 1
			if sidx > 0 && text.Get(sidx-1) == '_' {
				score -= deduct + 1
				deduct = 1
			}
			if eidx < textLen && text.Get(eidx) == '_' {
				score -= deduct
			}
			score += ScoreMatch*patLen + int(BonusBoundaryWhite)*(patLen+1)
		} else {
			score, _ = computeMatchScore(caseSensitive, normalize, text, pattern, sidx, eidx, false)
		}
		return MatchResult{sidx, eidx, score}, nil
	}
	return MatchResult{-1, -1, 0}, nil
}

func PrefixMatch(caseSensitive bool, normalize bool, forward bool, text *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	if len(pattern) == 0 {
		return MatchResult{0, 0, 0}, nil
	}
	trimmed := 0
	if !unicode.IsSpace(pattern[0]) {
		trimmed = text.LeadingWhitespaces()
	}
	if text.Length()-trimmed < len(pattern) {
		return MatchResult{-1, -1, 0}, nil
	}
	for i, r := range pattern {
		ch := text.Get(trimmed + i)
		if !caseSensitive {
			ch = unicode.ToLower(ch)
		}
		if normalize {
			ch = normalizeChar(ch)
		}
		if ch != r {
			return MatchResult{-1, -1, 0}, nil
		}
	}
	lp := len(pattern)
	score, _ := computeMatchScore(caseSensitive, normalize, text, pattern, trimmed, trimmed+lp, false)
	return MatchResult{trimmed, trimmed + lp, score}, nil
}

func SuffixMatch(caseSensitive bool, normalize bool, forward bool, text *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	textLen := text.Length()
	trimmed := textLen
	if len(pattern) == 0 || !unicode.IsSpace(pattern[len(pattern)-1]) {
		trimmed -= text.TrailingWhitespaces()
	}
	if len(pattern) == 0 {
		return MatchResult{trimmed, trimmed, 0}, nil
	}
	diff := trimmed - len(pattern)
	if diff < 0 {
		return MatchResult{-1, -1, 0}, nil
	}
	for i, r := range pattern {
		ch := text.Get(i + diff)
		if !caseSensitive {
			ch = unicode.ToLower(ch)
		}
		if normalize {
			ch = normalizeChar(ch)
		}
		if ch != r {
			return MatchResult{-1, -1, 0}, nil
		}
	}
	lp := len(pattern)
	sidx := trimmed - lp
	score, _ := computeMatchScore(caseSensitive, normalize, text, pattern, sidx, trimmed, false)
	return MatchResult{sidx, trimmed, score}, nil
}

func EqualMatch(caseSensitive bool, normalize bool, forward bool, text *charutil.Chars, pattern []rune, withPos bool, slab *charutil.Slab) (MatchResult, *[]int) {
	lp := len(pattern)
	if lp == 0 {
		return MatchResult{-1, -1, 0}, nil
	}
	leadTrim := 0
	if !unicode.IsSpace(pattern[0]) {
		leadTrim = text.LeadingWhitespaces()
	}
	trailTrim := 0
	if !unicode.IsSpace(pattern[lp-1]) {
		trailTrim = text.TrailingWhitespaces()
	}
	if text.Length()-leadTrim-trailTrim != lp {
		return MatchResult{-1, -1, 0}, nil
	}
	matched := true
	if normalize {
		runes := text.ToRunes()
		for i, pch := range pattern {
			ch := runes[leadTrim+i]
			if !caseSensitive {
				ch = unicode.To(unicode.LowerCase, ch)
			}
			if normalizeChar(pch) != normalizeChar(ch) {
				matched = false
				break
			}
		}
	} else {
		runes := text.ToRunes()
		runesStr := string(runes[leadTrim : len(runes)-trailTrim])
		if !caseSensitive {
			runesStr = strings.ToLower(runesStr)
		}
		matched = runesStr == string(pattern)
	}
	if matched {
		return MatchResult{leadTrim, leadTrim + lp,
			(ScoreMatch+int(BonusBoundaryWhite))*lp + (BonusFirstCharMultiplier-1)*int(BonusBoundaryWhite)}, nil
	}
	return MatchResult{-1, -1, 0}, nil
}
