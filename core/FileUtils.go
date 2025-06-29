package core

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math/rand"
	"os"
	"time"
)

const SizeThreshold = 2 * 1024 * 1024 * 1024 // 2GB

func CalculateFileHash(filePath string, fileSize int64) (string, error) {
	if fileSize > SizeThreshold {
		return CalculateFileHashSHA256(filePath)
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

func CalculateFileHashSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var h hash.Hash
	h = sha256.New()

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

func compareFilesBinarySampleSize(filePathA, filePathB string, sampleSize int) (bool, error) {

	// Open both files
	fileA, err := os.Open(filePathA)
	if err != nil {
		return false, fmt.Errorf("failed to open file A: %w", err)
	}
	defer fileA.Close()

	fileB, err := os.Open(filePathB)
	if err != nil {
		return false, fmt.Errorf("failed to open file B: %w", err)
	}
	defer fileB.Close()

	// Get file sizes
	statA, err := fileA.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to stat file A: %w", err)
	}

	statB, err := fileB.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to stat file B: %w", err)
	}

	// Files must be same size
	if statA.Size() != statB.Size() {
		return false, nil
	}

	fileSize := statA.Size()
	if fileSize == 0 {
		return true, nil // Both files are empty
	}

	// Adjust sample size if file is smaller
	if int64(sampleSize) > fileSize {
		sampleSize = int(fileSize)
	}

	// Generate random positions
	positions := make([]int64, sampleSize)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < sampleSize; i++ {
		positions[i] = rng.Int63n(fileSize)
	}

	// Compare bytes at random positions
	byteA := make([]byte, 1)
	byteB := make([]byte, 1)

	for _, pos := range positions {
		// Read byte from file A
		_, err := fileA.ReadAt(byteA, pos)
		if err != nil {
			return false, fmt.Errorf("failed to read from file A at position %d: %w", pos, err)
		}

		// Read byte from file B
		_, err = fileB.ReadAt(byteB, pos)
		if err != nil {
			return false, fmt.Errorf("failed to read from file B at position %d: %w", pos, err)
		}

		// Compare bytes
		if byteA[0] != byteB[0] {
			return false, nil
		}
	}

	return true, nil
}

//
//func compareFilesBinarySampleSize(path1, path2 string, sampleSize int64) (bool, error) {
//	// Get file info first
//	info1, err := os.Stat(path1)
//	if err != nil {
//		return false, err
//	}
//
//	// Commented out, because we already check this before calling this method
//	//info2, err := os.Stat(path2)
//	//if err != nil {
//	//	return false, err
//	//}
//	//
//	//// Different sizes = not identical
//	//if info1.Size() != info2.Size() {
//	//	return false, nil
//	//}
//
//	// If file is smaller than sample size, compare entire file
//	fileSize := info1.Size()
//	if fileSize <= int64(sampleSize) {
//		return CompareFilesBinary(path1, path2)
//	}
//
//	file1, err := os.Open(path1)
//	if err != nil {
//		return false, err
//	}
//	defer file1.Close()
//
//	file2, err := os.Open(path2)
//	if err != nil {
//		return false, err
//	}
//	defer file2.Close()
//
//	// Generate random positions
//	// Note: In Go 1.20+ rand.Seed is deprecated as the random number generator is automatically seeded
//	// This code uses a local random source for better practice
//	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
//	maxOffset := fileSize - int64(sampleSize)
//	offset := rng.Int63n(maxOffset + 1)
//
//	// Read samples
//	sample1 := make([]byte, sampleSize)
//	sample2 := make([]byte, sampleSize)
//
//	if _, err := file1.ReadAt(sample1, offset); err != nil {
//		return false, err
//	}
//	if _, err := file2.ReadAt(sample2, offset); err != nil {
//		return false, err
//	}
//
//	return bytes.Equal(sample1, sample2), nil
//}
