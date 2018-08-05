package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	mgo "gopkg.in/mgo.v2"
)

const (
	DB         = "monitor"
	COLLECTION = "healthchecks"
)

type Healthcheck struct {
	BP        BloodPressure `bson:"bp" json:"bp"`
	HeartRate int           `bson:"heart_rate" json:"heart_rate"`
	Date      time.Time     `bson:"date" json:"date"`
}

type BloodPressure struct {
	Systolic  int `bson:"systolic" json:"systolic"`
	Diastolic int `bson:"diastolic" json:"diastolic"`
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func healthcheckHandler(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		var healthcheck Healthcheck
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&healthcheck)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid payload")
			return
		}

		c := session.DB(DB).C(COLLECTION)
		err = c.Insert(healthcheck)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "ops, something went wrong!")
			fmt.Println("Failed to save healthcheck: ", err)
			return
		}
		respondWithJSON(w, http.StatusCreated, healthcheck)
	}
}

func main() {
	session, err := mgo.Dial(os.Getenv("MONGO_URL"))
	if err != nil {
		panic(err)
	}

	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	r := mux.NewRouter()
	r.HandleFunc("/healthcheck", healthcheckHandler(session)).Methods("POST")
	http.ListenAndServe(":5000", r)
}
