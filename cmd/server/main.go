package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	customHandlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
			return
		}

		urlParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/update/"), "/")
		typeName, paramName, paramValueStr := urlParts[0], urlParts[1], urlParts[2]

		fmt.Printf("Type: %s | Param: %s | Value: %s\n", typeName, paramName, paramValueStr)

	}

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: http.HandlerFunc(customHandlerFunc),
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Произошла ошибка при запуске сервера:", err)
		return
	}
}
