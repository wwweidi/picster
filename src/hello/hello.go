package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	exif "github.com/dsoprea/go-exif/v2"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// walkFiles starts a goroutine to walk the directory tree at root and send the
// path of each regular file on the string channel.  It sends the result of the
// walk on the error channel.  If done is closed, walkFiles abandons its work.
func walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)
	go func() { // HL
		// Close the paths channel after Walk returns.
		defer close(paths) // HL
		// No select needed for this send, since errc is buffered.
		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error { // HL
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			select {
			case paths <- path: // HL
			case <-done: // HL
				return errors.New("walk canceled")
			}
			return nil
		})
	}()
	return paths, errc
}

// Result A result is the product of reading and summing a file using MD5.
type Result struct {
	SourcePath string
	DestPath   string
	Md5        string //[md5.Size]byte
	Status     string
	Err        string //error
}

func readExifDate(data []byte) (dateString string, err error) {

	rawExif, err := exif.SearchAndExtractExif(data)
	if err != nil {
		return "", err
	}

	im := exif.NewIfdMapping()

	err = exif.LoadStandardIfds(im)
	if err != nil {
		return "", err
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return "", err
	}

	const tagName = "DateTime"

	rootIfd := index.RootIfd

	// We know the tag we want is on IFD0 (the first/root IFD).
	results, err := rootIfd.FindTagWithName(tagName)
	if err != nil {
		return "", err
	}

	// This should never happen.
	if len(results) != 1 {
		//log.Panicf("there wasn't exactly one result")
		return "", err
	}

	ite := results[0]

	valueRaw, err := ite.Value()
	if err != nil {
		return "", err
	}

	value := valueRaw.(string)
	//fmt.Println(value)
	return value, nil

}

func readFileDate(path string) (date string) {
	// get last modified time
	file, err := os.Stat(path)

	if err != nil {
		fmt.Println(err)
	}

	modifiedtime := file.ModTime()
	return modifiedtime.Format("2006:01:02 15:04:05")
}

func isFoto(path string) (itis bool) {
	return strings.HasSuffix(strings.ToLower(path), "jpg")
}

func parseDate(date string) (filename string, foldername string, err error) {
	dateAsFileName := strings.ReplaceAll(strings.ReplaceAll(date, ":", ""), " ", "_")
	//fmt.Println(dateAsFileName)

	folderParts := strings.Split(date, ":")[0:2]
	//TODO check length

	folderName := strings.Join(folderParts, "-")
	//fmt.Println(folderName)

	return dateAsFileName, folderName, err
}

func getMD5(data []byte) (md5Str string) {
	hasher := md5.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func errToStr(err error) (errSTr string) {
	var errStr string
	if err != nil {
		errStr = err.Error()
	} else {
		errStr = ""
	}
	return errStr
}

// digester reads path names from paths and sends digests of the corresponding
// files on c until either paths or done is closed.
func digester(done <-chan struct{}, paths <-chan string, dest string, c chan<- Result) {

	absDestPath, err := filepath.Abs(dest)
	if err != nil {
		fmt.Println(err)
	}

	for path := range paths { // HLpaths

		var res Result
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Println(err)
		}

		if isFoto(path) {

			data, err := ioutil.ReadFile(path)
			date, err := readExifDate(data)

			if date == "" || err != nil {
				date = readFileDate(path)
			}

			dateAsFileName, folderName, err := parseDate(date)

			destPath := filepath.Join(absDestPath, folderName, dateAsFileName+".jpg")

			md5Str := getMD5(data)

			errStr := errToStr(err)

			res = Result{absPath, destPath, md5Str, "", errStr}
		} else {
			res = Result{absPath, "", "", "", "Not a foto"}
		}
		select {
		case c <- res:
		case <-done:
			return
		}
	}
}

// MD5All reads all the files in the file tree rooted at root and returns a map
// from file path to the MD5 sum of the file's contents.  If the directory walk
// fails or any read operation fails, MD5All returns an error.  In that case,
// MD5All does not wait for inflight read operations to complete.
func MD5All(root string, dest string) ([]Result, error) {
	// MD5All closes the done channel when it returns; it may do so before
	// receiving all the values from c and errc.
	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, root)

	// Start a fixed number of goroutines to read and digest files.
	c := make(chan Result) // HLc
	var wg sync.WaitGroup
	const numDigesters = 20
	wg.Add(numDigesters)
	for i := 0; i < numDigesters; i++ {
		go func() {
			digester(done, paths, dest, c) // HLc
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c) // HLc
	}()
	// End of pipeline. OMIT

	var resultList []Result
	for r := range c {
		resultList = append(resultList, r)

		// out, err := json.Marshal(r)
		// if err != nil {
		// 	panic(err)
		// }

		//fmt.Println(string(out))
	}

	// Check whether the Walk failed.
	if err := <-errc; err != nil { // HLerrc
		return nil, err
	}
	return resultList, nil
}

//CopyFile create file, copy content, delete old
func CopyFile(sourcePath, destPath string, log *logrus.Entry) {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		log.Error("Couldn't open source file: %s", err)
		return
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		log.Error("Couldn't open dest file: %s", err)
		return
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		log.Error("Writing to output file failed: %s", err)
		return
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		log.Error("Failed removing original file: %s", err)
		return
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MoveFile move file by using os.Rename
// Fallback to copy in case of error
func MoveFile(source string, destination string, log *logrus.Entry) {

	if source == "" || destination == "" {
		log.Warn("Skipped - Source or destination path missing")
		return
	}

	//prepare directory
	err := os.MkdirAll(filepath.Dir(destination), os.ModePerm)
	if err != nil {
		log.Error("Error when creating directory: %s", err)
	}

	//check if already exists
	if destFileInfo, err := os.Stat(destination); err == nil {
		log.Info("File with same name already exists")

		srcFileInfo, err := os.Stat(source)
		if err != nil {
			log.Error(err)
		}

		if srcFileInfo.Size() == destFileInfo.Size() {
			log.Info("Same size - same file ?! skipping")
			err = os.Remove(source)
			if err != nil {
				log.Error("Failed removing original file: %s", err)
			}
			return
		} else {
			newDest := destination
			cnt := 0
			for fileExists(newDest) {
				cnt = cnt + 1
				newDest = strings.Replace(destination, ".jpg", fmt.Sprintf("_%03d.jpg", cnt), -1)
			}
			destination = newDest
		}
	}

	err = os.Rename(source, destination)
	if err != nil {
		log.Info("os.Rename not possible")
		CopyFile(source, destination, log)
	}
}

//Move moves the files
func Move(files []Result) {

	length := len(files)
	count := 0

	fmt.Printf("Processing %d files", length)

	for _, file := range files {

		fileLog := log.WithFields(logrus.Fields{"srcFile": file.SourcePath})

		MoveFile(file.SourcePath, file.DestPath, fileLog)

		if count%10 == 0 {
			fmt.Printf("Progress: %d/%d\n", count, length)
		}

		count = count + 1
	}
}

func initMoveLog() {
	file, err := os.OpenFile("move.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
}

func writeScanLog(results []Result) {
	file, err := os.OpenFile("scan.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {

		out, err := json.Marshal(results)
		if err != nil {
			log.Error(err)
		}

		file.Write(out)
	} else {
		log.Info("Failed to lwrite scan to file, using default stderr")
	}
}

func main() {
	// Calculate the MD5 sum of all files under the specified directory,
	// then print the results sorted by path name.
	results, err := MD5All(os.Args[1], os.Args[2])

	if err != nil {
		fmt.Println(err)
	}

	writeScanLog(results)

	initMoveLog()
	Move(results)

}
