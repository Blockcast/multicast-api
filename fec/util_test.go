package fec_test

import (
	"fmt"
	"github.com/Blockcast/multicast-api/fec"
	"github.com/stretchr/testify/assert"
	"math"
	"math/rand"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func TestContentRange(t *testing.T) {
	testsNominal := []struct {
		string
		fec.RangeList
		int64
	}{
		{"bytes=*/123", fec.RangeList{}, 123},
		{"bytes=112-115/*", fec.RangeList{{112, 115}}, -1},
		{"bytes=213-", fec.RangeList{{213, -1}}, -1},
		{"bytes */5242997", fec.RangeList{}, 5242997},
		{"bytes=1-2,3-4/5", fec.RangeList{{1, 2}, {3, 4}}, 5},
		{"bytes=3046761-3599361", fec.RangeList{{3046761, 3599361}}, -1},
	}
	for _, test := range testsNominal {
		cr, l, err := fec.ParseContentRange(test.string)
		assert.NoError(t, err, test.string)
		assert.Equal(t, test.RangeList, cr, test.string)
		assert.Equal(t, test.int64, l, test.string)
	}
}
func TestMissing(t *testing.T) {
	range0 := fec.RangeList{{0, 3}}
	missing1 := fec.Missing(range0, 0, 9)
	assert.Equal(t, fec.RangeList{{4, 9}}, missing1)
	missing2 := fec.Missing(range0, 10, 19)
	assert.Equal(t, fec.RangeList{{10, 19}}, missing2)

	range0 = fec.RangeList{{10, 10}}
	missing1 = fec.Missing(range0, 10, 19)
	assert.Equal(t, fec.RangeList{{11, 19}}, missing1)
}

func TestIntesection(t *testing.T) {
	r := "1589838-1592713,1599904-3537799"
	has := "0-111,1039680-1602839,3537800-7985319"
	exp := "1589838-1592713,1599904-1602839"
	rrange, _, _ := fec.ParseContentRange(r)
	hasrange, _, _ := fec.ParseContentRange(has)
	exprange, _, _ := fec.ParseContentRange(exp)
	assert.Equal(t, exprange, rrange.Intersection(hasrange))
}
func TestSubtract(t *testing.T) {
	range0 := fec.RangeList{{0, 99}}
	sub := fec.RangeList{{0, 9}, {20, 29}, {40, 49}, {60, 69}, {80, 89}}
	missing1 := range0.Subtract(sub)
	assert.Equal(t, fec.RangeList{{10, 19}, {30, 39}, {50, 59}, {70, 79}, {90, 99}}, missing1)

	range0 = fec.RangeList{{3062718, 3064155}}
	sub = fec.RangeList{{2902440, 3127022}}
	missing1 = range0.Subtract(sub)
	assert.Equal(t, fec.RangeList{}, missing1)
}
func TestSets(t *testing.T) {

	cases := [][]string{
		{"0-2402", "0-4423,5455-7009,8864-9179,9463-9875,9942-9970"},
		{"22-382,4365-5832,6170-6540,6590-6907,7522-7829", "22-1761,3922-9134,9571-9657,9961-9965,9998-10000"},
	}
	for _, c := range cases {
		var rl1, rl2 fec.RangeList
		assert.NoError(t, rl1.UnmarshalText([]byte(c[0])))
		assert.NoError(t, rl2.UnmarshalText([]byte(c[1])))
		intersection := rl1.Intersection(rl2)
		assert.Equal(t, intersection, rl2.Intersection(rl1), "rl1: %v, rl2: %v", rl1, rl2)
		rl1Sub := rl1.Subtract(intersection)
		rl2Sub := rl2.Subtract(intersection)
		assert.Equal(t, rl1, rl1Sub.Union(intersection), "rl1: %v, rl2: %v, intersection: %v, rl1-rl2: %v", rl1, rl2, intersection, rl1Sub)
		assert.Equal(t, rl2, rl2Sub.Union(intersection), "rl1: %v, rl2: %v, intersection: %v, rl2-rl1: %v", rl1, rl2, intersection, rl2Sub)
	}
}

func FuzzRangeList_SetOps(f *testing.F) {
	f.Add(0, 10000, 1, 1)
	f.Add(0, 10000, 2, 2)
	f.Fuzz(func(t *testing.T, start, end, l1, l2 int) {
		if start < 0 || end < 0 || l1 < 0 || l2 < 0 {
			t.Skip()
		}
		if start > end {
			start, end = end, start
		}
		rl1 := fec.MakeRandRangeList(start, end, l1)
		rl2 := fec.MakeRandRangeList(start, end, l2)
		intersection := rl1.Intersection(rl2)
		assert.Equal(t, intersection, rl2.Intersection(rl1), "rl1: %v, rl2: %v", rl1, rl2)
		rl1Sub := rl1.Subtract(intersection)
		rl2Sub := rl2.Subtract(intersection)
		x := rl1Sub.Union(intersection)
		y := rl2Sub.Union(intersection)
		assert.Equal(t, rl1, x, "rl1: %v, rl2: %v, intersection: %v, rl1-rl2: %v", rl1, rl2, intersection, rl1Sub)
		assert.Equal(t, rl2, y, "rl1: %v, rl2: %v, intersection: %v, rl2-rl1: %v", rl1, rl2, intersection, rl2Sub)

		rl1Sub.InplaceUnion(intersection)
		assert.Equal(t, x, rl1Sub)
		assert.Equal(t, rl1, rl1Sub, "rl1: %v, rl2: %v, intersection: %v", rl1, rl2, intersection)
		rl2Sub.InplaceUnion(intersection)
		assert.Equal(t, y, rl2Sub)
		assert.Equal(t, rl2, rl2Sub, "rl1: %v, rl2: %v, intersection: %v", rl1, rl2, intersection)
	})
}

func TestUnion(t *testing.T) {
	range0 := fec.RangeList{}
	copyRange0 := fec.RangeList{}
	for i := 0; i < 100; i++ {
		range0.InplaceUnion(fec.RangeList{fec.Range{Start: int64(i), End: int64(i)}})
		copyRange0 = copyRange0.Union(fec.RangeList{fec.Range{Start: int64(i), End: int64(i)}})
	}
	assert.Equal(t, fec.RangeList{{0, 99}}, range0)
	assert.Equal(t, fec.RangeList{{0, 99}}, copyRange0)

	range1 := fec.RangeList{}
	copyRange1 := fec.RangeList{}
	max := int64(0)
	min := int64(math.MaxInt)
	for i := 0; i < 100; i++ {
		start := int64(rand.Int63())
		if start < min {
			min = start
		}
		for j := 0; j < 10; j++ {
			step := int64(rand.Int63n(int64(math.MaxInt64) - start))
			if start+step > max {
				max = start + step
			}
			range1.InplaceUnion(fec.RangeList{{start, start + step}})
			copyRange1 = copyRange1.Union(fec.RangeList{{start, start + step}})
		}
	}
	assert.Equal(t, max, copyRange1[len(copyRange1)-1].End)
	assert.Equal(t, min, copyRange1[0].Start)
	assert.Equal(t, max, range1[len(range1)-1].End)
	assert.Equal(t, min, range1[0].Start)

	rl3 := fec.RangeList{{0, 1}, {3, 4}}
	rl4 := rl3.Union(fec.RangeList{{0, 7}})
	rl3.InplaceUnion(fec.RangeList{{0, 7}})
	assert.Equal(t, fec.RangeList{{0, 7}}, rl3)
	assert.Equal(t, fec.RangeList{{0, 7}}, rl4)

	rl5 := fec.RangeList{{0, 0}, {2, 2}, {4, 5}}
	rl6 := rl5.Union(fec.RangeList{{0, 7}})
	rl5.InplaceUnion(fec.RangeList{{0, 7}})
	expect := fec.RangeList{{0, 7}}
	assert.Equal(t, expect, rl5)
	assert.Equal(t, expect, rl6)

	rl5 = fec.RangeList{{12, 19}}
	rl6 = rl5.Union(fec.RangeList{{11, 11}})
	rl5.InplaceUnion(fec.RangeList{{11, 11}})
	expect = fec.RangeList{{11, 19}}
	assert.Equal(t, expect, rl5)
	assert.Equal(t, expect, rl6)
}

func TestESIRangeFromRangeList(t *testing.T) {
	var bsR fec.BlockingStructure
	bsR.N.Store(7509)
	bsR.I.Store(7506)
	bsR.A.Store(10)
	bsR.ESLen = 1430
	bsR.TransferLen.Store(107374182)
	bsR.T = 75087
	bsR.MaxSbLen = 10
	bsR.MaxNumEs = 12
	bsR.NumEsPerGroup = 1
	missing := fec.RangeList{
		{Start: 0, End: 1429},
		{Start: 1430, End: 2859},
		{Start: 2860, End: 4290}, // we are missing byte one of ESI 3 (inclusive)
		{Start: 10010, End: 11940},
		{Start: 14300, End: 15729},
		{Start: 28600, End: 42900},
		{Start: 42901, End: 45000},
		{Start: 45001, End: 59300},
		{Start: 75000, End: 89300},
		{Start: 100000, End: 114300},
		{Start: 114301, End: 128699},
		{Start: 128700, End: 137279},
		{Start: 140140, End: 141569},
	}
	expectedSrcSymMissing := fec.ESIRange{
		0: {{0, 3}, {7, 8}}, // {0, 0}. {1, 1}, {2, 2}, {3, 3}, {7, 8}
		1: {{0, 0}},
		2: {{0, 9}},
		3: {{0, 9}}, // {0, 1}, {1, 9}
		4: {{0, 1}},
		5: {{2, 9}},
		6: {{0, 2}, {9, 9}},
		7: {{0, 9}},
		8: {{0, 9}},
		9: {{0, 5}, {8, 8}},
	}

	// Get ESI range from missing range list
	srcSymMissing, err := fec.ESIRangeFromRangeList(&bsR, missing, true, true)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result srcSymMissing: %+v\n", srcSymMissing)
	}
	// Check if the srcSymMissing is not as expected
	if !reflect.DeepEqual(srcSymMissing, expectedSrcSymMissing) {
		fmt.Printf("Result of srcSymMissing failed:\nGot: %+v\nExpected: %+v\n", srcSymMissing, expectedSrcSymMissing)
	}
	hasR := fec.RangeList{
		{Start: 0, End: 2858}, // ESI 11 is 1 byte short
		{Start: 2860, End: 4289},
		{Start: 5720, End: 11439},
		{Start: 12870, End: 15729},
		{Start: 18590, End: 27169},
		{Start: 30030, End: 31459},
	}
	expectedRprSymHas := fec.ESIRange{
		0:  {{10, 10}},
		1:  {{10, 10}},
		2:  {{10, 11}},
		3:  {{10, 11}},
		4:  {{11, 11}},
		5:  {{10, 10}},
		6:  {{11, 11}},
		7:  {{10, 11}},
		8:  {{10, 11}},
		9:  {{10, 10}},
		10: {{11, 11}},
	}

	// Get ESI range from hasR range list
	rprSymHas, err := fec.ESIRangeFromRangeList(&bsR, hasR, false, false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result rprSymHas: %+v\n", rprSymHas)
	}
	// Check if the rprSymHas is not as expected
	if !reflect.DeepEqual(rprSymHas, expectedRprSymHas) {
		fmt.Printf("Result of rprSymHas failed:\nGot: %+v\nExpected: %+v\n", rprSymHas, expectedRprSymHas)
	}

	// Remove all the small blocks from the rprSymHas
	for i := bsR.I.Load(); i <= bsR.N.Load()-1; i++ {
		delete(rprSymHas, i)
	}

	missingESIs := fec.GetMissingESIs(srcSymMissing, rprSymHas, &bsR)
	fmt.Printf("Result missingESIs: %+v\n", missingESIs)

	expectedMissingESIs := fec.ESIRange{
		0: {{0, 3}, {7, 7}},
		2: {{0, 7}},
		3: {{0, 7}},
		4: {{0, 0}},
		5: {{2, 8}},
		6: {{0, 2}},
		7: {{0, 7}},
		8: {{0, 7}},
		9: {{0, 5}},
	}
	if !reflect.DeepEqual(missingESIs, expectedMissingESIs) {
		fmt.Printf("Result of missing failed:\nGot: %+v\nExpected: %+v\n", missingESIs, expectedMissingESIs)
	}
	assert.Equal(t, missingESIs, expectedMissingESIs)
	resMissing := missingESIs.ToRangeList(&bsR, true)
	expectedResMissing := fec.RangeList{
		{Start: 0, End: 5719},
		{Start: 10010, End: 11439},
		{Start: 28600, End: 40039},
		{Start: 42900, End: 54339},
		{Start: 57200, End: 58629},
		{Start: 74360, End: 84369},
		{Start: 85800, End: 90089},
		{Start: 100100, End: 111539},
		{Start: 114400, End: 125839},
		{Start: 128700, End: 137279},
	}
	assert.Equal(t, resMissing, expectedResMissing)
}

