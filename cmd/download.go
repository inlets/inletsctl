package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	inletsPro       bool
	downloadVersion string
	destination     string
	verbose         bool
)

func init() {
	inletsCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().BoolVar(&inletsPro, "pro", true, "Download inlets PRO")
	downloadCmd.Flags().StringVar(&downloadVersion, "version", "", "specific version to download")
	downloadCmd.Flags().StringVar(&destination, "download-to", "/usr/local/bin", "location to download to (Default: /usr/local/bin)")
	downloadCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show download URL")

}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Downloads the inlets PRO binary",
	Long:  `Downloads the inlets PRO binary from the GitHub releases page. Only inlets PRO is supported.`,
	Example: `  inletsctl download
  inletsctl download --version 0.2.6 
  inletsctl download --pro --version 0.2.6 
  inletsctl download --pro`,
	RunE:          downloadInlets,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func downloadInlets(_ *cobra.Command, _ []string) error {

	var versionUrl, downloadUrl, binaryName string

	if inletsPro {
		versionUrl = "https://github.com/inlets/inlets-pro/releases/latest"
		downloadUrl = "https://github.com/inlets/inlets-pro/releases/download/"
		binaryName = "inlets-pro"
	} else {
		versionUrl = "https://github.com/inlets/inlets-pro/releases/latest"
		downloadUrl = "https://github.com/inlets/inlets-pro/releases/download/"
		binaryName = "inlets-pro"
	}

	osVal := runtime.GOOS
	arch := runtime.GOARCH

	arch, extension := buildFilename(arch, osVal)

	if len(downloadVersion) == 0 {
		var err error
		downloadVersion, err = findRelease(versionUrl)
		if err != nil {
			return err
		}
	}

	url := downloadUrl + downloadVersion + "/" + binaryName + arch + extension
	if verbose {
		fmt.Printf("URL: %s.\n", url)
	}
	fmt.Printf("Starting download of %s %s, this could take a few moments.\n", binaryName, downloadVersion)
	output, err := downloadBinary(http.DefaultClient, url, binaryName)

	if err != nil {
		return err
	}

	var permissionErr bool
	err, permissionErr = moveFile(output, path.Join(destination, binaryName))
	if err != nil && !permissionErr {
		return err
	}

	if permissionErr {
		installLocation := path.Join(destination, binaryName)
		fmt.Printf(`==============================================================
    The command was run as a user who is unable to write
    to %s. To complete the installation run as a 
    user that can write to this location.
==============================================================

Alternatively you can move the file using these commands
  curl -SLsf %s > /tmp/%s
  chmod a+x %s
  %s version
  sudo mv %s  %s
`, destination, url, binaryName, output, output, output, installLocation)
		if err := os.Remove(output); err != nil {
			return err
		}
		return nil
	}

	fmt.Printf(`Download completed, make sure that %s is on your path. 
  %s version
`, destination, binaryName)

	return nil
}

func findRelease(url string) (string, error) {

	client := http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	loc := res.Header.Get("Location")
	if len(loc) == 0 {
		return "", fmt.Errorf("unable to determine release of inlets")
	}
	log.Println(loc)
	// if res.StatusCode != http.StatusFound && res.StatusCode != http.StatusMovedPermanently {
	// 	return "", fmt.Errorf("incorrect status code from HEAD got: %d", res.StatusCode)
	// }

	version := loc[strings.LastIndex(loc, "/")+1:]
	return version, nil
}

func downloadBinary(client *http.Client, url, name string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	tempDir := os.TempDir()
	outputPath := path.Join(tempDir, name)
	if res.Body != nil {
		defer res.Body.Close()
		res, _ := ioutil.ReadAll(res.Body)

		err := ioutil.WriteFile(outputPath, res, 0777)
		if err != nil {
			return "", err
		}
		return outputPath, nil
	}

	return "", fmt.Errorf("error downloading %s", url)
}

func moveFile(source, destination string) (error, bool) {
	src, err := os.Open(source)
	if err != nil {
		return err, false
	}
	defer src.Close()
	fi, err := src.Stat()
	if err != nil {
		return err, false
	}
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	perm := fi.Mode() & os.ModePerm

	dst, err := os.OpenFile(destination, flag, perm)
	if err != nil {
		return err, true
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		dst.Close()
		os.Remove(destination)
		return err, false
	}
	err = dst.Close()
	if err != nil {
		return err, false
	}
	err = src.Close()
	if err != nil {
		return err, false
	}
	err = os.Remove(source)
	if err != nil {
		return err, false
	}
	return nil, false
}

func buildFilename(arch, osVal string) (string, string) {
	extension := ""

	if osVal == "windows" {
		extension = ".exe"
	}

	if arch == "arm" {
		arch = "armhf"
	}

	if osVal == "darwin" {
		arch = "-" + osVal
	} else if arch == "amd64" {
		arch = ""
	} else {
		arch = "-" + arch
	}
	return arch, extension
}
