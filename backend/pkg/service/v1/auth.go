package v1

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/DagDigg/unpaper/backend/customers"
	"github.com/DagDigg/unpaper/backend/follows"
	"github.com/DagDigg/unpaper/backend/helpers"
	dbNotifications "github.com/DagDigg/unpaper/backend/notifications"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/claims"
	"github.com/DagDigg/unpaper/backend/pkg/logger"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/DagDigg/unpaper/backend/pkg/notifications"
	"github.com/DagDigg/unpaper/backend/pkg/stripeservice"
	"github.com/DagDigg/unpaper/backend/users"
	"github.com/DagDigg/unpaper/core/cookies"
	"github.com/DagDigg/unpaper/core/session"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GoogleLogin replies to the request with consent screen URL and generated state
func (s *unpaperServiceServer) GoogleLogin(ctx context.Context, req *v1API.GoogleLoginRequest) (*v1API.GoogleLoginResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "failed to get metadata from request")
	}
	consentURL, ok := mdutils.GetFirstMDValue(md, "ConsentURL")
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "missing consent in metadata")
	}
	state, ok := mdutils.GetFirstMDValue(md, "State")
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "missing state in metadata")
	}
	codeVerifier, ok := mdutils.GetFirstMDValue(md, "Code-Verifier")
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "missing code verifier in metadata")
	}
	// Create cookie code-verifier with a lifetime of 5 minutes
	cookiesMngr := &cookies.Manager{
		Domain: s.cfg.ClientDomain,
	}
	codeVerifierCookie, err := cookiesMngr.GetValue("x-code-verifier", codeVerifier, 5*time.Minute)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create x-code-verifier cookie: %v", err)
	}

	// Send code verifier as httpOnly cookie
	grpc.SendHeader(ctx, metadata.Pairs("Set-Cookie", codeVerifierCookie))
	_ = &session.User{ID: "ok"}
	return &v1API.GoogleLoginResponse{
		Api:        apiVersion,
		ConsentURL: consentURL,
		State:      state,
	}, nil
}

// GoogleCallback handles the /callback request for signing-up/in an user
func (s *unpaperServiceServer) GoogleCallback(ctx context.Context, req *v1API.GoogleCallbackRequest) (*v1API.User, error) {
	return s.googleAuthentication(ctx)
}

// GoogleOneTap handles the one-tap google authentication
func (s *unpaperServiceServer) GoogleOneTap(ctx context.Context, req *empty.Empty) (*v1API.User, error) {
	return s.googleAuthentication(ctx)
}

