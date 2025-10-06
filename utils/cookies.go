package utils

import "net/http"

func GetSessionCookie(cookies []*http.Cookie, name string) (*http.Cookie, bool) {
	for _, cookie := range cookies {
		// 精确匹配Cookie名称（区分大小写）
		if cookie.Name == name {
			return cookie, true
		}
	}
	return nil, false
}
