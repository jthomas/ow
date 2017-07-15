package ow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
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

	log.Printf("runHandler")

	params := Params{}

	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("read %d bytes", len(bodyBytes))


	if err := json.Unmarshal(bodyBytes, &params); err != nil {
		log.Printf("Error unmarshaling request: %v", err)
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("invoking callback with params: value: %+v", string(params.Value))

	response, err := action(params.Value)

	if err != nil {
		log.Printf("Error running action: %v", err)
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	b, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response: %+v. Err: %v", response, err)
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("writing %d byte response: %v.  response object: %+v", len(b), string(b), response)


	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%v", len(b)))

	numBytesWritten, err := w.Write(b)
	if err != nil {
		log.Printf("Error writing response:  Err: %v", err)
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if numBytesWritten != len(b) {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("numBytesWritten (%d) != len(b)", numBytesWritten))
		return
	}

	log.Printf("No errors and wrote %d bytes", numBytesWritten)

	// w.Header().Set("Content-Type", "application/json")

	// Workaround for nasty bug where OpenWhisk returns error:
	//   "error": "The action did not produce a valid response and exited unexpectedly."
	// Workaround source: https://stackoverflow.com/questions/34794647/disable-chunked-transfer-encoding-in-go-without-using-content-length
	// w.Header().Set("Transfer-Encoding", "identity")

	//enc := json.NewEncoder(w)
	//if err := enc.Encode(response); err != nil {
	//	log.Printf("Error encoding json response: %v", err)
	//	sendError(w, http.StatusInternalServerError, err.Error())
	//	return
	//}


}

func setupHandlers() {
	http.HandleFunc("/init", initHandler)
	http.HandleFunc("/run", runHandler)
	http.ListenAndServe(":8080", nil)
}
