// fileutils.go
package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

const SizeThreshold = 2 * 1024 * 1024 * 1024 // 2GB

func CalculateFileHash(filePath string, fileSize int64) (string, error) {
	if fileSize > SizeThreshold {
		return CalculateFileHashSHA512(filePath)
	} else {
		return CalculateFileHashMD5(filePath)
	}
}

func CalculateFileHashMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var h hash.Hash
	h = md5.New()

	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CalculateFileHashSHA512(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var h hash.Hash
	h = sha512.New()

	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CompareFilesBinary(path1, path2 string) (bool, error) {
	f1, err := os.Open(path1)
	if err != nil {
		return false, err
	}
	defer f1.Close()

	f2, err := os.Open(path2)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	const chunkSize = 64 * 1024
	buf1 := make([]byte, chunkSize)
	buf2 := make([]byte, chunkSize)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		if err1 != nil && err1 != io.EOF {
			return false, err1
		}
		if err2 != nil && err2 != io.EOF {
			return false, err2
		}
		if n1 != n2 {
			return false, nil
		}
		if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}
		if !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}
	}
}
