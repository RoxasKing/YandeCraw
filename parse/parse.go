package parse

import (
	"net/url"
	"regexp"
)

const yandePostRe = `https://yande.re/post/show/\d+`
const konachanPostRe = `https://konachan.com/post/show/\d+`
const yandeDownloadRe = `<a class="original-file-(.+)" href="(.+)">`
const konachanDownloadRe = `<a class="original-file-(\w+)" href="(.+)" id="(\w+)">`
const yandeFileNameRE = `(yande.re .+)(.\w+)$`
const konachanFileNameRE = `(Konachan.com.+)(.\w+)$`

func ParsePicList(site string, contents []byte) [][]byte {
	var re *regexp.Regexp
	switch site {
	case "y":
		re = regexp.MustCompile(yandePostRe)
	case "k":
		re = regexp.MustCompile(konachanPostRe)
	}
	matches := re.FindAllSubmatch(contents, -1)
	return matches[0]
}

func ParsePicFile(site string, contents []byte) (path string) {
	var re *regexp.Regexp
	switch site {
	case "y":
		re = regexp.MustCompile(yandeDownloadRe)
	case "k":
		re = regexp.MustCompile(konachanDownloadRe)
	}
	matches := re.FindSubmatch(contents)
	path, _ = url.QueryUnescape(string(matches[2]))
	return
}

func ParseFileName(site string, path string) string {
	var re *regexp.Regexp
	switch site {
	case "y":
		re = regexp.MustCompile(yandeFileNameRE)
	case "k":
		re = regexp.MustCompile(konachanFileNameRE)
	}
	matches := re.FindStringSubmatch(path)
	return matches[0]
}
