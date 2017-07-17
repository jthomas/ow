package ow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"io/ioutil"
)

type Params struct {
	Value json.RawMessage `json:"value"`
}

type ErrResponse struct {
	Error string `json:"error"`
}

type Callback func(json.RawMessage) (interface{}, error)

var action callback

func RegisterAction(cb Callback) {
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

	params := Params{}

	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error reading request body: %v", err))
		return
	}

	if err := json.Unmarshal(bodyBytes, &params); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Error unmarshaling request: %v", err))
		return
	}

	response, err := action(params.Value)

	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error executing action: %v", err))
		return
	}

	b, err := json.Marshal(response)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error marshaling response: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Set the content-length to avoid using chunked transfer encoding, which will cause OpenWhisk to return error:
	//   "error": "The action did not produce a valid response and exited unexpectedly."
	// Workaround source: https://stackoverflow.com/questions/34794647/disable-chunked-transfer-encoding-in-go-without-using-content-length
	// Related issue: https://github.com/jthomas/ow/issues/2
	w.Header().Set("Content-Length", fmt.Sprintf("%v", len(b)))

	numBytesWritten, err := w.Write(b)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error writing response: %v", err))
		return
	}

	if numBytesWritten != len(b) {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Only wrote %d of %d bytes to response", numBytesWritten, len(b)))
		return
	}

}

func setupHandlers() {
	http.HandleFunc("/init", initHandler)
	http.HandleFunc("/run", runHandler)
	http.ListenAndServe(":8080", nil)
}
