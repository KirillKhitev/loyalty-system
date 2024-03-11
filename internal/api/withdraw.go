package api

import (
	"encoding/json"
	"fmt"
	"loyalty-system/internal/auth"
	"loyalty-system/internal/models/money"
	"loyalty-system/internal/models/orders"
	"loyalty-system/internal/store"
	"net/http"
	"strconv"
)

type RequestForAddWithdraw = struct {
	Order string      `json:"order"`
	Sum   money.Money `json:"sum"`
}

const WithdrawErrPrefix = "Error by add Withdraw User"

func CreateWithdrawUser(w http.ResponseWriter, r *http.Request, s store.Store) ResponseType {
	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: error by Authorization - %v", WithdrawErrPrefix, err),
			Code:   http.StatusUnauthorized,
			Body:   "Ошибка авторизации!",
		}
	}

	var request RequestForAddWithdraw

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&request); err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: cannot decode request JSON body - %v", WithdrawErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	orderNumber, err := strconv.ParseUint(request.Order, 10, 64)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: wrong Order number - %s", WithdrawErrPrefix, err),
			Code:   http.StatusUnprocessableEntity,
			Body:   "Неверный формат номера заказа!",
		}
	}

	if !orders.ValidateNumber(orderNumber) {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: not valid Order number - %d", WithdrawErrPrefix, orderNumber),
			Code:   http.StatusUnprocessableEntity,
			Body:   "Неверный формат номера заказа!",
		}
	}

	balance, err := s.GetBalanceByUserID(r.Context(), userID)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: unable get balance by User - %s", WithdrawErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	if request.Sum > balance.Current {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: wrong sum Withdraw for Order '%v'", WithdrawErrPrefix, request.Order),
			Code:   http.StatusPaymentRequired,
			Body:   "На счету недостаточно средств!",
		}
	}

	_, err = s.AddWithdrawToUser(r.Context(), userID, request.Order, request.Sum)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s for order '%s': - %v", WithdrawErrPrefix, request.Order, err),
			Code:   http.StatusInternalServerError,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	return ResponseType{
		Code: http.StatusOK,
		Body: "Заказ успешно оплачен баллами!",
	}
}

type ResponseType struct {
	LogMsg string
	Body   string
	Code   int
}

const WithdrawalsErrPrefix = "Error by get withdrawals User"

func GetWithdrawalsUser(w http.ResponseWriter, r *http.Request, s store.Store) ResponseType {
	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: error by Authorization - %v", WithdrawalsErrPrefix, err),
			Code:   http.StatusUnauthorized,
			Body:   "Ошибка авторизации!",
		}
	}

	withdrawalsList, err := s.GetWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: error by get withdrawals User - %v", WithdrawalsErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	if len(withdrawalsList) == 0 {
		return ResponseType{
			Code: http.StatusNoContent,
			Body: "[]",
		}
	}

	response, err := json.MarshalIndent(withdrawalsList, "", "    ")
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: cannot enCode response JSON Body - %v", WithdrawalsErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	return ResponseType{
		Code: http.StatusOK,
		Body: string(response),
	}
}