func TestESIRangeFromRangeListBorderCase(t *testing.T) {
	var bsR fec.BlockingStructure
	bsR.N.Store(734)
	bsR.I.Store(727)
	bsR.A.Store(10)
	bsR.ASmall.Store(10)
	bsR.ESLen = 1430
	bsR.TransferLen.Store(10485760)
	bsR.T = 7333
	bsR.MaxSbLen = 10
	bsR.MaxNumEs = 12
	bsR.NumEsPerGroup = 1

	err := bsR.UpdateLength(10485760, true)

	hasString := "0-10479039,10480470-10483329,10484760-10485759"
	has, err := parseRangeList(hasString)
	missing := fec.Missing(has, 10467600, 10485759)
	srcSymMissing, err := fec.ESIRangeFromRangeList(&bsR, missing, true, true)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result srcSymMissing: %+v\n", srcSymMissing)
	}

	hasRString := "0-5719,7150-14299,20020-21449,22880-25739,28600-32889,34320-44329,45760-55769,61490-68639,71500-78649,80080-91519,95810-100099,104390-114399,117260-121549,122980-124409,131560-132989,134420-153009,160160-163019,167310-168739,170170-173029,174460-183039,185900-197339,200200-201629,204490-210209,214500-224509,225940-251679,257400-265979,274560-280279,283140-297439,300300-304589,306020-316029,320320-323179,324610-338909,340340-351779,354640-363219,366080-370369,371800-380379,381810-383239,386100-388959,391820-394679,396110-407549,408980-411839,413270-416129,423280-428999,431860-433289,436150-437579,439010-440439,443300-451879,457600-466179,469040-474759,476190-479049,480480-483339,484770-497639,500500-519089,521950-523379,526240-540539,543400-549119,551980-560559,563420-566279,567710-573429,574860-577719,580580-589159,592020-593449,597740-612039,613470-614899,616330-620619,626340-630629,633490-634919,637780-640639,643500-656369,657800-660659,666380-677819,680680-692119,700700-717859,722150-726439,727870-732159,735020-747889,752180-753609,755040-769339,772200-777919,780780-782209,783640-800799,802230-803659,806520-817959,823680-827969,832260-835119,837980-855139,860860-865149,866580-886599,889460-920919,923780-933789,938080-940939,942370-948089,952380-965249,969540-986699,989560-990989,992420-998139,999570-1009579,1018160-1026739,1032460-1041039,1043900-1063919,1066780-1075359,1081080-1089659,1092520-1098239,1103960-1109679,1115400-1121119,1125410-1129699,1132560-1136849,1138280-1142569,1144000-1146859,1149720-1155439,1158300-1159729,1161160-1165449,1166880-1174029,1178320-1189759,1191190-1192619,1194050-1195479,1198340-1199769,1201200-1204059,1205490-1209779,1212640-1226939,1228370-1231229,1232660-1234089,1235520-1241239,1244100-1246959,1251250-1252679,1254110-1259829,1261260-1272699,1274130-1295579,1297010-1304159,1307020-1308449,1309880-1322749,1327040-1329899,1335620-1341339,1345630-1352779,1354210-1355639,1358500-1364219,1365650-1368509,1369940-1379949,1381380-1389959,1392820-1398539,1399970-1402829,1407120-1411409,1412840-1418559,1421420-1427139,1430000-1441439,1444300-1447159,1450020-1451449,1457170-1465749,1470040-1478619,1481480-1482909,1484340-1492919,1495780-1498639,1501500-1520089,1521520-1527239,1534390-1554409,1555840-1562989,1564420-1568709,1571570-1601599,1607320-1613039,1615900-1620189,1624480-1630199,1633060-1641639,1644500-1647359,1654510-1657369,1661660-1670239,1671670-1698839,1701700-1713139,1716000-1721719,1724580-1726009,1728870-1730299,1736020-1738879,1740310-1756039,1758900-1761759,1766050-1770339,1773200-1774629,1776060-1784639,1787500-1797509,1798940-1801799,1807520-1813239,1814670-1827539,1830400-1833259,1836120-1837549,1841840-1844699,1846130-1847559,1850420-1851849,1856140-1861859,1867580-1884739,1886170-1890459,1891890-1901899,1903330-1914769,1920490-1930499,1933360-1934789,1936220-1941939,1943370-1944799,1947660-1950519,1953380-1956239,1961960-1964819,1966250-1970539,1973400-1977689,1980550-1991989,1993420-2004859,2010580-2019159,2022020-2027739,2029170-2030599,2033460-2036319,2037750-2042039,2044900-2047759,2052050-2060629,2063490-2070639,2073500-2076359,2077790-2082079,2086370-2087799,2089230-2099239"
	hasR, err := parseRangeList(hasRString)

	// Get ESI range from hasR range list
	rprSymHas, err := fec.ESIRangeFromRangeList(&bsR, hasR, false, false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result rprSymHas: %+v\n", rprSymHas)
	}

	// Remove all the small blocks from the rprSymHas
	for i := bsR.I.Load(); i <= bsR.N.Load()-1; i++ {
		delete(rprSymHas, i)
	}

	missingESIs := fec.GetMissingESIs(srcSymMissing, rprSymHas, &bsR)
	fmt.Printf("Result missingESIs: %+v\n", missingESIs)

	resMissing := missingESIs.ToRangeList(&bsR, true)
	fmt.Printf("Result resMissing: %+v\n", resMissing)

	expectedResMissing := fec.RangeList{
		{Start: 10479040, End: 10480469},
		{Start: 10483330, End: 10484759},
	}
	assert.Equal(t, resMissing, expectedResMissing)
}

