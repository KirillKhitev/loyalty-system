package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"loyalty-system/internal/errs"
	"loyalty-system/internal/models/orders"
	"loyalty-system/internal/store"
	"net/http"
	"strconv"
)

const GetOrdersErrPrefix = "Error by get Orders for User"

func GetOrders(w http.ResponseWriter, r *http.Request, userID string, s store.Store) ResponseType {
	ordersList, err := s.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: error by get orders User - %v", GetOrdersErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if len(ordersList) == 0 {
		return ResponseType{
			Code: http.StatusOK,
			Body: "[]",
		}
	}

	response, err := json.MarshalIndent(ordersList, "", "    ")
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", GetOrdersErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	return ResponseType{
		Code: http.StatusOK,
		Body: string(response),
	}
}

const AddOrderErrPrefix = "Error by Add Order for User"

func AddOrder(w http.ResponseWriter, r *http.Request, userID string, s store.Store) ResponseType {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: error by read Request Body - %v", AddOrderErrPrefix, err),
			Code:   http.StatusBadRequest,
			Body:   "Ошибка в запросе",
		}
	}

	orderNumberStr := string(body)
	orderNumber, err := strconv.ParseUint(orderNumberStr, 10, 64)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: wrong Order number - %s", AddOrderErrPrefix, err),
			Code:   http.StatusUnprocessableEntity,
			Body:   "Неверный формат номера заказа!",
		}
	}

	if !orders.ValidateNumber(orderNumber) {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: not valid Order number - %d", AddOrderErrPrefix, orderNumber),
			Code:   http.StatusUnprocessableEntity,
			Body:   "Неверный формат номера заказа!",
		}
	}

	_, err = s.AddOrderToUser(r.Context(), userID, orderNumberStr, orders.StatusList.New)

	if err != nil && errors.Is(err, errs.ErrOrderExistsByOtherUser) {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: %v", GetOrdersErrPrefix, err),
			Code:   http.StatusConflict,
			Body:   "Номер заказа уже был загружен другим пользователем!",
		}
	}

	if err != nil && errors.Is(err, errs.ErrOrderExistsByThisUser) {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: %v", GetOrdersErrPrefix, err),
			Code:   http.StatusOK,
			Body:   "Этот заказ был уже загружен вами!",
		}
	}

	return ResponseType{
		LogMsg: fmt.Sprintf("Успешно добавили заказ в систему: %s", orderNumberStr),
		Code:   http.StatusAccepted,
		Body:   "Заказ принят в обработку",
	}
}
