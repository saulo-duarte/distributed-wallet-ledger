package httpadapter

import "net/http"

func NewRouter(accountHandler *AccountHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /accounts", accountHandler.Create)

	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
