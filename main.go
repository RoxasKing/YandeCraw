package main

import (
	"./hub"
	"flag"
	"fmt"
	"os"
)

var site = flag.String("s", "y", "the site's name")
var tag = flag.String("t", "", "the keyword of search")
var pages = flag.Int("a", 0, "the number of pages")
var pics = flag.Int("i", 0, "the number of pictures")
var dir = flag.String("d", "", "image save location")

func main() {
	flag.Parse()

	//*site = "y"
	//*tag = "elf"
	//*pics = 1
	//*dir = `D:/test/`

	var url string

	switch *site {
	case "y":
		url = "https://yande.re/post?tags=" + *tag
	case "d":
		url = "https://danbooru.donmai.us/posts?tags=" + *tag
	case "k":
		url = "https://konachan.com/post?tags=" + *tag
	}

	if *pages == 0 && *pics == 0 {
		fmt.Println("please input pages or pics")
		return
	}

	if *dir == "" {
		fmt.Println("please input storage address")
	}

	if (*dir)[len(*dir)-1] != '/' {
		*dir += "/"
	}

	if _, err := os.Stat(*dir); err != nil {
		fmt.Println(err)
		return
	}

	hub.Start(*site, url, *pages, *pics, *dir)

}