func (s *unpaperServiceServer) googleAuthentication(ctx context.Context) (*v1API.User, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing metadata in context")
	}
	// An empty name "" is accepted
	name, _ := mdutils.GetFirstMDValue(md, "x-user-name")
	userID, ok := mdutils.GetFirstMDValue(md, "x-user-id")
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing userID in metadata")
	}
	email, ok := mdutils.GetFirstMDValue(md, "x-user-email")
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing email in metadata")
	}

	// Initialize customers dir
	customersDir := customers.NewDirectory(s.db)

	// Since GoogleCallback is called either for signup and signin,
	// we need to check if a customer has already been created (signin).
	// If so, a new customer with subscription should not be created
	isSignin := true
	_, err := customersDir.GetCustomerByUserID(ctx, userID)
	if err != nil {
		if err != customers.ErrNoCustomer {
			return nil, status.Errorf(codes.Internal, "failure occurred retrieving customer: %q", err)
		}
		// Error is customers.ErrNoCustomer,
		// since no customer can be found, the user is signing up
		isSignin = false
	}

	user := &v1API.User{}

	if isSignin {
		usersDir := users.NewDirectory(s.db)
		userRetrieved, err := usersDir.GetUser(ctx, userID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve user: %v", err)
		}

		user = userRetrieved
	}

	if !isSignin {
		// Get user since ext auth already have created it
		userCreated, err := s.createUser(ctx, createUserParams{
			email:     email,
			givenName: name,
			userID:    userID,
			isLocal:   false,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not get user: %q", err)
		}

		// Create stripe customer and subscription if feature flag is enabled, and if user is signing up
		if s.cfg.EnableStripe {
			// User is signing up.
			// Create stripe customer and store it on db
			stripesvc := stripeservice.New(ctx, s.db, s.cfg)
			stripeCus, err := stripesvc.CreateCustomer(&stripeservice.CreateCustomerParams{
				UserID:     userID,
				Email:      userCreated.Email,
				FamilyName: userCreated.FamilyName,
				GivenName:  userCreated.GivenName,
			})
			if err != nil {
				return nil, err
			}
			pbCus, err := stripesvc.StoreCustomer(stripeCus, stripeservice.StoreCustomerParams{
				UserID:     userID,
				FamilyName: userCreated.FamilyName,
				GivenName:  userCreated.GivenName,
			})
			if err != nil {
				return nil, err
			}

			// Create free stripe subscription, attach it to customer
			// and store it on db
			stripeSub, err := stripesvc.CreatePlatformSubscription(pbCus.CustomerId, userID, s.cfg.PriceIDFree)
			if err != nil {
				return nil, err
			}
			_, err = stripesvc.StoreSubscription(stripeSub, userID)
			if err != nil {
				return nil, err
			}
		}

		user = userCreated
	}

	// Create user session
	cookiesMngr := &cookies.Manager{
		Domain: s.cfg.ClientDomain,
	}
	sid, err := s.sm.SetNew(ctx, &session.User{ID: userID}, session.Lifetime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating session: %v", err)
	}
	sidCookieStr, err := cookiesMngr.GetValue(session.CookieName, sid, session.Lifetime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating session cookie: %v", err)
	}

	if err := grpc.SendHeader(ctx, metadata.Pairs("Set-Cookie", sidCookieStr)); err != nil {
		return nil, status.Errorf(codes.Internal, "error sending grpc header: %v", err)
	}
	return user, nil
}

// EmailSignup register the user on the platform, creates a stripe customer and free subscription
func (s *unpaperServiceServer) EmailSignup(ctx context.Context, req *v1API.EmailSignupRequest) (*v1API.User, error) {
	user, err := s.createUser(ctx, createUserParams{
		userID:   uuid.NewString(),
		email:    req.Email,
		username: req.Username,
		password: req.Password,
		isLocal:  true,
	})
	if err != nil {
		return nil, err
	}
	if s.cfg.EnableStripe {
		// Create customer only if flag is enabled
		_, err = s.createCustomer(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	verificationJWT, err := createVerificationJWT(user.Id, s.cfg.UnpaperClientSecret)
	if err != nil {
		return nil, err
	}

	fmt.Println(verificationJWT)
	if s.cfg.Environment == "prod" {
		// Send emails only on prod environment
		msg := []byte("To: " + "dontcare@gmail.com" + "\r\n" +
			"Subject: Verify your email\r\n" +
			"\r\n" +
			"Click here to verify your email: https://localhost:3000/verify?code=" + verificationJWT)

		auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPDomain)
		err := smtp.SendMail(s.cfg.SMTPDomain+":"+s.cfg.SMTPPort, auth, "unpaper@me.com", []string{"dontcare@gmail.com"}, msg)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not send email: '%v'", err)
		}
	}

	// Create new session for the user
	cookiesMngr := &cookies.Manager{
		Domain: s.cfg.ClientDomain,
	}
	sid, err := s.sm.SetNew(ctx, &session.User{ID: user.Id}, session.Lifetime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating session: %v", err)
	}

	cookieSIDString, err := cookiesMngr.GetValue(session.CookieName, sid, session.Lifetime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating session cookie: %v", err)
	}

	if err = grpc.SendHeader(ctx, metadata.Pairs("Set-Cookie", cookieSIDString)); err != nil {
		return nil, status.Errorf(codes.Internal, "error sending grpc header: %v", err)
	}
	return user, nil
}

// createUser validates incoming signup request, and on success insert a user into db
type createUserParams struct {
	userID    string
	email     string
	username  string
	givenName string
	password  string
	isLocal   bool
}

