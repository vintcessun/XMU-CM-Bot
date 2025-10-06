package utils

import "path/filepath"

func GetExtByFilepath(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > 0 {
		return ext[1:]
	}
	return ext
}
