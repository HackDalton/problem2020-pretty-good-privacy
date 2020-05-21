package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"golang.org/x/crypto/openpgp"
)

type homePageData struct {
	ShowError bool
	Err       string
}

var tmpl = template.Must(template.ParseFiles("./public/index.html"))
var privateKey *openpgp.Entity
var flag []byte

func main() {
	keyfile, err := os.Open("privatekey.asc")
	if err != nil {
		panic(err)
	}
	keyring, err := openpgp.ReadArmoredKeyRing(keyfile)
	if err != nil {
		panic(err)
	}

	privateKey = keyring[0]

	password, ok := os.LookupEnv("KEY_PASSWORD")
	if !ok {
		panic("Missing KEY_PASSWORD enviornment variable")
	}

	privateKey.PrivateKey.Decrypt([]byte(password))

	flagFile, err := os.Open("./flag.txt")
	if err != nil {
		panic(err)
	}

	flag, err = ioutil.ReadAll(flagFile)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/sendKey", sendKey)
	http.HandleFunc("/", sendIndex)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func sendIndex(rw http.ResponseWriter, req *http.Request) {
	err := tmpl.Execute(rw, homePageData{
		ShowError: false,
		Err:       "",
	})
	if err != nil {
		panic(err)
	}
}

func getIdentities(m map[string]*openpgp.Identity) []string {
	keys := make([]string, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func sendKey(rw http.ResponseWriter, req *http.Request) {
	if req.FormValue("key") == "" {
		rw.WriteHeader(http.StatusBadRequest)
		tmpl.Execute(rw, homePageData{
			ShowError: true,
			Err:       "You must submit your key",
		})
		return
	}
	keyReader := strings.NewReader(req.FormValue("key"))
	list, err := openpgp.ReadArmoredKeyRing(keyReader)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		tmpl.Execute(rw, homePageData{
			ShowError: true,
			Err:       "There was an error reading your key",
		})
		return
	}
	key := list[0]
	if key.PrimaryKey.PublicKey == nil {
		rw.WriteHeader(http.StatusBadRequest)
		tmpl.Execute(rw, homePageData{
			ShowError: true,
			Err:       "That doesn't look like a PGP public key.",
		})
		return
	}

	rw.Header().Add("Content-Disposition", "attachment; filename=\"flag.txt.gpg\"")
	rw.Header().Add("Content-Type", "application/pgp-encrypted") // RFC3156
	w, err := openpgp.Encrypt(rw, list, privateKey, nil, nil)
	if err != nil {
		panic(err)
	}
	identities := getIdentities(key.Identities)
	fmt.Fprintf(w, `
Hey %s,

This flag was made just for you: %s

Enjoy!`, strings.Split(identities[1], " <")[0], flag)
	err = w.Close()

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		tmpl.Execute(rw, homePageData{
			ShowError: true,
			Err:       "An internal server error occured while encrypting your flag.",
		})
		return
	}

}