func (s *unpaperServiceServer) createUser(ctx context.Context, params createUserParams) (*v1API.User, error) {
	// Init users directory
	usersDir := users.NewDirectory(s.db)
	var hashedPassword string

	// Users registered with IDP do not need some params
	if params.isLocal {
		if len(params.password) < 8 {
			// TODO: better password validation
			return nil, status.Errorf(codes.InvalidArgument, "invalid password")
		}
		if len(params.username) < 4 {
			// TODO: better username validation
			return nil, status.Errorf(codes.InvalidArgument, "invalid username")
		}
		// Check if username is available
		ok, err := usersDir.UsernameExists(ctx, params.username)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error checking username existence: %v", err)
		}
		if ok {
			return nil, status.Errorf(codes.Internal, "username already exists")
		}
		// Salt and hash the password using the bcrypt algorithm
		// The second argument is the cost of hashing, which is arbitrarily set as 8 (this value can be more or less, depending on the computing power)
		p, err := bcrypt.GenerateFromPassword([]byte(params.password), 8)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error hashing password: %v", err)
		}
		hashedPassword = string(p)
	}

	// Check email validity
	ok, err := helpers.IsEmailValid(params.email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error verifying email validity: %v", err)
	}
	if !ok {
		return nil, status.Error(codes.Internal, "invalid email")
	}

	// Check if email is available
	ok, err = usersDir.EmailExists(ctx, params.email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error checking email existence: %v", err)
	}
	if ok {
		return nil, status.Errorf(codes.Internal, "email already exists")
	}

	// Get user since ext auth already have created it
	user, err := usersDir.CreateUser(ctx, users.CreateUserParams{
		ID:         params.userID,
		GivenName:  helpers.NewNullString(params.givenName),
		FamilyName: sql.NullString{},
		Email:      params.email,
		Password:   helpers.NewNullString(hashedPassword),
		Username:   helpers.NewNullString(params.username),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create user: %q", err)
	}

	return user, nil
}

// createCustomer creates a new Stripe customer, a new free platform subscription
// and stores them on database
func (s *unpaperServiceServer) createCustomer(ctx context.Context, user *v1API.User) (*v1API.Customer, error) {
	// Create a customer on stripe, and store it on database
	stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	stripeCus, err := stripesvc.CreateCustomer(&stripeservice.CreateCustomerParams{
		UserID:     user.Id,
		Email:      user.Email,
		FamilyName: user.FamilyName,
		GivenName:  user.GivenName,
	})
	if err != nil {
		return nil, err
	}
	pbCus, err := stripesvc.StoreCustomer(stripeCus, stripeservice.StoreCustomerParams{
		UserID:     user.Id,
		FamilyName: user.FamilyName,
		GivenName:  user.GivenName,
	})
	if err != nil {
		return nil, err
	}

	// Create a subscription on stripe attached to customer, and store it on database
	stripeSub, err := stripesvc.CreatePlatformSubscription(pbCus.CustomerId, user.Id, s.cfg.PriceIDFree)
	if err != nil {
		return nil, err
	}
	_, err = stripesvc.StoreSubscription(stripeSub, user.Id)
	if err != nil {
		return nil, err
	}

	return pbCus, nil
}

// createVerificationJWT creates an email-verification JWT
func createVerificationJWT(userID, unpaperClientSecret string) (string, error) {
	verificationJWT, err := claims.CreateJWTTokenStr(jwt.StandardClaims{
		ExpiresAt: time.Now().Add(claims.VerificationTknExpiry).Unix(),
		Subject:   userID,
		Issuer:    "unpaper",
	}, unpaperClientSecret)
	if err != nil {
		return "", status.Errorf(codes.Internal, "error creating verification token: '%v'", err)
	}

	return verificationJWT, nil
}

