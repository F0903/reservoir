package utils

import "os"

func OpenWithSize(path string) (*os.File, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}

	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}

	return f, st.Size(), nil
}
