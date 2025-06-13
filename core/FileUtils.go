// fileutils.go
package core

import (
	"bytes"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"math/rand"
	"os"
	"time"
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

func CompareFilesBinaryRandom(path1, path2 string, sampleSize int64) (bool, error) {
	// Get file info first
	info1, err := os.Stat(path1)
	if err != nil {
		return false, err
	}
	// Commented out, because we already check this before calling this method
	//info2, err := os.Stat(path2)
	//if err != nil {
	//	return false, err
	//}
	//
	//// Different sizes = not identical
	//if info1.Size() != info2.Size() {
	//	return false, nil
	//}

	// If file is smaller than sample size, compare entire file
	fileSize := info1.Size()
	if fileSize <= int64(sampleSize) {
		return CompareFilesBinary(path1, path2)
	}

	file1, err := os.Open(path1)
	if err != nil {
		return false, err
	}
	defer file1.Close()

	file2, err := os.Open(path2)
	if err != nil {
		return false, err
	}
	defer file2.Close()

	// Generate random positions
	// Note: In Go 1.20+ rand.Seed is deprecated as the random number generator is automatically seeded
	// This code uses a local random source for better practice
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	maxOffset := fileSize - int64(sampleSize)
	offset := rng.Int63n(maxOffset + 1)

	// Read samples
	sample1 := make([]byte, sampleSize)
	sample2 := make([]byte, sampleSize)

	if _, err := file1.ReadAt(sample1, offset); err != nil {
		return false, err
	}
	if _, err := file2.ReadAt(sample2, offset); err != nil {
		return false, err
	}

	return bytes.Equal(sample1, sample2), nil
}