// EmailSignin checks if the user has enter a valid email/password. If so, a new pair of access/refresh token is generated.
// Access token is set as cookie, while the refresh token is upserted on database
func (s *unpaperServiceServer) EmailSignin(ctx context.Context, req *v1API.EmailSigninRequest) (*v1API.User, error) {
	// Check for bad request
	if req.Email == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing email or password")
	}

	usersDir := users.NewDirectory(s.db)

	userID, err := usersDir.GetUserIDFromEmail(ctx, req.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "email %q is not associated to any user", req.Email)
	}
	storedPassword, err := usersDir.GetPassword(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failure occurred during user retrieval")
	}

	// Check the correctness of the password
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Internal, "wrong password inserted")
	}

	// Create a session
	cookiesMngr := &cookies.Manager{
		Domain: s.cfg.ClientDomain,
	}
	sid, err := s.sm.SetNew(ctx, &session.User{ID: userID}, session.Lifetime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}
	sidCookieStr, err := cookiesMngr.GetValue(session.CookieName, sid, session.Lifetime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session cookie: %v", err)
	}

	if err := grpc.SendHeader(ctx, metadata.Pairs("Set-Cookie", sidCookieStr)); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send grpc header: %v", err)
	}

	// Retrieve user to send back
	user, err := usersDir.GetUser(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user from db: %v", err)
	}

	return user, nil
}

func (s *unpaperServiceServer) SignOut(ctx context.Context, req *empty.Empty) (*empty.Empty, error) {
	// ExtAuth already have deleted session and cookie
	return new(empty.Empty), nil
}

// Ping handler
func (s *unpaperServiceServer) Ping(ctx context.Context, req *v1API.PingRequest) (*v1API.User, error) {
	return &v1API.User{
		Id:    "foo",
		Email: "bar",
	}, nil
}

// EmailVerify checks the incoming verification token, and on success updates the `email_verified` db column
func (s *unpaperServiceServer) EmailVerify(ctx context.Context, req *v1API.EmailVerifyRequest) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	// Parse and verify the verification token in claims.
	// If the token has an invalid signature, or expired, or does not
	// belong to the user making the request, an error is returned
	claims := struct {
		jwt.StandardClaims
	}{}
	_, err := jwt.ParseWithClaims(req.VerificationToken, &claims, func(token *jwt.Token) (interface{}, error) {
		// Validate alg
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.UnpaperClientSecret), nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "an error occurred verifying the verification token: '%v'", err)
	}

	if claims.Subject != userID {
		// User submitting the request is not the same as
		// the one encoded in the token. Someone else could have
		// clicked the email verification link
		return nil, status.Error(codes.Internal, "verification code do not belong to this user")
	}

	// Token is valid and belongs to the user.
	// Update email_verified column on db
	usersDir := users.NewDirectory(s.db)
	if err := usersDir.VerifyEmail(ctx, userID); err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "error verifying email on db: %v", err)
	}

	return new(empty.Empty), nil
}

func (s *unpaperServiceServer) EmailCheck(ctx context.Context, req *v1API.EmailCheckRequest) (*empty.Empty, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "missing 'email' in request")
	}
	ok, err := helpers.IsEmailValid(req.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to exec regexp on the provided email: %q", err)
	}
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "the email provided is invalid: '%v'", req.Email)
	}
	return nil, nil
}

// ChangePassword updates the user password on db, updating the password_changed_at timestamp
// to avoid any subsequent token usage with other clients
func (s *unpaperServiceServer) ChangePassword(ctx context.Context, req *v1API.ChangePasswordRequest) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.NewPassword != req.Repeat {
		return nil, status.Errorf(codes.FailedPrecondition, "Passwords should match")
	}
	if len(req.NewPassword) < 8 {
		return nil, status.Errorf(codes.FailedPrecondition, "Password should have at least 8 characters")
	}

	usersDir := users.NewDirectory(s.db)

	// User must have verified its email in order to change password
	emailVerified, err := usersDir.GetEmailVerified(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "An error occurred retrieving email_verified: %v", err)
	}
	if !emailVerified {
		return nil, status.Error(codes.FailedPrecondition, "Your email should be verified in order to change password")
	}

	// Retrieve user password
	storedPsw, err := usersDir.GetPassword(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get user password: %v", err)
	}

	// Check the correctness of the password
	err = bcrypt.CompareHashAndPassword([]byte(storedPsw), []byte(req.OldPassword))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "The password you've entered is wrong: %v", err)
	}

	// Password is valid
	// Salt and hash the password using the bcrypt algorithm
	// The second argument is the cost of hashing, which is arbitrarily set as 8 (this value can be more or less, depending on the computing power)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 8)
	if err != nil {
		return nil, err
	}

	err = usersDir.UpdatePassword(ctx, userID, string(hashedPassword))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error storing new password on db: %v", err)
	}

	// Delete cookie, since the password_changed_at has been set as now
	// and thus the token has been issued at a time before that
	grpc.SendHeader(ctx, metadata.Pairs("Set-Cookie", "x-unpaper-token=deleted; Domain=localhost; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Secure; HttpOnly"))
	return new(empty.Empty), nil
}

