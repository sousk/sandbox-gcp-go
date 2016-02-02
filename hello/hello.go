package hello

import (
	// "appengine"
	// "appengine/user"
	// "fmt"
	"html/template"
	"io/ioutil"
	"net/http"
)

var (
	guestbookForm []byte
	signTemplate  = template.Must(template.ParseFiles("guestbook.html"))
)

func init() {
	content, err := ioutil.ReadFile("guestbookform.html")
	if err != nil {
		panic(err)
	}
	guestbookForm = content

	http.HandleFunc("/", root)
	http.HandleFunc("/sign", sign)
}

func root(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf(w, guestbookForm)
	w.Write(guestbookForm)
}

func sign(w http.ResponseWriter, r *http.Request) {
	err := signTemplate.Execute(w, r.FormValue("content"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/**
func handler(w http.ResponseWriter, r *http.Request) {
	// Create a new context
	c := appengine.NewContext(r)
	// Get the current user
	// returns a pointer to a user.User if the user is already signed in
	u := user.Current(c)

	if u == nil {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}

	fmt.Fprintf(w, "Hello, %v !", u)
}
**/
