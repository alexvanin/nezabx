package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var testCases = [...]struct {
	config  string
	isValid bool
	errMsg  string
}{
	{
		config: `
state:
  bolt: ./nz.state
notifications:
  email:
    smtp: smtp.gmail.com:587
    login: nzbx@corp.com
    password: secret
    groups:
      - name: developers
        addresses:
          - alex@corp.com
commands:
  - name: Test command
    exec: ./script.sh arg1
    cron: "*/5 * * * *"
    interval: 10s
    timeout: 2s
    threshold: 3
    threshold_sleep: 5s
    notifications:
      - email:developers`,
		isValid: true,
	},
	{
		config: `
notifications:
  email:
    groups:
      - name: developers`,
		errMsg: "contains no addresses",
	},
	{
		config: `
notifications:
  email:
    groups:
      - name: developers
        addresses:
          - foo@corp.com
      - name: developers
        addresses:
          - bar@corp.com
`,
		errMsg: "non unique email group name",
	},
	{
		config: `
commands:
  - name: Test command
    notifications:
      - hello wolrd`,
		errMsg: "invalid notification tuple",
	},
	{
		config: `
commands:
  - name: Test command
    notifications:
      - foo:bar`,
		errMsg: "invalid notification type",
	},
	{
		config: `
notifications:
  email:
    groups:
      - name: developers
        addresses:
          - foo@corp.com
commands:
  - name: Test command
    notifications:
      - email:ops`,
		errMsg: "invalid notification group",
	},
}

func TestParseConfig(t *testing.T) {
	for i, testCase := range testCases {
		_, err := parseConfig([]byte(testCase.config))
		require.Equal(t, testCase.isValid, err == nil, fmt.Sprintf("test:%d, err:%s", i, err))
		if !testCase.isValid {
			require.Contains(t, err.Error(), testCase.errMsg)
		}
	}
}
