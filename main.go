package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
	"io/ioutil"
	"github.com/bluele/gcache"
	"flag"
)

var gitTag string

var (
	listenAddress    = flag.String("listen_address", ":8080", "Address to listen proxy requests.")
	listenEndpoint   = flag.String("listen_endpoint", "/proxy/bages/measure", "Path under which proxy response to SonarQube.")
	sonarUrl         = flag.String("sonar_url", "http://127.0.0.1:9000", "SonarQube url.")
	insecure         = flag.Bool("insecure", false, "Allow insecure requests.")
	proxyToken       = flag.String("proxy_token", "", "Proxy authrization token.")
	debug            = flag.Bool("debug", false, "Enable debug logging.")
	version          = flag.Bool("version", false, "Show version number and quit.")
)


type BadgeToken struct {
	Token string
}


const (
	sonarBadgePath      = "/api/project_badges/measure"
	sonarBadgeTokenPath = "/api/project_badges/token"
	cacheSize           = 1000
	cacheExpre          = time.Hour * 12
)

var (
    targetUrl *url.URL
)

var tokensCache gcache.Cache = gcache.New(cacheSize).LRU().Build()


func main() {
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

    var err error
    targetUrl, err = url.Parse(*sonarUrl)
    if err != nil {
		log.Fatal("Unable to parse sonar_url: ", err)
    }

    handler := http.NewServeMux()
    handler.HandleFunc(*listenEndpoint, proxyHandler)

    server := &http.Server{
        Addr:         *listenAddress,
        Handler:      handler,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  15 * time.Second,
    }
    server.SetKeepAlivesEnabled(true)
    log.Printf("Listening on %s%s", server.Addr, *listenEndpoint)
    log.Fatal(server.ListenAndServe())
}

func proxyHandler(writer http.ResponseWriter, request *http.Request) {
    http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: *insecure}
	
	log.Debug("Receive request ", request.Host, request.URL.Path)
	query := request.URL.Query()

	if *proxyToken != "" {
		if query["proxy_token"] == nil {
			log.Debug("Authorization error")
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		if *proxyToken != query["proxy_token"][0] {
			log.Debug("Authorization error")
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		log.Debug("Authorization sucess")
	}

	if query["project"] == nil || query["token"] == nil {
		log.Warn("Wrong params")
		writer.WriteHeader(http.StatusBadRequest)
		writer.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)
		resp["message"] = fmt.Sprintf("Wrong params")
		jsonResp, _ := json.Marshal(resp)
		writer.Write(jsonResp)
		return
	}


	request.Host = targetUrl.Host
	request.URL.Path = sonarBadgePath
	projectName := query["project"][0]
	sonarToken := query["token"][0]

	log.Debug("Get badge token for project for ", projectName)
	sonarBadgeToken, err := getSonarBadgeToken(projectName, sonarToken)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)
		resp["message"] = fmt.Sprintf("%s", err)
		jsonResp, _ := json.Marshal(resp)
		writer.Write(jsonResp)
		return

	}

	query.Set("token", sonarBadgeToken)
	request.URL.RawQuery = query.Encode()
	log.Debug("Proxy request to ", request.Host, request.URL.Path)
    httputil.NewSingleHostReverseProxy(targetUrl).ServeHTTP(writer, request)
}

func getSonarBadgeToken(projectName string, sonarToken string) (string, error) {
    cachedToken, err := tokensCache.Get(projectName)
	if err == nil {
		log.Debug("Token found in cache")
		return cachedToken.(string), nil
	}
	log.Debug("Query project token")
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?project=%s", *sonarUrl, sonarBadgeTokenPath, projectName), nil)
	req.SetBasicAuth(sonarToken,"")
	resp, err := client.Do(req)

	if err != nil {
		log.Warn("Error while getting badge token: ", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	var badgeTokenResponse BadgeToken
	if err := json.Unmarshal(body, &badgeTokenResponse); err != nil {
		log.Error("Can not unmarshal JSON. ", err)
	}
	tokensCache.SetWithExpire(projectName, badgeTokenResponse.Token, cacheExpre)
	return badgeTokenResponse.Token, nil
}