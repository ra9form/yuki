package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/utrack/yuki/integration/one_location_with_go_package_alias/pb"
)

func main() {
	r := chi.NewMux()
	desc := pb.NewStrings().GetDescription()
	desc.RegisterHTTP(r)

	r.Handle("/swagger.json", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(desc.SwaggerDef())
	}))

	http.ListenAndServe(":8080", r)
}
