package actors

import (
	"bytes"
	"io"
	"os"

	"nostrocket/engine/library"
)

func Open(mind, db string) (*os.File, bool) {
	if err := os.MkdirAll(directory(mind), 0777); err != nil {
		library.LogCLI(err.Error(), 0)
	}
	_, err := os.Stat(directory(mind) + db + ".dat")
	if os.IsNotExist(err) {
		return nil, false
	}
	file, err := os.Open(directory(mind) + db + ".dat")
	if err != nil {
		library.LogCLI(err.Error(), 0)
		return nil, false //IDE helper
	}
	return file, true
}

func Write(mind, db string, b []byte) {
	os.Remove(directory(mind) + db + ".dat")
	if err := os.MkdirAll(directory(mind), 0777); err != nil {
		library.LogCLI(err.Error(), 0)
	}
	f, err := os.Create(directory(mind) + db + ".dat")
	if err != nil {
		library.LogCLI(err.Error(), 0)
		return //IDE helper
	}
	defer f.Close()
	_, err = io.Copy(f, bytes.NewReader(b))
	if err != nil {
		library.LogCLI(err.Error(), 0)
		return //IDE helper
	}
}

func directory(mind string) string {
	dir := MakeOrGetConfig().GetString("rootDir")
	dir = dir + MakeOrGetConfig().GetString("flatFileDir")
	dir = dir + mind + "/"
	return dir
}
