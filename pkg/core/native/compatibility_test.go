package native

import (
	"testing"
	"unicode"

	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/stretchr/testify/require"
)

// "C" and "O" can easily be typed by accident.
func TestNamesASCII(t *testing.T) {
	cfg := config.ProtocolConfiguration{P2PSigExtensions: true}
	cs := NewContracts(cfg)
	latestHF := config.LatestHardfork()
	for _, c := range cs.Contracts {
		require.True(t, isASCII(c.Metadata().Name))
		hfMD := c.Metadata().HFSpecificContractMD(&latestHF)
		for _, m := range hfMD.Methods {
			require.True(t, isASCII(m.MD.Name))
		}
		for _, e := range hfMD.Manifest.ABI.Events {
			require.True(t, isASCII(e.Name))
		}
	}
}

func isASCII(s string) bool {
	ok := true
	for i := 0; i < len(s); i++ {
		ok = ok && s[i] <= unicode.MaxASCII
	}
	return ok
}
