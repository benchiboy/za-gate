// qmlServer project main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	GConn2UserMap = &sync.Map{}
)

const (
	DEFAULTPORT  = 7788
	DEFAULT_PATH = "./file/"
	HEAD_LEN     = 5

	UP_FILE_INFO      = "up_file_info"
	UP_FILE_INFO_RESP = "up_file_info_resp"

	SIGN_IN      = "sign_in"
	SIGN_IN_RESP = "sign_in_resp"

	TEXT_MSG      = "text_msg"
	TEXT_MSG_RESP = "text_msg_resp"

	VOICE_MSG      = "voice_msg"
	VOICE_MSG_RESP = "voice_msg_resp"
)

//整形转换成字节
func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

//字节转换成整形
func BytesToInt(b []byte) int32 {
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.LittleEndian, &x)
	return int32(x)
}

type Command struct {
	Method string `json:"method"`
}

type SignIn struct {
	Method string `json:"method"`
	User   string `json:"user"`
	pass   int    `json:"pass"`
}

type SignInResp struct {
	Method string `json:"method"`
	Result bool   `json:"result"`
	Msg    string `json:"msg"`
}

type UpFileInfo struct {
	Method string `json:"method"`
	Sn     string `json:"sn"`
	Name   string `json:"name"`
	Length int    `json:"length"`
}

type UpFileInfoResp struct {
	Method  string `json:"method"`
	Sn      string `json:"sn"`
	Success bool   `json:"success"`
}

/*
	语音通知消息
*/
type VoiceMsg struct {
	Method string `json:"method"`
	Sn     string `json:"sn"`
	From   string `json:"from"`
	To     string `json:"to"`
	Name   string `json:"name"`
	Msg    string `json:"msg"`
}

type VoiceMsgResp struct {
	Method  string `json:"method"`
	Sn      string `json:"sn"`
	Success bool   `json:"success"`
}

/*
	文本消息
*/
type TextMsg struct {
	Method string `json:"method"`
	Sn     string `json:"sn"`
	From   string `json:"from"`
	To     string `json:"to"`
	Msg    string `json:"msg"`
}

type TextMsgResp struct {
	Method  string `json:"method"`
	Sn      string `json:"sn"`
	Success bool   `json:"success"`
}

/*
	1、上传用户头像图片
	2、如果上传成功，更新DB中存储文件名称
*/
func UploadPics_input(w http.ResponseWriter, r *http.Request) {
	r.FormFile("image")
	file, handle, err := r.FormFile("image")
	if err != nil {
		log.Println(err.Error())
		err.Error()
		return
	}
	r.FormFile("image")
	userName := r.FormValue("image")
	fileName := userName + "-" + handle.Filename

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
	io.Copy(f, file)
	if err != nil {
		err.Error()
	}
	defer f.Close()
	defer file.Close()
	fmt.Println("upload success")

	return
}

/*
	处理设备的各个指令，根据指令调用各自的函数进行处理
*/
func ProcData(conn *net.TCPConn, packBuf []byte) error {
	var f *os.File
	f, err := os.Create("./file/9999999999.wav")
	if err != nil {
		fmt.Println(err)
	}
	f.Write(packBuf)
	return nil
}

/*
	处理设备的各个指令，根据指令调用各自的函数进行处理
*/
func ProcPacket(conn *net.TCPConn, packBuf []byte) error {
	var command Command
	if err := json.Unmarshal(packBuf, &command); err != nil {
		log.Println(err)
	}
	switch command.Method {

	case SIGN_IN:
		var signIn SignIn
		if err := json.Unmarshal(packBuf, &signIn); err != nil {
			log.Println(err)
			return err
		}

	case UP_FILE_INFO:
		var upInfo UpFileInfo
		if err := json.Unmarshal(packBuf, &upInfo); err != nil {
			log.Println(err)
			return err
		}
		log.Println(upInfo)

	case VOICE_MSG:
		var voiceMsg VoiceMsg
		if err := json.Unmarshal(packBuf, &voiceMsg); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

/*
	处理TCP部分，是网关程序的核心模块
	1、接收设备发送的各个指令包
	2、按协议进行解析，根据命令进行处理
	3、如果出现超时或网络断开，进行清理处理
*/
func tcpPipe(conn *net.TCPConn) {
	ipStr := conn.RemoteAddr().String()
	defer func() {
		log.Println("Disconnect===>:"+ipStr, "Conn:", conn)
		conn.Close()
	}()
	reader := bufio.NewReader(conn)
	headBuf := make([]byte, 5)
	for {
		for {
			conn.SetReadDeadline(time.Now().Add(time.Second * 180))
			var nLen int
			nLen, err := reader.Read(headBuf)
			if err != nil || nLen <= 0 {
				log.Println(err)
				return
			}
			if nLen < HEAD_LEN {
				continue
			} else {
				break
			}
		}
		packTotal := BytesToInt(headBuf[1:5])
		packLen := int(packTotal) - 5
		log.Println("total---->packlen", packTotal, packLen)
		packBuf := make([]byte, packLen)
		tmpBuf := make([]byte, 1024*1024)
		var nSum int
		for packLen > 0 {
			nLen, err := reader.Read(tmpBuf)
			if err != nil || nLen <= 0 {
				log.Println(err)
				return
			}
			log.Println("recv====>", nLen)
			copy(packBuf[nSum:], tmpBuf)
			nSum += nLen
			packLen = packLen - nLen
		}
		if headBuf[0] == 'B' {
			log.Println("command packet")
			log.Println(string(packBuf))
			if err := ProcPacket(conn, packBuf); err != nil {
				log.Println(err)
			}
		} else {
			log.Println("data packet")
			ProcData(conn, packBuf)
		}
	}
}

func main() {
	os.Mkdir("file", 0777)

	go func() {
		log.Println("========>web start....")
		http.Handle("/pollux/", http.StripPrefix("/pollux/", http.FileServer(http.Dir("file"))))
		http.HandleFunc("/upload", UploadPics_input)
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()

	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", ":8089")
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	defer tcpListener.Close()
	log.Println("========>start....")
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			continue
		}
		log.Println("Has A New Connection===>:" + tcpConn.RemoteAddr().String())
		go tcpPipe(tcpConn)
	}

}
