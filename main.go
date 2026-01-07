package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

var dataCache []Complet

type Complet struct {
	Artist       Groupes
	Locations    LocationsData
	ConcertDates ConcertDatesData
	Relations    RelationsData
}
type Groupes struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Nom          string   `json:"name"`
	Membres      []string `json:"members"`
	DateCreation int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Locations    string   `json:"locations"`
	DatesConcert string   `json:"concertDates"`
	Relations    string   `json:"relations"`
}

type RelationsData struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

type ConcertDatesData struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

type LocationsData struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
}

func init() {
	dataCache = RecupData()
}

func Api[T any](url string, t *T) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, t)
	return err
}

func RecupData() []Complet {
	url := "https://groupietrackers.herokuapp.com/api/artists"
	var groupes []Groupes
	Api(url, &groupes)

	var dataComplet []Complet

	for _, a := range groupes {

		var loc LocationsData
		var dates ConcertDatesData
		var rel RelationsData

		Api(a.Locations, &loc)
		Api(a.DatesConcert, &dates)
		Api(a.Relations, &rel)

		dataComplet = append(dataComplet, Complet{
			Artist:       a,
			Locations:    loc,
			ConcertDates: dates,
			Relations:    rel,
		})
	}
	return dataComplet
}

func main() {
	http.HandleFunc("/", Handler)
	http.HandleFunc("/groupe/", PageMusicHandler)

	fs := http.FileServer(http.Dir("./statics"))
	http.Handle("/statics/", http.StripPrefix("/statics/", fs))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	http.ListenAndServe(":8080", nil)

}

// Page d'accueil
func Handler(w http.ResponseWriter, r *http.Request) {

	search := r.URL.Query().Get("search")
	filtreMembre := r.URL.Query().Get("membres")
	filtreDate := r.URL.Query().Get("date")
	filtreDateAlbum := r.URL.Query().Get("date_album")

	dataComplet := dataCache

	recherche := dataComplet

	//filtres
	if filtreMembre != "" {
		nb, _ := strconv.Atoi(filtreMembre)
		var filtre []Complet
		for _, artiste := range recherche {
			if len(artiste.Artist.Membres) == nb {
				filtre = append(filtre, artiste)
			}
		}
		recherche = filtre
	}

	if filtreDate != "2015" && filtreDate != "" {
		nb, _ := strconv.Atoi(filtreDate)
		var filtre []Complet
		for _, artiste := range recherche {
			if artiste.Artist.DateCreation <= nb {
				filtre = append(filtre, artiste)
			}
		}
		recherche = filtre
	}

	if filtreDateAlbum != "2018" && filtreDateAlbum != "" {
		nb, _ := strconv.Atoi(filtreDateAlbum)
		var filtre []Complet
		for _, artiste := range recherche {
			annee, _ := strconv.Atoi(artiste.Artist.FirstAlbum[len(artiste.Artist.FirstAlbum)-4:])
			if annee <= nb {
				filtre = append(filtre, artiste)
			}
		}
		recherche = filtre
	}

	//Barre de recherche
	if search != "" {
		search = strings.ToLower(search)
		var filtre []Complet
		for _, artiste := range recherche {
			if strings.HasPrefix(strings.ToLower(artiste.Artist.Nom), search) {
				filtre = append(filtre, artiste)
				continue
			}
			for _, membre := range artiste.Artist.Membres {
				if strings.HasPrefix(strings.ToLower(membre), search) {
					filtre = append(filtre, artiste)
					break
				}
			}
		}
		recherche = filtre
	}

	tmpl := template.Must(template.ParseFiles("./assets/index.html"))
	tmpl.Execute(w, recherche)
}

// Page d'affichage par groupes
func PageMusicHandler(w http.ResponseWriter, r *http.Request) {

	idStr := r.URL.Path[len("/groupe/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dataComplet := dataCache

	var bon_id *Complet
	for i := range dataComplet {
		if dataComplet[i].Artist.ID == id {
			bon_id = &dataComplet[i]
			break
		}
	}

	if bon_id == nil {
		http.NotFound(w, r)
		return
	}

	tmpl := template.Must(template.ParseFiles("./asstets/page_artiste.html"))
	tmpl.Execute(w, bon_id)
}
