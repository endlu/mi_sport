package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	http.HandleFunc("/mi_sport", miSportHandler)
	http.HandleFunc("/ip", ip)

	http.HandleFunc("/sport", sportWeb)
	http.ListenAndServe(":8088", nil)

}

func reco() {
	err := recover()
	if err != nil {
		log.Printf("panic!!, err:%s\n", err)
	}
}

func sportWeb(w http.ResponseWriter, r *http.Request) {
	defer reco()

	file, err := ioutil.ReadFile("./sport.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(file)
}

func ip(w http.ResponseWriter, r *http.Request) {
	defer reco()
	resp, err := http.DefaultClient.Get("https://ipinfo.io/ip")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	defer resp.Body.Close()
	bodyCon, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(bodyCon)
}

func miSportHandler(w http.ResponseWriter, r *http.Request) {
	defer reco()
	log.Println("receive req")
	if err := r.ParseForm(); err != nil {
		w.Write([]byte("ParseForm err:" + err.Error()))
		return
	}
	log.Printf("len of Form:%d PostForm:%d", len(r.Form), len(r.PostForm))
	tel, pwd, step := "", "", ""
	for k, v := range r.Form {
		if k == "tel" {
			tel = v[0]
		}
		if k == "pwd" {
			pwd = v[0]
		}
		if k == "step" {
			step = v[0]
		}
	}
	log.Printf("tel:%s, pwd:%s, step:%s \n", tel, pwd, step)

	if len(tel) == 0 || len(pwd) == 0 || len(step) == 0 {
		w.Write([]byte("param invalid"))
		return
	}

	code, err := getCode(tel, pwd)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	tokenInfo, err := login(code)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	stepInt, err := strconv.Atoi(step)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	if err := doChange(tokenInfo, stepInt); err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("ok"))
}

func pipe() {
	code, err := getCode("13269961236", "wl008421")
	if err != nil {
		panic(err)
	}
	tokenInfo, err := login(code)
	if err != nil {
		panic(err)
	}
	if err := doChange(tokenInfo, 3636); err != nil {
		panic(err)
	}
}

type TokenResult struct {
	TokenInfo *TokenInfo `json:"token_info"`
}

type TokenInfo struct {
	UserID     string `json:"user_id"`
	AppToken   string `json:"app_token"`
	LoginToken string `json:"login_token"`
}

type ChangeResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func doChange(tokenInfo *TokenInfo, step int) error {
	log.Printf("开始修改步数，目标：%d", step)
	bytes, err := ioutil.ReadFile("./data_json.txt")
	if err != nil {
		log.Printf("do change, read file err:%s \n", err.Error())
		return err
	}
	data := string(bytes)

	data = strings.Replace(data, "${datetime}", time.Now().Format("2006-01-02"), -1)
	data = strings.Replace(data, "${step}", strconv.Itoa(step), -1)

	req, err := http.NewRequest(http.MethodPost, "https://api-mifit-cn.huami.com/v1/data/band_data.json?&t="+strconv.FormatInt(time.Now().Unix(), 10),
		strings.NewReader("userid="+tokenInfo.UserID+"&"+
			"last_sync_data_time="+strconv.FormatInt(time.Now().Unix()-rand.Int63n(12*60*60)-12*60*60, 10)+"&"+
			"device_type=0&last_deviceid=DA932FFFFE8816E7&"+
			"data_json="+data))
	if err != nil {
		log.Printf("do change new req err:%s \n", err.Error())
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("apptoken", tokenInfo.AppToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("do change post http err:%s \n", err.Error())
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("do change read body err:%s \n", err.Error())
		return err
	}
	log.Printf("do change body: %s \n", string(body))
	var result ChangeResult
	if err = json.Unmarshal(body, &result); err != nil {
		log.Printf("do change json Unmarshal body err:%s \n", err.Error())
		return err
	}

	if result.Code == 1 && result.Message == "success" {
		log.Printf("修改步数, 成功\n")
		return nil
	}
	log.Printf("修改步数失败: %s\n", err.Error())
	return err
}

func login(code string) (*TokenInfo, error) {
	req, err := http.NewRequest(http.MethodPost, "https://account.huami.com/v2/client/login",
		strings.NewReader("app_name=com.xiaomi.hm.health&app_version=4.6.0&code="+code+"&country_code=CN&device_id=2C8B4939-0CCD-4E94-8CBA-CB8EA6E613A1&device_model=phone&grant_type=access_token&third_name=huami_phone"))
	if err != nil {
		log.Printf("login new request err:%s\n", err.Error())
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", "MiFit/4.6.0 (iPhone; iOS 14.0.1; Scale/2.00")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("login request err:%s\n", err.Error())
		return nil, err
	}
	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	var result TokenResult
	if err = json.Unmarshal(bytes, &result); err != nil {
		panic(err)
	}
	if result.TokenInfo == nil {
		log.Printf("获取login_token失败, 请检查账户名或密码是否正确\n")
		return nil, errors.New("获取login_token失败, 请检查账户名或密码是否正确")
	}
	return result.TokenInfo, nil
}

func getCode(tel, pwd string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, "https://api-user.huami.com/registrations/+86"+tel+"/tokens",
		strings.NewReader("client_id=HuaMi&password="+pwd+"&redirect_uri=https://s3-us-west-2.amazonaws.com/hm-registration/successsignin.html&token=access"))
	if err != nil {
		log.Printf("get code new request err:%s\n", err.Error())
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", "MiFit/4.6.0 (iPhone; iOS 14.0.1; Scale/2.00")

	// 禁用重定向
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Printf("get code http request err:%s\n", err.Error())
		return "", err
	}
	log.Printf("getcode, httpCode::%d, status:%s\n", response.StatusCode, response.Status)

	var code string
	location, ok := response.Header["Location"]
	if !ok {
		log.Printf("getcode 返回header没有location\n")
		return "", errors.New("get code 返回header没有location")
	}
	loList := strings.Split(location[0], "&")
	for _, lo := range loList {
		if strings.Contains(lo, "access=") {
			code = strings.Split(lo, "=")[1]
			break
		}
	}
	log.Printf("get code sucess, code::%s\n", code)
	return code, nil
}