func (s *unpaperServiceServer) SendResetLink(ctx context.Context, req *v1API.SendResetLinkRequest) (*empty.Empty, error) {
	// Do not return any error during this RPC. If the email exists, it will be sent,
	// otherwise, the service will fail silently, always succeeding
	valid, err := isEmailValid(req.Email)
	if err != nil {
		return new(empty.Empty), nil
	}
	if !valid {
		return new(empty.Empty), nil
	}

	usersDir := users.NewDirectory(s.db)

	// Check if email exists
	exists, err := usersDir.EmailExists(ctx, req.Email)
	if err != nil {
		return new(empty.Empty), nil
	}
	if !exists {
		// Return generic error message if email doesn't exists
		return new(empty.Empty), nil
	}

	// Since it's an unauthenticated request, we only
	// have the user email, which we need to use to get the userID
	userID, err := usersDir.GetUserIDFromEmail(ctx, req.Email)
	if err != nil {
		return new(empty.Empty), nil
	}

	// Create JWT for letting the future service know
	// if the user has clicked the email
	// Create verification token
	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Subject:   userID,
		Issuer:    "unpaper",
		ExpiresAt: time.Now().Add(60 * time.Minute).Unix(), // 1 hour lifespan
	})
	verificationToken, err := unsignedToken.SignedString([]byte(s.cfg.UnpaperClientSecret))
	if err != nil {
		return new(empty.Empty), nil
	}

	// TODO: send email
	// TODO: set state httpOnly secure cookie.
	// Otherwise the JWT can be history-hijacked since
	// it will be displayed as url search string

	fmt.Println("VERIFICATION TOKEN: ", verificationToken)
	return new(empty.Empty), nil
}

// TokenError represent a token parsing and or validation error
var TokenError = status.Error(codes.Internal, "Oops, an error occurred")

// ResetPassword validates the verification token, and performs database update for the password column
func (s *unpaperServiceServer) ResetPassword(ctx context.Context, req *v1API.ResetPasswordRequest) (*empty.Empty, error) {
	if len(req.NewPassword) < 8 {
		return new(empty.Empty), status.Error(codes.FailedPrecondition, "Password should be at least 8 characters long")
	}
	if req.NewPassword != req.Repeat {
		return new(empty.Empty), status.Error(codes.FailedPrecondition, "Passwords should match")
	}

	type claims struct {
		jwt.StandardClaims
	}
	c := &claims{}
	t, err := jwt.ParseWithClaims(req.VerificationToken, c, func(tok *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.UnpaperClientSecret), nil
	})
	if err != nil {
		return new(empty.Empty), TokenError
	}
	if !t.Valid {
		return new(empty.Empty), TokenError
	}

	// Token is valid, salt and hash new password
	hashedPsw, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 8)
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	usersDir := users.NewDirectory(s.db)

	// Perform password update
	err = usersDir.UpdatePassword(ctx, c.Subject, string(hashedPsw))
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to update password: %v", err)
	}

	return new(empty.Empty), nil
}

// ExtUserInfo returns external user info
func (s *unpaperServiceServer) ExtUserInfo(ctx context.Context, req *v1API.ExtUserInfoRequest) (*v1API.ExtUserInfoResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.UserId == "" && req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "missing target user_id or username")
	}

	return fetchExtUserInfo(fetchExtUserInfoParams{
		ctx:         ctx,
		db:          s.db,
		userID:      userID,
		extUserID:   req.UserId,
		extUsername: req.Username,
	})
}

type fetchExtUserInfoParams struct {
	ctx         context.Context
	db          *sql.DB
	userID      string
	extUserID   string
	extUsername string
}

