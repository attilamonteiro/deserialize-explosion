package main

import (
    "log"
    "net/http"

    "projeto0/controller"
    "projeto0/repository"
    "projeto0/service"
)

func main() {
    repo := repository.NewCacheRepo()
    svc := service.NewService(repo)
    ctrl := controller.NewController(svc)

    mux := http.NewServeMux()
    ctrl.Register(mux)

    addr := ":8080"
    log.Printf("starting server at %s", addr)
    log.Fatal(http.ListenAndServe(addr, mux))
}
