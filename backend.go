package main

import (
	"net/http"
	"os"
	"fmt"
	"io/ioutil"
	"net/url"
	"encoding/base64"
	"encoding/json"
	"bytes"
	"encoding/xml"
	"strings"
	"math/rand"
	"github.com/mediocregopher/radix.v2/pool"
)

const SESSION_COOKIE  = "YTTOOLSESS"
const SESSION_TOKEN_FIELD  = "access_token"

var ytCookies []*http.Cookie;

var ytURL string;

var ytId string;

var ytSecret string;

var ytCode string;

var redisPool *pool.Pool

func serveIndex(w http.ResponseWriter, r *http.Request) {

	if !hasSession(r) {
		startSession(w);
		index, _ := ioutil.ReadFile("login.html")
		fmt.Fprint(w, string(index));
		return;
	}

	if sessionGet(r, SESSION_TOKEN_FIELD) == "" {
		index, _ := ioutil.ReadFile("login.html")
		fmt.Fprint(w, string(index));
	} else {
		fmt.Printf("AccessToken: %s\n", sessionGet(r, SESSION_TOKEN_FIELD))
		index, _ := ioutil.ReadFile("index.html")
		fmt.Fprint(w, string(index));
	}

}
func auth(w http.ResponseWriter, r *http.Request) {
	//body,_ := ioutil.ReadAll(r.Body)
	//r.Body.Close();
	if !hasSession(r) {
		startSession(w);
	}

	query := r.URL.Query();

	ytCode = query["code"][0];

	fmt.Printf("Auth: %v\n", ytCode)

	authData := url.Values{}
	authData.Add("grant_type", "authorization_code")
	authData.Add("code", ytCode)
	authData.Add("redirect_uri", "http://"+r.Host+":8080/")

	basicAuth := base64.StdEncoding.EncodeToString([]byte(ytId + ":" + ytSecret))

	body := bytes.NewBufferString(authData.Encode())

	httpClient := http.Client{};
	postReq, _ := http.NewRequest("POST", "https://portable.myjetbrains.com/hub/api/rest/oauth2/token", body)
	postReq.Header.Add("Accept", "application/json")
	postReq.Header.Add("Authorization", "Basic " + basicAuth)
	postReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Add("Content-Length", string(len(authData.Encode())))

	resp, _ := httpClient.Do(postReq)

	respBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Resp: %s\n", respBody)

	type AuthResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Expires     int `json:"expires_in"`
		Scope       string `json:"scope"`
	}

	authResp := new(AuthResp)

	json.Unmarshal(respBody, authResp)

	ytAccessToken := authResp.AccessToken

	sessionSet(r, SESSION_TOKEN_FIELD, ytAccessToken)

	//ioutil.WriteFile(".session", []byte(ytAccessToken), os.FileMode(0755))
	fmt.Printf("Token: %s\n", authResp.AccessToken)

	resp.Body.Close()


	w.Header().Add("Location", "/")
	w.WriteHeader(http.StatusFound);

	//index,_ := ioutil.ReadFile("index.html")
	//fmt.Fprint(w, string(index));
}

func startSession(w http.ResponseWriter) string {
	sessionKey := RandStringBytesMaskImpr(30);
	sessionCookie := http.Cookie{
		Name:SESSION_COOKIE,
		MaxAge:3600,
		Value:sessionKey,
	}
	w.Header().Add("Set-Cookie", sessionCookie.String())
	return sessionKey
}

func hasSession(r *http.Request) bool {
	cookie,err := r.Cookie(SESSION_COOKIE);
	if err != nil || cookie.Value == "" {
		return false;
	}
	return true;
}

func sessionGet(r *http.Request, field string) string {
	if hasSession(r) {
		cookie,_ := r.Cookie(SESSION_COOKIE);
		redisClient,_ := redisPool.Get()
		resp := redisClient.Cmd("HGET", cookie.Value, field)
		if resp.Err == nil {
			str,_ := resp.Str()
			return str
		}
	}
	return ""
}
func sessionSet(r *http.Request, field, value string) string {
	if hasSession(r) {
		cookie,_ := r.Cookie(SESSION_COOKIE);
		redisClient,_ := redisPool.Get()
		fmt.Printf("\nSave cookie value: %s:%s = %s\n\n", cookie.Value, field, value)
		resp := redisClient.Cmd("HSET", cookie.Value, field, value)
		if resp.Err == nil {
			return resp.String()
		}
	}
	return ""
}

func findProjects(w http.ResponseWriter, r *http.Request) {


	if !hasSession(r) {
		fmt.Fprint(w, "403 Access Denied")
		return
	}
	ytAccessToken := sessionGet(r, SESSION_TOKEN_FIELD)
	if ytAccessToken == "" {
		fmt.Fprint(w, "403 Access Denied")
		return
	}

	httpClient := http.Client{};

	findReq, _ := http.NewRequest("GET", ytURL + "/rest/project/all", nil)
	findReq.Header.Add("Authorization", "Bearer " + ytAccessToken)

	resp, err := httpClient.Do(findReq);
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return;
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	p := new(Projects)

	err = xml.Unmarshal(respBody, p)

	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		return
	}

	jsonResp, _ := json.Marshal(p.Project);

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonResp))
}

func createTickets(w http.ResponseWriter, r *http.Request) {

	if !hasSession(r) {
		fmt.Fprint(w, "403 Access Denied")
		return
	}
	ytAccessToken := sessionGet(r, SESSION_TOKEN_FIELD)
	if ytAccessToken == "" {
		fmt.Fprint(w, "403 Access Denied")
		return
	}

	r.ParseForm()

	project := r.Form.Get("project");
	tickets := strings.Split(r.Form.Get("tickets"), "\n");

	//reqBody,_ := ioutil.ReadAll(r.Body)
	for _, ticket := range tickets {
		fmt.Printf("New %s %s\n", project, ticket)
	}
	//r.Body.Close()

	httpClient := http.Client{};

	for _, ticket := range tickets {

		ticketData := url.Values{}
		ticketData.Add("project", project)
		ticketData.Add("summary", ticket)

		findReq, _ := http.NewRequest("PUT", ytURL + "/rest/issue?"+ticketData.Encode(), nil)
		findReq.Header.Add("Authorization", "Bearer " + ytAccessToken)

		resp, err := httpClient.Do(findReq);
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return;
		}
		respBody, _ := ioutil.ReadAll(resp.Body)
		fmt.Fprint(w, string(respBody))
		resp.Body.Close()
	}

}

type Project struct {
	Name string `xml:"name,attr" json:"name"`
	Code string `xml:"shortName,attr" json:"code"`
}

type Projects struct {
	XMLName xml.Name `xml:"projects`
	Project []Project `xml:"project"`
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytesMaskImpr(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func main() {

	redisPool,_ = pool.New("tcp",os.Getenv("YTTOOL_REDIS"), 5);

	ytId = os.Getenv("YTTOOL_ID");
	ytSecret = os.Getenv("YTTOOL_SECRET");

	port := os.Getenv("YTTOOL_PORT")
	ytURL = os.Getenv("YTTOOL_URL")
	if port == "" {
		port = "8080"
	}
	if ytURL == "" {
		ytURL = "https://portable.myjetbrains.com/youtrack"
	}
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/projects", findProjects)
	http.HandleFunc("/auth", auth)
	http.HandleFunc("/create", createTickets)
	fmt.Printf("Listing in 0.0.0.0:%s\n", port)
	http.ListenAndServe("0.0.0.0:" + port, nil)
}