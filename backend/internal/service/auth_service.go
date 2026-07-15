package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fitmind/backend/pkg/conf"
	"fitmind/backend/pkg/header/dto"
	"fitmind/backend/pkg/manager"
	"fitmind/backend/pkg/model"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type ErrorKind string

const (
	KindInvalidInput      ErrorKind = "invalid input"
	KindConflict          ErrorKind = "conflict"
	KindUnauthorized      ErrorKind = "unauthorized"
	KindForbidden         ErrorKind = "forbidden"
	KindNotFound          ErrorKind = "not_found"
	KindInternal          ErrorKind = "internal error"
	defaultJWTIssurer               = "fitmind"
	defaultJWTSecret                = "fitmind"
	defaultAccessTokenTTL           = 24 * time.Hour
)

type ServiceError struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (err *ServiceError) Error() string {
	if err.Err != nil {
		return err.Message
	}
	return err.Message + ": " + err.Err.Error()
}

func (err *ServiceError) Unwrap() error {
	return err.Err
}

type AuthService struct {
	users *manager.UserManager
	auth  conf.AuthConfig
	now   func() time.Time
}

type accessTokenClaims struct {
	UserID    string `json:"user_id"`
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

func NewAuthService(users *manager.UserManager, auth conf.AuthConfig) *AuthService {
	if auth.JWTSecret == "" {
		auth.JWTSecret = defaultJWTSecret
	}
	if auth.JWTIssuer == "" {
		auth.JWTIssuer = defaultJWTIssurer
	}
	if auth.AccessTokenTTL == 0 {
		auth.AccessTokenTTL = defaultAccessTokenTTL
	}

	return &AuthService{
		users: users,
		auth:  auth,
		now:   time.Now,
	}
}

func (service *AuthService) Register(ctx context.Context, request dto.RegisterRequest) (*dto.AuthResponse, error) {
	email, err := normalizeEmail(request.Email)
	if err != nil {
		return nil, newServiceError(KindInvalidInput, err.Error(), err)
	}
	if err = validatePassword(request.Password); err != nil {
		return nil, newServiceError(KindInvalidInput, err.Error(), err)
	}

	nickname, err := normalizeNickname(request.Nickname, email)
	if err != nil {
		return nil, newServiceError(KindInvalidInput, err.Error(), err)
	}

	_, err = service.users.FindByEmail(ctx, email)
	if err == nil {
		return nil, newServiceError(KindConflict, "邮箱已注册", err)
	}
	if !errors.Is(err, manager.ErrUserNotFound) {
		return nil, newServiceError(KindInternal, "检查邮箱失败", err)

	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, newServiceError(KindInternal, "生成密码哈希失败", err)
	}

	user, err := service.users.CreateUser(ctx, manager.CreateUserParams{
		Email:        email,
		PasswordHash: string(passwordHash),
		Nickname:     nickname,
	})
	if manager.IsUniqueViolation(err) {
		return nil, newServiceError(KindConflict, "邮箱已注册", err)
	}
	if err != nil {
		return nil, newServiceError(KindInternal, "创建用户失败", err)
	}

	return service.buildAuthResponse(user)
}

func (service *AuthService) Login(ctx context.Context, request dto.LoginRequest) (*dto.AuthResponse, error) {
	email, err := normalizeEmail(request.Email)
	if err != nil {
		return nil, newServiceError(KindUnauthorized, "邮箱或密码错误", nil)
	}

	user, err := service.users.FindByEmail(ctx, email)
	if errors.Is(err, manager.ErrUserNotFound) {
		return nil, newServiceError(KindUnauthorized, "邮箱或密码错误", nil)
	}
	if err != nil {
		return nil, newServiceError(KindInternal, "查询用户失败", nil)
	}

	if user.Status != "active" {
		return nil, newServiceError(KindForbidden, "账号不可用", nil)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	if err != nil {
		return nil, newServiceError(KindUnauthorized, "邮箱或密码错误", nil)
	}

	loginAt := service.now()
	if err := service.users.UpdateLastLoginAt(ctx, user.ID, loginAt); err != nil {
		return nil, newServiceError(KindInternal, "更新登录时间失败", err)
	}
	user.LastLoginAt = &loginAt

	return service.buildAuthResponse(user)
}

func (service *AuthService) CurrentUser(ctx context.Context, userID string) (*dto.UserResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, newServiceError(KindUnauthorized, "未登录", nil)
	}

	user, err := service.users.FindByID(ctx, userID)
	if errors.Is(err, manager.ErrUserNotFound) {
		return nil, newServiceError(KindNotFound, "用户不存在", nil)
	}
	if err != nil {
		return nil, newServiceError(KindInternal, "查询用户失败", err)
	}
	if user.Status != "active" {
		return nil, newServiceError(KindForbidden, "账号不可用", nil)
	}

	response := toUserResponse(user)
	return &response, nil
}

func (service *AuthService) ParseAccessToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", newServiceError(KindUnauthorized, "访问令牌格式错误", nil)
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := service.sign([]byte(signingInput))

	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(actualSignature, expectedSignature) {
		return "", newServiceError(KindUnauthorized, "访问令牌无效", err)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", newServiceError(KindUnauthorized, "访问令牌载荷无效", err)
	}

	var claims accessTokenClaims
	if err = json.Unmarshal(payload, &claims); err != nil {
		return "", newServiceError(KindUnauthorized, "访问令牌载荷无效", err)
	}

	if claims.Issuer != service.auth.JWTIssuer {
		return "", newServiceError(KindUnauthorized, "访问令牌签发者无效", err)
	}
	if claims.UserID == "" || claims.Subject != claims.UserID {
		return "", newServiceError(KindUnauthorized, "访问令牌用户无效", err)
	}
	if claims.ExpiresAt <= service.now().Unix() {
		return "", newServiceError(KindUnauthorized, "访问令牌已过期", err)
	}

	return claims.UserID, nil
}

