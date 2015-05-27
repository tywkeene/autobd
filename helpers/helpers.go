package helpers

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

func LogHttp(r *http.Request) {
	log.Printf("%s %s %s %s", r.Method, r.URL, r.RemoteAddr, r.UserAgent())
}

func LogHttpErr(w http.ResponseWriter, r *http.Request, err error, status int) {
	log.Printf("Returned error \"%s\" (HTTP %s) to %s", err.Error(), http.StatusText(status), r.RemoteAddr)
	serialErr, _ := json.Marshal(err.Error())
	http.Error(w, string(serialErr), status)
}

func GetQueryValue(name string, w http.ResponseWriter, r *http.Request) string {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		LogHttpErr(w, r, err, http.StatusInternalServerError)
		return ""
	}
	value := query.Get(name)
	if len(value) == 0 || value == "" {
		LogHttpErr(w, r, fmt.Errorf("Must specify %s", name), http.StatusBadRequest)
		return ""
	}
	return value
}

//addTarFile() and PackDir() are from https://github.com/pivotal-golang/archiver
//I was originally going to bring in the whole package as a dependency but it turns out
//the extractor package doesn't entirely work the way I thought. This works, so I'm putting it
//in here to avoid the whole dependency thing until I can fix it.
//TL;DR: This isn't my code.

func addTarFile(path string, name string, tw *tar.Writer) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	link := ""
	if fi.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(path); err != nil {
			return err
		}
	}

	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return err
	}
	if fi.IsDir() && !os.IsPathSeparator(name[len(name)-1]) {
		name = name + "/"
	}
	if hdr.Typeflag == tar.TypeReg && name == "." {
		hdr.Name = filepath.ToSlash(filepath.Base(path))
	} else {
		hdr.Name = filepath.ToSlash(name)
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if hdr.Typeflag == tar.TypeReg {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func PackDir(srcPath string, dest io.Writer) error {
	absolutePath, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}
	tw := tar.NewWriter(dest)
	defer tw.Close()

	err = filepath.Walk(absolutePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var relativePath string
		if os.IsPathSeparator(srcPath[len(srcPath)-1]) {
			relativePath, err = filepath.Rel(absolutePath, path)
		} else {
			relativePath, err = filepath.Rel(filepath.Dir(absolutePath), path)
		}

		relativePath = filepath.ToSlash(relativePath)

		if err != nil {
			return err
		}
		return addTarFile(path, relativePath, tw)
	})

	return err
}
