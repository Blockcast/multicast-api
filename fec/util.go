package fec

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"go/token"
	"math"
	"math/rand"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type BlockReadError struct {
	Err       error
	SourceSBN uint32
	Missing   RangeList
	*BlockingStructure
}

func (b BlockReadError) Is(err error) bool {
	_, ok := err.(BlockReadError)
	return ok
}
func (b BlockReadError) Error() string {
	var N uint32
	var offset uint64
	if b.BlockingStructure != nil {
		N = b.N.Load()
		offset = b.BlockingStructure.SrcOffset(b.SourceSBN)
	}
	return fmt.Sprintf("err=%v, sbn=%d/%d, offset=%d missing %s", b.Err, b.SourceSBN, N, offset, b.Missing)
}

func NewBlockReadError(bs *BlockingStructure, sbn uint32, missing RangeList, err error) BlockReadError {
	return BlockReadError{BlockingStructure: bs, Err: err, SourceSBN: sbn, Missing: missing}
}

func NewBlockRangeReadError(esiRange ESIRange, err error) BlockRangeReadError {
	return BlockRangeReadError{err, esiRange}
}

type BlockRangeReadError struct {
	Err error
	ESIRange
}

func (b BlockRangeReadError) Is(err error) bool {
	_, ok := err.(BlockRangeReadError)
	return ok
}
func (b BlockRangeReadError) Error() string {
	return fmt.Sprintf("err=%v, missing %s", b.Err, b.ESIRange.String())
}

// IETF RFC 7233 section 2.1 Byte Ranges
// TODO nil start ->  end is length of suffix
// nil end -> start is length of offset to length
type Range struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

func (r Range) Contains(l Range) bool {
	return r.Start <= l.Start && l.End <= r.End
}
func (r Range) OverlapsFront(l Range) bool {
	return (r.Start <= l.Start && r.End+1 >= l.Start)
}
func (r Range) Union(l Range) Range {
	start := min(r.Start, l.Start)
	end := max(r.End, l.End)
	return Range{
		start, end,
	}
}

//func max(a, b int64) int64 {
//	if a > b {
//		return a
//	}
//	return b
//}
//func min(a, b int64) int64 {
//	if a < b {
//		return a
//	}
//	return b
//}

func (r Range) Intersection(l Range) *Range {
	if r.Start <= l.Start && r.End >= l.Start {
		return &Range{l.Start, min(l.End, r.End)}
	} else if l.Start <= r.Start && l.End >= r.Start {
		return &Range{r.Start, min(l.End, r.End)}
	}
	return nil
}

var rQuery = regexp.MustCompile(`(\d+)(-?)(\d*)`)

