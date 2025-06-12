package common

import "time"

//Brokers
type Broker string

const (
	Okx     Broker = "okx"
	Binance Broker = "binance"
)

//Roles
type Role string

const (
	SuperAdmin Role = "super_admin"
	Admin      Role = "admin"
	Developer  Role = "developer"
	User       Role = "user"
)

//User Status
type UserStatus string

const (
	Active      UserStatus = "active"
	Deactivated UserStatus = "deactivated"
)

//Subscriptions
type SubscriptionType string

const (
	Basic        SubscriptionType = "basic"
	Professional SubscriptionType = "professional"
	Ultimate     SubscriptionType = "ultimate"
)

//OTP Types

type OTPType string

const (
	RegistrationOTP  OTPType = "registration_otp"
	ResetPasswordOTP OTPType = "reset_password_otp"
)

//OTP Expired Time
const (
	OTPExpiredTime = 10 * time.Minute
)

//Email Types

type EmailType string

const (
	SimulationResult EmailType = "SimulationResult"
	OnboardingGQA    EmailType = "OnboardingGQA"
)
