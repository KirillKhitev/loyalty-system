package store

import (
	"context"
	"loyalty-system/internal/auth"
	"loyalty-system/internal/models/money"
	"loyalty-system/internal/models/orders"
	"loyalty-system/internal/models/users"
	"loyalty-system/internal/models/withdrawals"
)

type Store interface {
	GetUserByID(ctx context.Context, userID string) (users.User, error)
	GetUserByLogin(ctx context.Context, login string) (users.User, error)
	CreateUser(ctx context.Context, data auth.AuthorizingData) (users.User, error)
	AddOrderToUser(ctx context.Context, userID string, number string, status string) (orders.Order, error)
	AddWithdrawToUser(ctx context.Context, userID, number string, money money.Money) (withdrawals.Withdraw, error)
	GetWithdrawalsByUserID(ctx context.Context, userID string) ([]withdrawals.Withdraw, error)
	GetOrdersByUserID(ctx context.Context, userID string) ([]orders.Order, error)
	GetNewOrders(ctx context.Context) ([]orders.Order, error)
	ChangeStatusOrder(ctx context.Context, number string, status string) error
	UpdateStatusAndAccrualOrder(ctx context.Context, data orders.DataOrder) error
	GetBalanceByUserID(ctx context.Context, userID string) (users.Balance, error)
	Close() error
}
