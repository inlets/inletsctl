package env

import (
	"fmt"
	"github.com/spf13/pflag"
	"io/ioutil"
	"os"
	"strings"
)

func GetRequiredFileOrString(flags *pflag.FlagSet, file, value, envVarName string) (string, error) {
	return getFileOrString(flags, file, value, envVarName, true)
}

func getFileOrString(flags *pflag.FlagSet, file, value, envVarName string, required bool) (string, error) {
	var val string

	authFile, _ := flags.GetString(file)
	envToken := os.Getenv(envVarName)
	flagVal, _ := flags.GetString(value)

	if len(authFile) > 0 {
		// Fallback to the File Flag, then the Env Var
		res, err := ioutil.ReadFile(authFile)
		if err != nil {
			return "", err
		}
		val = strings.TrimSpace(string(res))
	} else {
		val = flagVal
	}

	// Finally if val isn't set we can look in the env variable
	if len(val) == 0 && len(envToken) > 0 {
		val = strings.TrimSpace(string(envToken))
	}

	if required && len(val) == 0 {
		return "", fmt.Errorf("give a value for --%s, --%s or set the environment variable %q", file, value, envVarName)
	}

	return val, nil
}
