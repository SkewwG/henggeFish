package main

import (
"bufio"
"bytes"
"context"
"crypto/rand"
"encoding/base64"
"fmt"
"github.com/tencentyun/scf-go-lib/cloudfunction"
"io/ioutil"
"log"
"math"
"math/big"
marand "math/rand"
"mime/quotedprintable"
"net/smtp"
"os"
"strings"
"time"
"github.com/Unknwon/goconfig"
)

var sendName string		// 发件人
var emailTitle string	// 邮件标题
var emailContent string	// 邮件正文
var fileName string		// 附件名称

// 每个163邮箱的结构体
type mail163 struct {
	user string		// 账号
	pass string		// 密码
	host string		// host
	success int		// 成功次数
	targetMails []string	// 成功发送的邮箱列表
}


// 发送邮件
/*
=?gb2312?B?xOO6ww==?=
第一个？同第二个？之间的gb2312代表标题内容所使用的字符集
第二个？和第三个？之间的B代表这部分内容采用的是base64编码方式，如果采用Quoted-printabel编码方式则显示Q
第三个？和第四个？之间则是"你好"经过base64编码后的字符串。
*/

func sendMail(eachMail163 mail163, targetMail string, Content []byte) string {
	boundary := GetRandomString(60)
	//boundary_two := GetRandomString(60)
	messageID, _ := generateMessageID(targetMail)
	mime := bytes.NewBuffer(nil)
	var bb = time.Now().UTC()
	var cstSh, _ = time.LoadLocation("Asia/Shanghai")

	//设置邮件
	mime.WriteString(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nMessage-Id: %s\r\nDate: %s\r\n",
		"=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(sendName)) + "?=<"+eachMail163.user+">",
		targetMail,
		"=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(emailTitle)) + "?=",
		messageID,
		bb.In(cstSh)))

	// 定义
	// mixed包含附件
	// alternative纯文本与超文本共存
	mime.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	mime.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	mime.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
	mime.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))

	// 文本内容-base64编码
	html := fmt.Sprintf(emailContent)
	mime.WriteString("Content-Transfer-Encoding: base64\r\n")
	mime.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
	html_body := base64.StdEncoding.EncodeToString([]byte(html))

	// 文本内容-qutoted协议
	//html := fmt.Sprintf("HTML正文内容%s", targetMail)
	//mime.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	//mime.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	//html_body,_ := toQuotedPrintable(html)

	mime.WriteString(html_body+"\r\n")
	mime.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))

	// 附件
	file_name := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(fileName)) + "?="
	mime.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	mime.WriteString("Content-Type: application/octet-stream\r\n")
	mime.WriteString("Content-ID: "+fmt.Sprintf("<%s>", file_name)+"\r\n")
	mime.WriteString("Content-Description: "+ file_name +"\r\n")
	mime.WriteString("Content-Transfer-Encoding: base64\r\n")
	mime.WriteString("Content-Disposition: attachment; filename=\"" + file_name + "\"\r\n\r\n")
	for index, line := 0, len(Content); index < line; index++ {
		mime.WriteByte(Content[index])
		if (index+1)%76 == 0 {
			mime.WriteString("\r\n")
		}
	}


	// 结束邮件声明
	mime.WriteString("\r\n--" + boundary + "--\r\n\r\n")

	//fmt.Println(mime)
	// 获取配置文件 email账户
	user := eachMail163.user
	// 获取配置文件 email密码
	password := eachMail163.pass
	// 获取配置文件 host发送地址
	host := eachMail163.host
	// 邮箱认证
	hp := strings.Split(host, ":")

	auth := smtp.PlainAuth("", user, password, hp[0])
	// 分割发送者邮箱
	send_to := strings.Split(targetMail, ";")
	// 发送email      账户   认证  发送者  发给谁   邮件内容
	err := smtp.SendMail(host, auth, user, send_to, mime.Bytes())
	//fmt.Println(a)
	if (err == nil){
		return "success"
	} else {
		//fmt.Println(err.Error())
		return "error"
	}
}

