package main

import (
	"encoding/json"
	"fmt"
	"github.com/st3fan/moz-go-persona"
	"log"
	"net/http"
	"strings"
)

type PersonaVerifyResponse struct {
	OriginalPath string `json:"originalPath"`
}

func HandlePersonaVerify(w http.ResponseWriter, r *http.Request) {
	verifier, err := persona.NewVerifier("https://verifier.login.persona.org/verify", "https://basement.sateh.com")
	if err != nil {
		log.Print(err)
		http.Error(w, fmt.Sprintf("Cannot create persona verifier: %v\n", err), 500)
		return
	}

	assertion := r.FormValue("assertion")

	personaResponse, err := verifier.VerifyAssertion(assertion)
	if err != nil {
		log.Print(err)
		http.Error(w, fmt.Sprintf("Cannot verify persona assertion: %v\n", err), 500)
		return
	}

	if personaResponse.Status != "okay" {
		http.Error(w, fmt.Sprintf("Persona verification failed: %v\n", err), 500)
		return
	}

	if !strings.HasSuffix(personaResponse.Email, "@mozilla.com") {
		http.Error(w, fmt.Sprintf("Invalid email\n"), 500)
		return
	}

	user, err := getUserByEmail(personaResponse.Email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot load user\n"), 500)
		return
	}
	if user == nil {
		http.Error(w, fmt.Sprintf("Unknown user\n"), 500)
		return
	}

	log.Printf("User logged in %+v", user)

	session, _ := cookieStore.Get(r, "session-name")
	session.Values["email"] = personaResponse.Email
	session.Save(r, w)

	personaVerifyResponse := &PersonaVerifyResponse{OriginalPath: session.Values["originalPath"].(string)}
	encoded, err := json.Marshal(personaVerifyResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot marshal response: %s\n", err), 500)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(encoded)
}
