package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/encoding/protojson"

	emailverifier "github.com/shepard-labs/go-mailsafe"
	emailverifierv1 "github.com/shepard-labs/go-mailsafe/proto/v1"
)

var verifier = emailverifier.NewVerifier().EnableDomainSuggest()

var marshaler = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	readTimeout := getEnvDuration("READ_TIMEOUT", 5*time.Second)
	writeTimeout := getEnvDuration("WRITE_TIMEOUT", 10*time.Second)
	idleTimeout := getEnvDuration("IDLE_TIMEOUT", 120*time.Second)

	addr := fmt.Sprintf(":%s", port)

	r := chi.NewRouter()
	r.Get("/v1/{email}/verify", handleVerify)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	log.Printf("Starting server on %s (read=%s, write=%s, idle=%s)", addr, readTimeout, writeTimeout, idleTimeout)
	log.Fatal(srv.ListenAndServe())
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return fallback
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	result, err := verifier.Verify(email)
	if err != nil {
		http.Error(w, "verification failed", http.StatusInternalServerError)
		return
	}

	var mxResult *emailverifierv1.MXResult
	mx, mxErr := verifier.CheckMX(result.Syntax.Domain)
	if mxErr != nil {
		var lookupErr *emailverifier.LookupError
		if errors.As(mxErr, &lookupErr) {
			log.Printf("MX lookup failed for %s: %s (%s)", result.Syntax.Domain, lookupErr.Message, lookupErr.Details)
		}
		mxResult = &emailverifierv1.MXResult{HasMxRecord: false}
	} else {
		mxResult = toProtoMX(mx)
	}

	resp := toProtoResponse(result, mxResult)

	jsonBytes, err := marshaler.Marshal(resp)
	if err != nil {
		http.Error(w, "serialization failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonBytes)
}

func toProtoMX(mx *emailverifier.Mx) *emailverifierv1.MXResult {
	if mx == nil {
		return &emailverifierv1.MXResult{}
	}
	result := &emailverifierv1.MXResult{
		HasMxRecord: mx.HasMXRecord,
	}
	for _, r := range mx.Records {
		result.Records = append(result.Records, &emailverifierv1.MXRecord{
			Host: r.Host,
			Pref: uint32(r.Pref),
		})
	}
	return result
}

func toProtoResponse(r *emailverifier.Result, mx *emailverifierv1.MXResult) *emailverifierv1.VerifyResponse {
	return &emailverifierv1.VerifyResponse{
		Email: r.Email,
		Syntax: &emailverifierv1.Syntax{
			Username: r.Syntax.Username,
			Domain:   r.Syntax.Domain,
			Valid:    r.Syntax.Valid,
		},
		Mx:          mx,
		Disposable:  r.Disposable,
		RoleAccount: r.RoleAccount,
		Free:        r.Free,
		Suggestion:  r.Suggestion,
	}
}
