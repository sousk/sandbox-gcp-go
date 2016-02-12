package wifiSearch

import (
	"appengine"
	"appengine/datastore"
	"appengine/search"
	"encoding/csv"
	"fmt"
	// "golang.org/x/net/context"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	//	"strings"
)

const (
	SEARCH_AREA_DISTANCE = 50000 // 50km

	ENTITY_KIND       = "WifiSpot"
	LISTING_NUM       = 20
	CSV_SRC_PATH      = "./data/jta_free_wifi.csv"
	CSV_INDEX_NAME_JA = 1
	CSV_INDEX_ADDR    = 8
	CSV_INDEX_LAT     = 18
	CSV_INDEX_LONG    = 19
)

var (
	searchTemplate = template.Must(template.ParseFiles("templates/search.html"))
)

type WifiSpot struct {
	Name    string
	Address string
	Coords  appengine.GeoPoint
}

type WifiSpotRepository struct {
	Context appengine.Context
	Index   *search.Index
}

func (r *WifiSpotRepository) GetIndex() *search.Index {
	// if r.Index == nil {
	index, err := search.Open("wifi_spots")
	if err != nil {
		panic(err.Error())
	}
	// r.Index = index
	return index
	/**
	} else {
		return r.Index
	}
	**/
}
func (r *WifiSpotRepository) Truncate() error {
	c := r.Context
	// q := datastore.NewQuery(ENTITY_KIND).Limit(10000).KeysOnly()
	q := datastore.NewQuery(ENTITY_KIND).KeysOnly()
	keys, err := q.GetAll(c, nil)
	if err != nil {
		return err
	} else {
		delerr := datastore.DeleteMulti(c, keys)
		return delerr
	}
}
func (r *WifiSpotRepository) TruncateIndex() error {
	index := r.GetIndex()
	c := r.Context
	for t := index.List(c, nil); ; {
		var ws WifiSpot
		id, err := t.Next(&ws)
		if err == search.Done {
			break
		}
		if err != nil {
			return err
		}

		if err := index.Delete(c, id); err != nil {
			return err
		} else {
			log.Println("Deleted ", id)
		}
	}
	return nil
}
func (r *WifiSpotRepository) Put(v WifiSpot) error {
	key := datastore.NewIncompleteKey(r.Context, ENTITY_KIND, nil)
	_, err := datastore.Put(r.Context, key, &v)
	return err
}
func (r *WifiSpotRepository) PutIndex(v WifiSpot) error {
	key, err := r.GetIndex().Put(r.Context, "", &v)
	log.Println("KEY: %v", key)
	return err
}

// func (r *WifiSpotRepository) SearchBy(coords appengine.GeoPoint, keyword string) []WifiSpot {
func (r *WifiSpotRepository) SearchBy(lat string, lng string, keyword string) []WifiSpot {
	c := r.Context
	index := r.GetIndex()

	// Restrict to 10km
	distance := "distance(Coords, geopoint(" + lat + "," + lng + "))"
	query := fmt.Sprintf("%s < %d", distance, SEARCH_AREA_DISTANCE)
	// see
	// https://cloud.google.com/appengine/docs/go/search/reference#SearchOptions
	sortopt := search.SortOptions{Expressions: []search.SortExpression{search.SortExpression{Expr: distance, Reverse: true}}}
	opt := search.SearchOptions{Sort: &sortopt}

	res := make([]WifiSpot, 0, LISTING_NUM)
	for t := index.Search(c, query, &opt); ; {
		doc := WifiSpot{}
		id, err := t.Next(&doc)
		if err == search.Done {
			log.Println("done !!")
			break
		}
		if err != nil {
			panic(err.Error())
			break
		}
		log.Printf("id: %v\n", id)
		log.Printf("loc: %v\n", doc.Coords)
		res = append(res, doc)
	}

	/**
	res := make([]WifiSpot, 0, LISTING_NUM)
	q := datastore.NewQuery(ENTITY_KIND).Limit(LISTING_NUM)
	q = q.Filter(filter)
	// q = q.Order()
	_, err := q.GetAll(r.Context, &res)
	if err != nil {
		panic(err.Error())
	}
	**/

	return res
}

func init() {
	http.HandleFunc("/", doSearch)
	http.HandleFunc("/setup", setupData)
}

func str2coords(latstr string, longstr string) appengine.GeoPoint {
	lat, _ := strconv.ParseFloat(latstr, 64)
	long, _ := strconv.ParseFloat(longstr, 64)
	return appengine.GeoPoint{Lat: lat, Lng: long}
}

func doSearch(w http.ResponseWriter, r *http.Request) {
	var spots []WifiSpot

	lat := r.FormValue("lat")
	long := r.FormValue("long")
	if lat != "" && long != "" {
		log.Printf("got params: %v -- %v\n", lat, long)
		repo := WifiSpotRepository{Context: appengine.NewContext(r)}
		// spots = repo.SearchBy(str2coords(lat, long), r.FormValue("keyword"))
		spots = repo.SearchBy(lat, long, r.FormValue("keyword"))
	}
	if err := searchTemplate.Execute(w, spots); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func setupData(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	repo := WifiSpotRepository{Context: c}

	if err := repo.Truncate(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := repo.TruncateIndex(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	csv := loadCSV()
	for _, dat := range csv {
		ws := WifiSpot{
			Name:    dat[CSV_INDEX_NAME_JA],
			Address: dat[CSV_INDEX_ADDR],
			Coords:  str2coords(dat[CSV_INDEX_LAT], dat[CSV_INDEX_LONG]),
		}
		// TBD: change to batch writing
		if err := repo.Put(ws); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if err := repo.PutIndex(ws); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	}
	log.Println("--> done.")
	fmt.Fprintf(w, "Done. The data repository has been refreshed.")
}

func loadCSV() [][]string {
	buf := [][]string{}
	file, err := os.Open(CSV_SRC_PATH)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		buf = append(buf, line)
		// log.Println(line[0])
	}

	return buf
}
