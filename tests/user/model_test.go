package user

import (
	"testing"
	"time"

	"github.com/charlottepl/blog-system/internal/user/model"
)

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		user model.User
		want bool
	}{
		{
			name: "Admin user",
			user: model.User{
				Role: "admin",
			},
			want: true,
		},
		{
			name: "Regular user",
			user: model.User{
				Role: "user",
			},
			want: false,
		},
		{
			name: "Empty role",
			user: model.User{
				Role: "",
			},
			want: false,
		},
		{
			name: "Editor role",
			user: model.User{
				Role: "editor",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsAdmin(); got != tt.want {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name string
		user model.User
		want bool
	}{
		{
			name: "Active user",
			user: model.User{
				Status: "active",
			},
			want: true,
		},
		{
			name: "Inactive user",
			user: model.User{
				Status: "inactive",
			},
			want: false,
		},
		{
			name: "Banned user",
			user: model.User{
				Status: "banned",
			},
			want: false,
		},
		{
			name: "Empty status",
			user: model.User{
				Status: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsActive(); got != tt.want {
				t.Errorf("User.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_CanLogin(t *testing.T) {
	tests := []struct {
		name string
		user model.User
		want bool
	}{
		{
			name: "Active user can login",
			user: model.User{
				Status: "active",
			},
			want: true,
		},
		{
			name: "Inactive user cannot login",
			user: model.User{
				Status: "inactive",
			},
			want: false,
		},
		{
			name: "Banned user cannot login",
			user: model.User{
				Status: "banned",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.CanLogin(); got != tt.want {
				t.Errorf("User.CanLogin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_GetDisplayName(t *testing.T) {
	tests := []struct {
		name string
		user model.User
		want string
	}{
		{
			name: "User with nickname",
			user: model.User{
				Username: "testuser",
				Nickname: "Test User",
			},
			want: "Test User",
		},
		{
			name: "User without nickname",
			user: model.User{
				Username: "testuser",
				Nickname: "",
			},
			want: "testuser",
		},
		{
			name: "User with empty username and nickname",
			user: model.User{
				Username: "",
				Nickname: "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.GetDisplayName(); got != tt.want {
				t.Errorf("User.GetDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_SetPassword(t *testing.T) {
	user := &model.User{
		ID:       1,
		Username: "testuser",
	}

	// 设置密码
	password := "testpassword123"
	err := user.SetPassword(password)
	if err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	// 验证密码哈希不为空
	if user.PasswordHash == "" {
		t.Error("Password hash should not be empty after setting password")
	}

	// 验证密码哈希不等于原密码
	if user.PasswordHash == password {
		t.Error("Password hash should not equal the original password")
	}

	// 验证密码检查功能
	if !user.CheckPassword(password) {
		t.Error("CheckPassword should return true for correct password")
	}

	// 验证错误密码检查
	if user.CheckPassword("wrongpassword") {
		t.Error("CheckPassword should return false for incorrect password")
	}
}

func TestUser_SetPassword_EmptyPassword(t *testing.T) {
	user := &model.User{}

	// 尝试设置空密码
	err := user.SetPassword("")
	if err == nil {
		t.Error("SetPassword should return error for empty password")
	}
}

func TestUser_ToSafeResponse(t *testing.T) {
	user := model.User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		Nickname:     "Test User",
		Avatar:       "http://example.com/avatar.jpg",
		Bio:          "Test bio",
		Role:         "user",
		Status:       "active",
		PasswordHash: "hashedpassword",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	response := user.ToSafeResponse()

	// 验证敏感信息被过滤
	if response.PasswordHash != "" {
		t.Error("Password hash should be empty in safe response")
	}

	// 验证公开信息存在
	if response.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, response.Username)
	}

	if response.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, response.Email)
	}

	if response.Nickname != user.Nickname {
		t.Errorf("Expected nickname %s, got %s", user.Nickname, response.Nickname)
	}
}

func TestNewUser(t *testing.T) {
	username := "newuser"
	email := "newuser@example.com"
	password := "password123"

	user, err := model.NewUser(username, email, password)
	if err != nil {
		t.Fatalf("NewUser failed: %v", err)
	}

	// 验证基本信息
	if user.Username != username {
		t.Errorf("Expected username %s, got %s", username, user.Username)
	}

	if user.Email != email {
		t.Errorf("Expected email %s, got %s", email, user.Email)
	}

	// 验证默认值
	if user.Role != "user" {
		t.Errorf("Expected default role 'user', got '%s'", user.Role)
	}

	if user.Status != "active" {
		t.Errorf("Expected default status 'active', got '%s'", user.Status)
	}

	// 验证密码已设置
	if user.PasswordHash == "" {
		t.Error("Password hash should not be empty for new user")
	}

	// 验证创建时间
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero for new user")
	}

	if user.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero for new user")
	}
}

func TestNewUser_Validation(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid user data",
			username: "validuser",
			email:    "valid@example.com",
			password: "validpassword123",
			wantErr:  false,
		},
		{
			name:     "Empty username",
			username: "",
			email:    "valid@example.com",
			password: "validpassword123",
			wantErr:  true,
		},
		{
			name:     "Invalid email",
			username: "validuser",
			email:    "invalid-email",
			password: "validpassword123",
			wantErr:  true,
		},
		{
			name:     "Empty password",
			username: "validuser",
			email:    "valid@example.com",
			password: "",
			wantErr:  true,
		},
		{
			name:     "Short password",
			username: "validuser",
			email:    "valid@example.com",
			password: "123",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.NewUser(tt.username, tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}