func parseRangeList(input string) (fec.RangeList, error) {
	var rangeList fec.RangeList
	ranges := strings.Split(input, ",")

	for _, r := range ranges {
		limits := strings.Split(r, "-")
		if len(limits) != 2 {
			return nil, fmt.Errorf("invalid range format: %s", r)
		}

		start, err := strconv.ParseInt(limits[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid start value: %s", limits[0])
		}

		end, err := strconv.ParseInt(limits[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid end value: %s", limits[1])
		}

		rangeList = append(rangeList, fec.Range{Start: start, End: end})
	}
	return rangeList, nil
}

func TestESI(t *testing.T) {
	s := "0;1:;2:3;4:5,6-9,11-11,12-14;5-6;7:0-1;8-10"
	actual := fec.ESIRange{}
	expect := fec.NewESIRange(map[uint32]fec.RangeList{
		0:  {},
		1:  {},
		2:  {{3, 3}},
		4:  {{5, 9}, {11, 14}},
		5:  {},
		6:  {},
		7:  {{0, 1}},
		8:  {},
		9:  {},
		10: {}},
	)
	err := actual.UnmarshalText([]byte(s))
	assert.NoError(t, err)
	assert.Equal(t, *expect, actual)
	assert.Equal(t, "0-1;2:3;4:5-9,11-14;5-6;7:0-1;8-10", actual.String())

	range2 := fec.RangeList{
		{1, 1},
		{3, 3},
		{9, 10}}
	union := actual[4].Union(range2)
	assert.Equal(t, "1,3,5-14", union.String())
	intersection := actual[4].Intersection(range2)
	assert.Equal(t, "9", intersection.String())

	range1 := fec.RangeList{
		{0, 4}}
	range2 = fec.RangeList{
		{0, 0},
		{2, 7}}
	assert.Equal(t, "0,2-4", range1.Intersection(range2).String())

	range1 = fec.RangeList{
		{0, 3}}
	range2 = fec.RangeList{
		{0, 0},
		{2, 7}}
	assert.Equal(t, "0,2-3", range1.Intersection(range2).String())

	range1 = fec.RangeList{
		{0, 2267}, {2268, 9569}}
	range2 = fec.RangeList{
		{0, 4126}, {4127, 4381}}
	assert.Equal(t, "0-4381", range1.Intersection(range2).String())

	s = "12-19;28:23-59;30:101"
	actual = fec.ESIRange{}
	expect = fec.NewESIRange(map[uint32]fec.RangeList{
		12: {},
		13: {},
		14: {},
		15: {},
		16: {},
		17: {},
		18: {},
		19: {},
		28: {{23, 59}},
		30: {{101, 101}},
	})
	err = actual.UnmarshalText([]byte(s))
	assert.NoError(t, err)
	assert.Equal(t, *expect, actual)
	assert.Equal(t, s, actual.String())

}

func TestMBMS(t *testing.T) {
	type testCase struct {
		input  string
		output string
	}
	testCases := []testCase{
		{"&SourceSBN=12;ESI=23",
			"12:23"},
		{"&SourceSBN=12;ESI=23-28",
			"12:23-28"},
		{"&SourceSBN=12;ESI=23,26,28",
			"12:23,26,28"},
		{"&SourceSBN=12",
			"12"},
		{"&SourceSBN=12-19",
			"12-19"},
		{"&SourceSBN=12;ESI=34&SourceSBN=20;ESI=23",
			"12:34;20:23"},
		{"&SourceSBN=12-19&SourceSBN=28%3BESI=23-59&SourceSBN=30;ESI=101",
			"12-19;28:23-59;30:101"},
		{"&SourceSBN=12%3bESI=120+10",
			"12:120-130"},
	}
	for i, tc := range testCases {
		_ = i
		result, err := fec.NewESIRangeFromMBMSQuery(tc.input)
		assert.NoError(t, err)
		assert.Equal(t, tc.output, result.String(), "input: %s", tc.input)
		// last cases is transformed to "12:120-130"
		if !strings.Contains(tc.input, "+") {
			query := result.ToMBMSRawQuery()
			query, err := url.QueryUnescape(query)
			assert.NoError(t, err)
			input, err := url.QueryUnescape(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, input, "&"+query)
		}
	}

}
