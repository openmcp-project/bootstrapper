package cmd_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/bootstrapper/cmd"
	testutil "github.com/openmcp-project/bootstrapper/test/utils"
)

func TestOcmTransfer(t *testing.T) {
	expectError := errors.New("expected error")

	testutil.DownloadOCMAndAddToPath(t)

	ctfIn := testutil.BuildComponent("./testdata/component-constructor.yaml", t)
	ctfOut := filepath.Join(t.TempDir(), "ctfOut")

	testCases := []struct {
		desc          string
		arguments     []string
		expectedError error
	}{
		{
			desc:          "No arguments specified",
			arguments:     []string{},
			expectedError: expectError,
		},
		{
			desc:          "One argument specified",
			arguments:     []string{"source"},
			expectedError: expectError,
		},
		{
			desc:          "Two arguments specified",
			arguments:     []string{ctfIn, ctfOut},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			root := cmd.RootCmd
			args := []string{"ocm-transfer"}
			if len(tc.arguments) > 0 {
				args = append(args, tc.arguments...)
			}
			root.SetArgs(args)

			err := root.Execute()
			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
