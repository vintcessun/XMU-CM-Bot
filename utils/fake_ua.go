package utils

import browser "github.com/EDDYCJY/fake-useragent"

func GetFakeUA() string {
	return browser.Random()
}

func GetFakeUAComputer() string {
	return browser.Computer()
}
