package http

import (
	"encoding/json"
	"fmt"
	"github.com/dinp/common/model"
	"github.com/dinp/server/g"
	"log"
	"net/http"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func nodesHandler(w http.ResponseWriter, r *http.Request) {
	js, err := json.Marshal(g.Nodes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func realStateHandler(w http.ResponseWriter, r *http.Request) {
	js, err := json.Marshal(g.RealState)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func appHandler(w http.ResponseWriter, r *http.Request) {
	appName := r.URL.Path[len("/app/"):]
	if appName == "" {
		http.NotFound(w, r)
		return
	}

	safeApp, exists := g.RealState.GetSafeApp(appName)
	if !exists {
		http.NotFound(w, r)
		return
	}

	cs := safeApp.Containers()
	vs := make([]*model.Container, len(cs))
	idx := 0
	for _, v := range cs {
		vs[idx] = v
		idx++
	}

	js, err := json.Marshal(vs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func Start() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/nodes", nodesHandler)
	http.HandleFunc("/real", realStateHandler)
	http.HandleFunc("/app/", appHandler)
	addr := fmt.Sprintf("%s:%d", g.Config().Http.Addr, g.Config().Http.Port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("ListenAndServe %s fail: %s", addr, err)
	}
}
