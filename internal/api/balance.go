package api

import (
	"encoding/json"
	"fmt"
	"loyalty-system/internal/auth"
	"loyalty-system/internal/store"
	"net/http"
)

const BalanceErrPrefix = "Error by get balance User"

func GetBalanceUser(w http.ResponseWriter, r *http.Request, s store.Store) ResponseType {
	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: error by Authorization - %v", BalanceErrPrefix, err),
			Code:   http.StatusUnauthorized,
			Body:   "Ошибка авторизации!",
		}
	}

	balanceInfo, err := s.GetBalanceByUserID(r.Context(), userID)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s - %v", BalanceErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	response, err := json.MarshalIndent(balanceInfo, "", "    ")
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: cannot encode response JSON body - %v", BalanceErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	return ResponseType{
		Code: http.StatusOK,
		Body: string(response),
	}
}