func fetchExtUserInfo(params fetchExtUserInfoParams) (*v1API.ExtUserInfoResponse, error) {
	usersDir := users.NewDirectory(params.db)
	followsDir := follows.NewDirectory(params.db)

	u, err := getExtUser(params.ctx, usersDir, params.extUserID, params.extUsername)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve user: %v", err)
	}
	isFollowingUser, err := followsDir.IsFollowingUser(params.ctx, follows.IsFollowingUserParams{
		FollowerUserID:  params.userID,
		FollowingUserID: params.extUserID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failure retrieving is following user: %v", err)
	}

	return &v1API.ExtUserInfoResponse{
		UserInfo: &v1API.ExtUserInfo{
			Id:         u.Id,
			Email:      u.Email,
			FamilyName: u.FamilyName,
			GivenName:  u.GivenName,
			Username:   u.Username,
			IsFollowed: isFollowingUser,
		},
	}, nil
}

func getExtUser(ctx context.Context, dir *users.Directory, userID, username string) (*v1API.User, error) {
	if userID != "" {
		return dir.GetUser(ctx, userID)
	}

	return dir.GetUserByUsername(ctx, username)
}

type addCustomerInfoParams struct {
	ctx         context.Context
	db          *sql.DB
	extUserInfo *v1API.ExtUserInfo
}

func addCustomerInfo(params addCustomerInfoParams) (*v1API.ExtUserCustomerInfoResponse, error) {
	customersDir := customers.NewDirectory(params.db)
	c, err := customersDir.GetCustomerByUserID(params.ctx, params.extUserInfo.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve customer: %v", err)
	}
	hasConnectedAccount, err := customersDir.ConnectedAccountExists(params.ctx, c.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve connected account: %v", err)
	}

	return &v1API.ExtUserCustomerInfoResponse{
		UserInfo: params.extUserInfo,
		CustomerInfo: &v1API.CustomerInfo{
			AccountId:           c.AccountId,
			HasConnectedAccount: hasConnectedAccount,
		},
	}, nil
}

// UpdateUsername updates the user's usarname db column
func (s *unpaperServiceServer) UpdateUsername(ctx context.Context, req *v1API.UpdateUsernameRequest) (*v1API.User, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username cannot be empty")
	}
	if len(req.Username) > 24 {
		return nil, status.Error(codes.InvalidArgument, "Username cannot be more than 24 characters long")
	}

	usersDir := users.NewDirectory(s.db)

	user, err := usersDir.UpdateUsername(ctx, &users.UpdateUsernameParams{
		ID:       userID,
		Username: sql.NullString{String: req.Username, Valid: true},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to update username: %v", err)
	}

	return user, nil
}

// SetUserOnline sets the user in the active users underlying rdb
func (s *unpaperServiceServer) SetUserOnline(ctx context.Context, req *empty.Empty) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return &empty.Empty{}, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	return &empty.Empty{}, s.usersession.Log(ctx, userID)
}

// SetUserOffline sets the user in the active users underlying rdb
func (s *unpaperServiceServer) SetUserOffline(ctx context.Context, req *empty.Empty) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return &empty.Empty{}, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	return &empty.Empty{}, s.usersession.Unlog(ctx, userID)
}