func parseLimits(limits []string) (int64, int64, error) {
	rangeStart, err := strconv.ParseInt(limits[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	rangeEnd := rangeStart
	if len(limits) > 1 && limits[2] == "-" {
		rangeEnd = -1
	}
	if len(limits) > 2 && limits[3] != "" {
		rangeEnd, err = strconv.ParseInt(limits[3], 10, 64)
	}
	return rangeStart, rangeEnd, nil
}

func (r *Range) UnmarshalText(text []byte) (err error) {
	if limits := rQuery.FindStringSubmatch(string(text)); limits != nil {
		r.Start, r.End, err = parseLimits(limits)
	}
	return
}

func (r Range) MarshalText() ([]byte, error) {
	s := fmt.Sprintf("%d", r.Start)
	if r.Start == r.End {
		return []byte(s), nil
	}
	if r.End == -1 {
		return []byte(s + "-"), nil
	}
	s = fmt.Sprintf("%d-%d", r.Start, r.End)
	return []byte(s), nil
}

func (r Range) String() string {
	s := fmt.Sprintf("%d", r.Start)
	if r.Start == r.End {
		return s
	}
	if r.End == -1 {
		return s + "-"
	}
	return fmt.Sprintf("%d-%d", r.Start, r.End)
}

func (r Range) StringHTTP() string {
	s := fmt.Sprintf("%d", r.Start)
	if r.End == -1 {
		return s + "-"
	}
	return fmt.Sprintf("%d-%d", r.Start, r.End)
}

func (r Range) Count() int64 {
	return r.End - r.Start + 1
}

var contentRangeRegex = regexp.MustCompile(`(\d+|\*)(-\d*)?(\/\d*)?`)

func ParseContentRange(contentRange string) (RangeList, int64, error) {
	allLimits := strings.Split(contentRange, ",")
	var length int64 = -1
	var err error
	ret := make(RangeList, 0, len(allLimits))
	for _, limitS := range allLimits {
		limits := contentRangeRegex.FindStringSubmatch(limitS)
		if limits == nil {
			return nil, 0, fmt.Errorf("invalid content range %s", contentRange)
		}
		var start, end int64 = 0, -1
		if limits[1] != "*" {
			start, err = strconv.ParseInt(limits[1], 10, 64)
			if err != nil {
				return nil, 0, err
			}
		}
		if len(limits[2]) > 1 {
			end, err = strconv.ParseInt(limits[2][1:], 10, 64)
			if err != nil {
				return nil, 0, err
			}
		}
		if length == -1 && len(limits[3]) > 1 && limits[3][1:] != "*" {
			length, err = strconv.ParseInt(limits[3][1:], 10, 64)
			if err != nil {
				return nil, 0, err
			}
		}
		if (start > 0 || end > 0) &&
			(len(ret) == 0 ||
				(ret[len(ret)-1].End < start) && (ret[len(ret)-1].End != -1)) {
			ret = append(ret, Range{start, end})
		}
	}
	return ret, length, nil
}

type RangeList []Range

func (rl RangeList) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: rl.String()}, nil
}

func (rl RangeList) Count() int64 {
	ret := int64(0)
	for _, r := range rl {
		ret += r.Count()
	}
	return ret
}

func (rl RangeList) String() string {
	if len(rl) == 0 {
		return ""
	}
	fms := make([]byte, 0, len(rl)*3)
	rli := make([]interface{}, len(rl))
	for i := range rl {
		rli[i] = rl[i]
		fms = append(fms, "%s,"...)
		//if r.Start <= last.End+1 {
		//	last.End = r.End
		//	if i < len(rl)-1 {
		//		continue
		//	} else {
		//		s = append(s, fmt.Sprintf("%s", last.String())...)
		//		break
		//	}
		//}
		//s = append(s, fmt.Sprintf("%s,", last.String())...)
		//last = r
		//if i == len(rl)-1 {
		//	s = append(s, fmt.Sprintf("%s", r.String())...)
		//}
	}
	fms = fms[:len(fms)-1]
	ret := fmt.Sprintf(string(fms), rli...)
	fms, rli = nil, nil
	return ret
}

func (rl RangeList) StringHTTP() string {
	if len(rl) == 0 {
		return ""
	}
	out := ""
	for i := range rl {
		out += rl[i].StringHTTP() + ","
		//if r.Start <= last.End+1 {
		//	last.End = r.End
		//	if i < len(rl)-1 {
		//		continue
		//	} else {
		//		s = append(s, fmt.Sprintf("%s", last.String())...)
		//		break
		//	}
		//}
		//s = append(s, fmt.Sprintf("%s,", last.String())...)
		//last = r
		//if i == len(rl)-1 {
		//	s = append(s, fmt.Sprintf("%s", r.String())...)
		//}
	}
	return out[:len(out)-1]
}

func (rl RangeList) Subtract(ll RangeList) RangeList {
	if rl == nil {
		return nil
	}
	if len(ll) == 0 {
		return rl
	}
	minLen := int64(len(rl)) + int64(len(ll))
	out := make([]Range, 0, minLen)
	var i, j int
	for {
		if i == len(rl) {
			break
		} else if j == len(ll) {
			out = append(out, rl[i])
			i++
			continue
		}
		r1 := Range{(rl)[i].Start, ll[j].Start - 1}
		if (rl)[i].Contains(ll[j]) {
			if ll[j].Start-1 >= (rl)[i].Start {
				out = append(out, Range{rl[i].Start, ll[j].Start - 1})
			}
			if rl[i].End >= ll[j].End+1 {
				out = append(out, RangeList{
					{ll[j].End + 1, rl[i].End}}.Subtract(ll[j+1:])...)
			}
			i++
			j++
			continue
		} else if (ll)[j].Contains(rl[i]) {
			i++
		} else if (rl)[i].OverlapsFront(ll[j]) {
			r1 = Range{rl[i].Start, ll[j].Start - 1}
			i++
		} else if ll[j].OverlapsFront(rl[i]) {
			r1 = Range{ll[j].End + 1, rl[i].End}
			i++
		} else if (rl)[i].End < ll[j].Start {
			r1 = (rl)[i]
			i++
		} else if (rl)[i].Start > ll[j].End {
			j++
			continue
		}
		if r1.End >= r1.Start {
			out = append(out, r1)
		}

	}
	return flatten(out)
}

