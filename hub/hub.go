package hub

import (
	"../parse"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 存放 ResponseBody
var bodys = make(chan interface{}, 2)

// 存放图片详情地址
var details = make(chan string, 21)

// 存放图片下载地址
var urls = make(chan string, 42)

// 需要下载图片数量
var tSum int

// 成功下载数量
var sSum int32

// 下载任务队列
var wg sync.WaitGroup

// 分段下载队列
var mg sync.WaitGroup

// 分段下载线程数
var threads int64 = 10

var client = http.Client{Timeout: time.Second * 1800}

// site: 站点缩写
// url: 站点地址
// pages: 下载页数
// pics: 下载图片数
// dir: 保存地址
func Start(site, url string, pages, pics int, dir string) {

	if pages != 0 { // 按页数

		go func(url string, pages int) {
			defer close(bodys)
			for i := 1; i <= pages; i++ {
				// 获取 ResponseBody
				body, err := fetch(url + "&page=" + strconv.Itoa(i))
				if err != nil {
					fmt.Println(err)
				} else {
					bodys <- body
				}
			}
		}(url, pages)

	} else { // 按图片数

		go func(url string, pics int) {
			defer close(bodys)
			for i := 1; tSum < pics; i++ {
				// 获取 ResponseBody
				body, err := fetch(url + "&page=" + strconv.Itoa(i))
				if err != nil {
					fmt.Println(err)
				} else {
					bodys <- body
				}
			}
		}(url, pics)

	}

	// 处理 ResponseBody
	go handleBodys(site, pages, pics)

	// 获取图片下载地址
	go handlePicPath(site)

	// 下载图片
	downloadPics(site, dir)

	if pages != 0 {
		fmt.Printf("需要下载页数 %d, 成功下载图片 %d", pages, sSum)
	} else {
		fmt.Printf("需要下载图片数 %d, 成功下载图片 %d", pics, sSum)
	}

}

// 通过 url 获取 responsebody 数据
func fetch(url string) (body []byte, err error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	} else {
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			err = fmt.Errorf("Error, Status Code is %d", res.StatusCode)
			return nil, err
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			err = fmt.Errorf("Error, can't read response body")
			return nil, err
		}
		return body, nil
	}
}

// 处理 responsebody 数据
func handleBodys(site string, pages, pics int) {
	defer close(details)
	for body := range bodys {
		matches := parse.ParsePicList(site, body.([]byte))
		for _, m := range matches {
			if pages != 0 || pics != 0 && tSum < pics {
				details <- string(m)
				tSum++
			} else {
				break
			}
		}
	}
}

// 处理图片详情地址
func handlePicPath(site string) {
	defer close(urls)
	for t := range details {
		body, err := fetch(t)
		if err != nil {
			fmt.Println(err)
		} else {
			path := parse.ParsePicFile(site, body)
			urls <- path
		}
	}
}

// 下载图片主程序
func downloadPics(site, dir string) {
	for url := range urls {
		// 获取图片名
		fileName := parse.ParseFileName(site, url)
		// 创建文件
		file, err := os.Create(dir + fileName)
		if err != nil {
			fmt.Printf("创建文件 %s 失败", fileName)
			continue
		}
		// 获取图片下载信息: 图片大小 是否支持多线程
		size, flag, err := getHeaderInform(url)
		if err != nil {
			fmt.Println(err)
			continue
		}
		// 如果图片支持多线程下载
		if flag {
			wg.Add(1)
			go multithreadedDownload(url, file, size)
		} else {
			wg.Add(1)
			go singlethreadedDownload(url, file, size)
		}
	}
	wg.Wait()
	return
}

// 获取报文头信息
func getHeaderInform(url string) (ContentLength int64, flag bool, err error) {
	//HEAD 方法请求服务端是否支持多线程下载,并获取文件大小
	if req, err := http.NewRequest("HEAD", url, nil); err != nil {
		return 0, false, err
	} else {
		if res, err := client.Do(req); err != nil {
			return 0, false, err
		} else {
			defer res.Body.Close()
			// 获取图片大小
			ContentLength := res.ContentLength
			// 判断是否支持多线程下载
			if strings.Compare(res.Header.Get("Accept-Ranges"), "bytes") == 0 {
				// 支持，走多线程下载流程
				return ContentLength, true, nil
			} else {
				return ContentLength, false, nil
			}
		}
	}
}

// 多线程下载
func multithreadedDownload(url string, file *os.File, contentLength int64) {
	defer wg.Done()
	defer file.Close()
	// 每块下载大小
	packageSize := contentLength / threads
	//下载成功线程统计
	var count int64
	//分配下载线程
	for i := 0; i < int(threads); i++ {
		//计算每个线程下载的区间,起始位置
		var start int64
		var end int64
		start = int64(int64(i) * packageSize)
		end = start + packageSize
		if i+1 == int(threads) {
			end = contentLength
		}
		// 分段请求下载
		if req, err := http.NewRequest("GET", url, nil); err != nil {
			fmt.Println(err)
			break
		} else {
			req.Header.Set(
				"Range", "bytes="+strconv.FormatInt(start, 10)+"-"+
					strconv.FormatInt(end, 10))
			mg.Add(1)
			go sliceDownload(req, file, &count, start)
		}
	}
	mg.Wait()
	// 如果完成下载进度
	if count == threads {
		// 成功下载图片数 +1
		atomic.AddInt32(&sSum, 1)
	}
}

// 分段下载队列
func sliceDownload(req *http.Request, file *os.File, count *int64, start int64) {
	defer mg.Done()
	if response, err := client.Do(req); err == nil && response.StatusCode == 206 {
		defer response.Body.Close()
		if bytes, i := ioutil.ReadAll(response.Body); i == nil {
			//从我们计算好的起点写入文件
			_, _ = file.WriteAt(bytes, start)
			atomic.AddInt64(count, 1)
		} else {
			panic(err)
		}
	} else {
		panic(err)
	}
}

// 单线程下载
func singlethreadedDownload(path string, file *os.File, size int64) {
	defer wg.Done()
	// 获取图片二进制数据
	body, err := fetch(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	n, err := file.Write(body)
	if err == nil && n < int(size) {
		fmt.Println(io.ErrShortWrite)
		return
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	file.Close()
	// 成功下载图片数 +1
	atomic.AddInt32(&sSum, 1)
}
