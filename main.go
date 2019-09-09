package main

//go:generate go-bindata -pkg $GOPACKAGE -o assets.go index.html application.js static/...

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	validator "gopkg.in/validator.v2"

	"github.com/Luzifer/rconfig"
)

const (
	requiredScope = "read:org,user"
	sessionName   = "vault-otp-ui"
)

var (
	cfg struct {
		Github struct {
			ClientID     string `flag:"client-id" default:"" env:"CLIENT_ID" description:"Github oAuth2 application Client ID" validate:"nonzero"`
			ClientSecret string `flag:"client-secret" default:"" env:"CLIENT_SECRET" description:"Github oAuth2 application Client Secret" validate:"nonzero"`
		}
		Listen        string `flag:"listen" default:":3000" description:"IP/Port to listen on"`
		LogLevel      string `flag:"log-level" default:"info" description:"Set log level (debug, info, warning, error)"`
		SessionSecret string `flag:"session-secret" default:"" env:"SESSION_SECRET" description:"Secret to encrypt the session with"`
		Vault         struct {
			Address     string `flag:"vault-addr" env:"VAULT_ADDR" default:"https://127.0.0.1:8200" description:"Vault API address"`
			Prefix      string `flag:"vault-prefix" env:"VAULT_PREFIX" default:"/totp" description:"Prefix to search for OTP secrets / tokens in"`
			SecretField string `flag:"vault-secret-field" env:"VAULT_SECRET_FIELD" default:"secret" description:"Field to search the secret in"`
		}
		VersionAndExit bool `flag:"version" default:"false" description:"Print version information and exit"`
	}

	version     = "dev"
	mini        = minify.New()
	cookieStore *sessions.CookieStore
)

func loadConfig() error {
	if err := rconfig.Parse(&cfg); err != nil {
		return err
	}

	if err := validator.Validate(cfg); err != nil {
		return err
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err == nil {
		log.SetLevel(l)
	} else {
		log.Fatalf("Invalid log level: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("vault-otp-ui %s\n", version)
		os.Exit(0)
	}

	if cfg.SessionSecret == "" {
		cookieStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32))
	} else {
		cookieStore = sessions.NewCookieStore([]byte(cfg.SessionSecret), []byte(fmt.Sprintf("%x", sha1.Sum([]byte(cfg.SessionSecret)))[0:32]))
	}

	mini.AddFunc("text/html", html.Minify)
	mini.AddFunc("application/javascript", js.Minify)

	return nil
}