func (rl RangeList) Intersection(ll RangeList) RangeList {
	if rl == nil {
		return nil
	}
	if len(ll) == 0 || len(rl) == 0 {
		return nil
	}
	var j = 0
	i := sort.Search(len(rl), func(i int) bool { return (rl)[i].End+1 >= ll[j].Start })
	if i == len(rl) {
		i--
	}
	minLen := min(int64(len(rl)-i), int64(len(ll)))
	out := make([]Range, 0, minLen)
	for {
		if i == len(rl) || j == len(ll) {
			break
		}
		if (rl)[i].Contains(ll[j]) {
			out = append(out, ll[j])
			j++
		} else if ll[j].Contains((rl)[i]) {
			out = append(out, (rl)[i])
			i++
		} else if (rl)[i].OverlapsFront(ll[j]) {
			intersection := (rl)[i].Intersection(ll[j])
			if intersection != nil {
				out = append(out, *intersection)
			}
			i++
		} else if ll[j].OverlapsFront((rl)[i]) {
			intersection := (rl)[i].Intersection(ll[j])
			if intersection != nil {
				out = append(out, *intersection)
			}
			j++
		} else if (rl)[i].End < ll[j].Start {
			i++
		} else {
			j++
		}
		out = flatten(out)
	}
	return out

}
func (rl *RangeList) InplaceUnion(ll RangeList) int {
	if rl == nil || len(ll) == 0 {
		return 0
	}
	if len(*rl) == 0 {
		(*rl) = append(*rl, ll...)
		return len(ll)
	}
	var j, n = 0, 0
	i := sort.Search(len(*rl), func(i int) bool { return (*rl)[i].End+1 >= ll[j].Start })
	if i > 0 {
		i--
	}
	for {
		if j < len(ll) {
			if (*rl)[i].Contains(ll[j]) {
				j++
			} else if (*rl)[i].OverlapsFront(ll[j]) {
				// rl is before ll and overlaps
				(*rl)[i].End = ll[j].End
				j++
			} else if ll[j].Contains((*rl)[i]) {
				(*rl)[i].Start = ll[j].Start
				(*rl)[i].End = ll[j].End
				i++
			} else if ll[j].OverlapsFront((*rl)[i]) {
				// ll is before rl and overlaps
				(*rl)[i].Start = ll[j].Start
				j++
			} else if (*rl)[i].End <= ll[j].Start {
				// rl is before ll starts, move on
				i++
			} else if ll[j].End <= (*rl)[i].Start {
				// ll is before rl starts, add to output
				(*rl) = append((*rl)[:i+1], (*rl)[i:]...)
				(*rl)[i] = ll[j]
				i++
				j++
				n++
			}
		} else {
			i++
		}
		if len(*rl) > 1 && i > 0 && i < len(*rl) && (*rl)[i].Start <= (*rl)[i-1].End+1 {
			if (*rl)[i].End > (*rl)[i-1].End {
				(*rl)[i-1].End = (*rl)[i].End
			}
			(*rl) = append((*rl)[:i], (*rl)[i+1:]...)
			i--
		}
		if i == len(*rl) {
			// finished with rl but have larger lls to add
			if j < len(ll) && (*rl)[i-1].End <= ll[j].End {
				if ll[j].Start <= (*rl)[i-1].End+1 {
					(*rl)[i-1].End = ll[j].End
					j++
				}
				(*rl) = append(*rl, ll[j:]...)
				n += len(ll[j:])
			}
			break
		}
	}
	*rl = flatten(*rl)
	return n
}

