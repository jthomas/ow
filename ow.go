package ow

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Params struct {
	Value json.RawMessage `json:"value"`
}

type ErrResponse struct {
	Error string `json:"error"`
}

type callback func(json.RawMessage) (interface{}, error)

var action callback

func RegisterAction(cb callback) {
	action = cb
	setupHandlers()
}

func initHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
}

func sendError(w http.ResponseWriter, code int, cause string) {
	fmt.Println("action error:", cause)
	errResponse := ErrResponse{Error: cause}
	b, err := json.Marshal(errResponse)
	if err != nil {
		fmt.Println("error marshalling error response:", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	params := new(Params)

	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := action(params.Value)

	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	b, err := json.Marshal(response)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func setupHandlers() {
	http.HandleFunc("/init", initHandler)
	http.HandleFunc("/run", runHandler)
	http.ListenAndServe(":8080", nil)
}
