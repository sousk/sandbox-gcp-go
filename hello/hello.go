package hello

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	// "fmt"
	"html/template"
	"net/http"
	"time"
)

type Greeting struct {
	Author  string
	Content string
	Date    time.Time
}

var (
	// guestbookForm []byte
	// signTemplate  = template.Must(template.ParseFiles("guestbook.html"))
	guestbookTemplate = template.Must(template.ParseFiles("guestbook2.html"))
)

func guestbookKey(c appengine.Context) *datastore.Key {
	// The string "default_guestbook" here could be varied to have multiple guestbooks.
	//
	// NewKey(c appengine.Context, kind, stringID string, intID int64, parent *Key) *Key
	// New Key creates a new key. kind cannot be empty.
	// Either one or both of stringID and intID must be zero.
	// If both are zero, the key returned is incomplete.
	// parent must either be a complete key or nil
	return datastore.NewKey(c, "Guestbook", "default_guestbook", 0, nil)
}

func init() {
	/**
	content, err := ioutil.ReadFile("guestbookform.html")
	if err != nil {
		panic(err)
	}
	guestbookForm = content
	**/

	http.HandleFunc("/", root)
	http.HandleFunc("/sign", sign)
}

func root(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	// Ancestor queries are strongly consistent.
	// Queries that span entity groups are eventually consistent.
	// If we omitted the .Ancestor from this query
	// there would be a slight chance that Greeting that had just been written
	// would not show up in a query
	q := datastore.NewQuery("Greeting").Ancestor(guestbookKey(c)).Order("-Date").Limit(10)
	greetings := make([]Greeting, 0, 10)
	if _, err := q.GetAll(c, &greetings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := guestbookTemplate.Execute(w, greetings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// fmt.Fprintf(w, guestbookForm)
	// w.Write(guestbookForm)
}

func sign(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := Greeting{
		Content: r.FormValue("content"),
		Date:    time.Now(),
	}

	if u := user.Current(c); u != nil {
		g.Author = u.String()
	}

	// We set the same parent key on every Greeting entity
	// to ensure each Greeting is in the same entity group.
	// Queries across the single entity group will be consistent.
	// However, the write rate to a signle entity group should be limited to ~1/sec
	key := datastore.NewIncompleteKey(c, "Greeting", guestbookKey(c))
	_, err := datastore.Put(c, key, &g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
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
