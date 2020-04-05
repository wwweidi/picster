package picster

import (
	"strings"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func getTempedFile(srcDir string, extension string, content []byte) (expectedPath string) {
	tmpfile, err := ioutil.TempFile(srcDir, "*."+extension)
	if err != nil {
		log.Fatal(err)
	}
	fileInfo, err := tmpfile.Stat()
	modTime := fileInfo.ModTime()

	if content != nil {
		tmpfile.Write(content)
	}

	tmpfile.Close()
	folderName := modTime.Format(GetConfiguration().folderNamePattern)
	fileName := modTime.Format(GetConfiguration().fileNamePattern)

	return filepath.Join(folderName, fileName+"."+extension)
}

func TestFileExtensions(t *testing.T) {

	//Arrange
	dir, err := ioutil.TempDir("", "picster_*")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)
	defer os.RemoveAll(dir) // clean up

	srcDir := filepath.Join(dir, "source")
	os.MkdirAll(srcDir, os.ModePerm)

	files := [...]string{"jpg", "avi", "mov", "mp4"}
	expectedPaths := []string{}

	for _, fileExt := range files {
		expectedPath := getTempedFile(srcDir, fileExt, nil)
		expectedPaths = append(expectedPaths, expectedPath)
	}

	dest := filepath.Join(dir, "destination")

	//Act
	results, err := ScanDir(srcDir, dest)
	Move(results)

	//Assert
	for _, expected := range expectedPaths {
		var subdir string
		if filepath.Ext(expected) == ".jpg" {
			subdir = GetConfiguration().pictureFolder
		} else {
			subdir = GetConfiguration().videoFolder
		}

		expectedPath := filepath.Join(dest, subdir, expected)

		if !FileExists(expectedPath) {
			t.Error("Expected file does not exist: ", expectedPath)
		}
	}

	sourceF, err := os.Open(srcDir)
	names, err := sourceF.Readdirnames(1)

	if len(names) >0 {
		t.Error("Expected source dir to be empty: ")
	}

}

func TestNameAndSize(t *testing.T) {

	//Arrange
	dir, err := ioutil.TempDir("", "picster_*")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)
	defer os.RemoveAll(dir) // clean up

	srcDir := filepath.Join(dir, "source")
	os.MkdirAll(srcDir, os.ModePerm)

	// Creating three files, so at least two have the same time (seconds) being created
	// the middle one gets some content to differ in size,
	// so there is one file ending with a suffix '_001.jpg'
	f1 := getTempedFile(srcDir, "jpg", nil)
	f2 := getTempedFile(srcDir, "jpg", []byte("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"))
	f3 := getTempedFile(srcDir, "jpg", nil)
	
	dest := filepath.Join(dir, "destination")

	//Act
	results, err := ScanDir(srcDir, dest)
	Move(results)

	//Assert

	
	expectedPath := filepath.Join(dest, GetConfiguration().pictureFolder, f1)
	if !FileExists(expectedPath) {
		t.Error("Expected file does not exist: ", expectedPath)
	}
	expectedPath = filepath.Join(dest, GetConfiguration().pictureFolder, f2)
	expectedPath = strings.Replace(expectedPath, ".jpg", "_001.jpg", -1)
	if !FileExists(expectedPath) {
		t.Error("Expected file does not exist: ", expectedPath)
	}
	expectedPath = filepath.Join(dest, GetConfiguration().pictureFolder, f3)
	if !FileExists(expectedPath) {
		t.Error("Expected file does not exist: ", expectedPath)
	}

	sourceF, err := os.Open(srcDir)
	names, err := sourceF.Readdirnames(1)

	if len(names) >0 {
		t.Error("Expected source dir to be empty: ")
	}
}


func TestLeaveOthers(t *testing.T) {

	//Arrange
	dir, err := ioutil.TempDir("", "picster_*")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)
	defer os.RemoveAll(dir) // clean up

	srcDir := filepath.Join(dir, "source")
	os.MkdirAll(srcDir, os.ModePerm)

	getTempedFile(srcDir, "txt", nil)
	
	dest := filepath.Join(dir, "destination")

	//Act
	results, err := ScanDir(srcDir, dest)
	Move(results)

	//Assert
	sourceF, err := os.Open(srcDir)
	names, err := sourceF.Readdirnames(1)

	if len(names) !=1 {
		t.Error("Expected source dir to contain one *.txt")
	}
}