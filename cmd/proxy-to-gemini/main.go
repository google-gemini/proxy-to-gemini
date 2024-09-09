// Copyright 2024 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google-gemini/proxy-to-gemini/ollama"
	"github.com/google-gemini/proxy-to-gemini/openai"
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

var (
	apikey   string
	hostport string
	api      string
)

func main() {
	ctx := context.Background()

	flag.StringVar(&hostport, "listen", ":5555", "host and port to listen on")
	flag.StringVar(&api, "api", "openai", "API proxocol; openai or ollama")
	flag.Parse()

	apikey = os.Getenv("GEMINI_API_KEY")
	if apikey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(apikey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	r := mux.NewRouter()
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	switch api {
	case "openai":
		openai.RegisterHandlers(r, client)
	case "ollama":
		ollama.RegisterHandlers(r, client)
	}
	r.HandleFunc("/", indexHandler)

	log.Printf("Starting server on %v", hostport)
	if err := http.ListenAndServe(hostport, r); err != nil {
		log.Printf("Error starting server: %v", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "You are running proxy-to-gemini at %q; api = %q", hostport, api)
}
