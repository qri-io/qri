package dataset

import (
	"fmt"
	"testing"
)

func TestParseFormatConfigMap(t *testing.T) {
	cases := []struct {
		df   DataFormat
		opts map[string]interface{}
		cfg  FormatConfig
		err  error
	}{
		{CsvDataFormat, map[string]interface{}{}, &CsvOptions{}, nil},
		{JsonDataFormat, map[string]interface{}{}, &JsonOptions{}, nil},
		{JsonDataFormat, map[string]interface{}{"arrayEntries": true}, &JsonOptions{ArrayEntries: true}, nil},
	}

	for i, c := range cases {
		cfg, err := ParseFormatConfigMap(c.df, c.opts)
		if err != c.err {
			t.Errorf("case %d error mismatch: %s != %s", i, c.err, err)
			continue
		}
		if err := CompareFormatConfigs(c.cfg, cfg); err != nil {
			t.Errorf("case %d config err: %s", i, err.Error())
			continue
		}
	}
}

func CompareFormatConfigs(a, b FormatConfig) error {
	if a == nil && b == nil {
		return nil
	} else if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("FormatConfig mismatch: %s != %s", a, b)
	}

	if a.Format() != b.Format() {
		return fmt.Errorf("FormatConfig mistmatch %s != %s", a.Format(), b.Format())
	}

	// TODO - exhaustive check

	return nil
}
