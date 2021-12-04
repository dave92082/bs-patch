package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	log.Info("Reading config.yaml")
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("Error reading config.yaml: %s", err.Error())
	}

	if viper.GetString("nodeBinPath") == "" {
		log.Fatal("nodeBinPath not set in config.yaml")
	}

	currentPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", currentPath, viper.GetString("nodeBinPath")))

	log.Info("Finding Latest BluOS Controller")
	l, err := getLatest()
	if err != nil {
		log.Fatalf("Error getting latest client URL: %s", err.Error())
	}

	log.Infof("Downloading Latest BlueOS controller: %s", *l)

	err = downloadFile("./controller.dmg", *l)
	if err != nil {
		log.Fatalf("Error downloading latest controller dmg: %s", err.Error())
	}

	log.Info("Extracting BlueOS Controller DMG")
	extractController()

	log.Info("Extracting ASAR")
	extractAsar()

	log.Info("Patching electron.js")
	patchFile(
		"./bluos/www/js/electron.js",
		`    if(platform=='mac'){
        path = "/Applications/Spotify.app"
    }`,
		`    if(platform=='mac'){
        path = "/Applications/Spotify.app"
    }
    if(platform=='lin64') { path = "/snap/bin/spotify" }`)

	log.Info("Patching app.js - Update Check")
	patchFile("./bluos/www/app.js",
		`f7.checkAppUpdate=function(e){var t;"macOS"==f7.appInfo.platform&&(t="http://upgrade.nadelectronics.com/desktop_app/osx/version.xml?currentVersion"),"Windows"==f7.appInfo.platform&&(t="http://upgrade.nadelectronics.com/desktop_app/windows/version.xml?currentVersion")`,
		`f7.checkAppUpdate=function(e){var t;"linux" == f7.appInfo.platform && (t = "http://upgrade.nadelectronics.com/desktop_app/osx/version.xml?currentVersion"), "macOS"==f7.appInfo.platform&&(t="http://upgrade.nadelectronics.com/desktop_app/osx/version.xml?currentVersion"),"Windows"==f7.appInfo.platform&&(t="http://upgrade.nadelectronics.com/desktop_app/windows/version.xml?currentVersion")`)

	log.Info("Patching app.js - Update Platform")
	patchFile("./bluos/www/app.js",
		`.autoupgrade,queue:{loading:!1,pagesize:200,total:0}}`,
		`.autoupgrade,queue:{loading:!1,pagesize:200,total:0},platform: "linux"}`)

	log.Info("Adding electron dependency")
	addNpmPackage("electron@^9.0.0")

	log.Info("Adding electron-builder dependency")
	addNpmPackage("electron-builder")

	log.Info("Building Snap and AppImage")
	buildPackage()

	log.Info("Cleaning up")
	cleanUp()

}

func cleanUp() {
	os.Rename("./bluos/dist", "./dist")
	os.RemoveAll("./bluos")
	os.RemoveAll("./controller")
	os.Remove("./controller.dmg")
}

func addNpmPackage(packageName string) error {
	cmd := exec.Command(fmt.Sprintf("%s/npm", viper.GetString("nodeBinPath")), "install", packageName, "--save-dev")
	cmd.Dir = "./bluos"
	var out bytes.Buffer
	var eout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &eout
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error installing electron: %s", eout.String())
		return err
	}
	return nil
}

func buildPackage() error {
	cmd := exec.Command(fmt.Sprintf("%s/electron-builder", viper.GetString("nodeBinPath")))
	cmd.Dir = "./bluos"
	var out bytes.Buffer
	var eout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &eout
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error extracting ASAR: %s", eout.String())
		return err
	}
	return nil
}

func patchFile(filePath, search, replace string) error {
	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	output := strings.Replace(string(input), search, replace, 1)
	err = ioutil.WriteFile(filePath, []byte(output), 0)
	if err != nil {
		return err
	}

	return nil
}

func extractAsar() error {
	folders, err := ioutil.ReadDir("./controller/")
	if err != nil {
		return err
	}

	for _, f := range folders {
		if strings.Contains(f.Name(), "BluOS Controller") {
			cmd := exec.Command(fmt.Sprintf("%s/npx", viper.GetString("nodeBinPath")), "asar", "extract", fmt.Sprintf("./controller/%s/BluOS Controller.app/Contents/Resources/app.asar", f.Name()), "./bluos")
			var out bytes.Buffer
			var eout bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &eout
			err := cmd.Run()
			if err != nil {
				log.Errorf("Error extracting ASAR: %s", eout.String())
				return err
			}
		}
	}
	return nil
}

func extractController() error {
	cmd := exec.Command("7z", "x", "-ocontroller", "./controller.dmg")
	var eout bytes.Buffer
	cmd.Stderr = &eout
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error extracting controller: %s", eout.String())
		return err
	}
	return nil
}

func getLatest() (*string, error) {
	resp, err := http.Get("https://www.bluesound.com/downloads/")
	if err != nil {
		log.Errorf("Error getting Bluesound download page: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading response body")
		return nil, err
	}

	re := regexp.MustCompile(`(http|ftp|https)://([\w_-]+(?:(?:\.[\w_-]+)+))([\w.,@?^=%&:/~+#-]*[\w@?^=%&/~+#-])?.dmg`)
	url := string(re.Find(body))

	return &url, nil
}

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

func downloadFile(filepath string, url string) error {

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return nil
}
