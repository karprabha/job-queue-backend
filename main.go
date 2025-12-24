package main

import (
	"encoding/json"
	"net/http"
)

type StatusResponse struct {
	Status string `json:"status"`
}

func main() {
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		responseData := StatusResponse{
			Status: "ok",
		}

		jsonData, err := json.Marshal(responseData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		w.Write(jsonData)
	})
	http.ListenAndServe(":8080", nil)
}