// 读取文件
func readfile_bufio_scanner(filename string)[]string  {
	var result = []string{}
	f,err := os.Open(filename)
	if err!= nil {
		fmt.Printf("Error: %s\n", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan(){
		line := scanner.Text()
		result = append(result,line)
	}
	if err := scanner.Err();err != nil{
		log.Printf("Cannot scanner text file: %s, err: [%v]", filename, err)
	}

	return result
}


// 读取文件中账户密码
func get_mailconns(kamis []string) map[int]mail163  {
	allMail163 := make(map[int]mail163)
	//var allMail163 = []mail163{}
	i := 0
	for _, x := range kamis {
		user_pass := strings.Split(x,"----")
		allMail163[i] = mail163{
			user: user_pass[0],
			pass: user_pass[1],
			host: "smtp.163.com:25",
			success: 0,
			targetMails: []string{},
		}
		i += 1
	}
	return allMail163
}

// 获取MessageID
func generateMessageID(targetMail string) (string, error) {
	t := time.Now().UnixNano()
	pid := os.Getpid()
	var maxBigInt = big.NewInt(math.MaxInt64)
	rint, err := rand.Int(rand.Reader, maxBigInt)
	if err != nil {
		return "", err
	}
	h := fmt.Sprintf("%s", targetMail)
	msgid := fmt.Sprintf("<%d.%d.%d.Coremail.%s>", t, pid, rint, h)
	return msgid, nil
}

// 随机生成字符串
func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := marand.New(marand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

// 生成qutoted协议格式
func toQuotedPrintable(s string) (string, string) {
	var ac bytes.Buffer
	w := quotedprintable.NewWriter(&ac)
	_, err := w.Write([]byte(s))
	if err != nil {
		return "", "write"
	}
	err = w.Close()
	if err != nil {
		return "", "close"
	}
	return ac.String(), ""
}


type DefineEvent struct {
	// test event define
	Key1 string `json:"key1"`
	Key2 string `json:"key2"`
}

func hello(ctx context.Context, event DefineEvent) (string, error) {
	// 附件内容
	atterData,_ := ioutil.ReadFile("./1.zip")
	b := make([]byte,base64.StdEncoding.EncodedLen(len(atterData)))
	base64.StdEncoding.Encode(b,atterData)
	Content := []byte(b)


	// 目标邮箱
	targetMails := readfile_bufio_scanner("target.txt")
	//fmt.Println(targetMails)

	// 邮箱池
	kamis := readfile_bufio_scanner("kami.txt")
	allMail163 := get_mailconns(kamis)

	cfg, err := goconfig.LoadConfigFile("./conf.ini")
	if err != nil{
		panic("错误")
	}
	sendName, _ = cfg.GetValue("发件人", "sendName")
	emailTitle, _ = cfg.GetValue("邮件标题", "emailTitle")
	fileName, _ = cfg.GetValue("附件名称", "fileName")
	emailContent_tmp, _ := cfg.GetValue("邮件正文", "emailContent")
	emailContent2, _ := base64.StdEncoding.DecodeString(emailContent_tmp)
	emailContent = string(emailContent2)

	// 没发送的邮箱列表
	notSendTargetMails := []string{}

	for _, targetMail := range targetMails{
		sendFlag := false
		for i, eachMail163 := range allMail163{
			if eachMail163.success < 10 {
				info := sendMail(eachMail163, targetMail, Content)
				//info := "success"
				if info == "success" {
					sendFlag = true
					eachMail163.success += 1
					eachMail163.targetMails = append(eachMail163.targetMails, targetMail)
					allMail163[i] = eachMail163
					fmt.Printf("[+] %s 发送成功 %s 已发送次数:%d\n", targetMail, eachMail163.user, eachMail163.success)
					break
				} else {
					fmt.Printf("[-] %s 发送失败 %s \n", targetMail, eachMail163.user)
					notSendTargetMails = append(notSendTargetMails, targetMail)
				}
			}else {
				fmt.Printf("[*] %s 已发送十封邮件\n", eachMail163.user)
			}
		}
		if sendFlag == false{
			notSendTargetMails = append(notSendTargetMails, targetMail)
		}
	}

	for _, v := range allMail163{
		fmt.Printf("%s 已发送邮件列表 %s\n\n", v.user, v.targetMails)
	}

	fmt.Printf("[%d] 没有发送的邮件列表 %s\n", len(notSendTargetMails), notSendTargetMails)

	return fmt.Sprintf("Hello %s!", event.Key1), nil
}

func main() {
	// Make the handler available for Remote Procedure Call by Cloud Function
	cloudfunction.Start(hello)
	//fmt.Printf("111")
}