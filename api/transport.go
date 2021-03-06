package api

import (
	"context"
	"encoding/json"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/manager"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc manager.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	registration := kithttp.NewServer(
		registrationEndpoint(svc),
		decodeCredentials,
		encodeResponse,
		opts...,
	)

	login := kithttp.NewServer(
		loginEndpoint(svc),
		decodeCredentials,
		encodeResponse,
		opts...,
	)

	addClient := kithttp.NewServer(
		addClientEndpoint(svc),
		decodeAddClient,
		encodeResponse,
		opts...,
	)

	viewClient := kithttp.NewServer(
		viewClientEndpoint(svc),
		decodeViewClient,
		encodeResponse,
		opts...,
	)

	removeClient := kithttp.NewServer(
		removeClientEndpoint(svc),
		decodeViewClient,
		encodeResponse,
		opts...,
	)

	r := bone.New()

	r.Post("/users", registration)
	r.Post("/tokens", login)
	r.Post("/clients", addClient)
	r.Get("/clients/:id", viewClient)
	r.Delete("/clients/:id", removeClient)
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	var user manager.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, err
	}

	return user, nil
}

func decodeAddClient(_ context.Context, r *http.Request) (interface{}, error) {
	var client manager.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		return nil, err
	}

	req := addClientReq{
		key:    r.Header.Get("Authorization"),
		client: client,
	}

	return req, nil
}

func decodeViewClient(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewClientReq{
		key: r.Header.Get("Authorization"),
		id:  bone.GetValue(r, "id"),
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(apiResponse); ok {
		for k, v := range ar.headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.code())

		if ar.empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", contentType)

	switch err {
	case manager.ErrInvalidCredentials, manager.ErrMalformedClient:
		w.WriteHeader(http.StatusBadRequest)
	case manager.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case manager.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	case manager.ErrConflict:
		w.WriteHeader(http.StatusConflict)
	default:
		if _, ok := err.(*json.SyntaxError); ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}
}
