package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"loyalty-system/internal/auth"
	"loyalty-system/internal/errs"
	"loyalty-system/internal/store"
	"net/http"
)

func GetUserFromAuthHeader(w http.ResponseWriter, r *http.Request, s store.Store) (string, ResponseType) {
	userID, err := auth.GetUserIDFromAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return userID, ResponseType{
			LogMsg: fmt.Sprintf("error by Authorization - %v", err),
			Code:   http.StatusUnauthorized,
			Body:   "Ошибка авторизации!",
		}
	}

	user, err := s.GetUserByID(r.Context(), userID)
	if err != nil {
		return user.ID, ResponseType{
			LogMsg: fmt.Sprintf("unable find User - %s", err),
			Code:   http.StatusInternalServerError,
		}
	}

	return user.ID, ResponseType{}
}

const LoginErrPrefix = "Error by login User"

func Login(w http.ResponseWriter, r *http.Request, s store.Store) ResponseType {
	requestData := auth.AuthorizingData{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&requestData); err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: unable decode json - %v", LoginErrPrefix, err),
			Code:   http.StatusBadRequest,
			Body:   "Ошибка в запросе",
		}
	}

	if requestData.Login == "" || requestData.Password == "" {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: empty required data", LoginErrPrefix),
			Code:   http.StatusBadRequest,
			Body:   "Не передали логин или пароль!",
		}
	}

	user, errFindUser := s.GetUserByLogin(r.Context(), requestData.Login)

	if errFindUser != nil {
		if !errors.Is(errFindUser, errs.ErrNotFound) {
			return ResponseType{
				LogMsg: fmt.Sprintf("%s: unable find User - %s", LoginErrPrefix, errFindUser),
				Code:   http.StatusInternalServerError,
			}
		}

		return ResponseType{
			LogMsg: fmt.Sprintf("%s: not find User - %s", LoginErrPrefix, requestData.Login),
			Code:   http.StatusUnauthorized,
			Body:   "Неправильные логин/пароль",
		}
	}

	hash := requestData.GenerateHashPassword()
	if hash != user.HashPassword {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: wrong password for User - %s", LoginErrPrefix, requestData.Login),
			Code:   http.StatusUnauthorized,
			Body:   "Неправильные логин/пароль",
		}
	}

	token, err := auth.BuildJWTString(user)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: unable create auth token - %s ", LoginErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.Header().Set("Content-Type", "application/json")

	return ResponseType{
		LogMsg: fmt.Sprintf("Успешно авторизовали пользователя '%s'\n", user.Login),
		Code:   http.StatusOK,
		Body:   "Вы успешно авторизованы!",
	}
}

const RegisterErrPrefix = "Error by register new User"

func RegisterUser(w http.ResponseWriter, r *http.Request, s store.Store) ResponseType {
	requestData := auth.AuthorizingData{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&requestData); err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: unable decode json - %v", RegisterErrPrefix, err),
			Code:   http.StatusBadRequest,
			Body:   "Ошибка в запросе",
		}
	}

	if requestData.Login == "" || requestData.Password == "" {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: empty required data", RegisterErrPrefix),
			Code:   http.StatusBadRequest,
			Body:   "Не передали логин или пароль!",
		}
	}

	user, errCreateUser := s.CreateUser(r.Context(), requestData)
	if errCreateUser != nil {
		if errors.Is(errCreateUser, errs.ErrAlreadyExist) {
			return ResponseType{
				LogMsg: fmt.Sprintf("%s: user '%s' already exists!", RegisterErrPrefix, requestData.Login),
				Code:   http.StatusConflict,
				Body:   "Данный пользователь уже зарегистрирован!",
			}
		}

		return ResponseType{
			LogMsg: fmt.Sprintf("%s: unable create new User - %s", RegisterErrPrefix, errCreateUser),
			Code:   http.StatusInternalServerError,
		}
	}

	token, err := auth.BuildJWTString(user)
	if err != nil {
		return ResponseType{
			LogMsg: fmt.Sprintf("%s: unable create auth token - %s ", RegisterErrPrefix, err),
			Code:   http.StatusInternalServerError,
		}
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.Header().Set("Content-Type", "application/json")

	return ResponseType{
		LogMsg: fmt.Sprintf("Успешно зарегистрировали и авторизовали нового пользователя '%s'\n", user.Login),
		Code:   http.StatusOK,
		Body:   "Вы успешно зарегистрированы и авторизованы!",
	}
}
