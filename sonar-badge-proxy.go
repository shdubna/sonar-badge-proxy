package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

var gitTag string

var (
	listenAddress  = flag.String("listen_address", ":8080", "Address to listen proxy requests.")
	listenEndpoint = flag.String("listen_endpoint", "/proxy/bages/measure", "Path under which proxy response to SonarQube.")
	sonarUrl       = flag.String("sonar_url", "http://127.0.0.1:9000", "SonarQube url.")
	sonarToken     = flag.String("sonar_token", "", "SonarQube token.")
	insecure       = flag.Bool("insecure", false, "Allow insecure requests.")
	proxyToken     = flag.String("proxy_token", "", "Proxy authrization token.")
	cacheExpire    = flag.Int64("cache_expire", 60, "Time to expire cached token, seconds.")
	debug          = flag.Bool("debug", false, "Enable debug logging.")
	version        = flag.Bool("version", false, "Show version number and quit.")
)

type BadgeToken struct {
	Token string
}

const (
	sonarBadgePath      = "/api/project_badges/measure"
	sonarBadgeTokenPath = "/api/project_badges/token"
)

var cacheExpre = time.Second * time.Duration(*cacheExpire)

var (
	targetUrl *url.URL
)

var tokensCache = cache.New(cacheExpre, 10*time.Minute)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(gitTag)
		os.Exit(0)
	}

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
		log.Debug("Authorization success")
	}

	if query["project"] == nil {
		log.Warn("Wrong parameters")
		writer.WriteHeader(http.StatusBadRequest)
		writer.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)
		resp["message"] = "Wrong params"
		jsonResp, _ := json.Marshal(resp)
		writer.Write(jsonResp)
		return
	}

	request.Host = targetUrl.Host
	request.URL.Path = sonarBadgePath
	projectName := query["project"][0]

	log.Debug("Get badge token for project for ", projectName)
	sonarBadgeToken, err := getSonarBadgeToken(projectName, *sonarToken)

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
	cachedToken, found := tokensCache.Get(projectName)
	if found {
		if cachedToken.(string) == "process" {
			time.Sleep(10 * time.Millisecond)
			log.Debug("Waiting for the issue token")
			return getSonarBadgeToken(projectName, sonarToken)
		}
		log.Debug("Token found in cache")
		return cachedToken.(string), nil
	} else {
		tokensCache.Set(projectName, "process", 1*time.Second)
	}

	log.Debug("Get project token from sonarqube")
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s%s?project=%s", *sonarUrl, sonarBadgeTokenPath, projectName), nil)
	req.SetBasicAuth(sonarToken, "")
	resp, err := client.Do(req)

	if err != nil {
		log.Warn("Error while getting badge token: ", err)
		tokensCache.Delete(projectName)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	var badgeTokenResponse BadgeToken
	if err := json.Unmarshal(body, &badgeTokenResponse); err != nil {
		log.Error("Can not unmarshal JSON. ", err)
	}
	tokensCache.Set(projectName, badgeTokenResponse.Token, cacheExpre)
	return badgeTokenResponse.Token, nil
}
