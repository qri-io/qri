package detect

var (
	egCorruptCsvData = []byte(`
		"""fhkajslfnakjlcdnajcl ashklj asdhcjklads ch,,,\dagfd
	`)
	egNaicsCsvData = []byte(`
STATE,FIRM,PAYR_N,PAYRFL_N,STATEDSCR,NAICSDSCR,entrsizedscr
00,--,74883.53,5621697325,United States,Total,01:  Total
00,--,35806.37,241347624,United States,Total,02:  0-4`)
	egNoHeaderData1 = []byte(`
example,false,other,stuff
ex,true,text,col
		`)
	egNoHeaderData2 = []byte(`
this,example,has,a,number,column,1
this,example,has,a,number,column,2
this,example,has,a,number,column,3`)
	egNoHeaderData3 = []byte(`
one, 1, three
one, 2, three`)
	egNonDeterministicHeader = []byte(`
not,possible,to,tell,if,this,csv,data,has,a,header
not,possible,to,tell,if,this,csv,data,has,a,header
not,possible,to,tell,if,this,csv,data,has,a,header
not,possible,to,tell,if,this,csv,data,has,a,header
`)
)

// func colsEqual(a, b []*ql.Column) error {
// 	for i, aCol := range a {
// 		if b[i] == nil {
// 			return errors.New(fmt.Sprintf("column %d doesn't exist", i))
// 		}
// 		bCol := b[i]

// 		if aCol.Type != bCol.Type || aCol.Name != bCol.Name || aCol.Description != bCol.Description {
// 			return errors.New(fmt.Sprintf("columns %d aren't equal. want: %s got: %s", i, aCol, bCol))
// 		}
// 	}

// 	return nil
// }

// func TestPossibleHeaderRow(t *testing.T) {
// 	cases := []struct {
// 		data   []byte
// 		expect bool
// 	}{
// 		// {egCorruptCsvData, false},
// 		// {egNaicsCsvData, true},
// 		// {egNoHeaderData1, false},
// 		// {egNoHeaderData2, false},
// 		{egNoHeaderData3, false},
// 		// {egNonDeterministicHeader, true},
// 	}

// 	for i, c := range cases {
// 		got := possibleHeaderRow(c.data)
// 		if got != c.expect {
// 			t.Errorf("case %d response mismatch. expected: %t, got: %t", i, c.expect, got)
// 		}
// 	}
// }

// func TestDetermineCsvSchema(t *testing.T) {
// 	var (
// 		egCorruptCsvData = []byte(`
// 		"""fhkajslfnakjlcdnajcl ashklj asdhcjklads ch,,,\dagfd
// 	`)
// 		egNaicsInput = []byte(`
// STATE,FIRM,PAYR_N,PAYRFL_N,STATEDSCR,NAICSDSCR,entrsizedscr
// 00,--,74883.53,5621697325,United States,Total,01:  Total
// 00,--,35806.37,241347624,United States,Total,02:  0-4`)

// 		egNaicsCols = []*ql.Column{
// 			{Name: "STATE", Type: dataTypeInt},
// 			{Name: "FIRM", Type: dataTypeString},
// 			{Name: "PAYR_N", Type: dataTypeFloat},
// 			{Name: "PAYRFL_N", Type: dataTypeInt},
// 			{Name: "STATEDSCR", Type: dataTypeString},
// 			{Name: "NAICSDSCR", Type: dataTypeString},
// 			{Name: "entrsizedscr", Type: dataTypeString},
// 		}
// 	)

// 	cases := []struct {
// 		data   []byte
// 		expect []*ql.Column
// 		err    error
// 	}{
// 		{egCorruptCsvData, nil, ErrCorruptCsvData},
// 		{egNaicsInput, egNaicsCols, nil},
// 	}

// 	for i, c := range cases {
// 		got, err := DetermineCsvSchema(c.data)
// 		if err != c.err {
// 			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
// 		}

// 		if err := colsEqual(c.expect, got); err != nil {
// 			t.Errorf("case %d: %s", i, err)
// 		}
// 	}
// }