func main() {
	var err error
	if err = loadConfig(); err != nil {
		log.Fatalf("Unable to parse CLI parameters: %s", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/oauth2", handleOAuthCallback)
	r.HandleFunc("/application.js", handleApplicationJS)
	r.HandleFunc("/vars.js", handleApplicationVars)
	r.HandleFunc("/codes.json", handleCodesJSON)
	r.PathPrefix("/static").HandlerFunc(handleStatics)
	r.HandleFunc("/", handleIndexPage)
	log.Fatalf("HTTP server exitted: %s", http.ListenAndServe(cfg.Listen, r))
}

func getFileContentFallback(filename string) (io.Reader, error) {
	if f, err := os.Open(filename); err == nil {
		defer f.Close()
		buf := new(bytes.Buffer)
		io.Copy(buf, f)
		return buf, nil
	}

	if b, err := Asset(filename); err == nil {
		return bytes.NewReader(b), nil
	}

	return nil, errors.New("No suitable index page found")
}

func handleIndexPage(res http.ResponseWriter, r *http.Request) {
	content, err := getFileContentFallback("index.html")

	if err != nil {
		http.Error(res, "No suitable index page found", http.StatusInternalServerError)
	}

	mini.Minify("text/html", res, content)
}

func handleApplicationJS(res http.ResponseWriter, r *http.Request) {
	content, err := getFileContentFallback("application.js")

	if err != nil {
		http.Error(res, "No suitable file found", http.StatusInternalServerError)
	}

	mini.Minify("application/javascript", res, content)
}

func handleApplicationVars(w http.ResponseWriter, r *http.Request) {
	sess, _ := cookieStore.Get(r, sessionName)
	_, hasAccessToken := sess.Values["access_token"]

	var buf = new(bytes.Buffer)

	fmt.Fprintf(buf, "const signedIn = %v\n", hasAccessToken)
	fmt.Fprintf(buf, "const authUrl = %q\n", getAuthenticationURL())

	mini.Minify("application/javascript", w, buf)
}

func handleOAuthCallback(res http.ResponseWriter, r *http.Request) {
	sess, _ := cookieStore.Get(r, sessionName)

	accessToken, err := getAccessToken(r.URL.Query().Get("code"))
	if err != nil {
		log.Errorf("An error occurred while fetching the access token: %s", err)
		http.Error(res, "Something went wrong when fetching your access token. Sorry.", http.StatusInternalServerError)
		return
	}

	if accessToken == "" {
		log.Errorf("Code %q was not resolved to an access token", r.URL.Query().Get("code"))
		http.Error(res, "Something went wrong when fetching your access token. Sorry.", http.StatusInternalServerError)
		return
	}

	sess.Values["access_token"] = accessToken
	if err := sess.Save(r, res); err != nil {
		log.Errorf("Was not able to set the cookie: %s", err)
		http.Error(res, "Something went wrong when fetching your access token. Sorry.", http.StatusInternalServerError)
		return
	}

	http.Redirect(res, r, "/", http.StatusFound)
}

func handleCodesJSON(res http.ResponseWriter, r *http.Request) {
	sess, _ := cookieStore.Get(r, sessionName)
	iAccessToken, hasAccessToken := sess.Values["access_token"]
	iToken := sess.Values["vault_token"]

	if !hasAccessToken {
		http.Error(res, `{"error":"Not logged in"}`, http.StatusUnauthorized)
		return
	}
	accessToken := iAccessToken.(string)

	var tok string
	if iToken != nil {
		tok = iToken.(string)
		log.WithFields(log.Fields{"token": hashSecret(tok)}).Debugf("Restored token from Session")
	}

	tok, err := useOrRenewToken(tok, accessToken)
	if err != nil {
		log.Errorf("Unable to authorize against vault: %s", err)
		http.Error(res, `{"error":"Unexpected error while fetching tokens"}`, http.StatusInternalServerError)
		return
	}
	log.WithFields(log.Fields{"token": hashSecret(tok)}).Debugf("Checked / renewed token")

	pointOfTime := time.Now()
	if r.URL.Query().Get("it") == "next" {
		pointOfTime = pointOfTime.Add(time.Duration(30-(pointOfTime.Second()%30)) * time.Second)
	}

	tokens, err := getSecretsFromVault(tok, pointOfTime)
	if err != nil {
		log.Errorf("Unable to fetch codes: %s", err)
		http.Error(res, `{"error":"Unexpected error while fetching tokens"}`, http.StatusInternalServerError)
		return
	}

	sess.Values["vault_token"] = tok
	if err := sess.Save(r, res); err != nil {
		log.Errorf("Was not able to set the cookie: %s", err)
		http.Error(res, "Something went wrong while fetching token. Sorry.", http.StatusInternalServerError)
		return
	}

	result := struct {
		Tokens   []*token  `json:"tokens"`
		NextWrap time.Time `json:"next_wrap"`
	}{
		Tokens:   tokens,
		NextWrap: pointOfTime.Add(time.Duration(30-(pointOfTime.Second()%30)) * time.Second),
	}

	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(res).Encode(result)
}

func handleStatics(res http.ResponseWriter, r *http.Request) {
	req := strings.TrimLeft(r.URL.Path, "/")
	ext := req[strings.LastIndex(req, "."):]
	t := mime.TypeByExtension(ext)

	b, err := Asset(req)
	if err != nil {
		log.WithFields(log.Fields{
			"file": req,
		}).Errorf("Static file not found")
		http.Error(res, "I don't have that.", http.StatusNotFound)
		return
	}

	res.Header().Set("Content-Type", t)
	res.Write(b)
}

func hashSecret(in string) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(in)))
}
