package logging

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDumpHexBytes(t *testing.T) {
	tm, err := time.Parse(time.RFC3339, "2025-05-07T00:43:18Z")
	require.NoError(t, err)

	t.Run("CompleteRows", func(t *testing.T) {
		text, err := dumpHexBytes(tm,
			true,
			[]byte{
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
				16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
			},
		)
		require.NoError(t, err)

		require.Equal(t, strings.Join([]string{
			"O " + tm.Format(time.RFC3339),
			"000000 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F ................",
			"000010 10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F ................",
		}, "\n")+"\n", text)
	})

	t.Run("IncompleteRows", func(t *testing.T) {
		text, err := dumpHexBytes(tm,
			true,
			[]byte{
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
				16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
			},
		)
		require.NoError(t, err)

		require.Equal(t, strings.Join([]string{
			"O " + tm.Format(time.RFC3339),
			"000000 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F ................",
			"000010 10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D ..............",
		}, "\n")+"\n", text)
		t.Logf("\n'%s'", text)
	})

	t.Run("Letters", func(t *testing.T) {
		text, err := dumpHexBytes(tm,
			false,
			[]byte{
				'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
				'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
				'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P',
				'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
			},
		)
		require.NoError(t, err)

		require.Equal(t, strings.Join([]string{
			"I " + tm.Format(time.RFC3339),
			"000000 61 62 63 64 65 66 67 68 69 6A 6B 6C 6D 6E 6F 70 abcdefghijklmnop",
			"000010 71 72 73 74 75 76 77 78 79 7A 41 42 43 44 45 46 qrstuvwxyzABCDEF",
			"000020 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 GHIJKLMNOPQRSTUV",
			"000030 57 58 59 5A WXYZ",
		}, "\n")+"\n", text)
	})

}