// FollowUser performs db actions to follow an user and returns an updated external user info
func (s *unpaperServiceServer) FollowUser(ctx context.Context, req *v1API.FollowUserRequest) (*v1API.ExtUserInfoResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	followsDir := follows.NewDirectory(s.db)
	usersDir := users.NewDirectory(s.db)

	senderUsr, err := usersDir.GetUser(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve is sender user: %v", err)
	}

	isFollowing, err := followsDir.IsFollowingUser(ctx, follows.IsFollowingUserParams{
		FollowerUserID:  userID,
		FollowingUserID: req.UserIdToFollow,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve is following user: %v", err)
	}

	res, err := toggleFollowUser(toggleFollowUserParams{
		ctx:            ctx,
		dir:            followsDir,
		isFollowing:    isFollowing,
		userID:         userID,
		userIDToFollow: req.UserIdToFollow,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failure occurred while following user: %v", err)
	}

	if res.IsFollowed {
		_, err = s.nm.Send(notifications.SendNotificationParams{
			Ctx:             ctx,
			SenderUserID:    userID,
			ReceiverUserID:  req.UserIdToFollow,
			TriggerID:       senderUsr.Username,
			EventID:         string(dbNotifications.EventIDFollow),
			ResendCondition: notifications.ResendConditionAfter(5 * time.Minute),
		})
		if err != nil {
			// Do not throw error on notification send failure
			logger.Log.Error(err.Error())
		}
	}

	return &v1API.ExtUserInfoResponse{
		UserInfo: res,
	}, nil
}

type toggleFollowUserParams struct {
	ctx            context.Context
	isFollowing    bool
	dir            *follows.Directory
	userID         string
	userIDToFollow string
}

func toggleFollowUser(params toggleFollowUserParams) (*v1API.ExtUserInfo, error) {
	if params.isFollowing {
		return params.dir.UnfollowUser(params.ctx, follows.UnfollowUserParams{
			FollowerUserID:  params.userID,
			FollowingUserID: params.userIDToFollow,
			UnfollowDate:    sql.NullTime{Time: time.Now(), Valid: true},
		})
	}

	return params.dir.FollowUser(params.ctx, follows.FollowUserParams{
		FollowerUserID:  params.userID,
		FollowingUserID: params.userIDToFollow,
		FollowDate:      time.Now(),
	})
}

func getFollowTargetUserID(ownUserID, extUserID string) string {
	if extUserID != "" {
		return extUserID
	}

	return ownUserID
}

// GetFollowers returns a list of external users info who follows the user
func (s *unpaperServiceServer) GetFollowers(ctx context.Context, req *v1API.GetFollowersRequest) (*v1API.GetFollowersResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	followsDir := follows.NewDirectory(s.db)

	res, err := followsDir.GetFollowers(ctx, getFollowTargetUserID(userID, req.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve followers: %v", err)
	}

	return &v1API.GetFollowersResponse{
		Users: res,
	}, nil
}

// GetFollowing returns a list of external users info who the user is following
func (s *unpaperServiceServer) GetFollowing(ctx context.Context, req *v1API.GetFollowingRequest) (*v1API.GetFollowingResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	followsDir := follows.NewDirectory(s.db)

	res, err := followsDir.GetFollowing(ctx, getFollowTargetUserID(userID, req.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve followers: %v", err)
	}

	return &v1API.GetFollowingResponse{
		Users: res,
	}, nil
}

// GetFollowersCount returns the number of external users info who follows the user
func (s *unpaperServiceServer) GetFollowersCount(ctx context.Context, req *v1API.GetFollowersCountRequest) (*v1API.GetFollowersCountResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	followsDir := follows.NewDirectory(s.db)

	res, err := followsDir.GetFollowersCount(ctx, getFollowTargetUserID(userID, req.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve followers: %v", err)
	}

	return &v1API.GetFollowersCountResponse{
		FollowersCount: res,
	}, nil
}

// GetFollowingCount returns the number of external users info who follows the user
func (s *unpaperServiceServer) GetFollowingCount(ctx context.Context, req *v1API.GetFollowingCountRequest) (*v1API.GetFollowingCountResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	followsDir := follows.NewDirectory(s.db)

	res, err := followsDir.GetFollowingCount(ctx, getFollowTargetUserID(userID, req.UserId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve followers: %v", err)
	}

	return &v1API.GetFollowingCountResponse{
		FollowingCount: res,
	}, nil
}

// IsEmailValid checks if the email provided passes the required structure
// and length test. It also checks the domain has a valid MX record.
func isEmailValid(e string) (bool, error) {
	emailRegex, err := regexp.Compile(
		"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
	)
	if err != nil {
		return false, err
	}

	if len(e) < 3 && len(e) > 254 {
		return false, nil
	}
	if !emailRegex.MatchString(e) {
		return false, nil
	}
	parts := strings.Split(e, "@")
	mx, err := net.LookupMX(parts[1])
	if err != nil || len(mx) == 0 {
		return false, err
	}

	return true, nil
}
