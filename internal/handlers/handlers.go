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

const REGISTER_ERR_PREFIX = "Error by register new User"

func (ch *Register) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requestData := auth.AuthorizingData{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&requestData); err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable decode json - %v", REGISTER_ERR_PREFIX, err),
			code:   http.StatusBadRequest,
			body:   "Ошибка в запросе",
			w:      &w,
		})
		return
	}

	if requestData.Login == "" || requestData.Password == "" {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: empty required data", REGISTER_ERR_PREFIX),
			code:   http.StatusBadRequest,
			body:   "Не передали логин или пароль!",
			w:      &w,
		})
		return
	}

	existUser, errFindUser := ch.Store.GetUserByLogin(r.Context(), requestData.Login)

	if errFindUser != nil && !errors.Is(errFindUser, sql.ErrNoRows) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable find exists User - %s", REGISTER_ERR_PREFIX, errFindUser),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	if existUser.Login != "" {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: user '%s' already exists!", REGISTER_ERR_PREFIX, existUser.Login),
			code:   http.StatusConflict,
			body:   "Данный пользователь уже зарегистрирован!",
			w:      &w,
		})
		return
	}

	user, errCreateUser := ch.Store.CreateUser(r.Context(), requestData)
	if errCreateUser != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable create new User - %s", REGISTER_ERR_PREFIX, errCreateUser),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	token, err := auth.BuildJWTString(user)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable create auth token - %s ", REGISTER_ERR_PREFIX, err),
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

const LOGIN_ERR_PREFIX = "Error by login User"

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
			logMsg: fmt.Sprintf("%s: unable decode json - %v", LOGIN_ERR_PREFIX, err),
			code:   http.StatusBadRequest,
			body:   "Ошибка в запросе",
			w:      &w,
		})
		return
	}

	if requestData.Login == "" || requestData.Password == "" {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: empty required data", LOGIN_ERR_PREFIX),
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
				logMsg: fmt.Sprintf("%s: unable find User - %s", LOGIN_ERR_PREFIX, errFindUser),
				code:   http.StatusInternalServerError,
				w:      &w,
			})
			return
		}

		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: not find User - %s", LOGIN_ERR_PREFIX, requestData.Login),
			code:   http.StatusUnauthorized,
			body:   "Неправильные логин/пароль",
			w:      &w,
		})

		return
	}

	hash := requestData.GenerateHashPassword()
	if hash != user.HashPassword {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: wrong password for User - %s", LOGIN_ERR_PREFIX, requestData.Login),
			code:   http.StatusUnauthorized,
			body:   "Неправильные логин/пароль",
			w:      &w,
		})
		return
	}

	token, err := auth.BuildJWTString(user)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: unable create auth token - %s ", LOGIN_ERR_PREFIX, err),
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
	userId, err := auth.GetUserIdFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", GET_ORDERS_ERR_PREFIX, err),
			code:   http.StatusUnauthorized,
			body:   "Ошибка авторизации!",
			w:      &w,
		})
		return
	}

	ch.user, err = ch.Store.GetUserById(r.Context(), userId)

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

const ADD_ORDER_ERR_PREFIX = "Error by Add Order for User"

func (ch *Orders) AddOrder(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by read Request Body - %v", ADD_ORDER_ERR_PREFIX, err),
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
			logMsg: fmt.Sprintf("%s: wrong Order number - %s", ADD_ORDER_ERR_PREFIX, err),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	if !orders.ValidateNumber(orderNumber) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: not valid Order number - %d", ADD_ORDER_ERR_PREFIX, orderNumber),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	_, err = ch.Store.AddOrderToUser(r.Context(), ch.user.Id, orderNumberStr, orders.StatusList.New)

	if err != nil && errors.Is(err, errs.ErrOrderExistsByOtherUser) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: %v", GET_ORDERS_ERR_PREFIX, err),
			code:   http.StatusConflict,
			body:   "Номер заказа уже был загружен другим пользователем!",
			w:      &w,
		})
		return
	}

	if err != nil && errors.Is(err, errs.ErrOrderExistsByThisUser) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: %v", GET_ORDERS_ERR_PREFIX, err),
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

const GET_ORDERS_ERR_PREFIX = "Error by get Orders for User"

func (ch *Orders) GetOrders(w http.ResponseWriter, r *http.Request) {
	ordersList, err := ch.Store.GetOrdersByUserId(r.Context(), ch.user.Id)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by get orders User - %v", GET_ORDERS_ERR_PREFIX, err),
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
			logMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", GET_ORDERS_ERR_PREFIX, err),
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

const BALANCE_ERR_PREFIX = "Error by get balance User"

func (ch *Balance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userId, err := auth.GetUserIdFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", BALANCE_ERR_PREFIX, err),
			code:   http.StatusUnauthorized,
			body:   "Ошибка авторизации!",
			w:      &w,
		})
		return
	}

	balanceInfo, err := ch.Store.GetBalanceByUserId(r.Context(), userId)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s - %v", BALANCE_ERR_PREFIX, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	response, err := json.MarshalIndent(balanceInfo, "", "    ")
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", BALANCE_ERR_PREFIX, err),
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

const WITHDRAW_ERR_PREFIX = "Error by add Withdraw User"

func (ch *Withdraw) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userId, err := auth.GetUserIdFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", WITHDRAW_ERR_PREFIX, err),
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
			logMsg: fmt.Sprintf("%s: cannot decode request JSON body - %v", WITHDRAW_ERR_PREFIX, err),
			code:   http.StatusInternalServerError,
			w:      &w,
		})
		return
	}

	orderNumber, err := strconv.ParseUint(request.Order, 10, 64)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: wrong Order number - %s", WITHDRAW_ERR_PREFIX, err),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	if !orders.ValidateNumber(orderNumber) {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: not valid Order number - %d", WITHDRAW_ERR_PREFIX, orderNumber),
			code:   http.StatusUnprocessableEntity,
			body:   "Неверный формат номера заказа!",
			w:      &w,
		})
		return
	}

	balance, err := ch.Store.GetBalanceByUserId(r.Context(), userId)

	if request.Sum > balance.Current {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: wrong sum Withdraw for Order '%v'", WITHDRAW_ERR_PREFIX, request.Order),
			code:   http.StatusPaymentRequired,
			body:   "На счету недостаточно средств!",
			w:      &w,
		})
		return
	}

	_, err = ch.Store.AddWithdrawToUser(r.Context(), userId, request.Order, request.Sum)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: - %v", WITHDRAW_ERR_PREFIX, err),
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

const WITHDRAWALS_ERR_PREFIX = "Error by get withdrawals User"

func (ch *Withdrawals) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userId, err := auth.GetUserIdFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by Authorization - %v", WITHDRAWALS_ERR_PREFIX, err),
			code:   http.StatusUnauthorized,
			body:   "Ошибка авторизации!",
			w:      &w,
		})
		return
	}

	withdrawalsList, err := ch.Store.GetWithdrawalsByUserId(r.Context(), userId)
	if err != nil {
		sendResponse(ResponseType{
			logMsg: fmt.Sprintf("%s: error by get withdrawals User - %v", WITHDRAWALS_ERR_PREFIX, err),
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
			logMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", WITHDRAWALS_ERR_PREFIX, err),
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
