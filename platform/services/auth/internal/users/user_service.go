package users

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func (us *UserService) RegisterUser(email, password string) error {
	var count int
	err := us.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("user already exists")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	id := uuid.NewString()
	_, err = us.db.Exec("INSERT INTO users (id, email, password, roles) VALUES ($1, $2, $3, $4)",
		id, email, string(hashed), "{user}")
	if err != nil {
		return err
	}
	return nil
}

func (us *UserService) LoginUser(email, password string) (*User, error) {
	row := us.db.QueryRow("SELECT id, password, roles FROM users WHERE email = $1", email)
	var id, hashedPwd string
	var rolesStr string
	if err := row.Scan(&id, &hashedPwd, &rolesStr); err != nil {
		return nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	roles := []string{"user"}
	if strings.Contains(rolesStr, "admin") {
		roles = append(roles, "admin")
	}

	return &User{
		ID:       id,
		Email:    email,
		Password: hashedPwd,
		Roles:    roles,
	}, nil
}
