package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"loyalty-system/internal/auth"
	"loyalty-system/internal/errs"
	"loyalty-system/internal/models/money"
	"loyalty-system/internal/models/orders"
	"loyalty-system/internal/models/users"
	"loyalty-system/internal/models/withdrawals"
	accrual "loyalty-system/internal/services"
)

type Store struct {
	conn *sql.DB
}

func (s *Store) AddOrderToUser(ctx context.Context, userId string, number string, status string) (*orders.Order, error) {
	order := &orders.Order{}

	var userIdExistsOrder string

	row := s.conn.QueryRowContext(
		ctx,
		`SELECT
			user_id
		FROM
			orders
		WHERE
			number = $1`,
		number,
	)

	err := row.Scan(&userIdExistsOrder)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return order, fmt.Errorf("unable query: %w", err)
	}

	if userIdExistsOrder != "" {
		if userIdExistsOrder != userId {
			return order, errs.ErrOrderExistsByOtherUser
		}

		return order, errs.ErrOrderExistsByThisUser
	}

	order = orders.NewOrder(number, userId, status)

	_, err = s.conn.ExecContext(
		ctx,
		`INSERT INTO orders
			(id, number, user_id, status, accrual, uploaded_date)
		VALUES
			($1, $2, $3, $4, $5, $6)`,
		order.Id, order.Number, order.UserId, order.Status, order.Accrual, order.UploadedDate,
	)

	if err != nil {
		return order, err
	}

	return order, nil
}

func (s *Store) CreateUser(ctx context.Context, data auth.AuthorizingData) (*users.User, error) {
	user := data.NewUserFromData()

	_, err := s.conn.ExecContext(
		ctx,
		`INSERT INTO users
			(id, login, hash_password, registration_date)
		VALUES
			($1, $2, $3, $4)`,
		user.Id, data.Login, user.HashPassword, user.RegistrationDate,
	)

	if err != nil {
		return user, err
	}

	return user, nil
}

func (s *Store) GetUserByLogin(ctx context.Context, login string) (*users.User, error) {
	var user users.User

	row := s.conn.QueryRowContext(
		ctx,
		`SELECT
			*
		FROM
			users
		WHERE
			login = $1`,
		login,
	)

	err := row.Scan(&user.Id, &user.Login, &user.HashPassword, &user.RegistrationDate)
	if err != nil {
		return &user, err
	}

	return &user, nil
}

