package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/ry/v8worker2"
)

func SourceCodeHash(filename string, sourceCodeBuf []byte) string {
	h := md5.New()
	h.Write([]byte(filename))
	h.Write(sourceCodeBuf)
	return hex.EncodeToString(h.Sum(nil))
}

func CacheFileName(filename string, sourceCodeBuf []byte) string {
	cacheKey := SourceCodeHash(filename, sourceCodeBuf)
	return path.Join(CompileDir, cacheKey+".js")
}

// Fetches a remoteUrl but also caches it to the localFilename.
func FetchRemoteSource(remoteUrl string, localFilename string) ([]byte, error) {
	//println("FetchRemoteSource", remoteUrl)
	Assert(strings.HasPrefix(localFilename, SrcDir), localFilename)
	var sourceReader io.Reader

	file, err := os.Open(localFilename)
	if *flagReload || os.IsNotExist(err) {
		// Fetch from HTTP.
		println("Downloading", remoteUrl)
		res, err := http.Get(remoteUrl)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		err = os.MkdirAll(path.Dir(localFilename), 0700)
		if err != nil {
			return nil, err
		}

		// Write to to file. Need to reopen it for writing.
		file, err = os.OpenFile(localFilename, os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return nil, err
		}
		sourceReader = io.TeeReader(res.Body, file) // Fancy!

	} else if err != nil {
		return nil, err
	} else {
		sourceReader = file
	}
	defer file.Close()
	return ioutil.ReadAll(sourceReader)
}

func LoadOutputCodeCache(filename string, sourceCodeBuf []byte) (outputCode string, err error) {
	cacheFn := CacheFileName(filename, sourceCodeBuf)
	outputCodeBuf, err := ioutil.ReadFile(cacheFn)
	if os.IsNotExist(err) {
		err = nil // Ignore error if we can't load the cache.
	} else if err != nil {
		outputCode = string(outputCodeBuf)
	}
	return
}

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func loadAsset(w *v8worker2.Worker, path string) {
	data, err := Asset(path)
	check(err)
	err = w.Load(path, string(data))
	check(err)
}

func createDirs() {
	DenoDir = path.Join(UserHomeDir(), ".deno")
	CompileDir = path.Join(DenoDir, "compile")
	err := os.MkdirAll(CompileDir, 0700)
	check(err)
	SrcDir = path.Join(DenoDir, "src")
	err = os.MkdirAll(SrcDir, 0700)
	check(err)
}
