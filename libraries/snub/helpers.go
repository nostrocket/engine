package snub

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
)

func GetCurrentDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}

func compressBytes(input []byte) ([]byte, error) {
	var compressed bytes.Buffer
	gzWriter := gzip.NewWriter(&compressed)
	_, err := gzWriter.Write(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compress string: %v", err)
	}

	err = gzWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}

	return compressed.Bytes(), nil
}
