package OverLoader


import (
"os"
"log"
"net/http"
"time"
"net"
"fmt"
"sync"
"strings"
"strconv"
)

type HttpDownloader struct {
	m_fileName string
	m_url string
	m_file *os.File
	m_downloadedSize int64
	m_contentLength int64
	m_dSizeMutex sync.Mutex
	m_finishedThread int
	m_threadNum int
	m_threadNumMutex sync.Mutex
	m_fileMutex sync.Mutex
}

//show the process of downloading
func (this *HttpDownloader)printSize()  {
	for {
		time.Sleep(time.Second)
		if this.m_finishedThread == this.m_threadNum {
			log.Println("process: ", "finish")
			return
		} else {
			log.Println("process: ", this.m_downloadedSize, "/", this.m_contentLength, "(bytes)")
		}
	}
}

//invoke this to start downloading
func (this *HttpDownloader)HttpDownload()  {
	this.m_downloadedSize = 0
	this.m_finishedThread = 0
	//create file
	var err error
	this.m_file, err = os.Create(this.m_fileName)
	defer this.m_file.Close()
	if err != nil {
		log.Println("Create file failed.")
		return
	}

	//get Content-Length
	var res *http.Response
	res, err = http.Get(this.m_url)
	this.m_contentLength = res.ContentLength
	fmt.Println("Content-Length:", this.m_contentLength)
	fmt.Println(res)

	time_begin := time.Now()

	go this.printSize()
	finishChan := make(chan int, 1)


	if res.Header.Get("Accept-Ranges") != ""{
		fmt.Println("支持分块下载")
	} else {
		//go this.HttpDownloadThread(0, finishChan)
		fmt.Println("不支持分块下载")
		this.m_threadNum = 1
	}

	//start thread
	// 如果支持分块下载
	for i := 0; i < this.m_threadNum; i++ {
		fmt.Println(i)
		go this.HttpDownloadThread(i, finishChan)
	}

	c := <- finishChan
	fmt.Println(c)
	time_finish := time.Now()
	time_consumption := time_finish.Sub(time_begin)
	log.Println(this.m_downloadedSize)
	log.Println("消耗时间: ", time_consumption)
	log.Println("平均下载速度: ", float64(this.m_contentLength)/ time_consumption.Seconds() / 1000, " kb/s")
}

//download thread
func (this *HttpDownloader)HttpDownloadThread(seq int, finishChan chan int)  {
	client := &http.Client{}
	req, err := http.NewRequest("GET", this.m_url, nil)
	from := (this.m_contentLength / int64(this.m_threadNum)) * int64(seq)
	var to int64
	if seq == this.m_threadNum - 1 {
		to = this.m_contentLength
	} else {
		to = (this.m_contentLength/int64(this.m_threadNum))*(int64(seq)+1) - 1
	}
	range_str := fmt.Sprintf("bytes=%d-%d", from, to)
	fmt.Println(range_str)
	offset := from
	req.Header.Set("Range", range_str)
	fmt.Println(req)
	res, err := client.Do(req)
	fmt.Println("a", seq)

	log.Println(res.Proto, res.Status)
	log.Println(res.Header)
	if err != nil {
		log.Println("Download failed.")
		return
	}

	buf := make([]byte, 4096)
	for {
		//log.Println("seq:", seq, "offset: ", offset)
		size, _ := res.Body.Read(buf)
		if size == 0 {
			log.Println("seq", seq, "Download success")
			this.m_threadNumMutex.Lock()
			this.m_finishedThread++
			if this.m_finishedThread == this.m_threadNum {
				finishChan <- 1
			}
			this.m_threadNumMutex.Unlock()
			return
		} else {
			this.m_fileMutex.Lock()
			this.m_file.WriteAt(buf[:size], offset)
			this.m_fileMutex.Unlock()

			offset += int64(size)

			this.m_dSizeMutex.Lock()
			this.m_downloadedSize += int64(size)
			this.m_dSizeMutex.Unlock()
		}
	}
}

func MagnetDownload(url string, filename string)  {
	conn, err := net.Dial("tcp4", url)
	defer conn.Close()
	if err != nil {
		log.Println("Connect failed.")
		return
	}

	buf := make([]byte, 4096)
	n, _ := conn.Read(buf)
	log.Println(buf[:n])
}

type FtpDownloader struct {
	m_conn net.Conn
	m_fileConn net.Conn
	m_controlConn net.Conn
}

func (this *FtpDownloader)Command(command string)  {
	command += "\r\n"
	buff := make([]byte, 1024)
	this.m_conn.Write([]byte(command))
	n, _ := this.m_conn.Read(buff)

	if n == 0 {
		fmt.Println("n == 0")
	}

	fmt.Println(string(buff[:n]))
	res := string(buff[:n])

	if string(buff[:3]) == "227" {
		delim_idx := strings.Index(res, ")")
		var port_str1 string
		var port_str2 string

		var i int
		for i = delim_idx - 1; res[i] != ','; i-- {
			port_str1 = string(res[i]) + port_str1
		}

		for i = i - 1; res[i] != ','; i-- {
			port_str2 = string(res[i]) + port_str2
		}

		fmt.Println(port_str1, port_str2)
		port_num2, _ := strconv.Atoi(port_str2)
		port_num1, _ := strconv.Atoi(port_str1)
		port_num := port_num2 * 256 + port_num1

		fmt.Println(port_num)
		var err error
		this.m_conn.Close()
		this.m_conn, err = net.Dial("tcp", "127.0.0.1:" + strconv.Itoa(port_num))
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (this *FtpDownloader)FtpDownload(url string)  {
	domain := url
	var err error
	this.m_conn, err = net.Dial("tcp", domain)
	if err != nil {
		log.Println(err)
	}
	buff := make([]byte, 1024)
	n, _:= this.m_conn.Read(buff)
	fmt.Println(string(buff[:n]))

	this.Command("user .....")
	this.Command("pass .....")
	this.Command("pwd")
	this.Command("PASV")
	this.Command("LIST")

	this.m_conn.Close()
}


func main() {
	downloader := HttpDownloader{m_fileName:"chrome", m_url:"http://sw.bos.baidu.com/sw-search-sp/software/2e5332c9d23dd/ChromeStandalone_65.0.3325.181_Setup.exe", m_threadNum: 4}
	downloader.HttpDownload()
}


