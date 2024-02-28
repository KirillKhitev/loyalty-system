package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"loyalty-system/internal/auth"
	"loyalty-system/internal/errs"
	"loyalty-system/internal/models/money"
	"loyalty-system/internal/models/orders"
	"loyalty-system/internal/models/users"
	"loyalty-system/internal/store"
	"net/http"
	"strconv"
)

type Handler struct {
	Store store.Store
	user  *users.User
}

type Register Handler

const RegisterErrPrefix = "Error by register new User"

func (ch *Register) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requestData := auth.AuthorizingData{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&requestData); err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable decode json - %v", RegisterErrPrefix, err),
			code:   http.StatusBadRequest,
			body:   "Ошибка в запросе",
			w:      &w,
		})
		return
	}

	if requestData.Login == "" || requestData.Password == "" {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: empty required data", RegisterErrPrefix),
			code:   http.StatusBadRequest,
			body:   "Не передали логин или пароль!",
			w:      &w,
		})
		return
	}

	existUser, errFindUser := ch.Store.GetUserByLogin(r.Context(), requestData.Login)

	if errFindUser != nil && !errors.Is(errFindUser, sql.ErrNoRows) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable find exists User - %s", RegisterErrPrefix, errFindUser),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	if existUser.Login != "" {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: user '%s' already exists!", RegisterErrPrefix, existUser.Login),
			code:   http.StatusConflict,
			body:   "Данный пользователь уже зарегистрирован!",
			w:      &w,
		})
		return
	}

	user, errCreateUser := ch.Store.CreateUser(r.Context(), requestData)
	if errCreateUser != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable create new User - %s", RegisterErrPrefix, errCreateUser),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	token, err := auth.BuildJWTString(user)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable create auth token - %s ", RegisterErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.Header().Set("Content-Type", "application/json")

	sendResponse(ResponseType{
		logMsg: fmt.Sprintf("Успешно зарегистрировали и авторизовали нового пользователя '%s'\n", user.Login),
		code:   http.StatusOK,
		body:   "Вы успешно зарегистрированы и авторизованы!",
		w:      &w,
	})
}

const LoginErrPrefix = "Error by login User"

type Login Handler

func (ch *Login) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requestData := auth.AuthorizingData{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&requestData); err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable decode json - %v", LoginErrPrefix, err),
			code:   http.StatusBadRequest,
			body:   "Ошибка в запросе",
			w:      &w,
		})
		return
	}

	if requestData.Login == "" || requestData.Password == "" {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: empty required data", LoginErrPrefix),
			code:   http.StatusBadRequest,
			body:   "Не передали логин или пароль!",
			w:      &w,
		})
		return
	}

	user, errFindUser := ch.Store.GetUserByLogin(r.Context(), requestData.Login)

	if errFindUser != nil {
		if !errors.Is(errFindUser, sql.ErrNoRows) {
			sendResponse(ResponseType{
				logMsg: fmt.Sprintf("%s: unable find User - %s", LoginErrPrefix, errFindUser),
				code:   http.StatusInternalServerError,
				w:      &w,
			})
			return
		}

		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: not find User - %s", LoginErrPrefix, requestData.Login),
			code:   http.StatusUnauthorized,
			body:   "Неправильные логин/пароль",
			w:      &w,
		})

		return
	}

	hash := requestData.GenerateHashPassword()
	if hash != user.HashPassword {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: wrong password for User - %s", LoginErrPrefix, requestData.Login),
			code:   http.StatusUnauthorized,
			body:   "Неправильные логин/пароль",
			w:      &w,
		})
		return
	}

	token, err := auth.BuildJWTString(user)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable create auth token - %s ", LoginErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.Header().Set("Content-Type", "application/json")

	sendResponse(ResponseType{
		logMsg: fmt.Sprintf("Успешно авторизовали пользователя '%s'\n", user.Login),
		code:   http.StatusOK,
		body:   "Вы успешно авторизованы!",
		w:      &w,
	})
}

type Orders Handler

func (ch *Orders) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", GetOrdersErrPrefix, err),
			code:   http.StatusUnauthorized,
			body:   "Ошибка авторизации!",
			w:      &w,
		})
		return
	}

	ch.user, err = ch.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable find User - %s", GetOrdersErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	if r.Method == http.MethodGet {
		ch.GetOrders(w, r)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ch.AddOrder(w, r)
}

const AddOrderErrPrefix = "Error by Add Order for User"

