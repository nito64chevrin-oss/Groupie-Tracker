package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Caching global data for performance
var dataCache []Complet

// Data structures for API responses
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

// Initializes the cache with API data when the server starts
func init() {
	dataCache = RecupData()
}

// Generic function to call an API and parse the response
func Api[T any](url string, t *T) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, t)
	return err
}

// Fetch and cache all data from the API
func RecupData() []Complet {
	const baseURL = "https://groupietrackers.herokuapp.com/api/artists"
	var groupes []Groupes

	// Fetch artist data
	if err := Api(baseURL, &groupes); err != nil {
		log.Fatalf("Erreur lors de la récupération des données des artistes : %v", err)
	}

	var dataComplet []Complet

	// Fetch additional data for each artist
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
	// Define routes
	http.HandleFunc("/", Handler)
	http.HandleFunc("/groupe/", PageMusicHandler)

	// Static file serving
	fs := http.FileServer(http.Dir("./statics"))
	http.Handle("/statics/", http.StripPrefix("/statics/", fs))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	// Start the server
	log.Println("Serveur démarré sur http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handler for the main page
func Handler(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	search := strings.ToLower(r.URL.Query().Get("search"))
	filtreMembre := r.URL.Query().Get("membres")
	filtreDate := r.URL.Query().Get("date")
	filtreDateAlbum := r.URL.Query().Get("date_album")

	recherche := dataCache

	// Filter by number of members
	if filtreMembre != "" {
		if nb, err := strconv.Atoi(filtreMembre); err == nil {
			var filtre []Complet
			for _, artiste := range recherche {
				if len(artiste.Artist.Membres) == nb {
					filtre = append(filtre, artiste)
				}
			}
			recherche = filtre
		}
	}

	// Filter by creation date
	if filtreDate != "" {
		if nb, err := strconv.Atoi(filtreDate); err == nil {
			var filtre []Complet
			for _, artiste := range recherche {
				if artiste.Artist.DateCreation <= nb {
					filtre = append(filtre, artiste)
				}
			}
			recherche = filtre
		}
	}

	// Filter by first album release year
	if filtreDateAlbum != "" {
		if nb, err := strconv.Atoi(filtreDateAlbum); err == nil {
			var filtre []Complet
			for _, artiste := range recherche {
				annee, _ := strconv.Atoi(artiste.Artist.FirstAlbum[len(artiste.Artist.FirstAlbum)-4:])
				if annee <= nb {
					filtre = append(filtre, artiste)
				}
			}
			recherche = filtre
		}
	}

	// Search filter
	if search != "" {
		var filtre []Complet
		for _, artiste := range recherche {
			if strings.Contains(strings.ToLower(artiste.Artist.Nom), search) {
				filtre = append(filtre, artiste)
				continue
			}
			for _, membre := range artiste.Artist.Membres {
				if strings.Contains(strings.ToLower(membre), search) {
					filtre = append(filtre, artiste)
					break
				}
			}
		}
		recherche = filtre
	}

	// Render the template
	tmpl := template.Must(template.ParseFiles("./assets/index.html"))
	if err := tmpl.Execute(w, recherche); err != nil {
		http.Error(w, "Erreur lors de l'affichage de la page", http.StatusInternalServerError)
	}
}

// Handler for artist detail pages
func PageMusicHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the artist's ID from the URL
	idStr := strings.TrimPrefix(r.URL.Path, "/groupe/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Find the corresponding artist
	var bonID *Complet
	for i := range dataCache {
		if dataCache[i].Artist.ID == id {
			bonID = &dataCache[i]
			break
		}
	}

	if bonID == nil {
		http.NotFound(w, r)
		return
	}

	// Create a custom template with functions
	funcMap := template.FuncMap{
		"formatLocation": func(location string) string {
			return strings.ReplaceAll(location, "-", ", ")
		},
		"formatDate": func(dateStr string) string {
			parts := strings.Split(dateStr, "-")
			if len(parts) == 3 {
				return fmt.Sprintf("%s/%s/%s", parts[2], parts[1], parts[0])
			}
			return dateStr
		},
		"getLocationName": func(location string) string {
			parts := strings.Split(location, "-")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
			return location
		},
		"sliceYear": func(date string) string {
			parts := strings.Split(date, "-")
			if len(parts) >= 1 {
				return parts[0]
			}
			return date
		},
		"getInitials": func(name string) string {
			if len(name) > 0 {
				return string(name[0])
			}
			return "?"
		},
	}

	// Render the artist page
	tmpl, err := template.New("page_artiste.html").Funcs(funcMap).ParseFiles("./assets/page_artiste.html")
	if err != nil {
		http.Error(w, "Erreur lors du chargement du template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, bonID); err != nil {
		http.Error(w, "Erreur lors de l'affichage de la page d'artiste", http.StatusInternalServerError)
	}
}