func (s *Store) GetUserById(ctx context.Context, userId string) (*users.User, error) {
	var user users.User

	row := s.conn.QueryRowContext(
		ctx,
		`SELECT
			*
		FROM
			users
		WHERE
			id = $1`,
		userId,
	)

	err := row.Scan(&user.Id, &user.Login, &user.HashPassword, &user.RegistrationDate)
	if err != nil {
		return &user, err
	}

	return &user, nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func NewStore(ctx context.Context, conn *sql.DB) *Store {
	store := &Store{
		conn: conn,
	}

	store.Bootstrap(ctx)

	return store
}

func (s *Store) Bootstrap(ctx context.Context) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	tx.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS users (
			id varchar(36) PRIMARY KEY,
			login varchar(255),
			hash_password varchar(255),
			registration_date timestamp
		)`,
	)
	tx.ExecContext(ctx, `CREATE UNIQUE INDEX login_idx ON users (login)`)

	tx.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS orders (
			id varchar(36) PRIMARY KEY,
			number varchar(255) NOT NULL,
			user_id varchar(36),
			status varchar(100),
			accrual bigint,
			uploaded_date timestamp
		)`,
	)
	tx.ExecContext(ctx, `CREATE INDEX user_status_idx ON orders (user_id, status)`)
	tx.ExecContext(ctx, `CREATE UNIQUE INDEX number_idx ON orders (number)`)

	tx.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			 id varchar(36) PRIMARY KEY,
			order_id varchar(36),
			sum bigint,
			processed_date timestamp
		)`,
	)
	tx.ExecContext(ctx, `CREATE INDEX order_idx ON withdrawals (order_id)`)

	return tx.Commit()
}

func (s *Store) GetOrdersByUserId(ctx context.Context, userId string) ([]orders.Order, error) {
	result := make([]orders.Order, 0)

	rows, err := s.conn.QueryContext(
		ctx,
		`SELECT
    		id,
    		user_id,
    		number,
    		status,
    		accrual,
    		uploaded_date
		FROM
			orders
		WHERE
			user_id = $1
		ORDER BY
			uploaded_date ASC`,
		userId,
	)
	if err != nil {
		return result, fmt.Errorf("unable query: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var order orders.Order

		err = rows.Scan(&order.Id, &order.UserId, &order.Number, &order.Status, &order.Accrual, &order.UploadedDate)
		if err != nil {
			return result, fmt.Errorf("unable to scan row: %w", err)
		}

		result = append(result, order)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("cursor error: %w", err)
	}

	return result, nil
}

func (s *Store) GetBalanceByUserId(ctx context.Context, userId string) (*users.Balance, error) {
	balance := &users.Balance{}

	row := s.conn.QueryRowContext(
		ctx,
		`SELECT
			COALESCE(SUM(o.accrual), 0) - COALESCE(SUM(w.sum), 0) AS current,
			COALESCE(SUM(w.sum), 0) AS withdrawn
		FROM
			orders AS o
		LEFT JOIN
			withdrawals AS w
		ON
			w.order_id = o.id
		WHERE
			o.status = 'PROCESSED' AND
			o.user_id = $1`,
		userId,
	)

	err := row.Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return balance, err
	}

	return balance, nil
}

func (s *Store) AddWithdrawToUser(ctx context.Context, userId, number string, sum money.Money) (*withdrawals.Withdraw, error) {
	order := orders.NewOrder(number, userId, orders.StatusList.Processed)
	withdraw := withdrawals.NewWithdraw(order.Id, order.Number, sum)

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return withdraw, err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO orders
			(id, number, user_id, status, accrual, uploaded_date)
		VALUES
			($1, $2, $3, $4, $5, $6)`,
		order.Id, order.Number, order.UserId, order.Status, order.Accrual, order.UploadedDate,
	)

	if err != nil {
		return withdraw, err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO withdrawals
			(id, order_id, sum, processed_date)
		VALUES
			($1, $2, $3, $4)`,
		withdraw.Id, withdraw.OrderId, withdraw.Sum, withdraw.ProcessedDate,
	)

	if err != nil {
		return withdraw, err
	}

	err = tx.Commit()

	return withdraw, err
}

func (s *Store) GetWithdrawalsByUserId(ctx context.Context, userId string) ([]*withdrawals.Withdraw, error) {
	result := make([]*withdrawals.Withdraw, 0)

	rows, err := s.conn.QueryContext(
		ctx,
		`SELECT
			w.id,
			w.order_id,
			o.number,
			w.sum,
			w.processed_date
		FROM
			withdrawals AS w
		INNER JOIN
			orders AS o
		ON
			o.id = w.order_id
		WHERE
			o.user_id = $1
		ORDER BY
			w.processed_date ASC`,
		userId,
	)
	if err != nil {
		return result, fmt.Errorf("unable query: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var withdraw withdrawals.Withdraw

		err = rows.Scan(&withdraw.Id, &withdraw.OrderId, &withdraw.Order, &withdraw.Sum, &withdraw.ProcessedDate)
		if err != nil {
			return result, fmt.Errorf("unable to scan row: %w", err)
		}

		result = append(result, &withdraw)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("cursor error: %w", err)
	}

	return result, nil
}

func (s *Store) GetNewOrders(ctx context.Context) ([]*orders.Order, error) {
	result := make([]*orders.Order, 0)

	rows, err := s.conn.QueryContext(
		ctx,
		`SELECT
			id,
    		user_id,
    		number,
    		status,
    		accrual,
    		uploaded_date
		FROM
			orders
		WHERE
			status = $1`,
		orders.StatusList.New,
	)
	if err != nil {
		return result, fmt.Errorf("unable query: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var order orders.Order

		err = rows.Scan(&order.Id, &order.UserId, &order.Number, &order.Status, &order.Accrual, &order.UploadedDate)
		if err != nil {
			return result, fmt.Errorf("unable to scan row: %w", err)
		}

		result = append(result, &order)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("cursor error: %w", err)
	}

	return result, nil
}

func (s *Store) ChangeStatusOrder(ctx context.Context, number, status string) error {
	_, err := s.conn.ExecContext(
		ctx,
		`UPDATE orders
		SET
			status = $1
		WHERE
			number = $2`,
		status, number,
	)

	return err
}

func (s *Store) UpdateStatusAndAccrualOrder(ctx context.Context, data accrual.DataOrder) error {
	_, err := s.conn.ExecContext(
		ctx,
		`UPDATE orders
		SET
			status = $1,
			accrual = $2
		WHERE
			number = $3`,
		data.Status, data.Accrual, data.Order,
	)

	return err
}