func (rl RangeList) Union(ll RangeList) RangeList {
	if len(ll) == 0 {
		out := make(RangeList, 0, len(rl))
		for _, r := range rl {
			out = append(out, r)
		}
		return out
	}
	if len(rl) == 0 {
		out := make(RangeList, 0, len(ll))
		for _, r := range ll {
			out = append(out, r)
		}
		return out
	}
	maxLen := max(int64(len(rl)), int64(len(ll)))
	out := make(RangeList, 0, maxLen)
	var i, j = 0, 0
	toAdd := Range{0, -1}
	for {
		if i == len(rl) {
			if j < len(ll) && (rl[i-1].End <= ll[j].End || rl[i-1].Count() == 0) {
				out = append(out, ll[j:]...)
			}
			break
		}
		if j == len(ll) {
			if i < len(rl) && (ll[j-1].End <= rl[i].End || ll[j-1].Count() == 0) {
				out = append(out, rl[i:]...)
			}
			break
		}
		if rl[i].Start > rl[i].End {
			i++
		} else if ll[j].Start > ll[j].End {
			j++
		} else if rl[i].Contains(ll[j]) {
			j++
		} else if ll[j].Contains(rl[i]) {
			i++
		} else if rl[i].OverlapsFront(ll[j]) || ll[j].OverlapsFront(rl[i]) {
			toAdd = rl[i].Union(ll[j])
			i++
			j++
		} else if rl[i].End <= ll[j].Start {
			toAdd = rl[i]
			i++
		} else {
			toAdd = ll[j]
			j++
		}
		if len(out) > 0 {
			last := out[len(out)-1]
			if last.OverlapsFront(toAdd) {
				out[len(out)-1] = last.Union(toAdd)
				continue
			}
		}
		if toAdd.End >= 0 {
			out = append(out, toAdd)
		}
	}
	// one reverse pass to flatten
	//for i := len(out) - 1; i > 0; i-- {
	//	if out[i].Start <= out[i-1].End {
	//		out[i-1].End = out[i].Start
	//		out = out[:i]
	//	}
	//}
	return flatten(out)
}

func (rl RangeList) Contains(ol RangeList) bool {
	j := 0
	if len(ol) == 0 {
		return true
	}
	i := sort.Search(len(rl), func(i int) bool { return (rl)[i].End+1 >= ol[0].Start })
	for ; i < len(rl) && j < len(ol); i++ {
		if rl[i].Contains(ol[j]) {
			j++
			i-- // retry current
		}
	}
	if j == len(ol) {
		return true
	}
	return false
}

func MakeRandRangeList(start, end, count int) RangeList {
	rl := RangeList{}
	for i := 0; i < count; i++ {
		if end-start <= 0 {
			break
		}
		inc := rand.Intn(end - start)
		if inc == 0 {
			inc++
		}
		rl = append(rl, Range{Start: int64(start), End: int64(start + inc)})
		start += inc + rand.Intn(end-start)
	}
	return flatten(rl)
}
func flatten(rl RangeList) RangeList {
	sort.Slice(rl, func(i, j int) bool { return rl[i].Start < rl[j].Start })
	for i := len(rl) - 1; i > 0; i-- {
		if rl[i].Start <= rl[i-1].End+1 {
			rl[i-1].End = max(rl[i].End, rl[i-1].End)
			for j := i; j < len(rl)-1; j++ {
				rl[j] = rl[j+1]
			}
			rl = rl[:len(rl)-1]
		}
	}
	return rl
}

type ESIRange map[uint32]RangeList