func (ch *Orders) AddOrder(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by read Request Body - %v", AddOrderErrPrefix, err),
			code:   http.StatusBadRequest,
			body:   "Ошибка в запросе",
			w:      &w,
		})
		return
	}

	orderNumberStr := string(body)
	orderNumber, err := strconv.ParseUint(orderNumberStr, 10, 64)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: wrong Order number - %s", AddOrderErrPrefix, err),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	if !orders.ValidateNumber(orderNumber) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: not valid Order number - %d", AddOrderErrPrefix, orderNumber),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	_, err = ch.Store.AddOrderToUser(r.Context(), ch.user.ID, orderNumberStr, orders.StatusList.New)

	if err != nil && errors.Is(err, errs.ErrOrderExistsByOtherUser) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: %v", GetOrdersErrPrefix, err),
			code:   http.StatusConflict,
			body:   "Номер заказа уже был загружен другим пользователем!",
			w:      &w,
		})
		return
	}

	if err != nil && errors.Is(err, errs.ErrOrderExistsByThisUser) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: %v", GetOrdersErrPrefix, err),
			code:   http.StatusOK,
			body:   "Этот заказ был уже загружен вами!",
			w:      &w,
		})
		return
	}

	sendResponse(ResponseType{
		logMsg: fmt.Sprintf("Успешно добавили заказ в систему: %s", orderNumberStr),
		code:   http.StatusAccepted,
		body:   "Заказ принят в обработку",
		w:      &w,
	})
}

const GetOrdersErrPrefix = "Error by get Orders for User"

func (ch *Orders) GetOrders(w http.ResponseWriter, r *http.Request) {
	ordersList, err := ch.Store.GetOrdersByUserID(r.Context(), ch.user.ID)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by get orders User - %v", GetOrdersErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(ordersList) == 0 {
		sendResponse(ResponseType{
			code: http.StatusOK,
			body: "[]",
			w:    &w,
		})
		return
	}

	response, err := json.MarshalIndent(ordersList, "", "    ")
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", GetOrdersErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	sendResponse(ResponseType{
		code: http.StatusOK,
		body: string(response),
		w:    &w,
	})
}

type Balance Handler

const BalanceErrPrefix = "Error by get balance User"

func (ch *Balance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", BalanceErrPrefix, err),
			code:   http.StatusUnauthorized,
			body:   "Ошибка авторизации!",
			w:      &w,
		})
		return
	}

	balanceInfo, err := ch.Store.GetBalanceByUserID(r.Context(), userID)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s - %v", BalanceErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	response, err := json.MarshalIndent(balanceInfo, "", "    ")
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", BalanceErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	sendResponse(ResponseType{
		code: http.StatusOK,
		body: string(response),
		w:    &w,
	})
}

type Withdraw Handler

type RequestForAddWithdraw = struct {
	Order string      `json:"order"`
	Sum   money.Money `json:"sum"`
}

const WithdrawErrPrefix = "Error by add Withdraw User"

func (ch *Withdraw) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", WithdrawErrPrefix, err),
			code:   http.StatusUnauthorized,
			body:   "Ошибка авторизации!",
			w:      &w,
		})
		return
	}

	var request RequestForAddWithdraw

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&request); err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: cannot decode request JSON body - %v", WithdrawErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	orderNumber, err := strconv.ParseUint(request.Order, 10, 64)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: wrong Order number - %s", WithdrawErrPrefix, err),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	if !orders.ValidateNumber(orderNumber) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: not valid Order number - %d", WithdrawErrPrefix, orderNumber),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	balance, err := ch.Store.GetBalanceByUserID(r.Context(), userID)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable get balance by User - %s", WithdrawErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	if request.Sum > balance.Current {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: wrong sum Withdraw for Order '%v'", WithdrawErrPrefix, request.Order),
			code:   http.StatusPaymentRequired,
			body:   "На счету недостаточно средств!",
			w:      &w,
		})
		return
	}

	_, err = ch.Store.AddWithdrawToUser(r.Context(), userID, request.Order, request.Sum)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: - %v", WithdrawErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	sendResponse(ResponseType{
		code: http.StatusOK,
		body: "Заказ успешно оплачен баллами!",
		w:    &w,
	})
}

type Withdrawals Handler

const WithdrawalsErrPrefix = "Error by get withdrawals User"

func (ch *Withdrawals) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", WithdrawalsErrPrefix, err),
			code:   http.StatusUnauthorized,
			body:   "Ошибка авторизации!",
			w:      &w,
		})
		return
	}

	withdrawalsList, err := ch.Store.GetWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by get withdrawals User - %v", WithdrawalsErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	if len(withdrawalsList) == 0 {
		sendResponse(ResponseType{
			code: http.StatusNoContent,
			body: "[]",
			w:    &w,
		})
		return
	}

	response, err := json.MarshalIndent(withdrawalsList, "", "    ")
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", WithdrawalsErrPrefix, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	sendResponse(ResponseType{
		code: http.StatusOK,
		body: string(response),
		w:    &w,
	})
}

type ResponseType struct {
	logMsg string
	body   string
	code   int
	w      *http.ResponseWriter
}

func sendResponse(res ResponseType) {
	if len(res.logMsg) > 0 {
		log.Println(res.logMsg)
	}

	writer := *res.w

	if res.code > 0 {
		writer.WriteHeader(res.code)
	}

	if len(res.body) > 0 {
		writer.Write([]byte(res.body))
	}
}
