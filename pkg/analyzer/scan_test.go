package analyzer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetworkAddressValue(t *testing.T) {

	type strBoolPair struct {
		str string
		b   bool
	}

	valuesToCheck := map[string]strBoolPair{
		"svc":                          {"svc", true},
		"svc:500":                      {"svc:500", true},
		"http://svc:500":               {"svc:500", true},
		"http://svc:500/something#abc": {"svc:500", true},
		strings.Repeat("abc", 500):     {"", false},
		"not%a*url":                    {"", false},
		"123":                          {"", false},
	}

	for val, expectedAnswer := range valuesToCheck {
		strRes, boolRes := NetworkAddressValue(val)
		require.Equal(t, expectedAnswer.b, boolRes)
		require.Equal(t, expectedAnswer.str, strRes)
	}
}
