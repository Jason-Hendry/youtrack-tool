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
)

var ytCookies []*http.Cookie;

var ytURL string;

var ytId string;

var ytSecret string;

var ytCode string;

var ytAccessToken string;

func serveIndex(w http.ResponseWriter, r *http.Request) {
	index, _ := ioutil.ReadFile("index.html")
	fmt.Fprint(w, string(index));
}
func auth(w http.ResponseWriter, r *http.Request) {
	//body,_ := ioutil.ReadAll(r.Body)
	//r.Body.Close();

	query := r.URL.Query();

	ytCode = query["code"][0];

	fmt.Printf("Auth: %v\n", ytCode)

	authData := url.Values{}
	authData.Add("grant_type", "authorization_code")
	authData.Add("code", ytCode)
	authData.Add("redirect_uri", "http://localhost:8080/")

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

	ytAccessToken = authResp.AccessToken
	ioutil.WriteFile(".session", []byte(ytAccessToken), os.FileMode(0755))
	fmt.Printf("Token: %s\n", authResp.AccessToken)

	resp.Body.Close()

	//index,_ := ioutil.ReadFile("index.html")
	//fmt.Fprint(w, string(index));
}

func findProjects(w http.ResponseWriter, r *http.Request) {

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

	r.ParseForm()

	project := r.Form.Get("project");
	tickets := strings.Split(r.Form.Get("tickets"), "\n");

	//reqBody,_ := ioutil.ReadAll(r.Body)
	for _, ticket := range tickets {
		fmt.Printf("New %s %s\n", project, ticket)
	}
	//r.Body.Close()

	if ytAccessToken == "" {
		fmt.Fprint(w, "403 Access Denied")
		return
	}

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

func main() {
	ytId = os.Getenv("YTTOOL_ID");
	ytSecret = os.Getenv("YTTOOL_SECRET");
	_, err := os.Stat(".session")
	if err == nil {
		token, _ := ioutil.ReadFile(".session")
		ytAccessToken = string(token);
	} else {
		ytAccessToken = ""
	}

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