func NewESIRange(sbnRange map[uint32]RangeList) *ESIRange {
	return (*ESIRange)(&sbnRange)
}
func ESRangeListForSBNFromRangeList(bs *BlockingStructure, curSBN uint32, rl RangeList) (RangeList, error) {
	if bs == nil {
		return nil, fmt.Errorf("nil blocking structure")
	}
	var curNumSym = bs.NumSrcSym(curSBN)
	var sbnOffset = int64(bs.SrcOffset(curSBN))
	var bSize = int64(bs.SrcBlockSize(curSBN))
	var ret RangeList
	var sbnOffsetHas = rl.Intersection(RangeList{{sbnOffset, sbnOffset + bSize - 1}})
	for _, r := range sbnOffsetHas {
		if sbnOffset >= r.Start || r.Start > r.End {
			return ret, fmt.Errorf("range list not sorted: %s", sbnOffsetHas)
		}
		symStartOffset := r.Start - sbnOffset
		esiStart := uint32(symStartOffset) / uint32(bs.ESLen)
		if symStartOffset%int64(bs.ESLen) != 0 {
			esiStart++
		}
		symEndOffset := r.End - sbnOffset
		esiEnd := uint32(symEndOffset+1) / uint32(bs.ESLen)
		if (symEndOffset+1)%int64(bs.ESLen) != 0 && r.End != int64(bs.TransferLen.Load())-1 {
			esiEnd--
		}
		if numSym := int(esiStart) - int(esiEnd); esiStart+uint32(numSym) > curNumSym {
			esiEnd = curNumSym - 1
		}
		if esiEnd < esiStart {
			continue
		}
		ret.InplaceUnion(RangeList{{int64(esiStart), int64(esiEnd)}})
	}
	return ret, nil
}

func GetMissingESIs(SrcSymMissing, RprSymHas ESIRange, bs *BlockingStructure) ESIRange {
	result := make(ESIRange)

	for sbn, missingRanges := range SrcSymMissing {
		// Count the missing ESIs in the block
		missingCount := missingRanges.Count()
		// Count the ESIs that can be repaired in the same block
		repairableRanges := RprSymHas[sbn]
		repairableCount := repairableRanges.Count()

		// If we have to repair the whole block
		esiPerBlock := int64(bs.NumSrcSym(sbn))
		if missingCount == esiPerBlock && repairableCount == 0 {
			result[sbn] = RangeList{Range{0, esiPerBlock - 1}}
		} else if missingCount > repairableCount {
			// If there are more missing ESIs than repairable ones, calculate how many remain to be requested
			remainingMissing := missingCount - repairableCount
			// Select the first 'remainingMissing' ESIs from the missing ESIs
			remainingRanges := RangeList{}
			for _, r := range missingRanges {
				for i := r.Start; i <= r.End && remainingMissing > 0; i++ {
					remainingRanges = append(remainingRanges, Range{Start: i, End: i})
					remainingMissing--
					if remainingMissing == 0 {
						break
					}
				}
				if remainingMissing == 0 {
					break
				}
			}
			result[sbn] = flatten(remainingRanges)
		}
	}
	return result
}

func ESIRangeFromRangeList(bs *BlockingStructure, rl RangeList, source, inclusive bool) (ESIRange, error) {
	if bs == nil || bs.ESLen == 0 {
		return nil, fmt.Errorf("invalid bs: %v, bs", bs)
	}
	er := make(ESIRange)
	var sbnStart, sbnEnd uint32
	for _, r := range rl {
		if r.Start > r.End {
			return er, fmt.Errorf("range list not sorted: %s", rl)
		}
		start := r.Start
		end := r.End
		var sbnRangeStart, sbnRangeEnd uint64
		if source {
			sbnStart = bs.SourceSBN(uint64(start))
			sbnEnd = bs.SourceSBN(uint64(end))
		} else {
			sbnStart = bs.RepairSBN(uint64(start))
			sbnEnd = bs.RepairSBN(uint64(end))
		}
		var esiStart, esiEnd int64
		for sbn := sbnStart; sbn <= sbnEnd; sbn++ {
			if source {
				sbnRangeStart = bs.SrcOffset(sbn)
				sbnRangeEnd = bs.SrcOffset(sbn+1) - 1
			} else {
				sbnRangeStart = bs.RprOffset(sbn)
				sbnRangeEnd = bs.RprOffset(sbn+1) - 1
			}
			esiStartOffset := max(start, int64(sbnRangeStart)) - int64(sbnRangeStart)
			esiStart = esiStartOffset / int64(bs.ESLen)
			esiEndOffset := min(end, int64(sbnRangeEnd)) - int64(sbnRangeStart)
			rangeLen := esiEndOffset - esiStartOffset + 1
			esiCount := rangeLen / int64(bs.ESLen)
			esiEnd = esiStart + esiCount - 1
			if rangeLen%int64(bs.ESLen) != 0 && inclusive {
				esiEnd++
			}
			if !source {
				numSrcSym := int64(bs.NumSrcSym(sbn))
				esiStart += numSrcSym
				esiEnd += numSrcSym
			}

			er[sbn] = append(er[sbn], Range{esiStart, esiEnd})
		}
	}
	// Merge all continuous ranges per RangeList
	for key, ranges := range er {
		er[key] = flatten(ranges)
	}
	return er, nil
}

