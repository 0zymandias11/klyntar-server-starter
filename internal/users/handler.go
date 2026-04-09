package users

import (
	"context"
	"database/sql"

	userspb "example.com/klyntar-server/gen/pb/users/v1"
	"example.com/klyntar-server/pkg/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	userspb.UnimplementedUsersServiceServer
	db *sql.DB
}

func NewHandler(database *sql.DB) *Handler {
	return &Handler{db: database}
}

func (h *Handler) GetUser(ctx context.Context, req *userspb.GetUserRequest) (*userspb.User, error) {
	if req.GetKeyFingerprint() == "" {
		return nil, status.Error(codes.InvalidArgument, "key_fingerprint is required")
	}

	var user userspb.User
	err := db.WithTx(h.db, ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx,
			"SELECT id, username, email, device_mac FROM users WHERE key_fingerprint = $1",
			req.GetKeyFingerprint(),
		).Scan(&user.Id, &user.Username, &user.Email, &user.DeviceMac)
	})
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &user, nil
}

func (h *Handler) CreateUser(ctx context.Context, req *userspb.CreateUserRequest) (*userspb.CreateUserResponse, error) {
	if req.GetUsername() == "" || req.GetEmail() == "" || req.GetKeyFingerprint() == "" {
		return nil, status.Error(codes.InvalidArgument, "username, email, and key_fingerprint are required")
	}

	var id string
	err := db.WithTx(h.db, ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx,
			"INSERT INTO users (username, email, key_fingerprint, device_mac) VALUES ($1, $2, $3, $4) RETURNING id",
			req.GetUsername(), req.GetEmail(), req.GetKeyFingerprint(), req.GetDeviceMac(),
		).Scan(&id)
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &userspb.CreateUserResponse{Id: id}, nil
}
