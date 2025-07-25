package general_server

import (
	"encoding/json"
	"net/http"
)

type RawPettiEnvelope struct {
	PettiVer string `json:"PettiVer"`
}

type RouterError struct {
	Status     string         `json:"Status"`
	StatusCode int            `json:"StatusCode"`
	Payload    map[string]any `json:"Payload"`
}

func WriteRouterError(w http.ResponseWriter, e *RouterError) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.StatusCode)

	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

// func WriteRouterError(w http.ResponseWriter, e *RouterError) error {
// 	resp := RouterError{
// 		Status:     e.Status,
// 		StatusCode: e.StatusCode,
// 		Payload:    e.Payload,
// 	}
// 	b, err := json.Marshal(resp)
// 	if err != nil {
// 		return err
// 	}
// 	var formatted bytes.Buffer
// 	if err := json.Indent(&formatted, b, "", "  "); err != nil {
// 		return err
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(e.StatusCode)
// 	formatted.WriteTo(w)
// 	return nil
// }
