// qmlServer project main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	//"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

/*
	SEESION存储信息
*/
type UserInfo struct {
	CurrConn   *net.TCPConn
	SignInTime time.Time
}

var (
	GConn2IdMap = &sync.Map{}
	GId2InfoMap = &sync.Map{}
)

const (
	DEFAULTPORT  = 7788
	DEFAULT_PATH = "./file/"
	HEAD_LEN     = 10

	CHECK_VER      = "check_ver"
	CHECK_VER_RESP = "check_ver_resp"

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
	ErrMsg string `json:"errMsg"`
}

type CheckVer struct {
	Method  string `json:"method"`
	CurrVer int    `json:"currver"`
}

type CheckVerResp struct {
	Method string `json:"method"`
	Result bool   `json:"result"`
	ErrMsg string `json:"errMsg"`
}

/*
	语音通知消息
*/
type VoiceMsg struct {
	Method   string `json:"method"`
	Sn       string `json:"sn"`
	From     string `json:"from"`
	To       string `json:"to"`
	FileName string `json:"filename"`
}

type VoiceMsgResp struct {
	Method string `json:"method"`
	Result bool   `json:"result"`
	ErrMsg string `json:"errMsg"`
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
	Method string `json:"method"`
	Result bool   `json:"result"`
	ErrMsg string `json:"errMsg"`
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
//	return ioutil.ReadFile("./file/game.apk")

/*
	处理设备的各个指令，根据指令调用各自的函数进行处理
*/
func ProcData(conn *net.TCPConn, packBuf []byte) error {
	var f *os.File
	f, err := os.Open("./file/9999999999.wav")
	if err != nil {
		fmt.Println(err)
	}
	f.Write(packBuf)
	return nil
}

func goSignMsg(conn *net.TCPConn, signIn SignIn) error {
	GConn2IdMap.Store(conn, signIn.User)
	GId2InfoMap.Store(signIn.User, UserInfo{CurrConn: conn, SignInTime: time.Now()})
	var resp SignInResp
	resp.Method = SIGN_IN_RESP
	resp.Result = true
	resp.ErrMsg = "登录成功"
	respBuf, _ := json.Marshal(resp)
	log.Println(string(respBuf))
	sendPacket(conn, respBuf)
	return nil
}

func getToConn(toUser string) *net.TCPConn {
	//
	currObj, ok := GId2InfoMap.Load(toUser)
	if !ok {
		log.Println(toUser + "缓存信息没有获取到")
	}
	currNode, ret := currObj.(UserInfo)
	if !ret {
		log.Println("类型断言错误")
	}
	return currNode.CurrConn
}

func goTextMsg(conn *net.TCPConn, texgMsg TextMsg) error {
	//
	toUser := texgMsg.To
	toConn := getToConn(toUser)

	transBuf, _ := json.Marshal(texgMsg)
	sendPacket(toConn, transBuf)

	log.Println("转发消息......")
	var resp TextMsgResp
	resp.Method = TEXT_MSG_RESP
	resp.Result = true
	resp.ErrMsg = "成功"
	respBuf, _ := json.Marshal(resp)
	sendPacket(conn, respBuf)
	return nil
}

func goCheckVer(conn *net.TCPConn, signIn SignIn) error {
	var resp SignInResp
	resp.Method = SIGN_IN_RESP
	resp.Result = true
	resp.ErrMsg = "登录成功"
	respBuf, _ := json.Marshal(resp)
	sendPacket(conn, respBuf)
	return nil
}

func goVoiceMsg(conn *net.TCPConn, voiceMsg VoiceMsg) error {

	toUser := voiceMsg.To
	toConn := getToConn(toUser)

	transBuf, _ := json.Marshal(voiceMsg)
	sendPacket(toConn, transBuf)

	var resp VoiceMsgResp
	resp.Method = TEXT_MSG_RESP
	resp.ErrMsg = "登录成功"
	respBuf, _ := json.Marshal(resp)
	sendPacket(conn, respBuf)
	return nil
}

/*
	1、处理各个指令
*/
func ProcPacket(conn *net.TCPConn, packBuf []byte) (string, error) {
	var command Command
	if err := json.Unmarshal(packBuf, &command); err != nil {
		log.Println(err)
	}
	switch command.Method {
	case SIGN_IN:
		var signIn SignIn
		if err := json.Unmarshal(packBuf, &signIn); err != nil {
			log.Println(err)
			return "", err
		}
		goSignMsg(conn, signIn)
	case CHECK_VER:
		var signIn SignIn
		if err := json.Unmarshal(packBuf, &signIn); err != nil {
			log.Println(err)
			return "", err
		}

	case TEXT_MSG:
		var texgMsg TextMsg
		if err := json.Unmarshal(packBuf, &texgMsg); err != nil {
			log.Println(err)
			return "abc1.png", err
		}
		log.Println(texgMsg)
		goTextMsg(conn, texgMsg)
	case VOICE_MSG:
		var voiceMsg VoiceMsg
		if err := json.Unmarshal(packBuf, &voiceMsg); err != nil {
			log.Println(err)
			return "", err
		}
		log.Println(voiceMsg.FileName)
		return voiceMsg.FileName, nil
	}
	return "", nil
}

func sendPacket(conn *net.TCPConn, cmdBuf []byte) error {
	head := make([]byte, 2)
	head[0] = 0x7E
	head[1] = 0x13
	allLenBuf := IntToBytes(HEAD_LEN + len(cmdBuf))
	dataLenBuf := IntToBytes(0)
	head = append(head, []byte(allLenBuf[0:4])...)
	head = append(head, []byte(dataLenBuf[0:4])...)
	head = append(head, []byte(cmdBuf)...)
	n, err := conn.Write([]byte(head))
	if err != nil {
		conn.Close()
	}
	log.Println("发送长度==>", n)
	return err
}

func recvPacket(reader io.Reader, conn *net.TCPConn, dataLen int32) ([]byte, error) {
	dataBuf := make([]byte, dataLen)
	nSum := 0
	for dataLen > 0 {
		conn.SetReadDeadline(time.Now().Add(time.Second * 180))
		tmpBuf := make([]byte, dataLen)
		if nLen, err := reader.Read(tmpBuf); err != nil {
			return nil, err
		} else {
			conn.SetReadDeadline(time.Time{})
			copy(dataBuf[nSum:], tmpBuf)
			nSum += nLen
			dataLen = dataLen - int32(nLen)
		}
	}
	return dataBuf, nil
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
	for {
		reader := bufio.NewReader(conn)
		var allLen, cmdLen, dataLen int32
		var fileName string
		if headBuf, err := recvPacket(reader, conn, HEAD_LEN); err != nil {
			log.Println(err)
			return
		} else {
			allLen = BytesToInt(headBuf[2:6])
			cmdLen = BytesToInt(headBuf[6:10])
			dataLen = allLen - cmdLen - HEAD_LEN
		}
		log.Println("总长度->命令包长度->", allLen, cmdLen)
		if cmdLen >= allLen {
			log.Println("错误：命令包长度>=总长度")
			return
		}
		if cmdBuf, err := recvPacket(reader, conn, cmdLen); err != nil {
			return
		} else {
			log.Println("命令包原始串===>", string(cmdBuf))
			if fileName, err = ProcPacket(conn, cmdBuf); err != nil {
				return
			}
		}
		if dataLen == 0 {
			continue
		}
		if dataBuf, err := recvPacket(reader, conn, dataLen); err != nil {
			return
		} else {
			var ff *os.File
			ff, err := os.Create("./file/" + fileName)
			if err != nil {
				log.Println(err)
				return
			}
			ff.Write(dataBuf)
			ff.Close()
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
