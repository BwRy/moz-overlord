package main

import (
	"fmt"
	"github.com/gorilla/sessions"
	"html/template"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"os"
	"time"
)

const OVERLORD_LISTEN_ADDRESS = "127.0.0.1"
const OVERLORD_LISTEN_PORT = 8123

const (
	RISK_RATING_CRITICAL = 10.0
	RISK_RATING_HIGH     = 7.0
	RISK_RATING_MODERATE = 4.0
	RISK_RATING_LOW      = 1.0
)

var cookieStore = sessions.NewCookieStore([]byte("c81f7327-1ec8-4f52-86dd-96a8b2dcd3fb"))

//

func NewSite(host string) *Site {
	return &Site{Host: host}
}

func (p *Site) CollectData() (*Results, error) {
	projectResults := &Results{Site: p.Host, Date: time.Now()}

	bugzillaResults, err := CollectBugzillaResults(p)
	if err != nil {
		return nil, err
	}
	if len(bugzillaResults) != 0 {
		projectResults.DataSourceResults = append(projectResults.DataSourceResults, bugzillaResults...)
		for _, results := range bugzillaResults {
			projectResults.Score += results.Score
		}
	}

	// minionResults, err := CollectMinionResults(p)
	// if err != nil {
	// 	return nil, err
	// }
	// if len(minionResults) != 0 {
	// 	projectResults.DataSourceResults = append(projectResults.DataSourceResults, minionResults...)
	// 	for _, results := range minionResults {
	// 		projectResults.Score += results.Score
	// 	}
	// }

	return projectResults, nil
}

func (r *Results) Persist() error {
	if r.Id != nil {
		log.Fatal("Cannot persist already stored results")
	}

	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	database := session.DB("overlord")

	// Store the results

	r.Id = bson.NewObjectId()
	err = database.C("results").Insert(r)
	if err != nil {
		return err
	}

	// Add a reference to the results to the site

	resultsReference := &ResultsReference{ResultsId: r.Id, Date: r.Date, Score: r.Score}
	change := bson.M{"$push": bson.M{"recentResults": resultsReference}}
	err = database.C("sites").Update(bson.M{"host": r.Site}, change)
	if err != nil {
		return err
	}

	return nil
}

//

func getUserByEmail(email string) (*User, error) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	database := session.DB("overlord")
	collection := database.C("users")

	user := User{}
	err = collection.Find(bson.M{"email": email}).One(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func getResultsById(id interface{}) (*Results, error) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	database := session.DB("overlord")

	results := Results{}
	err = database.C("results").FindId(id).One(&results)
	if err != nil {
		return nil, err
	}

	return &results, nil
}

func getSiteByHost(host string) (*Site, error) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	database := session.DB("overlord")
	collection := database.C("sites")

	site := Site{}
	err = collection.Find(bson.M{"host": host}).One(&site)
	if err != nil {
		return nil, err
	}

	return &site, nil
}

func getSitesInGroup(groupName string) ([]Site, error) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	database := session.DB("overlord")

	// Find the group

	group := Group{}
	err = database.C("groups").Find(bson.M{"name": groupName}).One(&group)
	if err != nil {
		return nil, err
	}

	// Grab all the sites for this group

	var sites []Site

	for _, host := range group.Sites {
		site := Site{}
		err = database.C("sites").Find(bson.M{"host": host}).One(&site)
		if err != nil {
			return nil, err
		}
		if site.Host != "" {
			sites = append(sites, site)
		}
	}

	return sites, nil
}

//

type IndexResult struct {
	Site    Site
	Results Results
}

type IndexData struct {
	Score   float64
	Results []IndexResult
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "session-name")
	if session.Values["email"] == nil {
		session.Values["originalPath"] = r.URL.Path
		session.Save(r, w)
		http.Redirect(w, r, "/overlord/login", 302)
		return
	}

	indexData := IndexData{}

	group := r.URL.Query().Get("group")
	if group == "" {
		group = "IT"
	}

	sites, err := getSitesInGroup(group)
	if err != nil {
		panic(err)
	}

	for _, site := range sites {
		if len(site.RecentResults) != 0 {
			results, _ := getResultsById(site.RecentResults[0].ResultsId)
			if results != nil {
				indexData.Score += site.RecentResults[0].Score
				indexData.Results = append(indexData.Results, IndexResult{Site: site, Results: *results})
			}
		}
	}

	w.Header().Set("Content-Type", "text/html")
	t, _ := template.ParseFiles("templates/index.html")
	t.Execute(w, indexData)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, _ := template.ParseFiles("templates/login.html")
	t.Execute(w, nil)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "session-name")
	delete(session.Values, "user")
	session.Save(r, w)
	http.Redirect(w, r, "/overlord/index?group=IT", 302)
}

//

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

//

func main() {
	if os.Args[1] == "serve" {
		http.Handle("/overlord/static/", http.StripPrefix("/overlord/static/", http.FileServer(http.Dir("static"))))
		http.HandleFunc("/overlord/index", handleIndex)
		http.HandleFunc("/overlord/login", handleLogin)
		http.HandleFunc("/overlord/logout", handleLogout)
		http.HandleFunc("/overlord/persona/verify", HandlePersonaVerify)
		addr := fmt.Sprintf("%s:%d", OVERLORD_LISTEN_ADDRESS, OVERLORD_LISTEN_PORT)
		log.Printf("Starting overlord server on %s", addr)
		err := http.ListenAndServe(addr, Log(http.DefaultServeMux))
		if err != nil {
			log.Fatal(err)
		}
	}

	if os.Args[1] == "collect" {
		host := os.Args[2]

		site, err := getSiteByHost(host)
		if err != nil {
			log.Fatal(err)
		}
		if site == nil {
			log.Fatal("Cannot find site " + host)
		}

		log.Printf("Found site %+v", site)

		results, err := site.CollectData()
		if err != nil {
			log.Fatalf("Cannot collect data for site: %v", err)
		}

		err = results.Persist()
		if err != nil {
			log.Fatalf("Cannot persist results: %v", err)
		}
	}
}
