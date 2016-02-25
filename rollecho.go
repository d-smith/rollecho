package main

import (
	"encoding/json"
	"flag"
	"fmt"
	az "github.com/xtraclabs/roll/authzwrapper"
	"github.com/xtraclabs/roll/repos"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

var jsonResponse accessTokenResponse

var templates = template.Must(template.ParseFiles("./echo.html"))

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func echoServiceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			defer r.Body.Close() //Check for resource leaks using lsof | grep xavi|wc -l
			echoTxt := r.FormValue("echo")
			log.Println("data to echo:", echoTxt)
			w.Write([]byte(echoTxt))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func readWhitelistClientIDFromEnv() string {
	return os.Getenv("ECHO_WHITELISTED_CLIENT_ID")
}

func oauthCallbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		codes := params["code"]

		resp, err := http.PostForm(os.Getenv("ROLL_ENDPOINT")+"/oauth2/token",
			url.Values{"grant_type": {"authorization_code"},
				"code": {codes[0]}, "client_id": {os.Getenv("ECHO_WHITELISTED_CLIENT_ID")},
				"client_secret": {os.Getenv("CLIENT_SECRET")},
				"redirect_uri":  {os.Getenv("REDIRECT_URI")}})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println(fmt.Sprintf("response body: %v", string(body)))

		err = json.Unmarshal(body, &jsonResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/echoclient", http.StatusFound)

		w.Write([]byte("now what?"))
	}
}

func echoClientHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":

			if err := templates.ExecuteTemplate(w, "echo.html", jsonResponse); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func main() {
	var port = flag.Int("port", -1, "Port to listen on")
	flag.Parse()
	if *port == -1 {
		fmt.Println("Must specify a -port argument")
		return
	}

	var whitelisted = readWhitelistClientIDFromEnv()

	mux := http.NewServeMux()
	mux.Handle("/echoclient", echoClientHandler())
	mux.Handle("/oauth2_callback", oauthCallbackHandler())
	mux.Handle("/echosvc", az.Wrap(repos.NewVaultSecretsRepo(), repos.NewDynamoAdminRepo(), []string{whitelisted}, echoServiceHandler()))
	http.ListenAndServe(fmt.Sprintf(":%d", *port), mux)
}