var plusRange = regexp.MustCompile(`([\d]+)[+]([\d]+)`)
var esiRange = regexp.MustCompile(`(%3[Bb]|;)(ESI|esi)=`)

func NewESIRangeFromMBMSQuery(query string) (ESIRange, error) {
	s := esiRange.ReplaceAllLiteralString(query, ":")
	idx := plusRange.FindAllStringSubmatchIndex(s, -1)
	offset := 0
	for _, i := range idx {
		start, _ := strconv.ParseUint(s[i[2]+offset:i[3]+offset], 10, 32)
		number, _ := strconv.ParseUint(s[i[4]+offset:i[5]+offset], 10, 32)
		end := fmt.Sprintf("-%d", start+number)
		s = s[:i[3]+offset] + end + s[i[5]+offset:]
		offset += len(end) - (i[5] - i[3])
	}
	params, err := url.ParseQuery(s)
	if err != nil {
		return nil, err
	}
	ranges := &ESIRange{}
	err = ranges.UnmarshalText([]byte(strings.Join(params["SourceSBN"], ";")))
	return *ranges, err
}

// esirange = rangelist | blockrange
var regex = regexp.MustCompile("")
var parseError = fmt.Errorf("esirange parse error")

func scanComma(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, ','); i >= 0 {
		// We have ASmall full newline-terminated line.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have ASmall final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func (rl *RangeList) UnmarshalText(text []byte) error {
	s := bufio.NewScanner(bytes.NewReader(text))
	s.Split(scanComma)
	for s.Scan() {
		r := Range{}
		if err := r.UnmarshalText(s.Bytes()); err != nil {
			return err
		}
		*rl = append(*rl, r)
	}
	if err := s.Err(); err != nil {
		return s.Err()
	}
	*rl = flatten(*rl)
	return nil
}

func (er *ESIRange) UnmarshalText(text []byte) error {
	for _, s := range strings.Split(string(text), token.SEMICOLON.String()) {
		partsSub := strings.SplitN(s, token.SUB.String(), 2)
		sbn, err := strconv.ParseUint(partsSub[0], 10, 32)
		if err == nil && len(partsSub) == 2 {
			if sbnEnd, errSbnRange := strconv.ParseUint(partsSub[1], 10, 32); errSbnRange == nil {
				for i := uint32(sbn); i <= uint32(sbnEnd); i++ {
					(*er)[i] = RangeList{}
				}
			}
		} else {
			parts := strings.SplitN(s, token.COLON.String(), 2)
			sbn, err = strconv.ParseUint(parts[0], 10, 32)
			if err != nil {
				return fmt.Errorf("%w: sbn `%s` is not an unsigned int", parseError, partsSub[0])
			}
			if len(parts) == 1 || len(parts[1]) == 0 {
				(*er)[uint32(sbn)] = RangeList{}
				continue
			}
			rl := &RangeList{}
			if err := rl.UnmarshalText([]byte(parts[1])); err != nil {
				return err
			}
			(*er)[uint32(sbn)] = *rl
		}
	}
	return nil
}