func KindOf(err error) ErrorKind {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.Kind
	}
	return KindInternal
}

func MessageOf(err error) string {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.Message
	}
	return "服务器内部错误"
}

func (service *AuthService) buildAuthResponse(user *model.User) (*dto.AuthResponse, error) {
	issuedAt := service.now()
	expiresAt := issuedAt.Add(service.auth.AccessTokenTTL)

	claims := accessTokenClaims{
		UserID:    user.ID,
		Issuer:    service.auth.JWTIssuer,
		Subject:   user.ID,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  issuedAt.Unix(),
	}

	token, err := service.signClaims(claims)
	if err != nil {
		return nil, newServiceError(KindInternal, "签发访问令牌失败", err)
	}

	return &dto.AuthResponse{
		User:        toUserResponse(user),
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(service.auth.AccessTokenTTL.Seconds()),
	}, nil
}

func (service *AuthService) signClaims(claims accessTokenClaims) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := header + "." + payload
	signature := base64.RawURLEncoding.EncodeToString(service.sign([]byte(signingInput)))

	return signingInput + "." + signature, nil
}

func (service *AuthService) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, []byte(service.auth.JWTSecret))
	mac.Write(data)
	return mac.Sum(nil)
}

func toUserResponse(user *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Gender:    user.Gender,
		AvatarURL: user.AvatarURL,
		Status:    user.Status,
	}
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", fmt.Errorf("邮箱不能为空")
	}
	if len(email) > 254 {
		return "", fmt.Errorf("邮箱长度不能超过254个字符")
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", fmt.Errorf("邮箱格式错误")
	}

	return email, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码至少需要8个字符")
	}
	if len(password) > 72 {
		return fmt.Errorf("密码不能超过72个字符")
	}
	return nil
}

func normalizeNickname(nickname string, email string) (string, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		localPart, _, found := strings.Cut(email, "@")
		if found && localPart != "" {
			return localPart, nil
		}
		return "FitMind用户", nil
	}

	if len([]rune(nickname)) > 40 {
		return "", fmt.Errorf("昵称不能超过40个字符")
	}

	return nickname, nil
}

func newServiceError(kind ErrorKind, message string, err error) *ServiceError {
	return &ServiceError{
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}
