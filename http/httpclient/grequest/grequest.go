package grequest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var version = "0.1"

type Header map[string]string

//type Data map[string]interface{}
type Param map[string]string
type BaseAuth struct {
	UserName string
	Password string
}

type RequestOptions struct {
	Header   Header
	Data     interface{}
	Json     bool
	Form     bool
	Params   Param
	Timeout  time.Duration
	BashAuth BaseAuth
}

type responseBody struct {
	Code interface{} `json:"code"`
	Msg  string      `json:"msg"`
}

func ResponseIsError(data []byte) error {
	var body = responseBody{Code: "1"}
	err := json.Unmarshal(data, &body)
	if err != nil {
		return fmt.Errorf("响应错误，响应的数据:%s", string(data))
	}
	switch body.Code.(type) {
	case int:
		if !(body.Code == 0 || body.Code == 200) {
			err = errors.New(body.Msg)
		}
	case string:
		if !(body.Code == "0" || body.Code == "200") {
			err = errors.New(body.Msg)
		}
	default:
		err = errors.New(body.Msg)
	}
	return err
}

func Request(url, method string, options ...*RequestOptions) (resp *Response, err error) {
	var option = RequestOptions{}
	if len(options) > 0 {
		option = *options[0]
	}
	var r *http.Request
	var response Response
	var params []string
	client := http.Client{Timeout: option.Timeout}
	// 设置params
	for k, v := range option.Params {
		params = append(params, k+"="+v)
	}
	if p := strings.Join(params, "&"); p != "" {
		url = url + "?" + p
	}
	if option.Json {
		data, _ := json.Marshal(option.Data)
		body := bytes.NewReader(data)
		if r, err = http.NewRequest(method, url, body); err != nil {
			return nil, err
		}
		r.Header.Set("Content-Type", "application/json")
	} else if option.Form {
		body := new(bytes.Buffer)
		w := multipart.NewWriter(body)
		data, ok := option.Data.(map[string]interface{})
		if ok {
			for k, v := range data {
				if vv, ok := v.(string); ok {
					w.WriteField(k, vv)
				}
				if vv, ok := v.(int); ok {
					w.WriteField(k, fmt.Sprintf("%d", vv))
				}
			}
		}
		w.Close()
		if r, err = http.NewRequest(method, url, body); err != nil {
			return nil, err
		}
		r.Header.Set("Content-Type", w.FormDataContentType())
	} else {
		data, _ := json.Marshal(option.Data)
		body := bytes.NewReader(data)
		r, err = http.NewRequest(method, url, body)
	}

	if err != nil {
		return resp, err
	}
	r.Header.Set("User-Agent", "go-request"+version)
	for k, v := range option.Header {
		r.Header.Set(k, v)
	}
	if option.BashAuth.UserName != "" && option.BashAuth.Password != "" {
		r.SetBasicAuth(option.BashAuth.UserName, option.BashAuth.Password)
	}
	response.Response, err = client.Do(r)
	if err != nil {
		return nil, err
	}
	// d, err := ioutil.ReadAll(response.Response.Body)
	//if err != nil {
	//	return nil, err
	//}
	//response.Response.Body = ioutil.NopCloser(bytes.NewReader(d))
	//err = ResponseIsError(d)
	code := response.StatusCode()
	if code >= 400 || code < 200 {
		msg, _ := response.Text()
		return &response, errors.New(msg)
	}
	return &response, err
}

func Get(url string, options ...*RequestOptions) (resp *Response, err error) {
	return Request(url, "GET", options...)
}

func Post(url string, options ...*RequestOptions) (resp *Response, err error) {
	return Request(url, "POST", options...)
}

func Put(url string, options ...*RequestOptions) (resp *Response, err error) {
	return Request(url, "PUT", options...)
}

func Patch(url string, options ...*RequestOptions) (resp *Response, err error) {
	return Request(url, "PATCH", options...)
}

func Delete(url string, options ...*RequestOptions) (resp *Response, err error) {
	return Request(url, "DELETE", options...)
}

func Head(url string, options ...*RequestOptions) (resp *Response, err error) {
	return Request(url, "HEAD", options...)
}

func destValidator(dest, filename string) (destFile string) {
	if dest == "" {
		dest = filename
	} else {
		f, err := os.Stat(dest)
		if err == nil {
			if f.IsDir() {
				fmt.Println("a")
				dest = path.Join(dest, filename)
			} else {
				dest = dest + strconv.Itoa(int(time.Now().Unix()))
			}
		}
	}
	return dest
}

// dest=""时，文件下载在本地
func DownloadFile(url, dest string) error {
	var buf = make([]byte, 32*1024)
	var written int
	fileName := path.Base(url)
	req, err := http.Get(url)
	if err != nil {
		return err
	}
	defer req.Body.Close()
	if req.StatusCode != 200 {
		return fmt.Errorf("下载错误，响应码是:%d", req.StatusCode)
	}
	if req.Body == nil {
		return errors.New("下载的类容为nil")
	}
	fsize, err := strconv.ParseInt(req.Header.Get("Content-Length"), 10, 32)
	destFile := destValidator(dest, fileName)
	destFileTemp := destFile + ".tmp"
	f, err := os.Create(destFileTemp)
	if err != nil {
		return nil
	}
	defer func() {
		f.Sync()
		f.Close()
		err = os.Rename(destFileTemp, destFile)
	}()
	for {
		nr, err := req.Body.Read(buf)
		written += nr
		if err != nil {
			if err == io.EOF {
				if fsize != int64(written) {
					return errors.New("下载文件大小不一致")
				}
				return nil
			}
			return err
		}
		if nr > 0 {
			if _, er := f.Write(buf[0:nr]); er != nil {
				return er
			}
		}
	}
}

// 通过form上传文件
func UploadFile(url, filePath string) error {
	var buf = &bytes.Buffer{}
	var bodyWrite = multipart.NewWriter(buf)
	fileWriter, _ := bodyWrite.CreateFormFile("file", path.Base(filePath))
	file, _ := os.Open(filePath)
	defer file.Close()
	io.Copy(fileWriter, file)
	bodyWrite.Close()
	resp, err := http.Post(url, bodyWrite.FormDataContentType(), buf)
	if err != nil {
		return err
	}
	defer func ()  {
		_ = resp.Body.Close()
	}()  
	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return errors.New(string(respBody))
	}
	return nil
}

type Response struct {
	Response *http.Response
}

func (r *Response) Text() (string, error) {
	if r == nil {
		return "", errors.New("response响应体是空")
	}
	defer r.Response.Body.Close()
	d, err := ioutil.ReadAll(r.Response.Body)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func (r *Response) Json(data interface{}) error {
	if r == nil {
		return errors.New("response响应体是空")
	}
	defer r.Response.Body.Close()
	d, err := ioutil.ReadAll(r.Response.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(d, data)
}

func (r *Response) Close() error {
	if r == nil {
		return errors.New("response响应体是空")
	}
	return r.Response.Body.Close()
}

func (r *Response) StatusCode() int {
	return r.Response.StatusCode
}

func (r *Response) Header() http.Header {
	return r.Response.Header
}

func (r *Response) Proto() string {
	return r.Response.Proto
}

func (r *Response) Body() io.Reader {
	return r.Response.Body
}

func (r *Response) Request() *http.Request {
	return r.Response.Request
}