func (er ESIRange) MarshalText() (s []byte, err error) {
	if len(er) == 0 {
		return []byte(""), nil
	}

	keys := make([]uint32, 0, len(er))
	for k := range er {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	var sbnRangeStart = keys[0]
	for i, sbn := range keys {
		rl := er[sbn]
		if sbnRangeStart < sbn && i > 0 && len(er[keys[i-1]]) == 0 &&
			((i < len(keys)-1 && len(er[keys[i+1]]) > 0) ||
				(i == len(keys)-1 && len(er[sbn]) == 0)) {
			s = append(s, fmt.Sprintf(";%d-%d", sbnRangeStart, sbn)...)
			sbnRangeStart = math.MaxUint32
		} else if len(rl) > 0 {
			s = append(s, fmt.Sprintf(";%d", sbn)...)
			if len(rl) > 1 || (len(rl) == 1 && (rl[0].Start != 0 || rl[0].End != -1)) {
				sbnRangeStart = sbn + 1
				s = append(append(s, ':'), rl.String()...)
			}
		} else if i == len(keys)-1 {
			s = append(s, fmt.Sprintf(";%d", sbn)...)
		}
	}
	if s != nil && s[0] == ';' {
		s = s[1:]
	}
	return
}

func (er ESIRange) ToRangeList(bs *BlockingStructure, source bool) RangeList {
	rl := make(RangeList, 0, len(er))
	var offset, blockSize int64
	for sbn, r := range er {
		if !source {
			N := int64(bs.MaxNumEs - bs.MaxSbLen)
			offset = int64(bs.RprOffset(sbn))
			blockSize = N * int64(bs.ESLen)
		} else {
			offset = int64(bs.SrcOffset(sbn))
			blockSize = int64(bs.SrcBlockSize(sbn))
		}
		k := int64(bs.NumSrcSym(sbn))
		if len(r) > 0 {
			for i := range r {
				esiStart := r[i].Start
				esiEnd := r[i].End + 1
				if !source {
					esiStart -= k
					esiEnd -= k
					if esiStart < 0 {
						esiStart = 0
					}
					if esiEnd < 0 {
						continue
					}
				}
				start := offset + esiStart*int64(bs.ESLen)
				end := offset + (esiEnd)*int64(bs.ESLen) - 1
				rl = append(rl, Range{start, end})
			}
		} else {
			rl = append(rl, Range{offset, offset + blockSize - 1})
		}
	}
	return flatten(rl)

}
func (er ESIRange) ToMBMSRawQuery() string {
	s := strings.ReplaceAll(er.String(), ";", "&SourceSBN=")
	s = strings.ReplaceAll(s, ":", "%3bESI=")
	if len(s) > 0 {
		s = "SourceSBN=" + s
	}
	return s
}

func (er ESIRange) String() string {
	s, _ := er.MarshalText()
	return string(s)
}

func (er *ESIRange) Count(bs *BlockingStructure) int {
	count := 0
	if er != nil {
		for sbn, rl := range *er {
			if len(rl) == 0 && bs != nil {
				rl = RangeList{{0, int64(bs.NumSrcSym(sbn) - 1)}}
			}
			count += int(rl.Count())
		}
	}
	return count
}

type URL url.URL

func (u *URL) UnmarshalText(text []byte) error {
	parsed, err := url.Parse(string(text))
	*u = URL(*parsed)
	return err
}
func (u URL) MarshalText() ([]byte, error) {
	u2 := url.URL(u)
	return (&u2).MarshalBinary()
}
func (er URL) String() string {
	s, _ := er.MarshalText()
	return string(s)
}

type MD5 []byte

func (m MD5) String() string {
	res, _ := m.MarshalText()
	return string(res)
}

func (m *MD5) UnmarshalText(text []byte) (err error) {
	*m, err = base64.StdEncoding.DecodeString(string(text))
	return err
}

func (m MD5) MarshalText() ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(m)), nil
}

func Missing(has RangeList, start int64, end int64) RangeList {
	ret := RangeList{}
	cur := start
	if cur > end || end < 0 {
		return ret
	}
	for i := 0; i < len(has); i++ {
		if has[i].Start > end {
			break
		} else if cur > has[i].End {
			continue
		} else if cur < has[i].Start {
			ret = append(ret, Range{cur, has[i].Start - 1})
			cur = has[i].End + 1
		} else if cur <= has[i].End {
			cur = has[i].End + 1
		}
		if cur > end {
			break
		}
	}
	if cur <= end {
		ret = append(ret, Range{cur, end})
	}
	return ret
}
