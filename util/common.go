package util

import "strings"

func RemoveComma(old string) (new string) {
	old = strings.ReplaceAll(old, ",", " ")
	new = strings.ReplaceAll(old, "，", " ")
	return new
}

func RemoveSpace(old string) (new string) {
	old = strings.ReplaceAll(old, " ", "")
	new = strings.ReplaceAll(old, "\n", "")
	return new
}

func SetNull(s string) string {
	if s == "暂无数据" {
		s = ""
	}
	return s
}
