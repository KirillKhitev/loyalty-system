package handlers

import (
	"log"
	"loyalty-system/internal/api"
	"loyalty-system/internal/store"
	"net/http"
)

type Handler struct {
	Store store.Store
}

type Register Handler

func (ch *Register) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	response := api.RegisterUser(w, r, ch.Store)

	sendResponse(response, w)
}

type Login Handler

func (ch *Login) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	response := api.Login(w, r, ch.Store)

	sendResponse(response, w)
}

type Orders Handler

func (ch *Orders) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, response := api.GetUserFromAuthHeader(w, r, ch.Store)
	if response.Code != 0 {
		sendResponse(response, w)
		return
	}

	if r.Method == http.MethodGet {
		response := api.GetOrders(w, r, userID, ch.Store)

		sendResponse(response, w)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	response = api.AddOrder(w, r, userID, ch.Store)

	sendResponse(response, w)
}

type Balance Handler

func (ch *Balance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	response := api.GetBalanceUser(w, r, ch.Store)

	sendResponse(response, w)
}

type Withdraw Handler

func (ch *Withdraw) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	response := api.CreateWithdrawUser(w, r, ch.Store)

	sendResponse(response, w)
}

type Withdrawals Handler

func (ch *Withdrawals) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	response := api.GetWithdrawalsUser(w, r, ch.Store)

	sendResponse(response, w)
}

func sendResponse(res api.ResponseType, writer http.ResponseWriter) {
	if len(res.LogMsg) > 0 {
		log.Println(res.LogMsg)
	}

	if res.Code > 0 {
		writer.WriteHeader(res.Code)
	}

	if len(res.Body) > 0 {
		writer.Write([]byte(res.Body))
	}
}
