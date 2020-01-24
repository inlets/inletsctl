package env

import (
	"github.com/spf13/pflag"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func Test_GetFileOrString_NoEnvVar_File(t *testing.T) {
	want := "some contents in this file"
	file := makeTempFile(want)

	flags := pflag.FlagSet{}

	flags.String("file-flag", file, "")

	got, err := GetRequiredFileOrString(&flags, "file-flag", "value-flag", "TEST_INLETS_NOT_SET")

	if err != nil {
		t.Errorf("got error when reading file: %s", err.Error())
	}

	if want != got {
		t.Errorf("want: %s, but got: %s", want, got)
	}

}

func Test_GetFileOrString_NoEnvVar_String(t *testing.T) {
	want := "this-value-is-set"
	flags := pflag.FlagSet{}

	flags.String("value-flag", want, "")

	got, err := GetRequiredFileOrString(&flags, "file-flag", "value-flag", "TEST_INLETS_NOT_SET")

	if err != nil {
		t.Errorf("got error when getting value: %s", err.Error())
	}

	if want != got {
		t.Errorf("want: %s, but got: %s", want, got)
	}
}

func Test_GetFileOrString_NoEnvVar_Nothing(t *testing.T) {
	flags := pflag.FlagSet{}

	_, err := GetRequiredFileOrString(&flags, "file-flag", "value-flag", "TEST_INLETS_NOT_SET")

	if err == nil {
		t.Errorf("expected error when trying to get value")
	}

}

func Test_GetFileOrString_InvalidFile(t *testing.T) {
	flags := pflag.FlagSet{}

	flags.String("file-flag", "/tmp/non-exists-file", "")

	_, err := GetRequiredFileOrString(&flags, "file-flag", "value-flag", "TEST_INLETS_NOT_SET")

	if err == nil {
		t.Errorf("expected error when trying to get value")
	}

}

func Test_GetFileOrString_EnvVar_File(t *testing.T) {
	want := "env var file should get found"
	file := makeTempFile(want)
	envVarName := "WE_SHOULD_FIND_THIS"

	flags := pflag.FlagSet{}

	flags.String("file-flag", file, "")

	os.Setenv(envVarName, file)
	got, err := GetRequiredFileOrString(&flags, "file-flag", "value-flag", envVarName)
	os.Unsetenv(envVarName)

	if err != nil {
		t.Errorf("got error when reading file: %s", err.Error())
	}

	if want != got {
		t.Errorf("want: %s, but got: %s", want, got)
	}
}

func Test_GetFileOrString_EnvVar_String(t *testing.T) {
	want := "we-want-this-value"
	envVarName := "VALUE_FLAG_SHOULD_OVERRIDE_THIS"

	flags := pflag.FlagSet{}

	flags.String("value-flag", want, "")

	os.Setenv(envVarName, "BLANK VALUE")
	got, err := GetRequiredFileOrString(&flags, "file-flag", "value-flag", envVarName)
	os.Unsetenv(envVarName)

	if err != nil {
		t.Errorf("got error when getting value: %s", err.Error())
	}

	if want != got {
		t.Errorf("want: %s, but got: %s", want, got)
	}
}

func Test_GetFileOrString_EnvVar_Nothing(t *testing.T) {
	want := "this file has some contents"
	file := makeTempFile(want)
	envVarName := "ENV_VAR_SET_NO_FLAGS"

	flags := pflag.FlagSet{}

	os.Setenv(envVarName, file)
	got, err := GetRequiredFileOrString(&flags, "file-flag", "value-flag", envVarName)
	os.Unsetenv(envVarName)

	if err != nil {
		t.Errorf("got error when getting value: %s", err.Error())
	}

	if file != got {
		t.Errorf("want: %s, but got: %s", want, got)
	}
}

func Test_GetFileOrString_NoVals_NotRequired(t *testing.T) {
	envVarName := "NO_VALS_ENV_VAR"

	flags := pflag.FlagSet{}

	got, err := getFileOrString(&flags, "file-flag", "value-flag", envVarName, false)

	if err != nil {
		t.Errorf("got error when getting value: %s", err.Error())
	}

	if got != "" {
		t.Errorf("want: \"\" but got: %s", got)
	}
}

func makeTempFile(contents string) string {
	file, err := ioutil.TempFile("", "prefix")
	if err != nil {
		log.Fatal(err)
	}
	file.Write([]byte(contents))

	return file.Name()
}
