package store

import (
	"context"
	"loyalty-system/internal/auth"
	"loyalty-system/internal/models/money"
	"loyalty-system/internal/models/orders"
	"loyalty-system/internal/models/users"
	"loyalty-system/internal/models/withdrawals"
	accrual "loyalty-system/internal/services"
)

type Store interface {
	GetUserById(ctx context.Context, userId string) (*users.User, error)
	GetUserByLogin(ctx context.Context, login string) (*users.User, error)
	CreateUser(ctx context.Context, data auth.AuthorizingData) (*users.User, error)
	AddOrderToUser(ctx context.Context, userId string, number string, status string) (*orders.Order, error)
	AddWithdrawToUser(ctx context.Context, userId, number string, money money.Money) (*withdrawals.Withdraw, error)
	GetWithdrawalsByUserId(ctx context.Context, userId string) ([]*withdrawals.Withdraw, error)
	GetOrdersByUserId(ctx context.Context, userId string) ([]orders.Order, error)
	GetNewOrders(ctx context.Context) ([]*orders.Order, error)
	ChangeStatusOrder(ctx context.Context, number string, status string) error
	UpdateStatusAndAccrualOrder(ctx context.Context, data accrual.DataOrder) error
	GetBalanceByUserId(ctx context.Context, userId string) (*users.Balance, error)
	Close() error
}
